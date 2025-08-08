package main

import (
	"fmt"
	"orly.dev/pkg/protocol/ws"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/log"
	"os/exec"
	"sync"
	"time"
)

type RelayType int

const (
	Khatru RelayType = iota
	Relayer
	Strfry
	RustNostr
)

func (r RelayType) String() string {
	switch r {
	case Khatru:
		return "khatru"
	case Relayer:
		return "relayer"
	case Strfry:
		return "strfry"
	case RustNostr:
		return "rust-nostr"
	default:
		return "unknown"
	}
}

type RelayConfig struct {
	Type    RelayType
	Binary  string
	Args    []string
	URL     string
	DataDir string
}

type RelayInstance struct {
	Config  RelayConfig
	Process *exec.Cmd
	Started time.Time
	Errors  []error
	mu      sync.RWMutex
}

type HarnessMetrics struct {
	StartupTime  time.Duration
	ShutdownTime time.Duration
	Errors       int
}

type MultiRelayHarness struct {
	relays  map[RelayType]*RelayInstance
	metrics map[RelayType]*HarnessMetrics
	mu      sync.RWMutex
}

func NewMultiRelayHarness() *MultiRelayHarness {
	return &MultiRelayHarness{
		relays:  make(map[RelayType]*RelayInstance),
		metrics: make(map[RelayType]*HarnessMetrics),
	}
}

func (h *MultiRelayHarness) AddRelay(config RelayConfig) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	instance := &RelayInstance{
		Config: config,
		Errors: make([]error, 0),
	}

	h.relays[config.Type] = instance
	h.metrics[config.Type] = &HarnessMetrics{}

	return nil
}

func (h *MultiRelayHarness) StartRelay(relayType RelayType) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	instance, exists := h.relays[relayType]
	if !exists {
		return fmt.Errorf("relay type %s not configured", relayType)
	}

	if instance.Process != nil {
		return fmt.Errorf("relay %s already running", relayType)
	}

	startTime := time.Now()
	cmd := exec.Command(instance.Config.Binary, instance.Config.Args...)

	if err := cmd.Start(); chk.E(err) {
		return fmt.Errorf("failed to start %s: %w", relayType, err)
	}

	instance.Process = cmd
	instance.Started = startTime

	time.Sleep(100 * time.Millisecond)

	metrics := h.metrics[relayType]
	metrics.StartupTime = time.Since(startTime)

	return nil
}

func (h *MultiRelayHarness) StopRelay(relayType RelayType) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	instance, exists := h.relays[relayType]
	if !exists {
		return fmt.Errorf("relay type %s not configured", relayType)
	}

	if instance.Process == nil {
		return nil
	}

	shutdownStart := time.Now()

	if err := instance.Process.Process.Kill(); chk.E(err) {
		return fmt.Errorf("failed to stop %s: %w", relayType, err)
	}

	instance.Process.Wait()
	instance.Process = nil

	metrics := h.metrics[relayType]
	metrics.ShutdownTime = time.Since(shutdownStart)

	return nil
}

func (h *MultiRelayHarness) ConnectToRelay(c context.T, relayType RelayType) error {
	h.mu.RLock()
	instance, exists := h.relays[relayType]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("relay type %s not configured", relayType)
	}

	if instance.Process == nil {
		return fmt.Errorf("relay %s not running", relayType)
	}

	_, err := ws.RelayConnect(c, instance.Config.URL)
	if chk.E(err) {
		h.mu.Lock()
		h.metrics[relayType].Errors++
		instance.Errors = append(instance.Errors, err)
		h.mu.Unlock()
		return fmt.Errorf("failed to connect to %s: %w", relayType, err)
	}

	return nil
}

func (h *MultiRelayHarness) StartAll() error {
	h.mu.RLock()
	relayTypes := make([]RelayType, 0, len(h.relays))
	for relayType := range h.relays {
		relayTypes = append(relayTypes, relayType)
	}
	h.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(relayTypes))

	for _, relayType := range relayTypes {
		wg.Add(1)
		go func(rt RelayType) {
			defer wg.Done()
			if err := h.StartRelay(rt); err != nil {
				errChan <- fmt.Errorf("failed to start %s: %w", rt, err)
			}
		}(relayType)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		for _, err := range errors {
			log.E.Ln(err)
		}
		return fmt.Errorf("failed to start %d relays", len(errors))
	}

	return nil
}

func (h *MultiRelayHarness) StopAll() error {
	h.mu.RLock()
	relayTypes := make([]RelayType, 0, len(h.relays))
	for relayType := range h.relays {
		relayTypes = append(relayTypes, relayType)
	}
	h.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(relayTypes))

	for _, relayType := range relayTypes {
		wg.Add(1)
		go func(rt RelayType) {
			defer wg.Done()
			if err := h.StopRelay(rt); err != nil {
				errChan <- fmt.Errorf("failed to stop %s: %w", rt.String(), err)
			}
		}(relayType)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		for _, err := range errors {
			log.E.Ln(err)
		}
		return fmt.Errorf("failed to stop %d relays", len(errors))
	}

	return nil
}

func (h *MultiRelayHarness) GetMetrics(relayType RelayType) *HarnessMetrics {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.metrics[relayType]
}

func (h *MultiRelayHarness) GetAllMetrics() map[RelayType]*HarnessMetrics {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[RelayType]*HarnessMetrics)
	for relayType, metrics := range h.metrics {
		result[relayType] = metrics
	}
	return result
}

func (h *MultiRelayHarness) IsRunning(relayType RelayType) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	instance, exists := h.relays[relayType]
	return exists && instance.Process != nil
}

func (h *MultiRelayHarness) GetErrors(relayType RelayType) []error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	instance, exists := h.relays[relayType]
	if !exists {
		return nil
	}

	return append([]error(nil), instance.Errors...)
}
