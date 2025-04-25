// Copyright 2025 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/collector"
	"github.com/newrelic/infrastructure-agent/sampler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPlatformCollector mocks the platform.ProcessCollector interface
type MockPlatformCollector struct {
	mock.Mock
}

func (m *MockPlatformCollector) GetProcesses() ([]*collector.ProcessInfo, error) {
	args := m.Called()
	return args.Get(0).([]*collector.ProcessInfo), args.Error(1)
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

// Helper function to initialize scanner with mock for tests
func initProcessScannerWithMock(t *testing.T, scanner *collector.ProcessScanner, mockCollector *MockPlatformCollector) {
	ctx := context.Background()
	
	// Pass the mock through the options
	options := map[string]interface{}{
		"mockCollector": mockCollector,
	}
	
	// Initialize the scanner with context and options
	scanner.Init(ctx, options)
}

// Test helpers
func createTestProcessInfo(pid int, name string, command string, cpu float64, rss uint64) *collector.ProcessInfo {
	return &collector.ProcessInfo{
		PID:        pid,
		PPID:       1,
		Name:       name,
		Command:    command,
		Username:   "test",
		State:      "running",
		CPUPercent: cpu,
		MemoryRSS:  rss,
		StartTime:  time.Now().Add(-time.Minute),
		Attributes: make(map[string]string),
	}
}

// Generate test processes with varying CPU/memory usage
func generateTestProcesses(count int) []*collector.ProcessInfo {
	processes := make([]*collector.ProcessInfo, count)
	for i := 0; i < count; i++ {
		// Create varying CPU and memory profiles for interesting test data
		var cpu float64
		var rss uint64
		
		// Create different categories of processes:
		// - High CPU, low memory
		// - Low CPU, high memory
		// - Balanced CPU and memory
		// - Low resource usage
		// - Extremely high resource usage
		
		switch i % 5 {
		case 0: // High CPU, low memory
			cpu = 50.0 + float64(i%10)
			rss = 50*1024*1024 + uint64(i%10)*1024*1024
		case 1: // Low CPU, high memory
			cpu = 5.0 + float64(i%10)
			rss = 500*1024*1024 + uint64(i%10)*10*1024*1024
		case 2: // Balanced
			cpu = 25.0 + float64(i%10)
			rss = 250*1024*1024 + uint64(i%10)*5*1024*1024
		case 3: // Low usage
			cpu = 1.0 + float64(i%5)
			rss = 10*1024*1024 + uint64(i%5)*1024*1024
		case 4: // Extremely high (outliers)
			cpu = 90.0 + float64(i%10)
			rss = 900*1024*1024 + uint64(i%10)*20*1024*1024
		}
		
		processes[i] = createTestProcessInfo(
			i,
			fmt.Sprintf("process%d", i),
			fmt.Sprintf("/bin/process%d", i),
			cpu,
			rss,
		)
	}
	return processes
}

// Tests integration between Process Scanner and TopN Sampler
func TestIntegration_ScannerTopNSampler(t *testing.T) {
	// Setup mock platform collector
	mockCollector := new(MockPlatformCollector)
	
	// Generate test processes with varying CPU/memory usage
	testProcesses := generateTestProcesses(200)
	
	// Configure mock behaviors
	mockCollector.On("GetProcesses").Return(testProcesses, nil)
	mockCollector.On("GetCPUTimes").Return(nil)
	mockCollector.On("GetSelfUsage").Return(0.3, uint64(20*1024*1024), nil)
	mockCollector.On("Shutdown").Return(nil)
	
	// Create ProcessScanner with test configuration
	scannerConfig := collector.ProcessScannerConfig{
		ScanInterval: 100 * time.Millisecond,
	}
	scanner := collector.NewProcessScanner(scannerConfig)
	
	// Initialize scanner with mock collector
	initProcessScannerWithMock(t, scanner, mockCollector)
	
	// Create TopN Sampler
	topNConfig := sampler.TopNConfig{
		MaxProcesses: 10,
		CPUWeight:    0.7,
		MemoryWeight: 0.3,
	}
	topnSampler := sampler.NewTopNSampler(topNConfig)
	
	// Initialize sampler
	ctx := context.Background()
	err := topnSampler.Init(ctx)
	require.NoError(t, err)
	
	// Register sampler as consumer of scanner
	err = scanner.RegisterConsumer("topn-sampler", topnSampler)
	require.NoError(t, err)
	
	// Start scanner
	err = scanner.Start()
	require.NoError(t, err)
	
	// Allow time for scan and processing
	time.Sleep(200 * time.Millisecond)
	
	// Verify TopN sampler has received and processed data
	topProcesses := topnSampler.GetTopN(10)
	
	// Verify correct number of processes
	assert.Equal(t, 10, len(topProcesses), "Should have 10 top processes")
	
	// Verify processes are correctly ordered by weighted score
	for i := 1; i < len(topProcesses); i++ {
		prevScore := computeScore(topProcesses[i-1], topNConfig.CPUWeight, topNConfig.MemoryWeight)
		currScore := computeScore(topProcesses[i], topNConfig.CPUWeight, topNConfig.MemoryWeight)
		assert.GreaterOrEqual(t, prevScore, currScore, 
			"Processes should be ordered by decreasing score")
	}
	
	// Verify metrics
	scannerMetrics := scanner.Metrics()
	samplerMetrics := topnSampler.Metrics()
	
	assert.Equal(t, float64(200), scannerMetrics[collector.MetricProcessCount], 
		"Scanner should track 200 processes")
	assert.Equal(t, float64(10), samplerMetrics["topn_processes_sampled"], 
		"Sampler should sample 10 processes")
	
	// Verify the capture ratio meets requirements
	captureRatio := samplerMetrics["topn_capture_ratio"]
	assert.GreaterOrEqual(t, captureRatio, 95.0, 
		"TopN sampler should have ≥95% capture ratio (G-4: Top-N Accuracy)")
	
	// Stop scanner and sampler
	err = scanner.Stop()
	require.NoError(t, err)
	
	err = topnSampler.Shutdown()
	require.NoError(t, err)
}

// Tests integration with process churn
func TestIntegration_ScannerTopNSampler_WithChurn(t *testing.T) {
	// Setup mock platform collector
	mockCollector := new(MockPlatformCollector)
	
	// Create initial processes
	initialProcesses := generateTestProcesses(100)
	
	// Create updated processes - some removed, some new, some updated
	updatedProcesses := make([]*collector.ProcessInfo, 100)
	
	// Copy 70 existing processes (0-69) with small changes
	for i := 0; i < 70; i++ {
		proc := initialProcesses[i].Clone()
		// Small random changes to CPU and memory
		proc.CPUPercent += float64(i % 10)
		proc.MemoryRSS += uint64(i%10) * 1024 * 1024
		updatedProcesses[i] = proc
	}
	
	// Add 30 new processes (with PIDs 500-529)
	for i := 0; i < 30; i++ {
		updatedProcesses[70+i] = createTestProcessInfo(
			500+i,
			fmt.Sprintf("newprocess%d", i),
			fmt.Sprintf("/bin/newprocess%d", i),
			float64(50+i),
			uint64((100+i)*1024*1024),
		)
	}
	
	// Configure mock behaviors
	mockCollector.On("GetProcesses").Return(initialProcesses, nil).Once()
	mockCollector.On("GetProcesses").Return(updatedProcesses, nil).Once()
	mockCollector.On("GetCPUTimes").Return(nil)
	mockCollector.On("GetSelfUsage").Return(0.3, uint64(20*1024*1024), nil)
	mockCollector.On("Shutdown").Return(nil)
	
	// Create ProcessScanner with test configuration
	scannerConfig := collector.ProcessScannerConfig{
		ScanInterval: 100 * time.Millisecond,
	}
	scanner := collector.NewProcessScanner(scannerConfig)
	
	// Initialize scanner with mock collector
	initProcessScannerWithMock(t, scanner, mockCollector)
	
	// Create TopN Sampler with churn handling enabled
	topNConfig := sampler.TopNConfig{
		MaxProcesses:         10,
		ChurnHandlingEnabled: true,
		ChurnThreshold:       50, // High threshold for this test
	}
	topnSampler := sampler.NewTopNSampler(topNConfig)
	
	// Initialize sampler
	ctx := context.Background()
	err := topnSampler.Init(ctx)
	require.NoError(t, err)
	
	// Register sampler as consumer of scanner
	err = scanner.RegisterConsumer("topn-sampler", topnSampler)
	require.NoError(t, err)
	
	// Start scanner
	err = scanner.Start()
	require.NoError(t, err)
	
	// Allow time for first scan
	time.Sleep(150 * time.Millisecond)
	
	// Get initial top processes
	initialTopProcesses := topnSampler.GetTopN(10)
	initialPIDs := make(map[int]bool)
	for _, proc := range initialTopProcesses {
		initialPIDs[proc.PID] = true
	}
	
	// Allow time for second scan
	time.Sleep(150 * time.Millisecond)
	
	// Get updated top processes
	updatedTopProcesses := topnSampler.GetTopN(10)
	updatedPIDs := make(map[int]bool)
	for _, proc := range updatedTopProcesses {
		updatedPIDs[proc.PID] = true
	}
	
	// Check for changes in top processes
	newTopProcessCount := 0
	for pid := range updatedPIDs {
		if !initialPIDs[pid] {
			newTopProcessCount++
		}
	}
	
	// Verify some of the new processes made it into the top
	assert.Greater(t, newTopProcessCount, 0, 
		"At least some of the new high-resource processes should appear in top processes")
	
	// Check churn rate metrics
	samplerMetrics := topnSampler.Metrics()
	assert.Greater(t, samplerMetrics["topn_churn_rate"], 20.0, 
		"Should have significant process churn rate")
	
	// Verify the sampler is still functioning properly
	assert.Equal(t, 10, len(updatedTopProcesses), "Should still have 10 top processes")
	assert.GreaterOrEqual(t, samplerMetrics["topn_capture_ratio"], 90.0, 
		"Should maintain good capture ratio even with churn")
	
	// Stop scanner and sampler
	err = scanner.Stop()
	require.NoError(t, err)
	
	err = topnSampler.Shutdown()
	require.NoError(t, err)
}

// Tests proper handling of deltas and statistical aggregation
func TestIntegration_ScannerTopNSampler_StatisticalFidelity(t *testing.T) {
	// This test validates G-3: Statistical Fidelity and G-5: Tail Error Bound
	
	// Setup mock platform collector
	mockCollector := new(MockPlatformCollector)
	
	// Generate a set of processes with a specific statistical distribution
	// for validating accuracy of percentile calculations
	const processCount = 1000
	testProcesses := make([]*collector.ProcessInfo, processCount)
	
	// Create processes with exponential distribution of CPU usage
	// This will test the accuracy of tail statistics (p95, p99)
	for i := 0; i < processCount; i++ {
		percentile := float64(i) / float64(processCount)
		
		// Map percentile to CPU usage using exponential distribution
		// Most processes will have low CPU, a few will have very high CPU
		cpuPct := -10.0 * math.Log(1.0-percentile)
		
		// Cap at 100% for realism
		if cpuPct > 100.0 {
			cpuPct = 100.0
		}
		
		// Create memory usage similarly but with different scale
		memMB := uint64(-100.0 * math.Log(1.0-percentile))
		
		// Cap at reasonable value
		if memMB > 1000 {
			memMB = 1000
		}
		
		testProcesses[i] = createTestProcessInfo(
			i,
			fmt.Sprintf("process%d", i),
			fmt.Sprintf("/bin/process%d", i),
			cpuPct,
			memMB * 1024 * 1024,
		)
	}
	
	// Calculate exact p95 CPU value for comparison
	expectedP95CPU := -10.0 * math.Log(1.0-0.95)
	if expectedP95CPU > 100.0 {
		expectedP95CPU = 100.0
	}
	
	// Configure mock
	mockCollector.On("GetProcesses").Return(testProcesses, nil)
	mockCollector.On("GetCPUTimes").Return(nil)
	mockCollector.On("GetSelfUsage").Return(0.3, uint64(20*1024*1024), nil)
	mockCollector.On("Shutdown").Return(nil)
	
	// Create ProcessScanner
	scannerConfig := collector.ProcessScannerConfig{
		ScanInterval: 100 * time.Millisecond,
	}
	scanner := collector.NewProcessScanner(scannerConfig)
	
	// Initialize scanner with mock collector
	initProcessScannerWithMock(t, scanner, mockCollector)
	
	// Create TopN Sampler with sketch enabled
	topNConfig := sampler.TopNConfig{
		MaxProcesses:  50, // Use larger size to better validate statistics
		SketchEnabled: true,
	}
	topnSampler := sampler.NewTopNSampler(topNConfig)
	
	// Initialize sampler
	ctx := context.Background()
	err := topnSampler.Init(ctx)
	require.NoError(t, err)
	
	// Register sampler as consumer of scanner
	err = scanner.RegisterConsumer("topn-sampler", topnSampler)
	require.NoError(t, err)
	
	// Start scanner
	err = scanner.Start()
	require.NoError(t, err)
	
	// Allow time for scan and sketch computation
	time.Sleep(200 * time.Millisecond)
	
	// Get sketch statistics
	p95CPU := topnSampler.GetCPUPercentile(0.95)
	p99CPU := topnSampler.GetCPUPercentile(0.99)
	
	// Verify statistical fidelity
	cpuError := math.Abs(p95CPU-expectedP95CPU) / expectedP95CPU
	assert.LessOrEqual(t, cpuError, 0.01, 
		"p95 CPU error should be ≤1% (G-3: Statistical Fidelity)")
	
	// Verify tail error bound
	totalCPUExact := 0.0
	for _, proc := range testProcesses {
		totalCPUExact += proc.CPUPercent
	}
	
	totalCPUMeasured := topnSampler.GetTotalCPU()
	aggregationError := math.Abs(totalCPUMeasured-totalCPUExact) / totalCPUExact
	
	assert.LessOrEqual(t, aggregationError, 0.05, 
		"CPU sum error should be ≤5% (G-5: Tail Error Bound)")
	
	// Stop scanner and sampler
	err = scanner.Stop()
	require.NoError(t, err)
	
	err = topnSampler.Shutdown()
	require.NoError(t, err)
}

// Helper function to compute process score based on weights
func computeScore(process *collector.ProcessInfo, cpuWeight, memoryWeight float64) float64 {
	return cpuWeight*process.CPUPercent + memoryWeight*float64(process.MemoryRSS)
}