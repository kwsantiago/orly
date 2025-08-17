package main

import (
	"flag"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/lol"
)

func main() {
	var (
		relayURL    = flag.String("relay", "ws://localhost:3334", "Relay URL")
		eventCount  = flag.Int("events", 10000, "Number of events")
		queryCount  = flag.Int("queries", 100, "Number of queries")
		concurrency = flag.Int("concurrency", 10, "Concurrent publishers")
		verbose     = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	if *verbose {
		lol.SetLogLevel("trace")
	}

	c := context.Bg()

	if *eventCount > 0 {
		fmt.Printf("Publishing %d events...\n", *eventCount)
		publishEvents(c, *relayURL, *eventCount, *concurrency)
	}

	if *queryCount > 0 {
		fmt.Printf("Executing %d queries...\n", *queryCount)
		runQueries(c, *relayURL, *queryCount)
	}
}

func publishEvents(c context.T, relayURL string, eventCount, concurrency int) {
	signers := make([]*testSigner, concurrency)
	for i := range signers {
		signers[i] = newTestSigner()
	}

	var published atomic.Int64
	var errors atomic.Int64
	var wg sync.WaitGroup

	eventsPerPublisher := eventCount / concurrency
	extraEvents := eventCount % concurrency
	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			relay, err := ws.RelayConnect(c, relayURL)
			if err != nil {
				log.E.F("Failed to connect: %v", err)
				errors.Add(1)
				return
			}
			defer relay.Close()

			count := eventsPerPublisher
			if id < extraEvents {
				count++
			}

			for j := 0; j < count; j++ {
				ev := generateSimpleEvent(signers[id], 1024)
				if err := relay.Publish(c, ev); err != nil {
					errors.Add(1)
					continue
				}
				published.Add(1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)
	rate := float64(published.Load()) / duration.Seconds()
	fmt.Printf("  Published: %d\n", published.Load())
	fmt.Printf("  Duration: %.2fs\n", duration.Seconds())
	fmt.Printf("  Rate: %.2f events/s\n", rate)
	if errors.Load() > 0 {
		fmt.Printf("  Errors: %d\n", errors.Load())
	}
}

func runQueries(c context.T, relayURL string, queryCount int) {
	relay, err := ws.RelayConnect(c, relayURL)
	if err != nil {
		log.E.F("Failed to connect: %v", err)
		return
	}
	defer relay.Close()

	var totalEvents atomic.Int64
	startTime := time.Now()

	for i := 0; i < queryCount; i++ {
		f := generateQueryFilter(i)
		events, err := relay.QuerySync(c, f)
		if err != nil {
			continue
		}
		totalEvents.Add(int64(len(events)))
	}

	duration := time.Since(startTime)
	rate := float64(queryCount) / duration.Seconds()
	fmt.Printf("  Executed: %d\n", queryCount)
	fmt.Printf("  Duration: %.2fs\n", duration.Seconds())
	fmt.Printf("  Rate: %.2f queries/s\n", rate)
	fmt.Printf("  Events returned: %d\n", totalEvents.Load())
}

func generateQueryFilter(index int) *filter.F {
	limit := uint(100)
	switch index % 5 {
	case 0:
		// Query all events by kind
		return &filter.F{Kinds: kinds.New(kind.TextNote), Limit: &limit}
	case 1:
		// Query recent events
		now := timestamp.Now()
		since := timestamp.New(now.I64() - 3600)
		return &filter.F{Since: since, Until: now, Limit: &limit}
	case 2:
		// Query all events (no filter)
		return &filter.F{Limit: &limit}
	case 3:
		// Query by multiple kinds
		return &filter.F{Kinds: kinds.New(kind.TextNote, kind.Repost, kind.Reaction), Limit: &limit}
	default:
		// Query older events
		now := timestamp.Now()
		until := timestamp.New(now.I64() - 1800)
		since := timestamp.New(now.I64() - 7200)
		return &filter.F{
			Since: since,
			Until: until,
			Limit: &limit,
		}
	}
}