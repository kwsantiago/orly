// +build ignore

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

// Simple event structure for benchmarking
type Event struct {
	ID        string     `json:"id"`
	Pubkey    string     `json:"pubkey"`
	CreatedAt int64      `json:"created_at"`
	Kind      int        `json:"kind"`
	Tags      [][]string `json:"tags"`
	Content   string     `json:"content"`
	Sig       string     `json:"sig"`
}

// Generate a test event
func generateTestEvent(size int) *Event {
	content := make([]byte, size)
	rand.Read(content)
	
	// Generate random pubkey and sig
	pubkey := make([]byte, 32)
	sig := make([]byte, 64)
	rand.Read(pubkey)
	rand.Read(sig)
	
	ev := &Event{
		Pubkey:    hex.EncodeToString(pubkey),
		CreatedAt: time.Now().Unix(),
		Kind:      1,
		Tags:      [][]string{},
		Content:   string(content),
		Sig:       hex.EncodeToString(sig),
	}
	
	// Generate ID (simplified)
	serialized, _ := json.Marshal([]interface{}{
		0,
		ev.Pubkey,
		ev.CreatedAt,
		ev.Kind,
		ev.Tags,
		ev.Content,
	})
	hash := sha256.Sum256(serialized)
	ev.ID = hex.EncodeToString(hash[:])
	
	return ev
}

func publishEvents(relayURL string, count int, size int, concurrency int) (int64, int64, time.Duration, error) {
	u, err := url.Parse(relayURL)
	if err != nil {
		return 0, 0, 0, err
	}

	var publishedEvents atomic.Int64
	var publishedBytes atomic.Int64
	var wg sync.WaitGroup
	
	eventsPerWorker := count / concurrency
	extraEvents := count % concurrency
	
	start := time.Now()
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		eventsToPublish := eventsPerWorker
		if i < extraEvents {
			eventsToPublish++
		}
		
		go func(workerID int, eventCount int) {
			defer wg.Done()
			
			// Connect to relay
			ctx := context.Background()
			conn, _, _, err := ws.Dial(ctx, u.String())
			if err != nil {
				log.Printf("Worker %d: connection error: %v", workerID, err)
				return
			}
			defer conn.Close()
			
			// Publish events
			for j := 0; j < eventCount; j++ {
				ev := generateTestEvent(size)
				
				// Create EVENT message
				msg, _ := json.Marshal([]interface{}{"EVENT", ev})
				
				err := wsutil.WriteClientMessage(conn, ws.OpText, msg)
				if err != nil {
					log.Printf("Worker %d: write error: %v", workerID, err)
					continue
				}
				
				publishedEvents.Add(1)
				publishedBytes.Add(int64(len(msg)))
				
				// Read response (OK or error)
				_, _, err = wsutil.ReadServerData(conn)
				if err != nil {
					log.Printf("Worker %d: read error: %v", workerID, err)
				}
			}
		}(i, eventsToPublish)
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	return publishedEvents.Load(), publishedBytes.Load(), duration, nil
}

func queryEvents(relayURL string, queries int, limit int) (int64, int64, time.Duration, error) {
	u, err := url.Parse(relayURL)
	if err != nil {
		return 0, 0, 0, err
	}

	ctx := context.Background()
	conn, _, _, err := ws.Dial(ctx, u.String())
	if err != nil {
		return 0, 0, 0, err
	}
	defer conn.Close()
	
	var totalQueries int64
	var totalEvents int64
	
	start := time.Now()
	
	for i := 0; i < queries; i++ {
		// Generate various filter types
		var filter map[string]interface{}
		
		switch i % 5 {
		case 0:
			// Query by kind
			filter = map[string]interface{}{
				"kinds": []int{1},
				"limit": limit,
			}
		case 1:
			// Query by time range
			now := time.Now().Unix()
			filter = map[string]interface{}{
				"since": now - 3600,
				"until": now,
				"limit": limit,
			}
		case 2:
			// Query by tag
			filter = map[string]interface{}{
				"#p": []string{hex.EncodeToString(randBytes(32))},
				"limit": limit,
			}
		case 3:
			// Query by author
			filter = map[string]interface{}{
				"authors": []string{hex.EncodeToString(randBytes(32))},
				"limit": limit,
			}
		case 4:
			// Complex query
			now := time.Now().Unix()
			filter = map[string]interface{}{
				"kinds":   []int{1, 6},
				"authors": []string{hex.EncodeToString(randBytes(32))},
				"since":   now - 7200,
				"limit":   limit,
			}
		}
		
		// Send REQ
		subID := fmt.Sprintf("bench-%d", i)
		msg, _ := json.Marshal([]interface{}{"REQ", subID, filter})
		
		err := wsutil.WriteClientMessage(conn, ws.OpText, msg)
		if err != nil {
			log.Printf("Query %d: write error: %v", i, err)
			continue
		}
		
		// Read events until EOSE
		eventCount := 0
		for {
			data, err := wsutil.ReadServerText(conn)
			if err != nil {
				log.Printf("Query %d: read error: %v", i, err)
				break
			}
			
			var msg []interface{}
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			
			if len(msg) < 2 {
				continue
			}
			
			msgType, ok := msg[0].(string)
			if !ok {
				continue
			}
			
			switch msgType {
			case "EVENT":
				eventCount++
			case "EOSE":
				goto done
			}
		}
		done:
		
		// Send CLOSE
		closeMsg, _ := json.Marshal([]interface{}{"CLOSE", subID})
		wsutil.WriteClientMessage(conn, ws.OpText, closeMsg)
		
		totalQueries++
		totalEvents += int64(eventCount)
		
		if totalQueries%20 == 0 {
			fmt.Printf("  Executed %d queries...\n", totalQueries)
		}
	}
	
	duration := time.Since(start)
	return totalQueries, totalEvents, duration, nil
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func main() {
	var (
		relayURL     = flag.String("relay", "ws://localhost:7447", "Relay URL to benchmark")
		eventCount   = flag.Int("events", 10000, "Number of events to publish")
		eventSize    = flag.Int("size", 1024, "Average size of event content in bytes")
		concurrency  = flag.Int("concurrency", 10, "Number of concurrent publishers")
		queryCount   = flag.Int("queries", 100, "Number of queries to execute")
		queryLimit   = flag.Int("query-limit", 100, "Limit for each query")
		skipPublish  = flag.Bool("skip-publish", false, "Skip publishing phase")
		skipQuery    = flag.Bool("skip-query", false, "Skip query phase")
	)
	flag.Parse()
	
	fmt.Printf("=== Nostr Relay Benchmark ===\n\n")
	
	// Phase 1: Publish events
	if !*skipPublish {
		fmt.Printf("Publishing %d events to %s...\n", *eventCount, *relayURL)
		published, bytes, duration, err := publishEvents(*relayURL, *eventCount, *eventSize, *concurrency)
		if err != nil {
			log.Fatalf("Publishing failed: %v", err)
		}
		
		fmt.Printf("\nPublish Performance:\n")
		fmt.Printf("  Events Published: %d\n", published)
		fmt.Printf("  Total Data: %.2f MB\n", float64(bytes)/1024/1024)
		fmt.Printf("  Duration: %s\n", duration)
		fmt.Printf("  Rate: %.2f events/second\n", float64(published)/duration.Seconds())
		fmt.Printf("  Bandwidth: %.2f MB/second\n", float64(bytes)/duration.Seconds()/1024/1024)
	}
	
	// Phase 2: Query events
	if !*skipQuery {
		fmt.Printf("\nQuerying events from %s...\n", *relayURL)
		queries, events, duration, err := queryEvents(*relayURL, *queryCount, *queryLimit)
		if err != nil {
			log.Fatalf("Querying failed: %v", err)
		}
		
		fmt.Printf("\nQuery Performance:\n")
		fmt.Printf("  Queries Executed: %d\n", queries)
		fmt.Printf("  Events Returned: %d\n", events)
		fmt.Printf("  Duration: %s\n", duration)
		fmt.Printf("  Rate: %.2f queries/second\n", float64(queries)/duration.Seconds())
		fmt.Printf("  Avg Events/Query: %.2f\n", float64(events)/float64(queries))
	}
}