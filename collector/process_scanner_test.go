package collector

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockProcessConsumer implements ProcessConsumer for testing
type MockProcessConsumer struct {
	events    []ProcessEvent
	eventsMux sync.Mutex
}

// NewMockProcessConsumer creates a new mock consumer
func NewMockProcessConsumer() *MockProcessConsumer {
	return &MockProcessConsumer{
		events: make([]ProcessEvent, 0),
	}
}

// HandleProcessEvent handles a process event
func (m *MockProcessConsumer) HandleProcessEvent(event ProcessEvent) error {
	m.eventsMux.Lock()
	defer m.eventsMux.Unlock()
	
	// Store a copy of the event
	eventCopy := ProcessEvent{
		Type:      event.Type,
		Process:   event.Process.Clone(),
		Timestamp: event.Timestamp,
	}
	
	m.events = append(m.events, eventCopy)
	return nil
}

// GetEvents returns all received events
func (m *MockProcessConsumer) GetEvents() []ProcessEvent {
	m.eventsMux.Lock()
	defer m.eventsMux.Unlock()
	
	// Return a copy to avoid race conditions
	eventsCopy := make([]ProcessEvent, len(m.events))
	copy(eventsCopy, m.events)
	
	return eventsCopy
}

// Reset clears all events
func (m *MockProcessConsumer) Reset() {
	m.eventsMux.Lock()
	defer m.eventsMux.Unlock()
	
	m.events = make([]ProcessEvent, 0)
}

// Count returns the number of events received
func (m *MockProcessConsumer) Count() int {
	m.eventsMux.Lock()
	defer m.eventsMux.Unlock()
	
	return len(m.events)
}

// CountByType returns the number of events of a specific type
func (m *MockProcessConsumer) CountByType(eventType ProcessEventType) int {
	m.eventsMux.Lock()
	defer m.eventsMux.Unlock()
	
	count := 0
	for _, event := range m.events {
		if event.Type == eventType {
			count++
		}
	}
	
	return count
}

// ErrorConsumer is a consumer that returns errors
type ErrorConsumer struct{}

// HandleProcessEvent always returns an error
func (e *ErrorConsumer) HandleProcessEvent(event ProcessEvent) error {
	return fmt.Errorf("intentional error from ErrorConsumer")
}

func TestProcessScanner_RegisterConsumer(t *testing.T) {
	// Create scanner with default config
	scanner := NewProcessScanner(DefaultConfig().ProcessScanner)
	
	// Create mock consumer
	consumer := NewMockProcessConsumer()
	
	// Register consumer
	err := scanner.RegisterConsumer("test", consumer)
	if err != nil {
		t.Errorf("Failed to register consumer: %v", err)
	}
	
	// Test duplicate registration
	err = scanner.RegisterConsumer("test", consumer)
	if err == nil {
		t.Errorf("Expected error when registering duplicate consumer")
	}
	
	// Test nil consumer
	err = scanner.RegisterConsumer("nil", nil)
	if err == nil {
		t.Errorf("Expected error when registering nil consumer")
	}
	
	// Test empty name
	err = scanner.RegisterConsumer("", consumer)
	if err == nil {
		t.Errorf("Expected error when registering with empty name")
	}
	
	// Unregister consumer
	err = scanner.UnregisterConsumer("test")
	if err != nil {
		t.Errorf("Failed to unregister consumer: %v", err)
	}
	
	// Test unregistering non-existent consumer
	err = scanner.UnregisterConsumer("nonexistent")
	if err == nil {
		t.Errorf("Expected error when unregistering non-existent consumer")
	}
}

func TestProcessScanner_StartStop(t *testing.T) {
	// Create scanner with default config
	config := DefaultConfig().ProcessScanner
	config.ScanInterval = time.Millisecond * 100 // Fast scanning for tests
	scanner := NewProcessScanner(config)
	
	// Initialize scanner
	err := scanner.Init(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize scanner: %v", err)
	}
	
	// Start scanner
	err = scanner.Start()
	if err != nil {
		t.Fatalf("Failed to start scanner: %v", err)
	}
	
	// Check status
	if scanner.Status() != StatusRunning {
		t.Errorf("Expected status to be running, got %s", scanner.Status())
	}
	
	// Try starting again
	err = scanner.Start()
	if err == nil {
		t.Errorf("Expected error when starting an already running scanner")
	}
	
	// Stop scanner
	err = scanner.Stop()
	if err != nil {
		t.Fatalf("Failed to stop scanner: %v", err)
	}
	
	// Check status
	if scanner.Status() != StatusStopped {
		t.Errorf("Expected status to be stopped, got %s", scanner.Status())
	}
	
	// Try stopping again
	err = scanner.Stop()
	if err == nil {
		t.Errorf("Expected error when stopping an already stopped scanner")
	}
	
	// Start, then shutdown
	err = scanner.Start()
	if err != nil {
		t.Fatalf("Failed to restart scanner: %v", err)
	}
	
	err = scanner.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown scanner: %v", err)
	}
}

func TestProcessScanner_ProcessEvents(t *testing.T) {
	// Create scanner with default config
	config := DefaultConfig().ProcessScanner
	config.ScanInterval = time.Millisecond * 100 // Fast scanning for tests
	scanner := NewProcessScanner(config)
	
	// Create mock consumer
	consumer := NewMockProcessConsumer()
	
	// Initialize scanner
	err := scanner.Init(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize scanner: %v", err)
	}
	
	// Register consumer
	err = scanner.RegisterConsumer("test", consumer)
	if err != nil {
		t.Fatalf("Failed to register consumer: %v", err)
	}
	
	// Start scanner
	err = scanner.Start()
	if err != nil {
		t.Fatalf("Failed to start scanner: %v", err)
	}
	
	// Wait for at least one scan cycle
	time.Sleep(time.Millisecond * 200)
	
	// Get events
	events := consumer.GetEvents()
	
	// We should have some events
	if len(events) == 0 {
		t.Errorf("Expected some events, got none")
	}
	
	// Check for created events
	createdCount := consumer.CountByType(ProcessCreated)
	if createdCount == 0 {
		t.Errorf("Expected some process created events, got none")
	}
	
	// Reset the consumer
	consumer.Reset()
	
	// Add an error consumer
	err = scanner.RegisterConsumer("error", &ErrorConsumer{})
	if err != nil {
		t.Fatalf("Failed to register error consumer: %v", err)
	}
	
	// Force a scan to generate events
	err = scanner.ForceScan()
	if err != nil {
		t.Fatalf("Failed to force scan: %v", err)
	}
	
	// Wait for events to be processed
	time.Sleep(time.Millisecond * 200)
	
	// Get events from the working consumer
	events = consumer.GetEvents()
	
	// We should still have some events despite the error consumer
	if len(events) == 0 {
		t.Errorf("Expected some events despite error consumer, got none")
	}
	
	// Stop scanner
	err = scanner.Stop()
	if err != nil {
		t.Fatalf("Failed to stop scanner: %v", err)
	}
}

func TestProcessScanner_Metrics(t *testing.T) {
	// Create scanner with default config
	config := DefaultConfig().ProcessScanner
	config.ScanInterval = time.Millisecond * 100 // Fast scanning for tests
	scanner := NewProcessScanner(config)
	
	// Initialize scanner
	err := scanner.Init(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize scanner: %v", err)
	}
	
	// Start scanner
	err = scanner.Start()
	if err != nil {
		t.Fatalf("Failed to start scanner: %v", err)
	}
	
	// Wait for some metrics to be collected
	time.Sleep(time.Millisecond * 200)
	
	// Get metrics
	metrics := scanner.Metrics()
	
	// Check for expected metrics
	expectedMetrics := []string{
		MetricScanDuration + "_ms",
		MetricProcessCount,
		MetricProcessCreated,
		MetricScanErrors,
		"uptime_seconds",
	}
	
	for _, metric := range expectedMetrics {
		if _, exists := metrics[metric]; !exists {
			t.Errorf("Expected metric %s not found", metric)
		}
	}
	
	// Check resource metrics
	resources := scanner.Resources()
	if _, exists := resources["cpu_percent"]; !exists {
		t.Errorf("Expected cpu_percent resource metric not found")
	}
	if _, exists := resources["memory_bytes"]; !exists {
		t.Errorf("Expected memory_bytes resource metric not found")
	}
	
	// Stop scanner
	err = scanner.Stop()
	if err != nil {
		t.Fatalf("Failed to stop scanner: %v", err)
	}
}

func TestProcessScanner_AdaptiveSampling(t *testing.T) {
	// This test is more of a functional test than a unit test
	// It tests the adaptive sampling feature by simulating high CPU usage
	
	// Create scanner with adaptive sampling enabled
	config := DefaultConfig().ProcessScanner
	config.ScanInterval = time.Millisecond * 100 // Fast scanning for tests
	config.AdaptiveSampling = true
	config.MaxCPUUsage = 0.1 // Set very low to trigger adaptation
	scanner := NewProcessScanner(config)
	
	// Initialize scanner
	err := scanner.Init(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize scanner: %v", err)
	}
	
	// Start scanner
	err = scanner.Start()
	if err != nil {
		t.Fatalf("Failed to start scanner: %v", err)
	}
	
	// Wait for at least one scan cycle
	time.Sleep(time.Millisecond * 200)
	
	// Force adaptivity by simulating high CPU
	p := scanner.(*ProcessScanner)
	p.adjustScanInterval(1.0) // 1.0% CPU, 10x higher than our 0.1% limit
	
	// Check if the scan interval was increased
	if p.config.ScanInterval <= time.Millisecond*100 {
		t.Errorf("Expected scan interval to increase, but it stayed at %v", p.config.ScanInterval)
	}
	
	// Stop scanner
	err = scanner.Stop()
	if err != nil {
		t.Fatalf("Failed to stop scanner: %v", err)
	}
}

func TestProcessScanner_FilterProcesses(t *testing.T) {
	// Create scanner with filters
	config := DefaultConfig().ProcessScanner
	config.ExcludePatterns = []string{"system"}
	config.IncludePatterns = []string{"ssh"}
	scanner := NewProcessScanner(config)
	
	// Initialize scanner
	err := scanner.Init(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize scanner: %v", err)
	}
	
	// Create test processes
	processes := []*ProcessInfo{
		{
			PID:     1,
			Name:    "systemd",
			Command: "/usr/lib/systemd/systemd",
		},
		{
			PID:     100,
			Name:    "sshd",
			Command: "/usr/sbin/sshd",
		},
		{
			PID:     200,
			Name:    "bash",
			Command: "/bin/bash",
		},
	}
	
	// Apply filters
	p := scanner.(*ProcessScanner)
	filtered := p.filterProcesses(processes)
	
	// Only sshd should pass the filters
	if len(filtered) != 1 {
		t.Errorf("Expected 1 process after filtering, got %d", len(filtered))
	}
	
	if len(filtered) > 0 && filtered[0].Name != "sshd" {
		t.Errorf("Expected 'sshd' to pass the filter, got '%s'", filtered[0].Name)
	}
}

func TestProcessScanner_ProcessNewScan(t *testing.T) {
	// Create scanner 
	scanner := NewProcessScanner(DefaultConfig().ProcessScanner)
	
	// Initialize scanner
	err := scanner.Init(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize scanner: %v", err)
	}
	
	// Create mock consumer
	consumer := NewMockProcessConsumer()
	
	// Register consumer
	err = scanner.RegisterConsumer("test", consumer)
	if err != nil {
		t.Fatalf("Failed to register consumer: %v", err)
	}
	
	// Get access to internal scanner
	p := scanner.(*ProcessScanner)
	
	// Initial scan with new processes
	initialProcesses := []*ProcessInfo{
		{
			PID:     1,
			Name:    "process1",
			Command: "/bin/process1",
		},
		{
			PID:     2,
			Name:    "process2",
			Command: "/bin/process2",
		},
	}
	
	// Process the scan
	count, created, updated, terminated := p.processNewScan(initialProcesses)
	
	// Verify counts
	if count != 2 {
		t.Errorf("Expected 2 processes in cache, got %d", count)
	}
	if created != 2 {
		t.Errorf("Expected 2 created processes, got %d", created)
	}
	if updated != 0 {
		t.Errorf("Expected 0 updated processes, got %d", updated)
	}
	if terminated != 0 {
		t.Errorf("Expected 0 terminated processes, got %d", terminated)
	}
	
	// Wait for events to be processed
	time.Sleep(time.Millisecond * 50)
	
	// Check events
	events := consumer.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}
	if consumer.CountByType(ProcessCreated) != 2 {
		t.Errorf("Expected 2 created events, got %d", consumer.CountByType(ProcessCreated))
	}
	
	// Clear events
	consumer.Reset()
	
	// Second scan with one process updated, one removed, one added
	updatedProcesses := []*ProcessInfo{
		{
			PID:     1,
			Name:    "process1-updated",
			Command: "/bin/process1",
		},
		{
			PID:     3,
			Name:    "process3",
			Command: "/bin/process3",
		},
	}
	
	// Process the scan
	count, created, updated, terminated = p.processNewScan(updatedProcesses)
	
	// Verify counts
	if count != 2 {
		t.Errorf("Expected 2 processes in cache, got %d", count)
	}
	if created != 1 {
		t.Errorf("Expected 1 created process, got %d", created)
	}
	if updated != 1 {
		t.Errorf("Expected 1 updated process, got %d", updated)
	}
	if terminated != 1 {
		t.Errorf("Expected 1 terminated process, got %d", terminated)
	}
	
	// Wait for events to be processed
	time.Sleep(time.Millisecond * 50)
	
	// Check events
	events = consumer.GetEvents()
	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}
	if consumer.CountByType(ProcessCreated) != 1 {
		t.Errorf("Expected 1 created event, got %d", consumer.CountByType(ProcessCreated))
	}
	if consumer.CountByType(ProcessUpdated) != 1 {
		t.Errorf("Expected 1 updated event, got %d", consumer.CountByType(ProcessUpdated))
	}
	if consumer.CountByType(ProcessTerminated) != 1 {
		t.Errorf("Expected 1 terminated event, got %d", consumer.CountByType(ProcessTerminated))
	}
}

func TestProcessInfo_Clone(t *testing.T) {
	// Create a process info
	proc := &ProcessInfo{
		PID:         1,
		PPID:        0,
		Name:        "test",
		Executable:  "/bin/test",
		Command:     "/bin/test --arg=value",
		User:        "root",
		CPU:         1.0,
		RSS:         1024,
		VMS:         2048,
		FDs:         10,
		Threads:     2,
		StartTime:   time.Now(),
		State:       "S",
		LastUpdated: time.Now(),
		IOReadBytes: 100,
		IOWriteBytes: 200,
		Labels: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}
	
	// Clone it
	clone := proc.Clone()
	
	// Check equality
	if !proc.Equal(clone) {
		t.Errorf("Clone not equal to original")
	}
	
	// Modify the original
	proc.CPU = 2.0
	proc.Labels["key1"] = "modified"
	
	// Clone should remain unchanged
	if clone.CPU != 1.0 {
		t.Errorf("Clone CPU changed with original")
	}
	if clone.Labels["key1"] != "value1" {
		t.Errorf("Clone labels changed with original")
	}
	
	// Nil case
	var nilProc *ProcessInfo
	nilClone := nilProc.Clone()
	if nilClone != nil {
		t.Errorf("Clone of nil should be nil")
	}
}

func TestCalculateDelta(t *testing.T) {
	// Create current and previous process info
	now := time.Now()
	prev := time.Now().Add(-1 * time.Second)
	
	current := &ProcessInfo{
		PID:         1,
		CPU:         2.0,
		RSS:         2048,
		IOReadBytes: 200,
		IOWriteBytes: 300,
		LastUpdated: now,
	}
	
	previous := &ProcessInfo{
		PID:         1,
		CPU:         1.0,
		RSS:         1024,
		IOReadBytes: 100,
		IOWriteBytes: 200,
		LastUpdated: prev,
	}
	
	// Calculate delta
	delta, err := CalculateDelta(current, previous)
	if err != nil {
		t.Fatalf("Failed to calculate delta: %v", err)
	}
	
	// Check delta values
	if delta.PID != 1 {
		t.Errorf("Expected PID 1, got %d", delta.PID)
	}
	if delta.CPU != 1.0 {
		t.Errorf("Expected CPU delta 1.0, got %f", delta.CPU)
	}
	if delta.RSS != 1024 {
		t.Errorf("Expected RSS delta 1024, got %d", delta.RSS)
	}
	if delta.IOReadBytes != 100 {
		t.Errorf("Expected IOReadBytes delta 100, got %d", delta.IOReadBytes)
	}
	if delta.IOWriteBytes != 100 {
		t.Errorf("Expected IOWriteBytes delta 100, got %d", delta.IOWriteBytes)
	}
	
	// Test error cases
	_, err = CalculateDelta(nil, previous)
	if err == nil {
		t.Errorf("Expected error with nil current process")
	}
	
	_, err = CalculateDelta(current, nil)
	if err == nil {
		t.Errorf("Expected error with nil previous process")
	}
	
	differentPID := &ProcessInfo{
		PID:         2,
		LastUpdated: prev,
	}
	_, err = CalculateDelta(current, differentPID)
	if err == nil {
		t.Errorf("Expected error with different PIDs")
	}
	
	sameTime := &ProcessInfo{
		PID:         1,
		LastUpdated: now,
	}
	_, err = CalculateDelta(current, sameTime)
	if err == nil {
		t.Errorf("Expected error with same timestamp")
	}
}
