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
	"orly.dev/pkg/encoders/text"
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
			"relay", "ws://localhost:7447", "Relay URL to benchmark",
		)
		eventCount = flag.Int("events", 10000, "Number of events to publish")
		eventSize  = flag.Int(
			"size", 1024, "Average size of event content in bytes",
		)
		concurrency = flag.Int(
			"concurrency", 10, "Number of concurrent publishers",
		)
		queryCount  = flag.Int("queries", 100, "Number of queries to execute")
		queryLimit  = flag.Int("query-limit", 100, "Limit for each query")
		skipPublish = flag.Bool("skip-publish", false, "Skip publishing phase")
		skipQuery   = flag.Bool("skip-query", false, "Skip query phase")
		verbose     = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	if *verbose {
		lol.SetLogLevel("trace")
	}

	c := context.Bg()
	results := &BenchmarkResults{}

	// Phase 1: Publish events
	if !*skipPublish {
		fmt.Printf("Publishing %d events to %s...\n", *eventCount, *relayURL)
		if err := benchmarkPublish(
			c, *relayURL, *eventCount, *eventSize, *concurrency, results,
		); chk.E(err) {
			fmt.Fprintf(os.Stderr, "Error during publish benchmark: %v\n", err)
			os.Exit(1)
		}
	}

	// Phase 2: Query events
	if !*skipQuery {
		fmt.Printf("\nQuerying events from %s...\n", *relayURL)
		if err := benchmarkQuery(
			c, *relayURL, *queryCount, *queryLimit, results,
		); chk.E(err) {
			fmt.Fprintf(os.Stderr, "Error during query benchmark: %v\n", err)
			os.Exit(1)
		}
	}

	// Print results
	printResults(results)
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
				ev := generateEvent(signer, eventSize)

				if err := relay.Publish(c, ev); err != nil {
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

func generateEvent(signer *testSigner, contentSize int) *event.E {
	// Generate content with some variation
	size := contentSize + frand.Intn(contentSize/2) - contentSize/4
	if size < 10 {
		size = 10
	}

	content := text.NostrEscape(nil, frand.Bytes(size))

	ev := &event.E{
		Pubkey:    signer.Pub(),
		Kind:      kind.TextNote,
		CreatedAt: timestamp.Now(),
		Content:   content,
		Tags:      generateRandomTags(),
	}

	if err := ev.Sign(signer); chk.E(err) {
		panic(fmt.Sprintf("failed to sign event: %v", err))
	}

	return ev
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
