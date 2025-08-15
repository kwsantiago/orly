package main

import (
	"fmt"
	"lukechampine.com/frand"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/kinds"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type QueryMetrics struct {
	Latencies      []time.Duration
	TotalQueries   int64
	FailedQueries  int64
	EventsReturned int64
	MemoryBefore   uint64
	MemoryAfter    uint64
	MemoryPeak     uint64
	P50            time.Duration
	P95            time.Duration
	P99            time.Duration
	Min            time.Duration
	Max            time.Duration
	Mean           time.Duration
}

type FilterType int

const (
	SimpleKindFilter FilterType = iota
	TimeRangeFilter
	AuthorFilter
	TagFilter
	ComplexFilter
	IDFilter
	PrefixFilter
	MultiKindFilter
	LargeTagSetFilter
	DeepTimeRangeFilter
)

type QueryProfiler struct {
	relay          string
	subscriptions  map[string]*ws.Subscription
	metrics        *QueryMetrics
	mu             sync.RWMutex
	memTicker      *time.Ticker
	stopMemMonitor chan struct{}
}

func NewQueryProfiler(relayURL string) *QueryProfiler {
	return &QueryProfiler{
		relay:         relayURL,
		subscriptions: make(map[string]*ws.Subscription),
		metrics: &QueryMetrics{
			Latencies: make(
				[]time.Duration, 0, 10000,
			),
		},
		stopMemMonitor: make(chan struct{}),
	}
}

func (qp *QueryProfiler) ExecuteProfile(
	c context.T, iterations int, concurrency int,
) error {
	qp.startMemoryMonitor()
	defer qp.stopMemoryMonitor()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	qp.metrics.MemoryBefore = m.Alloc

	filterTypes := []FilterType{
		SimpleKindFilter,
		TimeRangeFilter,
		AuthorFilter,
		TagFilter,
		ComplexFilter,
		IDFilter,
		PrefixFilter,
		MultiKindFilter,
		LargeTagSetFilter,
		DeepTimeRangeFilter,
	}

	var wg sync.WaitGroup
	latencyChan := make(chan time.Duration, iterations)
	errorChan := make(chan error, iterations)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			relay, err := ws.RelayConnect(c, qp.relay)
			if chk.E(err) {
				errorChan <- fmt.Errorf(
					"worker %d connection failed: %w", workerID, err,
				)
				return
			}
			defer relay.Close()

			iterationsPerWorker := iterations / concurrency
			if workerID == 0 {
				iterationsPerWorker += iterations % concurrency
			}

			for j := 0; j < iterationsPerWorker; j++ {
				filterType := filterTypes[frand.Intn(len(filterTypes))]
				f := qp.generateFilter(filterType)

				startTime := time.Now()
				events, err := relay.QuerySync(
					c, f,
				) // , ws.WithLabel(fmt.Sprintf("profiler-%d-%d", workerID, j)))
				latency := time.Since(startTime)

				if err != nil {
					errorChan <- err
					atomic.AddInt64(&qp.metrics.FailedQueries, 1)
				} else {
					latencyChan <- latency
					atomic.AddInt64(
						&qp.metrics.EventsReturned, int64(len(events)),
					)
					atomic.AddInt64(&qp.metrics.TotalQueries, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	close(latencyChan)
	close(errorChan)

	for latency := range latencyChan {
		qp.mu.Lock()
		qp.metrics.Latencies = append(qp.metrics.Latencies, latency)
		qp.mu.Unlock()
	}

	errorCount := 0
	for range errorChan {
		errorCount++
	}

	runtime.ReadMemStats(&m)
	qp.metrics.MemoryAfter = m.Alloc

	qp.calculatePercentiles()

	return nil
}

func (qp *QueryProfiler) generateFilter(filterType FilterType) *filter.F {
	switch filterType {
	case SimpleKindFilter:
		limit := uint(100)
		return &filter.F{
			Kinds: kinds.New(kind.TextNote),
			Limit: &limit,
		}

	case TimeRangeFilter:
		now := timestamp.Now()
		since := timestamp.New(now.I64() - 3600)
		limit := uint(50)
		return &filter.F{
			Since: since,
			Until: now,
			Limit: &limit,
		}

	case AuthorFilter:
		limit := uint(100)
		authors := tag.New(frand.Bytes(32))
		for i := 0; i < 2; i++ {
			authors.Append(frand.Bytes(32))
		}
		return &filter.F{
			Authors: authors,
			Limit:   &limit,
		}

	case TagFilter:
		limit := uint(50)
		t := tags.New()
		t.AppendUnique(tag.New([]byte("p"), frand.Bytes(32)))
		t.AppendUnique(tag.New([]byte("e"), frand.Bytes(32)))
		return &filter.F{
			Tags:  t,
			Limit: &limit,
		}

	case ComplexFilter:
		now := timestamp.Now()
		since := timestamp.New(now.I64() - 7200)
		limit := uint(25)
		authors := tag.New(frand.Bytes(32))
		return &filter.F{
			Kinds:   kinds.New(kind.TextNote, kind.Repost, kind.Reaction),
			Authors: authors,
			Since:   since,
			Until:   now,
			Limit:   &limit,
		}

	case IDFilter:
		limit := uint(10)
		ids := tag.New(frand.Bytes(32))
		for i := 0; i < 4; i++ {
			ids.Append(frand.Bytes(32))
		}
		return &filter.F{
			Ids:   ids,
			Limit: &limit,
		}

	case PrefixFilter:
		limit := uint(100)
		prefix := frand.Bytes(4)
		return &filter.F{
			Ids:   tag.New(prefix),
			Limit: &limit,
		}

	case MultiKindFilter:
		limit := uint(75)
		return &filter.F{
			Kinds: kinds.New(
				kind.TextNote,
				kind.SetMetadata,
				kind.FollowList,
				kind.Reaction,
				kind.Repost,
			),
			Limit: &limit,
		}

	case LargeTagSetFilter:
		limit := uint(20)
		t := tags.New()
		for i := 0; i < 10; i++ {
			t.AppendUnique(tag.New([]byte("p"), frand.Bytes(32)))
		}
		return &filter.F{
			Tags:  t,
			Limit: &limit,
		}

	case DeepTimeRangeFilter:
		now := timestamp.Now()
		since := timestamp.New(now.I64() - 86400*30)
		until := timestamp.New(now.I64() - 86400*20)
		limit := uint(100)
		return &filter.F{
			Since: since,
			Until: until,
			Limit: &limit,
		}

	default:
		limit := uint(100)
		return &filter.F{
			Kinds: kinds.New(kind.TextNote),
			Limit: &limit,
		}
	}
}

func (qp *QueryProfiler) TestSubscriptionPerformance(
	c context.T, duration time.Duration, subscriptionCount int,
) error {
	qp.startMemoryMonitor()
	defer qp.stopMemoryMonitor()

	relay, err := ws.RelayConnect(c, qp.relay)
	if chk.E(err) {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer relay.Close()

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	for i := 0; i < subscriptionCount; i++ {
		wg.Add(1)
		go func(subID int) {
			defer wg.Done()

			f := qp.generateFilter(FilterType(subID % 10))
			label := fmt.Sprintf("sub-perf-%d", subID)

			eventChan := make(chan *event.E, 100)
			sub, err := relay.Subscribe(
				c, &filters.T{F: []*filter.F{f}}, ws.WithLabel(label),
			)
			if chk.E(err) {
				return
			}

			go func() {
				for {
					select {
					case ev := <-sub.Events:
						eventChan <- ev
						atomic.AddInt64(&qp.metrics.EventsReturned, 1)
					case <-stopChan:
						sub.Unsub()
						return
					}
				}
			}()

			qp.mu.Lock()
			qp.subscriptions[label] = sub
			qp.mu.Unlock()
		}(i)
	}

	time.Sleep(duration)
	close(stopChan)
	wg.Wait()

	return nil
}

func (qp *QueryProfiler) startMemoryMonitor() {
	qp.memTicker = time.NewTicker(100 * time.Millisecond)
	go func() {
		for {
			select {
			case <-qp.memTicker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				qp.mu.Lock()
				if m.Alloc > qp.metrics.MemoryPeak {
					qp.metrics.MemoryPeak = m.Alloc
				}
				qp.mu.Unlock()
			case <-qp.stopMemMonitor:
				return
			}
		}
	}()
}

func (qp *QueryProfiler) stopMemoryMonitor() {
	if qp.memTicker != nil {
		qp.memTicker.Stop()
	}
	close(qp.stopMemMonitor)
}

func (qp *QueryProfiler) calculatePercentiles() {
	qp.mu.Lock()
	defer qp.mu.Unlock()

	if len(qp.metrics.Latencies) == 0 {
		return
	}

	sort.Slice(
		qp.metrics.Latencies, func(i, j int) bool {
			return qp.metrics.Latencies[i] < qp.metrics.Latencies[j]
		},
	)

	qp.metrics.Min = qp.metrics.Latencies[0]
	qp.metrics.Max = qp.metrics.Latencies[len(qp.metrics.Latencies)-1]

	p50Index := len(qp.metrics.Latencies) * 50 / 100
	p95Index := len(qp.metrics.Latencies) * 95 / 100
	p99Index := len(qp.metrics.Latencies) * 99 / 100

	if p50Index < len(qp.metrics.Latencies) {
		qp.metrics.P50 = qp.metrics.Latencies[p50Index]
	}
	if p95Index < len(qp.metrics.Latencies) {
		qp.metrics.P95 = qp.metrics.Latencies[p95Index]
	}
	if p99Index < len(qp.metrics.Latencies) {
		qp.metrics.P99 = qp.metrics.Latencies[p99Index]
	}

	var total time.Duration
	for _, latency := range qp.metrics.Latencies {
		total += latency
	}
	qp.metrics.Mean = total / time.Duration(len(qp.metrics.Latencies))
}

func (qp *QueryProfiler) GetMetrics() *QueryMetrics {
	qp.mu.RLock()
	defer qp.mu.RUnlock()
	return qp.metrics
}

func (qp *QueryProfiler) PrintReport() {
	metrics := qp.GetMetrics()

	fmt.Println("\n=== Query Performance Profile ===")
	fmt.Printf("Total Queries: %d\n", metrics.TotalQueries)
	fmt.Printf("Failed Queries: %d\n", metrics.FailedQueries)
	fmt.Printf("Events Returned: %d\n", metrics.EventsReturned)

	if metrics.TotalQueries > 0 {
		fmt.Println("\nLatency Percentiles:")
		fmt.Printf("  P50: %v\n", metrics.P50)
		fmt.Printf("  P95: %v\n", metrics.P95)
		fmt.Printf("  P99: %v\n", metrics.P99)
		fmt.Printf("  Min: %v\n", metrics.Min)
		fmt.Printf("  Max: %v\n", metrics.Max)
		fmt.Printf("  Mean: %v\n", metrics.Mean)
	}

	fmt.Println("\nMemory Usage:")
	fmt.Printf("  Before: %.2f MB\n", float64(metrics.MemoryBefore)/1024/1024)
	fmt.Printf("  After: %.2f MB\n", float64(metrics.MemoryAfter)/1024/1024)
	fmt.Printf("  Peak: %.2f MB\n", float64(metrics.MemoryPeak)/1024/1024)
	fmt.Printf(
		"  Delta: %.2f MB\n",
		float64(int64(metrics.MemoryAfter)-int64(metrics.MemoryBefore))/1024/1024,
	)
}
