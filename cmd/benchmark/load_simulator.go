package main

import (
	"fmt"
	"math"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"sync"
	"sync/atomic"
	"time"
)

type LoadPattern int

const (
	Constant LoadPattern = iota
	Spike
	Burst
	Sine
	Ramp
)

func (lp LoadPattern) String() string {
	switch lp {
	case Constant:
		return "constant"
	case Spike:
		return "spike"
	case Burst:
		return "burst"
	case Sine:
		return "sine"
	case Ramp:
		return "ramp"
	default:
		return "unknown"
	}
}

type ConnectionPool struct {
	relayURL    string
	poolSize    int
	connections []*ws.Client
	active      []bool
	mu          sync.RWMutex
	created     int64
	failed      int64
}

func NewConnectionPool(relayURL string, poolSize int) *ConnectionPool {
	return &ConnectionPool{
		relayURL:    relayURL,
		poolSize:    poolSize,
		connections: make([]*ws.Client, poolSize),
		active:      make([]bool, poolSize),
	}
}

func (cp *ConnectionPool) Initialize(c context.T) error {
	var wg sync.WaitGroup
	errors := make(chan error, cp.poolSize)

	for i := 0; i < cp.poolSize; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			conn, err := ws.RelayConnect(c, cp.relayURL)
			if err != nil {
				errors <- fmt.Errorf("connection %d failed: %w", idx, err)
				atomic.AddInt64(&cp.failed, 1)
				return
			}

			cp.mu.Lock()
			cp.connections[idx] = conn
			cp.active[idx] = true
			cp.mu.Unlock()

			atomic.AddInt64(&cp.created, 1)
		}(i)
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for range errors {
		errorCount++
	}

	if errorCount > 0 {
		return fmt.Errorf("failed to create %d connections", errorCount)
	}

	return nil
}

func (cp *ConnectionPool) GetConnection(idx int) *ws.Client {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if idx >= 0 && idx < len(cp.connections) && cp.active[idx] {
		return cp.connections[idx]
	}
	return nil
}

func (cp *ConnectionPool) CloseAll() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	for i, conn := range cp.connections {
		if conn != nil && cp.active[i] {
			conn.Close()
			cp.active[i] = false
		}
	}
}

func (cp *ConnectionPool) Stats() (created, failed int64) {
	return atomic.LoadInt64(&cp.created), atomic.LoadInt64(&cp.failed)
}

type LoadSimulator struct {
	relayURL       string
	pattern        LoadPattern
	duration       time.Duration
	baseLoad       int
	peakLoad       int
	poolSize       int
	eventSize      int
	connectionPool *ConnectionPool
	metrics        LoadMetrics
	running        atomic.Bool
}

type LoadMetrics struct {
	EventsSent       atomic.Int64
	EventsFailed     atomic.Int64
	ConnectionErrors atomic.Int64
	AvgLatency       atomic.Int64
	PeakLatency      atomic.Int64
	StartTime        time.Time
	EndTime          time.Time
}

func NewLoadSimulator(relayURL string, pattern LoadPattern, duration time.Duration, baseLoad, peakLoad, poolSize, eventSize int) *LoadSimulator {
	return &LoadSimulator{
		relayURL:  relayURL,
		pattern:   pattern,
		duration:  duration,
		baseLoad:  baseLoad,
		peakLoad:  peakLoad,
		poolSize:  poolSize,
		eventSize: eventSize,
	}
}

func (ls *LoadSimulator) Run(c context.T) error {
	fmt.Printf("Starting %s load simulation for %v...\n", ls.pattern, ls.duration)
	fmt.Printf("Base load: %d events/sec, Peak load: %d events/sec\n", ls.baseLoad, ls.peakLoad)
	fmt.Printf("Connection pool size: %d\n", ls.poolSize)

	ls.connectionPool = NewConnectionPool(ls.relayURL, ls.poolSize)
	if err := ls.connectionPool.Initialize(c); err != nil {
		return fmt.Errorf("failed to initialize connection pool: %w", err)
	}
	defer ls.connectionPool.CloseAll()

	created, failed := ls.connectionPool.Stats()
	fmt.Printf("Connections established: %d, failed: %d\n", created, failed)

	ls.metrics.StartTime = time.Now()
	ls.running.Store(true)

	switch ls.pattern {
	case Constant:
		return ls.runConstant(c)
	case Spike:
		return ls.runSpike(c)
	case Burst:
		return ls.runBurst(c)
	case Sine:
		return ls.runSine(c)
	case Ramp:
		return ls.runRamp(c)
	default:
		return fmt.Errorf("unsupported load pattern: %s", ls.pattern)
	}
}

func (ls *LoadSimulator) runConstant(c context.T) error {
	interval := time.Second / time.Duration(ls.baseLoad)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timeout := time.After(ls.duration)
	connectionIdx := 0

	for {
		select {
		case <-timeout:
			return ls.finalize()
		case <-ticker.C:
			go ls.sendEvent(c, connectionIdx%ls.poolSize)
			connectionIdx++
		}
	}
}

func (ls *LoadSimulator) runSpike(c context.T) error {
	baseInterval := time.Second / time.Duration(ls.baseLoad)
	spikeDuration := ls.duration / 10
	spikeStart := ls.duration / 2

	baseTicker := time.NewTicker(baseInterval)
	defer baseTicker.Stop()

	timeout := time.After(ls.duration)
	spikeTimeout := time.After(spikeStart)
	spikeEnd := time.After(spikeStart + spikeDuration)

	connectionIdx := 0
	inSpike := false

	for {
		select {
		case <-timeout:
			return ls.finalize()
		case <-spikeTimeout:
			if !inSpike {
				inSpike = true
				baseTicker.Stop()
				spikeInterval := time.Second / time.Duration(ls.peakLoad)
				baseTicker = time.NewTicker(spikeInterval)
			}
		case <-spikeEnd:
			if inSpike {
				inSpike = false
				baseTicker.Stop()
				baseTicker = time.NewTicker(baseInterval)
			}
		case <-baseTicker.C:
			go ls.sendEvent(c, connectionIdx%ls.poolSize)
			connectionIdx++
		}
	}
}

func (ls *LoadSimulator) runBurst(c context.T) error {
	burstInterval := ls.duration / 5
	burstSize := ls.peakLoad / 2

	ticker := time.NewTicker(burstInterval)
	defer ticker.Stop()

	timeout := time.After(ls.duration)
	connectionIdx := 0

	for {
		select {
		case <-timeout:
			return ls.finalize()
		case <-ticker.C:
			for i := 0; i < burstSize; i++ {
				go ls.sendEvent(c, connectionIdx%ls.poolSize)
				connectionIdx++
			}
		}
	}
}

func (ls *LoadSimulator) runSine(c context.T) error {
	startTime := time.Now()
	baseTicker := time.NewTicker(50 * time.Millisecond)
	defer baseTicker.Stop()

	timeout := time.After(ls.duration)
	connectionIdx := 0
	lastSend := time.Now()

	for {
		select {
		case <-timeout:
			return ls.finalize()
		case now := <-baseTicker.C:
			elapsed := now.Sub(startTime)
			progress := float64(elapsed) / float64(ls.duration)
			sineValue := math.Sin(progress * 4 * math.Pi)

			currentLoad := ls.baseLoad + int(float64(ls.peakLoad-ls.baseLoad)*((sineValue+1)/2))

			if currentLoad > 0 {
				interval := time.Second / time.Duration(currentLoad)
				if now.Sub(lastSend) >= interval {
					go ls.sendEvent(c, connectionIdx%ls.poolSize)
					connectionIdx++
					lastSend = now
				}
			}
		}
	}
}

func (ls *LoadSimulator) runRamp(c context.T) error {
	startTime := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(ls.duration)
	connectionIdx := 0
	lastSend := time.Now()

	for {
		select {
		case <-timeout:
			return ls.finalize()
		case now := <-ticker.C:
			elapsed := now.Sub(startTime)
			progress := float64(elapsed) / float64(ls.duration)

			currentLoad := ls.baseLoad + int(float64(ls.peakLoad-ls.baseLoad)*progress)

			if currentLoad > 0 {
				interval := time.Second / time.Duration(currentLoad)
				if now.Sub(lastSend) >= interval {
					go ls.sendEvent(c, connectionIdx%ls.poolSize)
					connectionIdx++
					lastSend = now
				}
			}
		}
	}
}

func (ls *LoadSimulator) sendEvent(c context.T, connIdx int) {
	startTime := time.Now()

	conn := ls.connectionPool.GetConnection(connIdx)
	if conn == nil {
		ls.metrics.ConnectionErrors.Add(1)
		return
	}

	signer := newTestSigner()
	ev := generateEvent(signer, ls.eventSize, 0, 0)

	err := conn.Publish(c, ev)
	latency := time.Since(startTime)

	if err != nil {
		ls.metrics.EventsFailed.Add(1)
		log.E.F("Event publish failed: %v", err)
		return
	}

	ls.metrics.EventsSent.Add(1)

	latencyMs := latency.Milliseconds()
	ls.metrics.AvgLatency.Store(latencyMs)

	if latencyMs > ls.metrics.PeakLatency.Load() {
		ls.metrics.PeakLatency.Store(latencyMs)
	}
}

func (ls *LoadSimulator) finalize() error {
	ls.metrics.EndTime = time.Now()
	ls.running.Store(false)

	duration := ls.metrics.EndTime.Sub(ls.metrics.StartTime)
	eventsSent := ls.metrics.EventsSent.Load()
	eventsFailed := ls.metrics.EventsFailed.Load()
	connectionErrors := ls.metrics.ConnectionErrors.Load()

	fmt.Printf("\n=== Load Simulation Results ===\n")
	fmt.Printf("Pattern: %s\n", ls.pattern)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Events Sent: %d\n", eventsSent)
	fmt.Printf("Events Failed: %d\n", eventsFailed)
	fmt.Printf("Connection Errors: %d\n", connectionErrors)

	if eventsSent > 0 {
		rate := float64(eventsSent) / duration.Seconds()
		successRate := float64(eventsSent) / float64(eventsSent+eventsFailed) * 100
		fmt.Printf("Average Rate: %.2f events/sec\n", rate)
		fmt.Printf("Success Rate: %.1f%%\n", successRate)
		fmt.Printf("Average Latency: %dms\n", ls.metrics.AvgLatency.Load())
		fmt.Printf("Peak Latency: %dms\n", ls.metrics.PeakLatency.Load())
	}

	return nil
}

func (ls *LoadSimulator) SimulateResourceConstraints(c context.T, memoryLimit, cpuLimit int) error {
	fmt.Printf("\n=== Resource Constraint Simulation ===\n")
	fmt.Printf("Memory limit: %d MB, CPU limit: %d%%\n", memoryLimit, cpuLimit)

	constraintTests := []struct {
		name     string
		duration time.Duration
		load     int
	}{
		{"baseline", 30 * time.Second, ls.baseLoad},
		{"memory_stress", 60 * time.Second, ls.peakLoad * 2},
		{"cpu_stress", 45 * time.Second, ls.peakLoad * 3},
		{"combined_stress", 90 * time.Second, ls.peakLoad * 4},
	}

	for _, test := range constraintTests {
		fmt.Printf("\nRunning %s test...\n", test.name)

		simulator := NewLoadSimulator(ls.relayURL, Constant, test.duration, test.load, test.load, ls.poolSize, ls.eventSize)

		if err := simulator.Run(c); err != nil {
			fmt.Printf("Test %s failed: %v\n", test.name, err)
			continue
		}

		time.Sleep(10 * time.Second)
	}

	return nil
}

func (ls *LoadSimulator) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	metrics["pattern"] = ls.pattern.String()
	metrics["events_sent"] = ls.metrics.EventsSent.Load()
	metrics["events_failed"] = ls.metrics.EventsFailed.Load()
	metrics["connection_errors"] = ls.metrics.ConnectionErrors.Load()
	metrics["avg_latency_ms"] = ls.metrics.AvgLatency.Load()
	metrics["peak_latency_ms"] = ls.metrics.PeakLatency.Load()

	if !ls.metrics.StartTime.IsZero() && !ls.metrics.EndTime.IsZero() {
		duration := ls.metrics.EndTime.Sub(ls.metrics.StartTime)
		metrics["duration_seconds"] = duration.Seconds()

		if eventsSent := ls.metrics.EventsSent.Load(); eventsSent > 0 {
			metrics["events_per_second"] = float64(eventsSent) / duration.Seconds()
		}
	}

	return metrics
}

type LoadTestSuite struct {
	relayURL  string
	poolSize  int
	eventSize int
}

func NewLoadTestSuite(relayURL string, poolSize, eventSize int) *LoadTestSuite {
	return &LoadTestSuite{
		relayURL:  relayURL,
		poolSize:  poolSize,
		eventSize: eventSize,
	}
}

func (lts *LoadTestSuite) RunAllPatterns(c context.T) error {
	patterns := []struct {
		pattern  LoadPattern
		baseLoad int
		peakLoad int
		duration time.Duration
	}{
		{Constant, 50, 50, 60 * time.Second},
		{Spike, 50, 500, 90 * time.Second},
		{Burst, 20, 400, 75 * time.Second},
		{Sine, 50, 300, 120 * time.Second},
		{Ramp, 10, 200, 90 * time.Second},
	}

	fmt.Printf("Running comprehensive load test suite...\n")

	for _, p := range patterns {
		fmt.Printf("\n--- Testing %s pattern ---\n", p.pattern)

		simulator := NewLoadSimulator(lts.relayURL, p.pattern, p.duration, p.baseLoad, p.peakLoad, lts.poolSize, lts.eventSize)

		if err := simulator.Run(c); err != nil {
			fmt.Printf("Pattern %s failed: %v\n", p.pattern, err)
			continue
		}

		time.Sleep(5 * time.Second)
	}

	return nil
}
