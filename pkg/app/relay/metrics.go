package relay

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"orly.dev/pkg/database"
	"orly.dev/pkg/utils/log"
)

// MetricsCollector tracks subscription system metrics
type MetricsCollector struct {
	mu sync.RWMutex
	db *database.D

	// Subscription metrics
	totalTrialSubscriptions int64
	totalPaidSubscriptions  int64

	// Payment metrics
	paymentSuccessCount int64
	paymentFailureCount int64

	// Conversion metrics
	trialToPaidConversions int64
	totalTrialsStarted     int64

	// Duration metrics
	subscriptionDurations []time.Duration
	maxDurationSamples    int

	// Health status
	lastHealthCheck   time.Time
	isHealthy         bool
	healthCheckErrors []string
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(db *database.D) *MetricsCollector {
	return &MetricsCollector{
		db:                 db,
		maxDurationSamples: 1000,
		isHealthy:          true,
		lastHealthCheck:    time.Now(),
	}
}

// RecordTrialStarted increments trial subscription counter
func (mc *MetricsCollector) RecordTrialStarted() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.totalTrialsStarted++
	mc.totalTrialSubscriptions++
}

// RecordPaidSubscription increments paid subscription counter
func (mc *MetricsCollector) RecordPaidSubscription() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.totalPaidSubscriptions++
}

// RecordTrialExpired decrements trial subscription counter
func (mc *MetricsCollector) RecordTrialExpired() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if mc.totalTrialSubscriptions > 0 {
		mc.totalTrialSubscriptions--
	}
}

// RecordPaidExpired decrements paid subscription counter
func (mc *MetricsCollector) RecordPaidExpired() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if mc.totalPaidSubscriptions > 0 {
		mc.totalPaidSubscriptions--
	}
}

// RecordPaymentSuccess increments successful payment counter
func (mc *MetricsCollector) RecordPaymentSuccess() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.paymentSuccessCount++
}

// RecordPaymentFailure increments failed payment counter
func (mc *MetricsCollector) RecordPaymentFailure() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.paymentFailureCount++
}

// RecordTrialToPaidConversion records when a trial user becomes paid
func (mc *MetricsCollector) RecordTrialToPaidConversion() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.trialToPaidConversions++
	// Move from trial to paid
	if mc.totalTrialSubscriptions > 0 {
		mc.totalTrialSubscriptions--
	}
	mc.totalPaidSubscriptions++
}

// RecordSubscriptionDuration adds a subscription duration sample
func (mc *MetricsCollector) RecordSubscriptionDuration(duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Keep only the most recent samples to prevent memory growth
	mc.subscriptionDurations = append(mc.subscriptionDurations, duration)
	if len(mc.subscriptionDurations) > mc.maxDurationSamples {
		mc.subscriptionDurations = mc.subscriptionDurations[1:]
	}
}

// GetMetrics returns current metrics snapshot
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	totalPayments := mc.paymentSuccessCount + mc.paymentFailureCount
	var paymentSuccessRate float64
	if totalPayments > 0 {
		paymentSuccessRate = float64(mc.paymentSuccessCount) / float64(totalPayments)
	}

	var conversionRate float64
	if mc.totalTrialsStarted > 0 {
		conversionRate = float64(mc.trialToPaidConversions) / float64(mc.totalTrialsStarted)
	}

	var avgDuration time.Duration
	if len(mc.subscriptionDurations) > 0 {
		var total time.Duration
		for _, d := range mc.subscriptionDurations {
			total += d
		}
		avgDuration = total / time.Duration(len(mc.subscriptionDurations))
	}

	return map[string]interface{}{
		"total_trial_subscriptions":             mc.totalTrialSubscriptions,
		"total_paid_subscriptions":              mc.totalPaidSubscriptions,
		"total_active_subscriptions":            mc.totalTrialSubscriptions + mc.totalPaidSubscriptions,
		"payment_success_count":                 mc.paymentSuccessCount,
		"payment_failure_count":                 mc.paymentFailureCount,
		"payment_success_rate":                  paymentSuccessRate,
		"trial_to_paid_conversions":             mc.trialToPaidConversions,
		"total_trials_started":                  mc.totalTrialsStarted,
		"conversion_rate":                       conversionRate,
		"average_subscription_duration_seconds": avgDuration.Seconds(),
		"last_health_check":                     mc.lastHealthCheck.Unix(),
		"is_healthy":                            mc.isHealthy,
	}
}

// GetPrometheusMetrics returns metrics in Prometheus format
func (mc *MetricsCollector) GetPrometheusMetrics() string {
	metrics := mc.GetMetrics()

	promMetrics := `# HELP orly_trial_subscriptions_total Total number of active trial subscriptions
# TYPE orly_trial_subscriptions_total gauge
orly_trial_subscriptions_total %d

# HELP orly_paid_subscriptions_total Total number of active paid subscriptions
# TYPE orly_paid_subscriptions_total gauge
orly_paid_subscriptions_total %d

# HELP orly_active_subscriptions_total Total number of active subscriptions (trial + paid)
# TYPE orly_active_subscriptions_total gauge
orly_active_subscriptions_total %d

# HELP orly_payment_success_total Total number of successful payments
# TYPE orly_payment_success_total counter
orly_payment_success_total %d

# HELP orly_payment_failure_total Total number of failed payments
# TYPE orly_payment_failure_total counter
orly_payment_failure_total %d

# HELP orly_payment_success_rate Payment success rate (0.0 to 1.0)
# TYPE orly_payment_success_rate gauge
orly_payment_success_rate %.6f

# HELP orly_trial_to_paid_conversions_total Total number of trial to paid conversions
# TYPE orly_trial_to_paid_conversions_total counter
orly_trial_to_paid_conversions_total %d

# HELP orly_trials_started_total Total number of trials started
# TYPE orly_trials_started_total counter
orly_trials_started_total %d

# HELP orly_conversion_rate Trial to paid conversion rate (0.0 to 1.0)
# TYPE orly_conversion_rate gauge
orly_conversion_rate %.6f

# HELP orly_avg_subscription_duration_seconds Average subscription duration in seconds
# TYPE orly_avg_subscription_duration_seconds gauge
orly_avg_subscription_duration_seconds %.2f

# HELP orly_last_health_check_timestamp Last health check timestamp
# TYPE orly_last_health_check_timestamp gauge
orly_last_health_check_timestamp %d

# HELP orly_health_status Health status (1 = healthy, 0 = unhealthy)
# TYPE orly_health_status gauge
orly_health_status %d
`

	healthStatus := 0
	if metrics["is_healthy"].(bool) {
		healthStatus = 1
	}

	return fmt.Sprintf(promMetrics,
		metrics["total_trial_subscriptions"],
		metrics["total_paid_subscriptions"],
		metrics["total_active_subscriptions"],
		metrics["payment_success_count"],
		metrics["payment_failure_count"],
		metrics["payment_success_rate"],
		metrics["trial_to_paid_conversions"],
		metrics["total_trials_started"],
		metrics["conversion_rate"],
		metrics["average_subscription_duration_seconds"],
		metrics["last_health_check"],
		healthStatus,
	)
}

// PerformHealthCheck checks system health
func (mc *MetricsCollector) PerformHealthCheck() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.lastHealthCheck = time.Now()
	mc.healthCheckErrors = []string{}
	mc.isHealthy = true

	if mc.db != nil {
		testPubkey := make([]byte, 32)
		_, err := mc.db.GetSubscription(testPubkey)
		if err != nil {
			mc.isHealthy = false
			mc.healthCheckErrors = append(mc.healthCheckErrors, fmt.Sprintf("database error: %v", err))
		}
	} else {
		mc.isHealthy = false
		mc.healthCheckErrors = append(mc.healthCheckErrors, "database not initialized")
	}

	if mc.isHealthy {
		log.D.Ln("health check passed")
	} else {
		log.W.F("health check failed: %v", mc.healthCheckErrors)
	}
}

// GetHealthStatus returns current health status
func (mc *MetricsCollector) GetHealthStatus() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return map[string]interface{}{
		"healthy":        mc.isHealthy,
		"last_check":     mc.lastHealthCheck.Format(time.RFC3339),
		"errors":         mc.healthCheckErrors,
		"uptime_seconds": time.Since(mc.lastHealthCheck).Seconds(),
	}
}

// StartPeriodicHealthChecks runs health checks periodically
func (mc *MetricsCollector) StartPeriodicHealthChecks(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Perform initial health check
	mc.PerformHealthCheck()

	for {
		select {
		case <-ticker.C:
			mc.PerformHealthCheck()
		case <-stopCh:
			log.D.Ln("stopping periodic health checks")
			return
		}
	}
}

// MetricsHandler handles HTTP requests for metrics endpoint
func (mc *MetricsCollector) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	metrics := mc.GetPrometheusMetrics()
	w.Write([]byte(metrics))
}

// HealthHandler handles HTTP requests for health check endpoint
func (mc *MetricsCollector) HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Perform real-time health check
	mc.PerformHealthCheck()

	status := mc.GetHealthStatus()

	w.Header().Set("Content-Type", "application/json")

	if status["healthy"].(bool) {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Simple JSON formatting without external dependencies
	healthy := "true"
	if !status["healthy"].(bool) {
		healthy = "false"
	}

	errorsJson := "[]"
	if errors, ok := status["errors"].([]string); ok && len(errors) > 0 {
		errorsJson = `["`
		for i, err := range errors {
			if i > 0 {
				errorsJson += `", "`
			}
			errorsJson += err
		}
		errorsJson += `"]`
	}

	response := fmt.Sprintf(`{
  "healthy": %s,
  "last_check": "%s",
  "errors": %s,
  "uptime_seconds": %.2f
}`, healthy, status["last_check"], errorsJson, status["uptime_seconds"])

	w.Write([]byte(response))
}
