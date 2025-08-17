package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"lukechampine.com/frand"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/lol"
)

type BenchmarkResults struct {
	EventsPublished      int64
	EventsPublishedBytes int64
	PublishDuration      time.Duration
	PublishRate          float64
	PublishBandwidth     float64

	QueriesExecuted int64
	QueryDuration   time.Duration
	QueryRate       float64
	EventsReturned  int64
}

func main() {
	var (
		relayURL = flag.String(
			"relay", "ws://localhost:7447", "Client URL to benchmark",
		)
		eventCount = flag.Int(
			"events", 10000, "Number of events to publish",
		)
		eventSize = flag.Int(
			"size", 1024, "Average size of event content in bytes",
		)
		concurrency = flag.Int(
			"concurrency", 10, "Number of concurrent publishers",
		)
		queryCount = flag.Int(
			"queries", 100, "Number of queries to execute",
		)
		queryLimit  = flag.Int("query-limit", 100, "Limit for each query")
		skipPublish = flag.Bool(
			"skip-publish", false, "Skip publishing phase",
		)
		skipQuery  = flag.Bool("skip-query", false, "Skip query phase")
		verbose    = flag.Bool("v", false, "Verbose output")
		multiRelay = flag.Bool(
			"multi-relay", false, "Use multi-relay harness",
		)
		relayBinPath = flag.String(
			"relay-bin", "", "Path to relay binary (for multi-relay mode)",
		)
		profileQueries = flag.Bool(
			"profile", false, "Run query performance profiling",
		)
		profileSubs = flag.Bool(
			"profile-subs", false, "Profile subscription performance",
		)
		subCount = flag.Int(
			"sub-count", 100,
			"Number of concurrent subscriptions for profiling",
		)
		subDuration = flag.Duration(
			"sub-duration", 30*time.Second,
			"Duration for subscription profiling",
		)
		installRelays = flag.Bool(
			"install", false, "Install relay dependencies and binaries",
		)
		installSecp = flag.Bool(
			"install-secp", false, "Install only secp256k1 library",
		)
		workDir = flag.String(
			"work-dir", "/tmp/relay-build", "Working directory for builds",
		)
		installDir = flag.String(
			"install-dir", "/usr/local/bin",
			"Installation directory for binaries",
		)
		generateReport = flag.Bool(
			"report", false, "Generate comparative report",
		)
		reportFormat = flag.String(
			"report-format", "markdown", "Report format: markdown, json, csv",
		)
		reportFile = flag.String(
			"report-file", "benchmark_report",
			"Report output filename (without extension)",
		)
		reportTitle = flag.String(
			"report-title", "Client Benchmark Comparison", "Report title",
		)
		timingMode = flag.Bool(
			"timing", false, "Run end-to-end timing instrumentation",
		)
		timingEvents = flag.Int(
			"timing-events", 100, "Number of events for timing instrumentation",
		)
		timingSubs = flag.Bool(
			"timing-subs", false, "Test subscription timing",
		)
		timingDuration = flag.Duration(
			"timing-duration", 10*time.Second,
			"Duration for subscription timing test",
		)
		loadTest = flag.Bool(
			"load", false, "Run load pattern simulation",
		)
		loadPattern = flag.String(
			"load-pattern", "constant",
			"Load pattern: constant, spike, burst, sine, ramp",
		)
		loadDuration = flag.Duration(
			"load-duration", 60*time.Second, "Duration for load test",
		)
		loadBase = flag.Int("load-base", 50, "Base load (events/sec)")
		loadPeak = flag.Int("load-peak", 200, "Peak load (events/sec)")
		loadPool = flag.Int(
			"load-pool", 10, "Connection pool size for load testing",
		)
		loadSuite = flag.Bool(
			"load-suite", false, "Run comprehensive load test suite",
		)
		loadConstraints = flag.Bool(
			"load-constraints", false, "Test under resource constraints",
		)
	)
	flag.Parse()

	if *verbose {
		lol.SetLogLevel("trace")
	}

	c := context.Bg()

	if *installRelays {
		runInstaller(*workDir, *installDir)
	} else if *installSecp {
		runSecp256k1Installer(*workDir, *installDir)
	} else if *generateReport {
		runReportGeneration(*reportTitle, *reportFormat, *reportFile)
	} else if *loadTest || *loadSuite || *loadConstraints {
		runLoadSimulation(
			c, *relayURL, *loadPattern, *loadDuration, *loadBase, *loadPeak,
			*loadPool, *eventSize, *loadSuite, *loadConstraints,
		)
	} else if *timingMode || *timingSubs {
		runTimingInstrumentation(
			c, *relayURL, *timingEvents, *eventSize, *timingSubs,
			*timingDuration,
		)
	} else if *profileQueries || *profileSubs {
		runQueryProfiler(
			c, *relayURL, *queryCount, *concurrency, *profileSubs, *subCount,
			*subDuration,
		)
	} else if *multiRelay {
		runMultiRelayBenchmark(
			c, *relayBinPath, *eventCount, *eventSize, *concurrency,
			*queryCount, *queryLimit, *skipPublish, *skipQuery,
		)
	} else {
		runSingleRelayBenchmark(
			c, *relayURL, *eventCount, *eventSize, *concurrency, *queryCount,
			*queryLimit, *skipPublish, *skipQuery,
		)
	}
}

func runSingleRelayBenchmark(
	c context.T, relayURL string,
	eventCount, eventSize, concurrency, queryCount, queryLimit int,
	skipPublish, skipQuery bool,
) {
	results := &BenchmarkResults{}

	// Phase 1: Publish events
	if !skipPublish {
		fmt.Printf("Publishing %d events to %s...\n", eventCount, relayURL)
		if err := benchmarkPublish(
			c, relayURL, eventCount, eventSize, concurrency, results,
		); chk.E(err) {
			fmt.Fprintf(os.Stderr, "Error during publish benchmark: %v\n", err)
			os.Exit(1)
		}
	}

	// Phase 2: Query events
	if !skipQuery {
		fmt.Printf("\nQuerying events from %s...\n", relayURL)
		if err := benchmarkQuery(
			c, relayURL, queryCount, queryLimit, results,
		); chk.E(err) {
			fmt.Fprintf(os.Stderr, "Error during query benchmark: %v\n", err)
			os.Exit(1)
		}
	}

	// Print results
	printResults(results)
}

func runMultiRelayBenchmark(
	c context.T, relayBinPath string,
	eventCount, eventSize, concurrency, queryCount, queryLimit int,
	skipPublish, skipQuery bool,
) {
	harness := NewMultiRelayHarness()
	generator := NewReportGenerator()

	if relayBinPath != "" {
		config := RelayConfig{
			Type:   Khatru,
			Binary: relayBinPath,
			Args:   []string{},
			URL:    "ws://localhost:7447",
		}
		if err := harness.AddRelay(config); chk.E(err) {
			fmt.Fprintf(os.Stderr, "Failed to add relay: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Starting relay harness...\n")
		if err := harness.StartAll(); chk.E(err) {
			fmt.Fprintf(os.Stderr, "Failed to start relays: %v\n", err)
			os.Exit(1)
		}
		defer harness.StopAll()

		time.Sleep(2 * time.Second)
	}

	relayTypes := []RelayType{Khatru}
	if relayBinPath == "" {
		fmt.Printf("Running multi-relay benchmark without starting relays (external relays expected)\n")
	}

	for _, relayType := range relayTypes {
		fmt.Printf("\n=== Benchmarking %s ===\n", relayType)

		results := &BenchmarkResults{}
		relayURL := "ws://localhost:7447"

		if !skipPublish {
			fmt.Printf("Publishing %d events to %s...\n", eventCount, relayURL)
			if err := benchmarkPublish(
				c, relayURL, eventCount, eventSize, concurrency, results,
			); chk.E(err) {
				fmt.Fprintf(
					os.Stderr, "Error during publish benchmark for %s: %v\n",
					relayType, err,
				)
				continue
			}
		}

		if !skipQuery {
			fmt.Printf("\nQuerying events from %s...\n", relayURL)
			if err := benchmarkQuery(
				c, relayURL, queryCount, queryLimit, results,
			); chk.E(err) {
				fmt.Fprintf(
					os.Stderr, "Error during query benchmark for %s: %v\n",
					relayType, err,
				)
				continue
			}
		}

		fmt.Printf("\n=== %s Results ===\n", relayType)
		printResults(results)

		metrics := harness.GetMetrics(relayType)
		if metrics != nil {
			printHarnessMetrics(relayType, metrics)
		}

		generator.AddRelayData(relayType.String(), results, metrics, nil)
	}

	generator.GenerateReport("Multi-Client Benchmark Results")

	if err := SaveReportToFile(
		"BENCHMARK_RESULTS.md", "markdown", generator,
	); chk.E(err) {
		fmt.Printf("Warning: Failed to save benchmark results: %v\n", err)
	} else {
		fmt.Printf("\nBenchmark results saved to: BENCHMARK_RESULTS.md\n")
	}
}

func benchmarkPublish(
	c context.T, relayURL string, eventCount, eventSize, concurrency int,
	results *BenchmarkResults,
) error {
	// Generate signers for each concurrent publisher
	signers := make([]*testSigner, concurrency)
	for i := range signers {
		signers[i] = newTestSigner()
	}

	// Track published events
	var publishedEvents atomic.Int64
	var publishedBytes atomic.Int64
	var errors atomic.Int64

	// Create wait group for concurrent publishers
	var wg sync.WaitGroup
	eventsPerPublisher := eventCount / concurrency
	extraEvents := eventCount % concurrency

	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(publisherID int) {
			defer wg.Done()

			// Connect to relay
			relay, err := ws.RelayConnect(c, relayURL)
			if err != nil {
				log.E.F("Publisher %d failed to connect: %v", publisherID, err)
				errors.Add(1)
				return
			}
			defer relay.Close()

			// Calculate events for this publisher
			eventsToPublish := eventsPerPublisher
			if publisherID < extraEvents {
				eventsToPublish++
			}

			signer := signers[publisherID]

			// Publish events
			for j := 0; j < eventsToPublish; j++ {
				ev := generateEvent(signer, eventSize, time.Duration(0), 0)

				if err = relay.Publish(c, ev); err != nil {
					log.E.F(
						"Publisher %d failed to publish event: %v", publisherID,
						err,
					)
					errors.Add(1)
					continue
				}

				evBytes := ev.Marshal(nil)
				publishedEvents.Add(1)
				publishedBytes.Add(int64(len(evBytes)))

				if publishedEvents.Load()%1000 == 0 {
					fmt.Printf(
						"  Published %d events...\n", publishedEvents.Load(),
					)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	results.EventsPublished = publishedEvents.Load()
	results.EventsPublishedBytes = publishedBytes.Load()
	results.PublishDuration = duration
	results.PublishRate = float64(results.EventsPublished) / duration.Seconds()
	results.PublishBandwidth = float64(results.EventsPublishedBytes) / duration.Seconds() / 1024 / 1024 // MB/s

	if errors.Load() > 0 {
		fmt.Printf(
			"  Warning: %d errors occurred during publishing\n", errors.Load(),
		)
	}

	return nil
}

func benchmarkQuery(
	c context.T, relayURL string, queryCount, queryLimit int,
	results *BenchmarkResults,
) error {
	relay, err := ws.RelayConnect(c, relayURL)
	if err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}
	defer relay.Close()

	var totalEvents atomic.Int64
	var totalQueries atomic.Int64

	startTime := time.Now()

	for i := 0; i < queryCount; i++ {
		// Generate various filter types
		var f *filter.F
		switch i % 5 {
		case 0:
			// Query by kind
			limit := uint(queryLimit)
			f = &filter.F{
				Kinds: kinds.New(kind.TextNote),
				Limit: &limit,
			}
		case 1:
			// Query by time range
			now := timestamp.Now()
			since := timestamp.New(now.I64() - 3600) // last hour
			limit := uint(queryLimit)
			f = &filter.F{
				Since: since,
				Until: now,
				Limit: &limit,
			}
		case 2:
			// Query by tag
			limit := uint(queryLimit)
			f = &filter.F{
				Tags:  tags.New(tag.New([]byte("p"), generateRandomPubkey())),
				Limit: &limit,
			}
		case 3:
			// Query by author
			limit := uint(queryLimit)
			f = &filter.F{
				Authors: tag.New(generateRandomPubkey()),
				Limit:   &limit,
			}
		case 4:
			// Complex query with multiple conditions
			now := timestamp.Now()
			since := timestamp.New(now.I64() - 7200)
			limit := uint(queryLimit)
			f = &filter.F{
				Kinds:   kinds.New(kind.TextNote, kind.Repost),
				Authors: tag.New(generateRandomPubkey()),
				Since:   since,
				Limit:   &limit,
			}
		}

		// Execute query
		events, err := relay.QuerySync(c, f)
		if err != nil {
			log.E.F("Query %d failed: %v", i, err)
			continue
		}

		totalEvents.Add(int64(len(events)))
		totalQueries.Add(1)

		if totalQueries.Load()%20 == 0 {
			fmt.Printf("  Executed %d queries...\n", totalQueries.Load())
		}
	}

	duration := time.Since(startTime)

	results.QueriesExecuted = totalQueries.Load()
	results.QueryDuration = duration
	results.QueryRate = float64(results.QueriesExecuted) / duration.Seconds()
	results.EventsReturned = totalEvents.Load()

	return nil
}

func generateEvent(
	signer *testSigner, contentSize int, rateLimit time.Duration, burstSize int,
) *event.E {
	return generateSimpleEvent(signer, contentSize)
}

func generateRandomTags() *tags.T {
	t := tags.New()

	// Add some random tags
	numTags := frand.Intn(5)
	for i := 0; i < numTags; i++ {
		switch frand.Intn(3) {
		case 0:
			// p tag
			t.AppendUnique(tag.New([]byte("p"), generateRandomPubkey()))
		case 1:
			// e tag
			t.AppendUnique(tag.New([]byte("e"), generateRandomEventID()))
		case 2:
			// t tag
			t.AppendUnique(
				tag.New(
					[]byte("t"),
					[]byte(fmt.Sprintf("topic%d", frand.Intn(100))),
				),
			)
		}
	}

	return t
}

func generateRandomPubkey() []byte {
	return frand.Bytes(32)
}

func generateRandomEventID() []byte {
	return frand.Bytes(32)
}

func printResults(results *BenchmarkResults) {
	fmt.Println("\n=== Benchmark Results ===")

	if results.EventsPublished > 0 {
		fmt.Println("\nPublish Performance:")
		fmt.Printf("  Events Published: %d\n", results.EventsPublished)
		fmt.Printf(
			"  Total Data: %.2f MB\n",
			float64(results.EventsPublishedBytes)/1024/1024,
		)
		fmt.Printf("  Duration: %s\n", results.PublishDuration)
		fmt.Printf("  Rate: %.2f events/second\n", results.PublishRate)
		fmt.Printf("  Bandwidth: %.2f MB/second\n", results.PublishBandwidth)
	}

	if results.QueriesExecuted > 0 {
		fmt.Println("\nQuery Performance:")
		fmt.Printf("  Queries Executed: %d\n", results.QueriesExecuted)
		fmt.Printf("  Events Returned: %d\n", results.EventsReturned)
		fmt.Printf("  Duration: %s\n", results.QueryDuration)
		fmt.Printf("  Rate: %.2f queries/second\n", results.QueryRate)
		avgEventsPerQuery := float64(results.EventsReturned) / float64(results.QueriesExecuted)
		fmt.Printf("  Avg Events/Query: %.2f\n", avgEventsPerQuery)
	}
}

func printHarnessMetrics(relayType RelayType, metrics *HarnessMetrics) {
	fmt.Printf("\nHarness Metrics for %s:\n", relayType)
	if metrics.StartupTime > 0 {
		fmt.Printf("  Startup Time: %s\n", metrics.StartupTime)
	}
	if metrics.ShutdownTime > 0 {
		fmt.Printf("  Shutdown Time: %s\n", metrics.ShutdownTime)
	}
	if metrics.Errors > 0 {
		fmt.Printf("  Errors: %d\n", metrics.Errors)
	}
}

func runQueryProfiler(
	c context.T, relayURL string, queryCount, concurrency int, profileSubs bool,
	subCount int, subDuration time.Duration,
) {
	profiler := NewQueryProfiler(relayURL)

	if profileSubs {
		fmt.Printf(
			"Profiling %d concurrent subscriptions for %v...\n", subCount,
			subDuration,
		)
		if err := profiler.TestSubscriptionPerformance(
			c, subDuration, subCount,
		); chk.E(err) {
			fmt.Fprintf(os.Stderr, "Subscription profiling failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf(
			"Profiling %d queries with %d concurrent workers...\n", queryCount,
			concurrency,
		)
		if err := profiler.ExecuteProfile(
			c, queryCount, concurrency,
		); chk.E(err) {
			fmt.Fprintf(os.Stderr, "Query profiling failed: %v\n", err)
			os.Exit(1)
		}
	}

	profiler.PrintReport()
}

func runInstaller(workDir, installDir string) {
	installer := NewRelayInstaller(workDir, installDir)

	if err := installer.InstallAll(); chk.E(err) {
		fmt.Fprintf(os.Stderr, "Installation failed: %v\n", err)
		os.Exit(1)
	}
}

func runSecp256k1Installer(workDir, installDir string) {
	installer := NewRelayInstaller(workDir, installDir)

	if err := installer.InstallSecp256k1Only(); chk.E(err) {
		fmt.Fprintf(os.Stderr, "secp256k1 installation failed: %v\n", err)
		os.Exit(1)
	}
}

func runLoadSimulation(
	c context.T, relayURL, patternStr string, duration time.Duration,
	baseLoad, peakLoad, poolSize, eventSize int, runSuite, runConstraints bool,
) {
	if runSuite {
		suite := NewLoadTestSuite(relayURL, poolSize, eventSize)
		if err := suite.RunAllPatterns(c); chk.E(err) {
			fmt.Fprintf(os.Stderr, "Load test suite failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	var pattern LoadPattern
	switch patternStr {
	case "constant":
		pattern = Constant
	case "spike":
		pattern = Spike
	case "burst":
		pattern = Burst
	case "sine":
		pattern = Sine
	case "ramp":
		pattern = Ramp
	default:
		fmt.Fprintf(os.Stderr, "Invalid load pattern: %s\n", patternStr)
		os.Exit(1)
	}

	simulator := NewLoadSimulator(
		relayURL, pattern, duration, baseLoad, peakLoad, poolSize, eventSize,
	)

	if err := simulator.Run(c); chk.E(err) {
		fmt.Fprintf(os.Stderr, "Load simulation failed: %v\n", err)
		os.Exit(1)
	}

	if runConstraints {
		fmt.Printf("\n")
		if err := simulator.SimulateResourceConstraints(
			c, 512, 80,
		); chk.E(err) {
			fmt.Fprintf(
				os.Stderr, "Resource constraint simulation failed: %v\n", err,
			)
		}
	}

	metrics := simulator.GetMetrics()
	fmt.Printf("\n=== Load Simulation Summary ===\n")
	fmt.Printf("Pattern: %v\n", metrics["pattern"])
	fmt.Printf("Events sent: %v\n", metrics["events_sent"])
	fmt.Printf("Events failed: %v\n", metrics["events_failed"])
	fmt.Printf("Connection errors: %v\n", metrics["connection_errors"])
	fmt.Printf("Events/second: %.2f\n", metrics["events_per_second"])
	fmt.Printf("Average latency: %vms\n", metrics["avg_latency_ms"])
	fmt.Printf("Peak latency: %vms\n", metrics["peak_latency_ms"])
}

func runTimingInstrumentation(
	c context.T, relayURL string, eventCount, eventSize int, testSubs bool,
	duration time.Duration,
) {
	instrumentation := NewTimingInstrumentation(relayURL)

	fmt.Printf("Connecting to relay at %s...\n", relayURL)
	if err := instrumentation.Connect(c, relayURL); chk.E(err) {
		fmt.Fprintf(os.Stderr, "Failed to connect to relay: %v\n", err)
		os.Exit(1)
	}
	defer instrumentation.Close()

	if testSubs {
		fmt.Printf("\n=== Subscription Timing Test ===\n")
		if err := instrumentation.TestSubscriptionTiming(
			c, duration,
		); chk.E(err) {
			fmt.Fprintf(os.Stderr, "Subscription timing test failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("\n=== Full Event Lifecycle Instrumentation ===\n")
		if err := instrumentation.RunFullInstrumentation(
			c, eventCount, eventSize,
		); chk.E(err) {
			fmt.Fprintf(os.Stderr, "Timing instrumentation failed: %v\n", err)
			os.Exit(1)
		}
	}

	metrics := instrumentation.GetMetrics()
	fmt.Printf("\n=== Instrumentation Metrics Summary ===\n")
	fmt.Printf("Total Events Tracked: %v\n", metrics["tracked_events"])
	fmt.Printf("Lifecycles Recorded: %v\n", metrics["lifecycles_count"])
	fmt.Printf("WebSocket Frames: %v\n", metrics["frames_tracked"])
	fmt.Printf("Write Amplifications: %v\n", metrics["write_amplifications"])

	if bottlenecks, ok := metrics["bottlenecks"].(map[string]map[string]interface{}); ok {
		fmt.Printf("\n=== Pipeline Stage Analysis ===\n")
		for stage, data := range bottlenecks {
			fmt.Printf(
				"%s: avg=%vms, p95=%vms, p99=%vms, throughput=%.2f ops/s\n",
				stage,
				data["avg_latency_ms"],
				data["p95_latency_ms"],
				data["p99_latency_ms"],
				data["throughput_ops_sec"],
			)
		}
	}
}

func runReportGeneration(title, format, filename string) {
	generator := NewReportGenerator()

	resultsFile := "BENCHMARK_RESULTS.md"
	if _, err := os.Stat(resultsFile); os.IsNotExist(err) {
		fmt.Printf("No benchmark results found. Run benchmarks first to generate data.\n")
		fmt.Printf("Example: ./benchmark --multi-relay --relay-bin /path/to/relay\n")
		os.Exit(1)
	}

	fmt.Printf("Generating %s report: %s\n", format, filename)

	sampleData := []RelayBenchmarkData{
		{
			RelayType:         "khatru",
			EventsPublished:   10000,
			EventsPublishedMB: 15.2,
			PublishDuration:   "12.5s",
			PublishRate:       800.0,
			PublishBandwidth:  1.22,
			QueriesExecuted:   100,
			EventsReturned:    8500,
			QueryDuration:     "2.1s",
			QueryRate:         47.6,
			AvgEventsPerQuery: 85.0,
			MemoryUsageMB:     245.6,
			P50Latency:        "15ms",
			P95Latency:        "45ms",
			P99Latency:        "120ms",
			StartupTime:       "1.2s",
			Errors:            0,
			Timestamp:         time.Now(),
		},
	}

	generator.report.Title = title
	generator.report.RelayData = sampleData
	generator.analyzePerfomance()
	generator.detectAnomalies()
	generator.generateRecommendations()

	ext := format
	if format == "markdown" {
		ext = "md"
	}

	outputFile := fmt.Sprintf("%s.%s", filename, ext)
	if err := SaveReportToFile(outputFile, format, generator); chk.E(err) {
		fmt.Fprintf(os.Stderr, "Failed to save report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Report saved to: %s\n", outputFile)

	if format == "markdown" {
		fmt.Printf("\nTIP: View with: cat %s\n", outputFile)
	}
}
