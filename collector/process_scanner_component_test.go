// Copyright 2025 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build component
// +build component

package collector

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/collector/platform"
	"github.com/newrelic/infrastructure-agent/internal/agent/diagnostics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPlatformCollector mocks the platform.ProcessCollector interface
type MockPlatformCollector struct {
	mock.Mock
}

func (m *MockPlatformCollector) GetProcesses() ([]*ProcessInfo, error) {
	args := m.Called()
	return args.Get(0).([]*ProcessInfo), args.Error(1)
}

func (m *MockPlatformCollector) GetCPUTimes() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPlatformCollector) GetSelfUsage() (float64, uint64, error) {
	args := m.Called()
	return args.Get(0).(float64), args.Get(1).(uint64), args.Error(2)
}

func (m *MockPlatformCollector) Shutdown() error {
	args := m.Called()
	return args.Error(0)
}

// MockProcessConsumer mocks the ProcessConsumer interface
type MockProcessConsumer struct {
	mock.Mock
	eventsMu       sync.Mutex
	receivedEvents []ProcessEvent
}

func (m *MockProcessConsumer) OnProcessEvent(event ProcessEvent) error {
	args := m.Called(event)
	
	// Record the event for later verification
	m.eventsMu.Lock()
	m.receivedEvents = append(m.receivedEvents, event)
	m.eventsMu.Unlock()
	
	return args.Error(0)
}

func (m *MockProcessConsumer) GetReceivedEvents() []ProcessEvent {
	m.eventsMu.Lock()
	defer m.eventsMu.Unlock()
	
	// Return a copy to prevent modification
	result := make([]ProcessEvent, len(m.receivedEvents))
	copy(result, m.receivedEvents)
	return result
}

// Helper function to initialize scanner with mock for tests
func initProcessScannerWithMock(t *testing.T, scanner *ProcessScanner, mockCollector platform.ProcessCollector) {
	ctx := context.Background()
	
	// Create a derived context
	scanner.ctx, scanner.cancel = context.WithCancel(ctx)
	
	// Use the provided mock collector
	scanner.platformCollector = mockCollector
	
	// Compile any patterns
	scanner.excludeRegexps = make([]*regexp.Regexp, 0, len(scanner.config.ExcludePatterns))
	for _, pattern := range scanner.config.ExcludePatterns {
		re, err := regexp.Compile(pattern)
		require.NoError(t, err)
		scanner.excludeRegexps = append(scanner.excludeRegexps, re)
	}
	
	scanner.includeRegexps = make([]*regexp.Regexp, 0, len(scanner.config.IncludePatterns))
	for _, pattern := range scanner.config.IncludePatterns {
		re, err := regexp.Compile(pattern)
		require.NoError(t, err)
		scanner.includeRegexps = append(scanner.includeRegexps, re)
	}
	
	// Set status to initialized
	scanner.statusLock.Lock()
	scanner.status = StatusInitialized
	scanner.statusLock.Unlock()
}

// Test helpers
func createTestProcessInfo(pid int, name string, command string, cpuPercent float64, memoryRSS uint64) *ProcessInfo {
	return &ProcessInfo{
		PID:        pid,
		PPID:       1,
		Name:       name,
		Command:    command,
		Username:   "test",
		State:      "running",
		CPUPercent: cpuPercent,
		MemoryRSS:  memoryRSS,
		StartTime:  time.Now().Add(-time.Minute),
		Attributes: make(map[string]string),
	}
}

// Tests
func TestProcessScanner_WithMockedCollector(t *testing.T) {
	// Setup mock platform collector
	mockCollector := new(MockPlatformCollector)
	
	// Create test processes
	testProcesses := []*ProcessInfo{
		createTestProcessInfo(1, "process1", "/bin/process1", 1.0, 1024*1024),
		createTestProcessInfo(2, "process2", "/bin/process2", 2.0, 2*1024*1024),
		createTestProcessInfo(3, "process3", "/bin/process3", 3.0, 3*1024*1024),
		createTestProcessInfo(4, "process4", "/bin/process4", 4.0, 4*1024*1024),
		createTestProcessInfo(5, "process5", "/bin/process5", 5.0, 5*1024*1024),
	}
	
	// Configure mock behaviors
	mockCollector.On("GetProcesses").Return(testProcesses, nil)
	mockCollector.On("GetCPUTimes").Return(nil)
	mockCollector.On("GetSelfUsage").Return(0.3, uint64(20*1024*1024), nil)
	mockCollector.On("Shutdown").Return(nil)
	
	// Create scanner with test configuration
	config := ProcessScannerConfig{
		ScanInterval: 100 * time.Millisecond,
	}
	scanner := NewProcessScanner(config)
	
	// Initialize scanner with mock
	initProcessScannerWithMock(t, scanner, mockCollector)
	
	// Setup mock consumer
	mockConsumer := new(MockProcessConsumer)
	mockConsumer.On("OnProcessEvent", mock.Anything).Return(nil)
	
	// Register consumer
	err := scanner.RegisterConsumer("test-consumer", mockConsumer)
	require.NoError(t, err)
	
	// Start scanner 
	err = scanner.Start()
	require.NoError(t, err)
	
	// Allow scanner to run
	time.Sleep(200 * time.Millisecond)
	
	// Verify expected calls
	mockCollector.AssertCalled(t, "GetProcesses")
	mockConsumer.AssertCalled(t, "OnProcessEvent", mock.Anything)
	
	// Verify process events
	receivedEvents := mockConsumer.GetReceivedEvents()
	
	// Count event types
	createdCount := 0
	for _, event := range receivedEvents {
		if event.Type == ProcessCreated {
			createdCount++
		}
	}
	
	assert.Equal(t, 5, createdCount, "Should receive 5 process created events")
	
	// Check cached processes
	cachedProcesses := scanner.GetCachedProcesses()
	assert.Equal(t, 5, len(cachedProcesses), "Should have 5 cached processes")
	
	// Verify metrics
	metrics := scanner.Metrics()
	assert.Equal(t, float64(5), metrics[MetricProcessCount], "Process count should be 5")
	
	// Verify specific process retrieval
	cachedProcess, exists := scanner.GetCachedProcess(1)
	assert.True(t, exists, "Process 1 should exist in cache")
	assert.Equal(t, "process1", cachedProcess.Name, "Process 1 should have correct name")
	
	// Shutdown
	err = scanner.Shutdown()
	require.NoError(t, err)
}

func TestProcessScanner_LifecycleEvents(t *testing.T) {
	// Setup mock collector with changing process lists
	mockCollector := new(MockPlatformCollector)
	
	// First scan: initial processes
	initialProcesses := []*ProcessInfo{
		createTestProcessInfo(1, "process1", "/bin/process1", 10.0, 1024*1024),
		createTestProcessInfo(2, "process2", "/bin/process2", 20.0, 2*1024*1024),
	}
	
	// Second scan: one process changed, one terminated, one new
	updatedProcesses := []*ProcessInfo{
		createTestProcessInfo(1, "process1", "/bin/process1", 15.0, 1.5*1024*1024), // Changed
		createTestProcessInfo(3, "process3", "/bin/process3", 30.0, 3*1024*1024),   // New
	}
	
	// Configure mock to return different sets on consecutive calls
	mockCollector.On("GetProcesses").Return(initialProcesses, nil).Once()
	mockCollector.On("GetProcesses").Return(updatedProcesses, nil).Once()
	mockCollector.On("GetCPUTimes").Return(nil)
	mockCollector.On("GetSelfUsage").Return(0.2, uint64(15*1024*1024), nil)
	mockCollector.On("Shutdown").Return(nil)
	
	// Create scanner with test configuration
	config := ProcessScannerConfig{
		ScanInterval: 100 * time.Millisecond, // Fast scanning for test
	}
	scanner := NewProcessScanner(config)
	
	// Initialize scanner with mock
	initProcessScannerWithMock(t, scanner, mockCollector)
	
	// Setup mock consumer
	mockConsumer := new(MockProcessConsumer)
	mockConsumer.On("OnProcessEvent", mock.Anything).Return(nil)
	
	// Register consumer
	err := scanner.RegisterConsumer("test-consumer", mockConsumer)
	require.NoError(t, err)
	
	// Start scanner 
	err = scanner.Start()
	require.NoError(t, err)
	
	// Wait for both scans to complete
	time.Sleep(250 * time.Millisecond)
	
	// Stop scanner
	err = scanner.Stop()
	require.NoError(t, err)
	
	// Analyze received events
	receivedEvents := mockConsumer.GetReceivedEvents()
	
	// Count event types
	createdCount := 0
	updatedCount := 0
	terminatedCount := 0
	
	// Track PIDs for each event type
	createdPIDs := make(map[int]bool)
	updatedPIDs := make(map[int]bool)
	terminatedPIDs := make(map[int]bool)
	
	for _, event := range receivedEvents {
		pid := event.Process.PID
		
		switch event.Type {
		case ProcessCreated:
			createdCount++
			createdPIDs[pid] = true
		case ProcessUpdated:
			updatedCount++
			updatedPIDs[pid] = true
		case ProcessTerminated:
			terminatedCount++
			terminatedPIDs[pid] = true
		}
	}
	
	// Verify counts and specific PIDs
	assert.Equal(t, 3, createdCount, "Should have 3 created events (2 initial + 1 new)")
	assert.Equal(t, 1, updatedCount, "Should have 1 updated event")
	assert.Equal(t, 1, terminatedCount, "Should have 1 terminated event")
	
	assert.True(t, createdPIDs[1], "Process 1 should have created event")
	assert.True(t, createdPIDs[2], "Process 2 should have created event")
	assert.True(t, createdPIDs[3], "Process 3 should have created event")
	
	assert.True(t, updatedPIDs[1], "Process 1 should have updated event")
	assert.True(t, terminatedPIDs[2], "Process 2 should have terminated event")
	
	// Verify delta data for updated event
	var updateEvent *ProcessEvent
	for _, event := range receivedEvents {
		if event.Type == ProcessUpdated && event.Process.PID == 1 {
			updateEvent = &event
			break
		}
	}
	
	require.NotNil(t, updateEvent, "Should have update event for PID 1")
	require.NotNil(t, updateEvent.Delta, "Update event should have delta information")
	assert.Greater(t, updateEvent.Delta.CPUPercent, 0.0, "Delta should show CPU increase")
	assert.Greater(t, updateEvent.Delta.MemoryRSS, uint64(0), "Delta should show memory increase")
}

func TestProcessScanner_FilteringWithPatterns(t *testing.T) {
	// Setup mock platform collector
	mockCollector := new(MockPlatformCollector)
	
	// Create test processes
	testProcesses := []*ProcessInfo{
		createTestProcessInfo(1, "include-proc1", "/bin/include-proc1", 1.0, 1024*1024),
		createTestProcessInfo(2, "include-proc2", "/bin/include-proc2", 2.0, 2*1024*1024),
		createTestProcessInfo(3, "exclude-proc1", "/bin/exclude-proc1", 3.0, 3*1024*1024),
		createTestProcessInfo(4, "normal-proc1", "/bin/normal-proc1", 4.0, 4*1024*1024),
		createTestProcessInfo(5, "exclude-proc2", "/bin/exclude-proc2", 5.0, 5*1024*1024),
	}
	
	// Configure mock behaviors
	mockCollector.On("GetProcesses").Return(testProcesses, nil)
	mockCollector.On("GetCPUTimes").Return(nil)
	mockCollector.On("GetSelfUsage").Return(0.3, uint64(20*1024*1024), nil)
	mockCollector.On("Shutdown").Return(nil)
	
	// Create scanner with filtering configuration
	config := ProcessScannerConfig{
		ScanInterval:    100 * time.Millisecond,
		IncludePatterns: []string{"^include.*$"},
		ExcludePatterns: []string{"^exclude.*$"},
	}
	scanner := NewProcessScanner(config)
	
	// Initialize scanner with mock
	initProcessScannerWithMock(t, scanner, mockCollector)
	
	// Setup mock consumer
	mockConsumer := new(MockProcessConsumer)
	mockConsumer.On("OnProcessEvent", mock.Anything).Return(nil)
	
	// Register consumer
	err := scanner.RegisterConsumer("test-consumer", mockConsumer)
	require.NoError(t, err)
	
	// Start scanner 
	err = scanner.Start()
	require.NoError(t, err)
	
	// Allow scanner to run
	time.Sleep(200 * time.Millisecond)
	
	// Stop scanner
	err = scanner.Stop()
	require.NoError(t, err)
	
	// Verify filtering results
	cachedProcesses := scanner.GetCachedProcesses()
	assert.Equal(t, 2, len(cachedProcesses), "Should have 2 processes after filtering")
	
	// Check which processes are cached
	pids := make(map[int]bool)
	for _, proc := range cachedProcesses {
		pids[proc.PID] = true
	}
	
	assert.True(t, pids[1], "Process 1 (include-proc1) should be included")
	assert.True(t, pids[2], "Process 2 (include-proc2) should be included")
	assert.False(t, pids[3], "Process 3 (exclude-proc1) should be excluded")
	assert.False(t, pids[4], "Process 4 (normal-proc1) should be excluded (doesn't match include pattern)")
	assert.False(t, pids[5], "Process 5 (exclude-proc2) should be excluded")
	
	// Verify events match filtering
	receivedEvents := mockConsumer.GetReceivedEvents()
	eventPids := make(map[int]bool)
	
	for _, event := range receivedEvents {
		if event.Type == ProcessCreated {
			eventPids[event.Process.PID] = true
		}
	}
	
	assert.True(t, eventPids[1], "Should have event for process 1")
	assert.True(t, eventPids[2], "Should have event for process 2")
	assert.False(t, eventPids[3], "Should not have event for process 3")
	assert.False(t, eventPids[4], "Should not have event for process 4")
	assert.False(t, eventPids[5], "Should not have event for process 5")
}

func TestProcessScanner_ErrorHandling(t *testing.T) {
	// Setup mock collector
	mockCollector := new(MockPlatformCollector)
	
	// Create test processes
	testProcesses := []*ProcessInfo{
		createTestProcessInfo(1, "process1", "/bin/process1", 1.0, 1024*1024),
	}
	
	// Configure mock behaviors
	// First call - success
	mockCollector.On("GetProcesses").Return(testProcesses, nil).Once()
	
	// Second call - error
	mockCollector.On("GetProcesses").Return([]*ProcessInfo{}, fmt.Errorf("simulated error")).Once()
	
	// Third call - success again
	mockCollector.On("GetProcesses").Return(testProcesses, nil).Once()
	
	mockCollector.On("GetCPUTimes").Return(nil)
	mockCollector.On("GetSelfUsage").Return(0.1, uint64(10*1024*1024), nil)
	mockCollector.On("Shutdown").Return(nil)
	
	// Create mock diagnostics service
	mockDiagnostics := new(MockDiagnosticsService)
	mockDiagnostics.On("EmitEvent", mock.Anything, mock.Anything, mock.Anything).Return()
	
	// Create scanner with test configuration
	config := ProcessScannerConfig{
		ScanInterval: 100 * time.Millisecond,
	}
	scanner := NewProcessScanner(config)
	scanner.diagnostics = mockDiagnostics
	
	// Initialize scanner with mock
	initProcessScannerWithMock(t, scanner, mockCollector)
	
	// Setup mock consumer
	mockConsumer := new(MockProcessConsumer)
	mockConsumer.On("OnProcessEvent", mock.Anything).Return(nil)
	
	// Register consumer
	err := scanner.RegisterConsumer("test-consumer", mockConsumer)
	require.NoError(t, err)
	
	// Start scanner 
	err = scanner.Start()
	require.NoError(t, err)
	
	// Wait for all three scans to complete
	time.Sleep(350 * time.Millisecond)
	
	// Stop scanner
	err = scanner.Stop()
	require.NoError(t, err)
	
	// Verify scanner is still running after error
	mockCollector.AssertNumberOfCalls(t, "GetProcesses", 3)
	
	// Verify error was recorded in metrics
	metrics := scanner.Metrics()
	assert.GreaterOrEqual(t, metrics[MetricScanErrors], 1.0, 
		"Should have at least one scan error recorded")
	
	// Verify diagnostic event was emitted
	mockDiagnostics.AssertCalled(t, "EmitEvent", "ProcessScanner", "ScanError", mock.Anything)
}

func TestProcessScanner_ResourceManagement(t *testing.T) {
	// Setup mock collector
	mockCollector := new(MockPlatformCollector)
	
	// Create test processes
	testProcesses := []*ProcessInfo{
		createTestProcessInfo(1, "process1", "/bin/process1", 1.0, 1024*1024),
		createTestProcessInfo(2, "process2", "/bin/process2", 2.0, 2*1024*1024),
	}
	
	// Configure mock behaviors
	mockCollector.On("GetProcesses").Return(testProcesses, nil)
	
	// First call - normal CPU usage
	mockCollector.On("GetSelfUsage").Return(0.3, uint64(10*1024*1024), nil).Once()
	
	// Second call - high CPU usage
	mockCollector.On("GetSelfUsage").Return(1.0, uint64(10*1024*1024), nil).Once()
	
	mockCollector.On("GetCPUTimes").Return(nil)
	mockCollector.On("Shutdown").Return(nil)
	
	// Create scanner with adaptive sampling config
	config := ProcessScannerConfig{
		ScanInterval:     100 * time.Millisecond,
		MaxCPUUsage:      0.5,
		AdaptiveSampling: true,
	}
	scanner := NewProcessScanner(config)
	
	// Initialize scanner with mock
	initProcessScannerWithMock(t, scanner, mockCollector)
	
	// Capture initial scan interval
	initialInterval := scanner.config.ScanInterval
	
	// Start scanner
	err := scanner.Start()
	require.NoError(t, err)
	
	// Wait for scans with different CPU usage
	time.Sleep(250 * time.Millisecond)
	
	// Stop scanner
	err = scanner.Stop()
	require.NoError(t, err)
	
	// Verify adaptive scanning adjusted the interval when CPU usage was high
	assert.Greater(t, scanner.config.ScanInterval, initialInterval, 
		"Scan interval should have increased due to high CPU usage")
}

// Mock DiagnosticsService for testing
type MockDiagnosticsService struct {
	mock.Mock
}

func (m *MockDiagnosticsService) EmitEvent(component, eventType string, description string) {
	m.Called(component, eventType, description)
}