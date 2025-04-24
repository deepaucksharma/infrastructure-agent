package sampler

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"
)

// Register the TopN sampler at package initialization
func init() {
	RegisterSampler("topn", func() Sampler {
		return NewTopNSampler(DefaultConfig().TopN)
	})
}

// TopNSampler implements a sampler that tracks the top N processes
// based on a configurable scoring function.
type TopNSampler struct {
	config        TopNConfig
	heap          *ProcessHeap
	metrics       map[string]float64
	pidHistory    map[int]bool
	seenPIDs      map[int]time.Time
	churnRate     float64
	lastUpdate    time.Time
	samplerStart  time.Time
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	circuitOpen   bool
	totalCPUUsage float64 // Total CPU usage as percentage
	totalRSSUsage int64   // Total RSS in bytes
}

// NewTopNSampler creates a new TopN sampler with the given configuration.
func NewTopNSampler(config TopNConfig) *TopNSampler {
	return &TopNSampler{
		config:       config,
		heap:         NewProcessHeap(config.MaxProcesses),
		metrics:      make(map[string]float64),
		pidHistory:   make(map[int]bool),
		seenPIDs:     make(map[int]time.Time),
		lastUpdate:   time.Now(),
		samplerStart: time.Now(),
		circuitOpen:  false,
	}
}

// Init initializes the sampler with a context.
func (s *TopNSampler) Init(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.samplerStart = time.Now()

	return nil
}

// Update updates the internal state with new process information.
func (s *TopNSampler) Update(processes []*ProcessInfo) error {
	start := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate churn rate if enabled
	now := time.Now()
	elapsed := now.Sub(s.lastUpdate).Seconds()
	s.lastUpdate = now

	// Track PIDs for churn calculation
	newPIDs := make(map[int]bool, len(processes))
	for _, p := range processes {
		newPIDs[p.PID] = true
		s.seenPIDs[p.PID] = now
	}

	// Calculate churn (PIDs/second)
	if s.config.ChurnHandlingEnabled && elapsed > 0 {
		added := 0
		for pid := range newPIDs {
			if !s.pidHistory[pid] {
				added++
			}
		}
		s.churnRate = 0.7*s.churnRate + 0.3*float64(added)/elapsed
	}

	// Check circuit breaker conditions
	s.checkCircuitBreaker()

	// If circuit breaker is open, reduce processing
	if s.circuitOpen {
		// Process only a subset when under high load
		sampleSize := int(math.Max(10, float64(len(processes))*0.1))
		if sampleSize < len(processes) {
			processes = processes[:sampleSize]
		}
	}

	// Prepare counters for metrics
	totalCPU := 0.0
	totalRSS := int64(0)
	processesUpdated := 0

	// Score and update processes
	for _, p := range processes {
		// Calculate score based on weighted CPU and RSS
		p.Score = s.calculateScore(p)

		// Update process in heap
		if s.heap.Update(p) {
			processesUpdated++
		}

		// Track totals for metrics
		totalCPU += p.CPU
		totalRSS += p.RSS
	}

	// Clean up old PIDs
	if s.config.ChurnHandlingEnabled {
		// Remove PIDs that weren't seen in this update
		for pid := range s.pidHistory {
			if !newPIDs[pid] {
				s.heap.Remove(pid)
				// Only delete from history after some time to handle process cycling
				if lastSeen, ok := s.seenPIDs[pid]; ok && now.Sub(lastSeen) > time.Minute {
					delete(s.seenPIDs, pid)
					delete(s.pidHistory, pid)
				}
			}
		}
	}

	// Update PID history for next churn calculation
	s.pidHistory = newPIDs

	// Store CPU and RSS totals
	s.totalCPUUsage = totalCPU
	s.totalRSSUsage = totalRSS

	// Calculate metrics
	processingTime := time.Since(start).Seconds()
	s.metrics["topn_update_time_seconds"] = processingTime
	s.metrics["topn_processes_tracked"] = float64(s.heap.Len())
	s.metrics["topn_processes_updated"] = float64(processesUpdated)
	s.metrics["topn_churn_rate"] = s.churnRate
	s.metrics["topn_circuit_breaker"] = 0
	if s.circuitOpen {
		s.metrics["topn_circuit_breaker"] = 1
	}

	// Calculate capture ratio (percentage of total resource captured by tracked processes)
	if s.totalCPUUsage > 0 {
		trackedCPU := 0.0
		for _, p := range s.heap.TopN(s.config.MaxProcesses) {
			trackedCPU += p.CPU
		}
		s.metrics["topn_capture_ratio"] = (trackedCPU / s.totalCPUUsage) * 100
	} else {
		s.metrics["topn_capture_ratio"] = 100 // If no CPU usage, we capture 100%
	}

	return nil
}

// GetTopN returns the top N processes according to the sampling strategy.
func (s *TopNSampler) GetTopN(n int) []*ProcessInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.heap.TopN(n)
}

// Metrics returns performance metrics for the sampler.
func (s *TopNSampler) Metrics() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Make a copy to avoid concurrent modification
	metrics := make(map[string]float64, len(s.metrics))
	for k, v := range s.metrics {
		metrics[k] = v
	}
	return metrics
}

// Resources returns resource usage of the sampler itself.
func (s *TopNSampler) Resources() map[string]float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]float64{
		"sampler_cpu_percent": s.metrics["topn_update_time_seconds"] * 100, // Approximation based on update time
		"sampler_rss_bytes":   float64(m.Sys),                              // Total memory obtained from system
		"sampler_uptime_seconds": time.Since(s.samplerStart).Seconds(),
	}
}

// Shutdown gracefully shuts down the sampler.
func (s *TopNSampler) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

// calculateScore computes a score for a process based on CPU and RSS.
func (s *TopNSampler) calculateScore(p *ProcessInfo) float64 {
	// Apply weights to CPU and RSS
	// Normalize RSS to be in similar range as CPU percentage
	normalizedRSS := 0.0
	if s.totalRSSUsage > 0 {
		normalizedRSS = (float64(p.RSS) / float64(s.totalRSSUsage)) * 100
	}

	// Calculate new score
	score := (s.config.CPUWeight * p.CPU) + (s.config.RSSWeight * normalizedRSS)

	// Apply minimum score threshold
	if score < s.config.MinScore {
		score = 0
	}

	return score
}

// checkCircuitBreaker checks if the circuit breaker should be activated.
func (s *TopNSampler) checkCircuitBreaker() {
	// Check if we need to open the circuit breaker
	if !s.circuitOpen {
		// Open circuit breaker if CPU usage too high or churn rate exceeds threshold
		if s.metrics["topn_update_time_seconds"]*100 > s.config.MaxSamplerCPU ||
			(s.config.ChurnHandlingEnabled && s.churnRate > float64(s.config.ChurnThreshold)) {
			s.circuitOpen = true
			// Log circuit breaker activation as an agent diagnostic event
			fmt.Printf("AgentDiagEvent: ModuleOverLimit detected in sampler. "+
				"CPU: %.2f%%, ChurnRate: %.2f PIDs/s\n",
				s.metrics["topn_update_time_seconds"]*100, s.churnRate)
		}
	} else {
		// Check if we can close the circuit breaker
		if s.metrics["topn_update_time_seconds"]*100 < s.config.MaxSamplerCPU*0.7 &&
			(!s.config.ChurnHandlingEnabled || s.churnRate < float64(s.config.ChurnThreshold)*0.7) {
			s.circuitOpen = false
			// Log circuit breaker deactivation
			fmt.Printf("AgentDiagEvent: ModuleOverLimit resolved in sampler. "+
				"CPU: %.2f%%, ChurnRate: %.2f PIDs/s\n",
				s.metrics["topn_update_time_seconds"]*100, s.churnRate)
		}
	}
}
