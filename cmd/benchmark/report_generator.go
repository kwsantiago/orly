package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

type RelayBenchmarkData struct {
	RelayType         string    `json:"relay_type"`
	EventsPublished   int64     `json:"events_published"`
	EventsPublishedMB float64   `json:"events_published_mb"`
	PublishDuration   string    `json:"publish_duration"`
	PublishRate       float64   `json:"publish_rate"`
	PublishBandwidth  float64   `json:"publish_bandwidth"`
	QueriesExecuted   int64     `json:"queries_executed"`
	EventsReturned    int64     `json:"events_returned"`
	QueryDuration     string    `json:"query_duration"`
	QueryRate         float64   `json:"query_rate"`
	AvgEventsPerQuery float64   `json:"avg_events_per_query"`
	StartupTime       string    `json:"startup_time,omitempty"`
	ShutdownTime      string    `json:"shutdown_time,omitempty"`
	Errors            int64     `json:"errors,omitempty"`
	MemoryUsageMB     float64   `json:"memory_usage_mb,omitempty"`
	P50Latency        string    `json:"p50_latency,omitempty"`
	P95Latency        string    `json:"p95_latency,omitempty"`
	P99Latency        string    `json:"p99_latency,omitempty"`
	Timestamp         time.Time `json:"timestamp"`
}

type ComparisonReport struct {
	Title           string               `json:"title"`
	GeneratedAt     time.Time            `json:"generated_at"`
	RelayData       []RelayBenchmarkData `json:"relay_data"`
	WinnerPublish   string               `json:"winner_publish"`
	WinnerQuery     string               `json:"winner_query"`
	Anomalies       []string             `json:"anomalies"`
	Recommendations []string             `json:"recommendations"`
}

type ReportGenerator struct {
	data   []RelayBenchmarkData
	report ComparisonReport
}

func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{
		data: make([]RelayBenchmarkData, 0),
		report: ComparisonReport{
			GeneratedAt:     time.Now(),
			Anomalies:       make([]string, 0),
			Recommendations: make([]string, 0),
		},
	}
}

func (rg *ReportGenerator) AddRelayData(relayType string, results *BenchmarkResults, metrics *HarnessMetrics, profilerMetrics *QueryMetrics) {
	data := RelayBenchmarkData{
		RelayType:         relayType,
		EventsPublished:   results.EventsPublished,
		EventsPublishedMB: float64(results.EventsPublishedBytes) / 1024 / 1024,
		PublishDuration:   results.PublishDuration.String(),
		PublishRate:       results.PublishRate,
		PublishBandwidth:  results.PublishBandwidth,
		QueriesExecuted:   results.QueriesExecuted,
		EventsReturned:    results.EventsReturned,
		QueryDuration:     results.QueryDuration.String(),
		QueryRate:         results.QueryRate,
		Timestamp:         time.Now(),
	}

	if results.QueriesExecuted > 0 {
		data.AvgEventsPerQuery = float64(results.EventsReturned) / float64(results.QueriesExecuted)
	}

	if metrics != nil {
		data.StartupTime = metrics.StartupTime.String()
		data.ShutdownTime = metrics.ShutdownTime.String()
		data.Errors = int64(metrics.Errors)
	}

	if profilerMetrics != nil {
		data.MemoryUsageMB = float64(profilerMetrics.MemoryPeak) / 1024 / 1024
		data.P50Latency = profilerMetrics.P50.String()
		data.P95Latency = profilerMetrics.P95.String()
		data.P99Latency = profilerMetrics.P99.String()
	}

	rg.data = append(rg.data, data)
}

func (rg *ReportGenerator) GenerateReport(title string) {
	rg.report.Title = title
	rg.report.RelayData = rg.data
	rg.analyzePerfomance()
	rg.detectAnomalies()
	rg.generateRecommendations()
}

func (rg *ReportGenerator) analyzePerfomance() {
	if len(rg.data) == 0 {
		return
	}

	var bestPublishRate float64
	var bestQueryRate float64
	bestPublishRelay := ""
	bestQueryRelay := ""

	for _, data := range rg.data {
		if data.PublishRate > bestPublishRate {
			bestPublishRate = data.PublishRate
			bestPublishRelay = data.RelayType
		}
		if data.QueryRate > bestQueryRate {
			bestQueryRate = data.QueryRate
			bestQueryRelay = data.RelayType
		}
	}

	rg.report.WinnerPublish = bestPublishRelay
	rg.report.WinnerQuery = bestQueryRelay
}

func (rg *ReportGenerator) detectAnomalies() {
	if len(rg.data) < 2 {
		return
	}

	publishRates := make([]float64, len(rg.data))
	queryRates := make([]float64, len(rg.data))

	for i, data := range rg.data {
		publishRates[i] = data.PublishRate
		queryRates[i] = data.QueryRate
	}

	publishMean := mean(publishRates)
	publishStdDev := stdDev(publishRates, publishMean)
	queryMean := mean(queryRates)
	queryStdDev := stdDev(queryRates, queryMean)

	for _, data := range rg.data {
		if math.Abs(data.PublishRate-publishMean) > 2*publishStdDev {
			anomaly := fmt.Sprintf("%s publish rate (%.2f) deviates significantly from average (%.2f)",
				data.RelayType, data.PublishRate, publishMean)
			rg.report.Anomalies = append(rg.report.Anomalies, anomaly)
		}

		if math.Abs(data.QueryRate-queryMean) > 2*queryStdDev {
			anomaly := fmt.Sprintf("%s query rate (%.2f) deviates significantly from average (%.2f)",
				data.RelayType, data.QueryRate, queryMean)
			rg.report.Anomalies = append(rg.report.Anomalies, anomaly)
		}

		if data.Errors > 0 {
			anomaly := fmt.Sprintf("%s had %d errors during benchmark", data.RelayType, data.Errors)
			rg.report.Anomalies = append(rg.report.Anomalies, anomaly)
		}
	}
}

func (rg *ReportGenerator) generateRecommendations() {
	if len(rg.data) == 0 {
		return
	}

	sort.Slice(rg.data, func(i, j int) bool {
		return rg.data[i].PublishRate > rg.data[j].PublishRate
	})

	if len(rg.data) > 1 {
		best := rg.data[0]
		worst := rg.data[len(rg.data)-1]

		improvement := (best.PublishRate - worst.PublishRate) / worst.PublishRate * 100
		if improvement > 20 {
			rec := fmt.Sprintf("Consider using %s for high-throughput scenarios (%.1f%% faster than %s)",
				best.RelayType, improvement, worst.RelayType)
			rg.report.Recommendations = append(rg.report.Recommendations, rec)
		}
	}

	for _, data := range rg.data {
		if data.MemoryUsageMB > 500 {
			rec := fmt.Sprintf("%s shows high memory usage (%.1f MB) - monitor for memory leaks",
				data.RelayType, data.MemoryUsageMB)
			rg.report.Recommendations = append(rg.report.Recommendations, rec)
		}
	}
}

func (rg *ReportGenerator) OutputMarkdown(writer io.Writer) error {
	fmt.Fprintf(writer, "# %s\n\n", rg.report.Title)
	fmt.Fprintf(writer, "Generated: %s\n\n", rg.report.GeneratedAt.Format(time.RFC3339))

	fmt.Fprintf(writer, "## Performance Summary\n\n")
	fmt.Fprintf(writer, "| Relay | Publish Rate | Publish BW | Query Rate | Avg Events/Query | Memory (MB) |\n")
	fmt.Fprintf(writer, "|-------|--------------|------------|------------|------------------|-------------|\n")

	for _, data := range rg.data {
		fmt.Fprintf(writer, "| %s | %.2f/s | %.2f MB/s | %.2f/s | %.2f | %.1f |\n",
			data.RelayType, data.PublishRate, data.PublishBandwidth,
			data.QueryRate, data.AvgEventsPerQuery, data.MemoryUsageMB)
	}

	if rg.report.WinnerPublish != "" || rg.report.WinnerQuery != "" {
		fmt.Fprintf(writer, "\n## Winners\n\n")
		if rg.report.WinnerPublish != "" {
			fmt.Fprintf(writer, "- **Best Publisher**: %s\n", rg.report.WinnerPublish)
		}
		if rg.report.WinnerQuery != "" {
			fmt.Fprintf(writer, "- **Best Query Engine**: %s\n", rg.report.WinnerQuery)
		}
	}

	if len(rg.report.Anomalies) > 0 {
		fmt.Fprintf(writer, "\n## Anomalies\n\n")
		for _, anomaly := range rg.report.Anomalies {
			fmt.Fprintf(writer, "- %s\n", anomaly)
		}
	}

	if len(rg.report.Recommendations) > 0 {
		fmt.Fprintf(writer, "\n## Recommendations\n\n")
		for _, rec := range rg.report.Recommendations {
			fmt.Fprintf(writer, "- %s\n", rec)
		}
	}

	fmt.Fprintf(writer, "\n## Detailed Results\n\n")
	for _, data := range rg.data {
		fmt.Fprintf(writer, "### %s\n\n", data.RelayType)
		fmt.Fprintf(writer, "- Events Published: %d (%.2f MB)\n", data.EventsPublished, data.EventsPublishedMB)
		fmt.Fprintf(writer, "- Publish Duration: %s\n", data.PublishDuration)
		fmt.Fprintf(writer, "- Queries Executed: %d\n", data.QueriesExecuted)
		fmt.Fprintf(writer, "- Query Duration: %s\n", data.QueryDuration)
		if data.P50Latency != "" {
			fmt.Fprintf(writer, "- Latency P50/P95/P99: %s/%s/%s\n", data.P50Latency, data.P95Latency, data.P99Latency)
		}
		if data.StartupTime != "" {
			fmt.Fprintf(writer, "- Startup Time: %s\n", data.StartupTime)
		}
		fmt.Fprintf(writer, "\n")
	}

	return nil
}

func (rg *ReportGenerator) OutputJSON(writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rg.report)
}

func (rg *ReportGenerator) OutputCSV(writer io.Writer) error {
	w := csv.NewWriter(writer)
	defer w.Flush()

	header := []string{
		"relay_type", "events_published", "events_published_mb", "publish_duration",
		"publish_rate", "publish_bandwidth", "queries_executed", "events_returned",
		"query_duration", "query_rate", "avg_events_per_query", "memory_usage_mb",
		"p50_latency", "p95_latency", "p99_latency", "startup_time", "errors",
	}

	if err := w.Write(header); err != nil {
		return err
	}

	for _, data := range rg.data {
		row := []string{
			data.RelayType,
			fmt.Sprintf("%d", data.EventsPublished),
			fmt.Sprintf("%.2f", data.EventsPublishedMB),
			data.PublishDuration,
			fmt.Sprintf("%.2f", data.PublishRate),
			fmt.Sprintf("%.2f", data.PublishBandwidth),
			fmt.Sprintf("%d", data.QueriesExecuted),
			fmt.Sprintf("%d", data.EventsReturned),
			data.QueryDuration,
			fmt.Sprintf("%.2f", data.QueryRate),
			fmt.Sprintf("%.2f", data.AvgEventsPerQuery),
			fmt.Sprintf("%.1f", data.MemoryUsageMB),
			data.P50Latency,
			data.P95Latency,
			data.P99Latency,
			data.StartupTime,
			fmt.Sprintf("%d", data.Errors),
		}

		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func (rg *ReportGenerator) GenerateThroughputCurve() []ThroughputPoint {
	points := make([]ThroughputPoint, 0)

	for _, data := range rg.data {
		point := ThroughputPoint{
			RelayType:  data.RelayType,
			Throughput: data.PublishRate,
			Latency:    parseLatency(data.P95Latency),
		}
		points = append(points, point)
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Throughput < points[j].Throughput
	})

	return points
}

type ThroughputPoint struct {
	RelayType  string  `json:"relay_type"`
	Throughput float64 `json:"throughput"`
	Latency    float64 `json:"latency_ms"`
}

func parseLatency(latencyStr string) float64 {
	if latencyStr == "" {
		return 0
	}

	latencyStr = strings.TrimSuffix(latencyStr, "ms")
	latencyStr = strings.TrimSuffix(latencyStr, "µs")
	latencyStr = strings.TrimSuffix(latencyStr, "ns")

	if dur, err := time.ParseDuration(latencyStr); err == nil {
		return float64(dur.Nanoseconds()) / 1e6
	}

	return 0
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	variance /= float64(len(values) - 1)

	return math.Sqrt(variance)
}

func SaveReportToFile(filename, format string, generator *ReportGenerator) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	switch format {
	case "json":
		return generator.OutputJSON(file)
	case "csv":
		return generator.OutputCSV(file)
	case "markdown", "md":
		return generator.OutputMarkdown(file)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}
