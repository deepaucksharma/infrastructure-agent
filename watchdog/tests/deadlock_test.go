package tests

import (
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/watchdog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockComponentMonitor mocks the ComponentMonitor for testing
type MockComponentMonitor struct {
	mock.Mock
}

// AddDeadlockDetectedHandler implements the interface
func (m *MockComponentMonitor) AddDeadlockDetectedHandler(handler func(string, watchdog.ComponentMetrics)) {
	m.Called(handler)
}

// TestDeadlockDetectorCreation tests creating a deadlock detector
func TestDeadlockDetectorCreation(t *testing.T) {
	config := watchdog.Config{
		DeadlockDetection: watchdog.DeadlockConfig{
			Enabled:              true,
			CheckInterval:        10 * time.Second,
			GoroutineThreshold:   100,
		},
	}
	
	monitor := new(MockComponentMonitor)
	monitor.On("AddDeadlockDetectedHandler", mock.Anything).Return()
	
	detector := watchdog.NewDeadlockDetector(config, monitor)
	assert.NotNil(t, detector)
	
	monitor.AssertExpectations(t)
}

// TestDeadlockDetection tests detecting a deadlock
func TestDeadlockDetection(t *testing.T) {
	config := watchdog.Config{
		DeadlockDetection: watchdog.DeadlockConfig{
			Enabled:              true,
			CheckInterval:        10 * time.Second,
			GoroutineThreshold:   100,
		},
	}
	
	monitor := new(MockComponentMonitor)
	
	// Capture the handler function when it's registered
	var deadlockHandler func(string, watchdog.ComponentMetrics)
	monitor.On("AddDeadlockDetectedHandler", mock.Anything).Run(func(args mock.Arguments) {
		deadlockHandler = args.Get(0).(func(string, watchdog.ComponentMetrics))
	}).Return()
	
	detector := watchdog.NewDeadlockDetector(config, monitor)
	assert.NotNil(t, detector)
	
	// Make sure the handler was captured
	assert.NotNil(t, deadlockHandler)
	
	// Simulate a deadlock by calling the handler
	metrics := watchdog.ComponentMetrics{
		LastResponseTime: 30 * time.Second,
		GoroutineCount:   200,
		State:            "blocked",
		HealthStatus:     "degraded",
	}
	
	deadlockHandler("test-component", metrics)
	
	// Check that the deadlock was detected
	deadlocks := detector.GetDetectedDeadlocks()
	assert.Len(t, deadlocks, 1)
	assert.Contains(t, deadlocks, "test-component")
	
	// Verify deadlock info
	deadlock := deadlocks["test-component"]
	assert.Equal(t, "test-component", deadlock.ComponentName)
	assert.Equal(t, 30*time.Second, deadlock.LastResponseTime)
	assert.NotEmpty(t, deadlock.GoroutineStacks)
	assert.NotNil(t, deadlock.AdditionalInfo)
	assert.Equal(t, "blocked", deadlock.AdditionalInfo["state"])
	assert.Equal(t, "degraded", deadlock.AdditionalInfo["health"])
}

// TestClearDeadlock tests clearing a detected deadlock
func TestClearDeadlock(t *testing.T) {
	config := watchdog.Config{
		DeadlockDetection: watchdog.DeadlockConfig{
			Enabled:              true,
			CheckInterval:        10 * time.Second,
			GoroutineThreshold:   100,
		},
	}
	
	monitor := new(MockComponentMonitor)
	
	// Capture the handler function when it's registered
	var deadlockHandler func(string, watchdog.ComponentMetrics)
	monitor.On("AddDeadlockDetectedHandler", mock.Anything).Run(func(args mock.Arguments) {
		deadlockHandler = args.Get(0).(func(string, watchdog.ComponentMetrics))
	}).Return()
	
	detector := watchdog.NewDeadlockDetector(config, monitor)
	
	// Simulate a deadlock
	metrics := watchdog.ComponentMetrics{
		LastResponseTime: 30 * time.Second,
		GoroutineCount:   200,
		State:            "blocked",
		HealthStatus:     "degraded",
	}
	
	deadlockHandler("test-component", metrics)
	
	// Verify the deadlock was detected
	assert.True(t, detector.HasDeadlock("test-component"))
	
	// Clear the deadlock
	detector.ClearDeadlock("test-component")
	
	// Verify it was cleared
	assert.False(t, detector.HasDeadlock("test-component"))
	assert.Empty(t, detector.GetDetectedDeadlocks())
}

// TestMultipleDeadlocks tests detecting multiple deadlocks
func TestMultipleDeadlocks(t *testing.T) {
	config := watchdog.Config{
		DeadlockDetection: watchdog.DeadlockConfig{
			Enabled:              true,
			CheckInterval:        10 * time.Second,
			GoroutineThreshold:   100,
		},
	}
	
	monitor := new(MockComponentMonitor)
	
	// Capture the handler function when it's registered
	var deadlockHandler func(string, watchdog.ComponentMetrics)
	monitor.On("AddDeadlockDetectedHandler", mock.Anything).Run(func(args mock.Arguments) {
		deadlockHandler = args.Get(0).(func(string, watchdog.ComponentMetrics))
	}).Return()
	
	detector := watchdog.NewDeadlockDetector(config, monitor)
	
	// Simulate deadlocks in multiple components
	components := []string{"component1", "component2", "component3"}
	
	for _, component := range components {
		metrics := watchdog.ComponentMetrics{
			LastResponseTime: 30 * time.Second,
			GoroutineCount:   200,
			State:            "blocked",
			HealthStatus:     "degraded",
		}
		
		deadlockHandler(component, metrics)
	}
	
	// Verify all deadlocks were detected
	deadlocks := detector.GetDetectedDeadlocks()
	assert.Len(t, deadlocks, 3)
	
	for _, component := range components {
		assert.True(t, detector.HasDeadlock(component))
		assert.Contains(t, deadlocks, component)
	}
	
	// Clear one deadlock
	detector.ClearDeadlock("component2")
	
	// Verify only that one was cleared
	deadlocks = detector.GetDetectedDeadlocks()
	assert.Len(t, deadlocks, 2)
	assert.True(t, detector.HasDeadlock("component1"))
	assert.False(t, detector.HasDeadlock("component2"))
	assert.True(t, detector.HasDeadlock("component3"))
}

// TestDeadlockTimestamps tests that deadlock timestamps are set correctly
func TestDeadlockTimestamps(t *testing.T) {
	config := watchdog.Config{
		DeadlockDetection: watchdog.DeadlockConfig{
			Enabled:              true,
			CheckInterval:        10 * time.Second,
			GoroutineThreshold:   100,
		},
	}
	
	monitor := new(MockComponentMonitor)
	
	// Capture the handler function when it's registered
	var deadlockHandler func(string, watchdog.ComponentMetrics)
	monitor.On("AddDeadlockDetectedHandler", mock.Anything).Run(func(args mock.Arguments) {
		deadlockHandler = args.Get(0).(func(string, watchdog.ComponentMetrics))
	}).Return()
	
	detector := watchdog.NewDeadlockDetector(config, monitor)
	
	// Remember the current time
	beforeTime := time.Now()
	
	// Wait a bit
	time.Sleep(10 * time.Millisecond)
	
	// Simulate a deadlock
	metrics := watchdog.ComponentMetrics{
		LastResponseTime: 30 * time.Second,
		GoroutineCount:   200,
		State:            "blocked",
		HealthStatus:     "degraded",
	}
	
	deadlockHandler("test-component", metrics)
	
	// Wait a bit more
	time.Sleep(10 * time.Millisecond)
	afterTime := time.Now()
	
	// Get the deadlock info
	deadlocks := detector.GetDetectedDeadlocks()
	deadlock := deadlocks["test-component"]
	
	// Verify the timestamp is between the before and after times
	assert.True(t, deadlock.DetectedAt.After(beforeTime))
	assert.True(t, deadlock.DetectedAt.Before(afterTime))
}
