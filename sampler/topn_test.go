package sampler

import (
	"context"
	"testing"
	"time"
)

func TestTopNSampler_Init(t *testing.T) {
	// Create a sampler with default config
	s := NewTopNSampler(DefaultConfig().TopN)

	// Initialize with context
	ctx := context.Background()
	err := s.Init(ctx)
	if err != nil {
		t.Errorf("Init failed: %v", err)
	}

	// Verify initialization state
	if s.ctx == nil {
		t.Errorf("Context should be set after Init")
	}
	if s.cancel == nil {
		t.Errorf("Cancel function should be set after Init")
	}
}

func TestTopNSampler_Update(t *testing.T) {
	// Create a sampler with default config
	config := DefaultConfig().TopN
	config.MaxProcesses = 10
	s := NewTopNSampler(config)

	// Initialize
	ctx := context.Background()
	err := s.Init(ctx)
	if err != nil {
		t.Errorf("Init failed: %v", err)
	}

	// Create test processes
	processes := []*ProcessInfo{
		{PID: 1, Name: "Process1", Command: "cmd1", CPU: 10.0, RSS: 1000000},
		{PID: 2, Name: "Process2", Command: "cmd2", CPU: 20.0, RSS: 2000000},
		{PID: 3, Name: "Process3", Command: "cmd3", CPU: 5.0, RSS: 500000},
		{PID: 4, Name: "Process4", Command: "cmd4", CPU: 15.0, RSS: 1500000},
		{PID: 5, Name: "Process5", Command: "cmd5", CPU: 1.0, RSS: 100000},
	}

	// Update the sampler
	err = s.Update(processes)
	if err != nil {
		t.Errorf("Update failed: %v", err)
	}

	// Check metrics
	metrics := s.Metrics()
	if metrics["topn_processes_tracked"] != 5 {
		t.Errorf("Expected 5 processes tracked, got %.1f", metrics["topn_processes_tracked"])
	}
	if metrics["topn_processes_updated"] != 5 {
		t.Errorf("Expected 5 processes updated, got %.1f", metrics["topn_processes_updated"])
	}
	if metrics["topn_capture_ratio"] != 100.0 {
		t.Errorf("Expected 100%% capture ratio, got %.1f%%", metrics["topn_capture_ratio"])
	}

	// Check top processes
	top := s.GetTopN(3)
	if len(top) != 3 {
		t.Errorf("Expected 3 top processes, got %d", len(top))
	}

	// Verify order by score (should be influenced by both CPU and RSS)
	if top[0].PID != 2 {
		t.Errorf("Expected PID 2 as top process, got %d", top[0].PID)
	}
	if top[1].PID != 4 {
		t.Errorf("Expected PID 4 as second process, got %d", top[1].PID)
	}
	if top[2].PID != 1 {
		t.Errorf("Expected PID 1 as third process, got %d", top[2].PID)
	}

	// Update with fewer processes
	newProcesses := []*ProcessInfo{
		{PID: 1, Name: "Process1", Command: "cmd1", CPU: 5.0, RSS: 500000},   // Decreased
		{PID: 2, Name: "Process2", Command: "cmd2", CPU: 25.0, RSS: 2500000}, // Increased
		{PID: 6, Name: "Process6", Command: "cmd6", CPU: 30.0, RSS: 3000000}, // New high-resource
	}

	err = s.Update(newProcesses)
	if err != nil {
		t.Errorf("Second update failed: %v", err)
	}

	// Check metrics after update
	metrics = s.Metrics()
	if metrics["topn_processes_tracked"] != 5 { // Still tracking 5 processes (PIDs 1-5 + 6, minus one)
		t.Errorf("Expected 5 processes tracked, got %.1f", metrics["topn_processes_tracked"])
	}

	// Check top processes after update
	top = s.GetTopN(3)
	if len(top) != 3 {
		t.Errorf("Expected 3 top processes, got %d", len(top))
	}

	// New process should be at the top now
	if top[0].PID != 6 {
		t.Errorf("Expected PID 6 as new top process, got %d", top[0].PID)
	}
	if top[1].PID != 2 {
		t.Errorf("Expected PID 2 as second process, got %d", top[1].PID)
	}
}

func TestTopNSampler_ChurnHandling(t *testing.T) {
	// Create a sampler with churn handling enabled
	config := DefaultConfig().TopN
	config.ChurnHandlingEnabled = true
	config.ChurnThreshold = 10 // Low threshold for testing
	s := NewTopNSampler(config)

	// Initialize
	ctx := context.Background()
	err := s.Init(ctx)
	if err != nil {
		t.Errorf("Init failed: %v", err)
	}

	// Create initial processes
	processes := []*ProcessInfo{
		{PID: 1, Name: "Process1", Command: "cmd1", CPU: 10.0, RSS: 1000000},
		{PID: 2, Name: "Process2", Command: "cmd2", CPU: 20.0, RSS: 2000000},
	}

	// Update the sampler
	err = s.Update(processes)
	if err != nil {
		t.Errorf("Update failed: %v", err)
	}

	// Wait briefly to simulate time passing
	time.Sleep(10 * time.Millisecond)

	// Create new set with high churn (many new PIDs)
	newProcesses := []*ProcessInfo{
		{PID: 1, Name: "Process1", Command: "cmd1", CPU: 10.0, RSS: 1000000},  // Kept
		{PID: 10, Name: "Process10", Command: "cmd10", CPU: 5.0, RSS: 500000}, // New
		{PID: 11, Name: "Process11", Command: "cmd11", CPU: 6.0, RSS: 600000}, // New
		{PID: 12, Name: "Process12", Command: "cmd12", CPU: 7.0, RSS: 700000}, // New
		{PID: 13, Name: "Process13", Command: "cmd13", CPU: 8.0, RSS: 800000}, // New
		{PID: 14, Name: "Process14", Command: "cmd14", CPU: 9.0, RSS: 900000}, // New
	}

	// Update the sampler
	err = s.Update(newProcesses)
	if err != nil {
		t.Errorf("High churn update failed: %v", err)
	}

	// Check churn rate
	metrics := s.Metrics()
	if metrics["topn_churn_rate"] <= 0 {
		t.Errorf("Expected positive churn rate, got %.1f", metrics["topn_churn_rate"])
	}

	// Circuit breaker might be open due to high churn
	if metrics["topn_churn_rate"] > float64(config.ChurnThreshold) && metrics["topn_circuit_breaker"] != 1 {
		t.Errorf("Circuit breaker should be open when churn rate (%.1f) exceeds threshold (%d)",
			metrics["topn_churn_rate"], config.ChurnThreshold)
	}
}

func TestTopNSampler_CircuitBreaker(t *testing.T) {
	// Create a sampler with strict CPU limits
	config := DefaultConfig().TopN
	config.MaxSamplerCPU = 0.1 // Very low threshold to force circuit breaker
	s := NewTopNSampler(config)

	// Initialize
	ctx := context.Background()
	err := s.Init(ctx)
	if err != nil {
		t.Errorf("Init failed: %v", err)
	}

	// Create a large number of processes to increase processing time
	processes := make([]*ProcessInfo, 10000)
	for i := 0; i < 10000; i++ {
		processes[i] = &ProcessInfo{
			PID:     i,
			Name:    "Process",
			Command: "cmd",
			CPU:     float64(i % 100),
			RSS:     int64(i * 1000),
		}
	}

	// Update the sampler
	err = s.Update(processes)
	if err != nil {
		t.Errorf("Update failed: %v", err)
	}

	// Force multiple updates to possibly trigger circuit breaker
	for i := 0; i < 3; i++ {
		err = s.Update(processes)
		if err != nil {
			t.Errorf("Update %d failed: %v", i, err)
		}
	}

	// Check if circuit breaker was activated
	metrics := s.Metrics()
	t.Logf("Update time: %.6f seconds", metrics["topn_update_time_seconds"])
	t.Logf("Circuit breaker: %.0f", metrics["topn_circuit_breaker"])

	// Even if circuit breaker is activated, we should still be tracking processes
	if metrics["topn_processes_tracked"] <= 0 {
		t.Errorf("Expected to track some processes even with circuit breaker, got %.0f",
			metrics["topn_processes_tracked"])
	}
}

func TestTopNSampler_Resources(t *testing.T) {
	// Create a sampler
	s := NewTopNSampler(DefaultConfig().TopN)

	// Initialize
	ctx := context.Background()
	err := s.Init(ctx)
	if err != nil {
		t.Errorf("Init failed: %v", err)
	}

	// Get resource usage
	resources := s.Resources()

	// Verify resource keys exist
	keys := []string{"sampler_cpu_percent", "sampler_rss_bytes", "sampler_uptime_seconds"}
	for _, key := range keys {
		if _, ok := resources[key]; !ok {
			t.Errorf("Expected resource key %s not found", key)
		}
	}
}

func TestTopNSampler_Shutdown(t *testing.T) {
	// Create a sampler
	s := NewTopNSampler(DefaultConfig().TopN)

	// Initialize
	ctx := context.Background()
	err := s.Init(ctx)
	if err != nil {
		t.Errorf("Init failed: %v", err)
	}

	// Shutdown
	err = s.Shutdown()
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Context should be canceled
	select {
	case <-s.ctx.Done():
		// Context was canceled, as expected
	default:
		t.Errorf("Context not canceled after shutdown")
	}
}

func BenchmarkTopNSampler_Update(b *testing.B) {
	// Create a sampler with default config
	s := NewTopNSampler(DefaultConfig().TopN)

	// Initialize
	ctx := context.Background()
	s.Init(ctx)

	// Create test processes
	processes := make([]*ProcessInfo, 1000)
	for i := 0; i < 1000; i++ {
		processes[i] = &ProcessInfo{
			PID:     i,
			Name:    "BenchProcess",
			Command: "cmd",
			CPU:     float64(i % 100),
			RSS:     int64(i * 10000),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Update(processes)
	}
}

func BenchmarkTopNSampler_GetTopN(b *testing.B) {
	// Create a sampler with default config
	s := NewTopNSampler(DefaultConfig().TopN)

	// Initialize
	ctx := context.Background()
	s.Init(ctx)

	// Create and update test processes
	processes := make([]*ProcessInfo, 1000)
	for i := 0; i < 1000; i++ {
		processes[i] = &ProcessInfo{
			PID:     i,
			Name:    "BenchProcess",
			Command: "cmd",
			CPU:     float64(i % 100),
			RSS:     int64(i * 10000),
		}
	}
	s.Update(processes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.GetTopN(100)
	}
}
