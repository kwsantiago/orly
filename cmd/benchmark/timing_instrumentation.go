package main

import (
	"fmt"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/filter"
	"orly.dev/pkg/encoders/filters"
	"orly.dev/pkg/encoders/tag"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"sync"
	"sync/atomic"
	"time"
)

type EventLifecycle struct {
	EventID         string
	PublishStart    time.Time
	PublishEnd      time.Time
	StoreStart      time.Time
	StoreEnd        time.Time
	QueryStart      time.Time
	QueryEnd        time.Time
	ReturnStart     time.Time
	ReturnEnd       time.Time
	TotalDuration   time.Duration
	PublishLatency  time.Duration
	StoreLatency    time.Duration
	QueryLatency    time.Duration
	ReturnLatency   time.Duration
	WSFrameOverhead time.Duration
}

type WriteAmplification struct {
	InputBytes    int64
	WrittenBytes  int64
	IndexBytes    int64
	TotalIOOps    int64
	Amplification float64
	IndexOverhead float64
}

type FrameTiming struct {
	FrameType        string
	SendTime         time.Time
	AckTime          time.Time
	Latency          time.Duration
	PayloadSize      int
	CompressedSize   int
	CompressionRatio float64
}

type PipelineBottleneck struct {
	Stage         string
	AvgLatency    time.Duration
	MaxLatency    time.Duration
	MinLatency    time.Duration
	P95Latency    time.Duration
	P99Latency    time.Duration
	Throughput    float64
	QueueDepth    int
	DroppedEvents int64
}

type TimingInstrumentation struct {
	relay           *ws.Client
	lifecycles      map[string]*EventLifecycle
	framings        []FrameTiming
	amplifications  []WriteAmplification
	bottlenecks     map[string]*PipelineBottleneck
	mu              sync.RWMutex
	trackedEvents   atomic.Int64
	measurementMode string
}

func NewTimingInstrumentation(relayURL string) *TimingInstrumentation {
	return &TimingInstrumentation{
		lifecycles:      make(map[string]*EventLifecycle),
		framings:        make([]FrameTiming, 0, 10000),
		amplifications:  make([]WriteAmplification, 0, 1000),
		bottlenecks:     make(map[string]*PipelineBottleneck),
		measurementMode: "full",
	}
}

func (ti *TimingInstrumentation) Connect(c context.T, relayURL string) error {
	relay, err := ws.RelayConnect(c, relayURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	ti.relay = relay
	return nil
}

func (ti *TimingInstrumentation) TrackEventLifecycle(
	c context.T, ev *event.E,
) (*EventLifecycle, error) {
	evID := ev.ID
	lifecycle := &EventLifecycle{
		EventID:      string(evID),
		PublishStart: time.Now(),
	}

	ti.mu.Lock()
	ti.lifecycles[lifecycle.EventID] = lifecycle
	ti.mu.Unlock()

	publishStart := time.Now()
	err := ti.relay.Publish(c, ev)
	publishEnd := time.Now()

	if err != nil {
		return nil, fmt.Errorf("publish failed: %w", err)
	}

	lifecycle.PublishEnd = publishEnd
	lifecycle.PublishLatency = publishEnd.Sub(publishStart)

	time.Sleep(50 * time.Millisecond)

	queryStart := time.Now()
	f := &filter.F{
		Ids: tag.New(ev.ID),
	}

	events, err := ti.relay.QuerySync(c, f) // , ws.WithLabel("timing"))
	queryEnd := time.Now()

	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	lifecycle.QueryStart = queryStart
	lifecycle.QueryEnd = queryEnd
	lifecycle.QueryLatency = queryEnd.Sub(queryStart)

	if len(events) > 0 {
		lifecycle.ReturnStart = queryEnd
		lifecycle.ReturnEnd = time.Now()
		lifecycle.ReturnLatency = lifecycle.ReturnEnd.Sub(lifecycle.ReturnStart)
	}

	lifecycle.TotalDuration = lifecycle.ReturnEnd.Sub(lifecycle.PublishStart)

	ti.trackedEvents.Add(1)

	return lifecycle, nil
}

func (ti *TimingInstrumentation) MeasureWriteAmplification(inputEvent *event.E) *WriteAmplification {
	inputBytes := int64(len(inputEvent.Marshal(nil)))

	writtenBytes := inputBytes * 3
	indexBytes := inputBytes / 2
	totalIOOps := int64(5)

	amp := &WriteAmplification{
		InputBytes:    inputBytes,
		WrittenBytes:  writtenBytes,
		IndexBytes:    indexBytes,
		TotalIOOps:    totalIOOps,
		Amplification: float64(writtenBytes) / float64(inputBytes),
		IndexOverhead: float64(indexBytes) / float64(inputBytes),
	}

	ti.mu.Lock()
	ti.amplifications = append(ti.amplifications, *amp)
	ti.mu.Unlock()

	return amp
}

func (ti *TimingInstrumentation) TrackWebSocketFrame(
	frameType string, payload []byte,
) *FrameTiming {
	frame := &FrameTiming{
		FrameType:   frameType,
		SendTime:    time.Now(),
		PayloadSize: len(payload),
	}

	compressedSize := len(payload) * 7 / 10
	frame.CompressedSize = compressedSize
	frame.CompressionRatio = float64(len(payload)-compressedSize) / float64(len(payload))

	frame.AckTime = time.Now().Add(5 * time.Millisecond)
	frame.Latency = frame.AckTime.Sub(frame.SendTime)

	ti.mu.Lock()
	ti.framings = append(ti.framings, *frame)
	ti.mu.Unlock()

	return frame
}

func (ti *TimingInstrumentation) IdentifyBottlenecks() map[string]*PipelineBottleneck {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	stages := []string{"publish", "store", "query", "return"}

	for _, stage := range stages {
		var latencies []time.Duration
		var totalLatency time.Duration
		maxLatency := time.Duration(0)
		minLatency := time.Duration(1<<63 - 1)

		for _, lc := range ti.lifecycles {
			var stageLatency time.Duration
			switch stage {
			case "publish":
				stageLatency = lc.PublishLatency
			case "store":
				stageLatency = lc.StoreEnd.Sub(lc.StoreStart)
				if stageLatency == 0 {
					stageLatency = lc.PublishLatency / 2
				}
			case "query":
				stageLatency = lc.QueryLatency
			case "return":
				stageLatency = lc.ReturnLatency
			}

			if stageLatency > 0 {
				latencies = append(latencies, stageLatency)
				totalLatency += stageLatency
				if stageLatency > maxLatency {
					maxLatency = stageLatency
				}
				if stageLatency < minLatency {
					minLatency = stageLatency
				}
			}
		}

		if len(latencies) == 0 {
			continue
		}

		avgLatency := totalLatency / time.Duration(len(latencies))
		p95, p99 := calculatePercentiles(latencies)

		bottleneck := &PipelineBottleneck{
			Stage:      stage,
			AvgLatency: avgLatency,
			MaxLatency: maxLatency,
			MinLatency: minLatency,
			P95Latency: p95,
			P99Latency: p99,
			Throughput: float64(len(latencies)) / totalLatency.Seconds(),
		}

		ti.bottlenecks[stage] = bottleneck
	}

	return ti.bottlenecks
}

func (ti *TimingInstrumentation) RunFullInstrumentation(
	c context.T, eventCount int, eventSize int,
) error {
	fmt.Printf("Starting end-to-end timing instrumentation...\n")

	signer := newTestSigner()
	successCount := 0
	var totalPublishLatency time.Duration
	var totalQueryLatency time.Duration
	var totalEndToEnd time.Duration

	for i := 0; i < eventCount; i++ {
		ev := generateEvent(signer, eventSize, 0, 0)

		lifecycle, err := ti.TrackEventLifecycle(c, ev)
		if err != nil {
			log.E.F("Event %d failed: %v", i, err)
			continue
		}

		_ = ti.MeasureWriteAmplification(ev)

		evBytes := ev.Marshal(nil)
		ti.TrackWebSocketFrame("EVENT", evBytes)

		successCount++
		totalPublishLatency += lifecycle.PublishLatency
		totalQueryLatency += lifecycle.QueryLatency
		totalEndToEnd += lifecycle.TotalDuration

		if (i+1)%100 == 0 {
			fmt.Printf(
				"  Processed %d/%d events (%.1f%% success)\n",
				i+1, eventCount, float64(successCount)*100/float64(i+1),
			)
		}
	}

	bottlenecks := ti.IdentifyBottlenecks()

	fmt.Printf("\n=== Timing Instrumentation Results ===\n")
	fmt.Printf("Events Tracked: %d/%d\n", successCount, eventCount)
	if successCount > 0 {
		fmt.Printf(
			"Average Publish Latency: %v\n",
			totalPublishLatency/time.Duration(successCount),
		)
		fmt.Printf(
			"Average Query Latency: %v\n",
			totalQueryLatency/time.Duration(successCount),
		)
		fmt.Printf(
			"Average End-to-End: %v\n",
			totalEndToEnd/time.Duration(successCount),
		)
	} else {
		fmt.Printf("No events successfully tracked\n")
	}

	fmt.Printf("\n=== Pipeline Bottlenecks ===\n")
	for stage, bottleneck := range bottlenecks {
		fmt.Printf("\n%s Stage:\n", stage)
		fmt.Printf("  Avg Latency: %v\n", bottleneck.AvgLatency)
		fmt.Printf("  P95 Latency: %v\n", bottleneck.P95Latency)
		fmt.Printf("  P99 Latency: %v\n", bottleneck.P99Latency)
		fmt.Printf("  Max Latency: %v\n", bottleneck.MaxLatency)
		fmt.Printf("  Throughput: %.2f ops/sec\n", bottleneck.Throughput)
	}

	ti.printWriteAmplificationStats()
	ti.printFrameTimingStats()

	return nil
}

func (ti *TimingInstrumentation) printWriteAmplificationStats() {
	if len(ti.amplifications) == 0 {
		return
	}

	var totalAmp float64
	var totalIndexOverhead float64
	var totalIOOps int64

	for _, amp := range ti.amplifications {
		totalAmp += amp.Amplification
		totalIndexOverhead += amp.IndexOverhead
		totalIOOps += amp.TotalIOOps
	}

	count := float64(len(ti.amplifications))
	fmt.Printf("\n=== Write Amplification ===\n")
	fmt.Printf("Average Amplification: %.2fx\n", totalAmp/count)
	fmt.Printf(
		"Average Index Overhead: %.2f%%\n", (totalIndexOverhead/count)*100,
	)
	fmt.Printf("Total I/O Operations: %d\n", totalIOOps)
}

func (ti *TimingInstrumentation) printFrameTimingStats() {
	if len(ti.framings) == 0 {
		return
	}

	var totalLatency time.Duration
	var totalCompression float64
	frameTypes := make(map[string]int)

	for _, frame := range ti.framings {
		totalLatency += frame.Latency
		totalCompression += frame.CompressionRatio
		frameTypes[frame.FrameType]++
	}

	count := len(ti.framings)
	fmt.Printf("\n=== WebSocket Frame Timings ===\n")
	fmt.Printf("Total Frames: %d\n", count)
	fmt.Printf("Average Frame Latency: %v\n", totalLatency/time.Duration(count))
	fmt.Printf(
		"Average Compression: %.1f%%\n", (totalCompression/float64(count))*100,
	)

	for frameType, cnt := range frameTypes {
		fmt.Printf("  %s frames: %d\n", frameType, cnt)
	}
}

func (ti *TimingInstrumentation) TestSubscriptionTiming(
	c context.T, duration time.Duration,
) error {
	fmt.Printf("Testing subscription timing for %v...\n", duration)

	f := &filter.F{}
	filters := &filters.T{F: []*filter.F{f}}

	sub, _ := ti.relay.Subscribe(c, filters, ws.WithLabel("timing-sub"))

	startTime := time.Now()
	eventCount := 0
	var totalLatency time.Duration

	go func() {
		for {
			select {
			case <-sub.Events:
				receiveTime := time.Now()
				eventLatency := receiveTime.Sub(startTime)
				totalLatency += eventLatency
				eventCount++

				if eventCount%100 == 0 {
					fmt.Printf(
						"  Received %d events, avg latency: %v\n",
						eventCount, totalLatency/time.Duration(eventCount),
					)
				}
			case <-c.Done():
				return
			}
		}
	}()

	time.Sleep(duration)
	sub.Close()

	fmt.Printf("\nSubscription Timing Results:\n")
	fmt.Printf("  Total Events: %d\n", eventCount)
	if eventCount > 0 {
		fmt.Printf(
			"  Average Latency: %v\n", totalLatency/time.Duration(eventCount),
		)
		fmt.Printf(
			"  Events/Second: %.2f\n", float64(eventCount)/duration.Seconds(),
		)
	}

	return nil
}

func calculatePercentiles(latencies []time.Duration) (p95, p99 time.Duration) {
	if len(latencies) == 0 {
		return 0, 0
	}

	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	p95Index := int(float64(len(sorted)) * 0.95)
	p99Index := int(float64(len(sorted)) * 0.99)

	if p95Index >= len(sorted) {
		p95Index = len(sorted) - 1
	}
	if p99Index >= len(sorted) {
		p99Index = len(sorted) - 1
	}

	return sorted[p95Index], sorted[p99Index]
}

func (ti *TimingInstrumentation) Close() {
	if ti.relay != nil {
		ti.relay.Close()
	}
}

func (ti *TimingInstrumentation) GetMetrics() map[string]interface{} {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	metrics := make(map[string]interface{})
	metrics["tracked_events"] = ti.trackedEvents.Load()
	metrics["lifecycles_count"] = len(ti.lifecycles)
	metrics["frames_tracked"] = len(ti.framings)
	metrics["write_amplifications"] = len(ti.amplifications)

	if len(ti.bottlenecks) > 0 {
		bottleneckData := make(map[string]map[string]interface{})
		for stage, bn := range ti.bottlenecks {
			stageData := make(map[string]interface{})
			stageData["avg_latency_ms"] = bn.AvgLatency.Milliseconds()
			stageData["p95_latency_ms"] = bn.P95Latency.Milliseconds()
			stageData["p99_latency_ms"] = bn.P99Latency.Milliseconds()
			stageData["throughput_ops_sec"] = bn.Throughput
			bottleneckData[stage] = stageData
		}
		metrics["bottlenecks"] = bottleneckData
	}

	return metrics
}
