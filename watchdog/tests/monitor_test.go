package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/watchdog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMonitorableComponent implements the Component interface for testing
type MockMonitorableComponent struct {
	mock.Mock
	name string
	resourceUsage watchdog.ResourceUsage
	mutex sync.RWMutex
	running bool
}

// Name implements Component interface
func (m *MockMonitorableComponent) Name() string {
	return m.name
}

// ResourceUsage implements Component interface
func (m *MockMonitorableComponent) ResourceUsage() watchdog.ResourceUsage {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.resourceUsage
}

// SetResourceUsage sets the resource usage for testing
func (m *MockMonitorableComponent) SetResourceUsage(usage watchdog.ResourceUsage) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.resourceUsage = usage
}

// Heartbeat implements Component interface
func (m *MockMonitorableComponent) Heartbeat() error {
	args := m.Called()
	return args.Error(0)
}

// Shutdown implements Component interface
func (m *MockMonitorableComponent) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	m.mutex.Lock()
	m.running = false
	m.mutex.Unlock()
	return args.Error(0)
}

// Start implements Component interface
func (m *MockMonitorableComponent) Start() error {
	args := m.Called()
	m.mutex.Lock()
	m.running = true
	m.mutex.Unlock()
	return args.Error(0)
}

// NewMockMonitorableComponent creates a new mock monitorable component
func NewMockMonitorableComponent(name string) *MockMonitorableComponent {
	component := &MockMonitorableComponent{
		name: name,
		resourceUsage: watchdog.ResourceUsage{
			CPUPercent: 10.0,
			MemoryBytes: 100 * 1024 * 1024, // 100 MB
			FileDescriptors: 10,
			Goroutines: 5,
			LastUpdated: time.Now(),
		},
		running: true,
	}
	
	component.On("Heartbeat").Return(nil)
	component.On("Shutdown", mock.Anything).Return(nil)
	component.On("Start").Return(nil)
	
	return component
}

func TestMonitorCreation(t *testing.T) {
	config := watchdog.Config{
		MonitoringInterval: 100 * time.Millisecond,
		ComponentConfigs: map[string]watchdog.ComponentConfig{
			"test-component": {
				Enabled: true,
				MaxCPUPercent: 80.0,
				MaxMemoryMB: 200,
				MaxFileDescriptors: 1000,
				MaxGoroutines: 100,
			},
		},
	}
	
	monitor := watchdog.NewResourceMonitor(config)
	assert.NotNil(t, monitor)
}

func TestAddRemoveComponent(t *testing.T) {
	config := watchdog.Config{
		MonitoringInterval: 100 * time.Millisecond,
		ComponentConfigs: map[string]watchdog.ComponentConfig{
			"test-component": {
				Enabled: true,
				MaxCPUPercent: 80.0,
				MaxMemoryMB: 200,
				MaxFileDescriptors: 1000,
				MaxGoroutines: 100,
			},
		},
	}
	
	monitor := watchdog.NewResourceMonitor(config)
	
	// Create a component
	component := NewMockMonitorableComponent("test-component")
	
	// Add the component
	err := monitor.AddComponent(component)
	assert.NoError(t, err)
	
	// Get resource usage for the component
	usage, ok := monitor.GetResourceUsage("test-component")
	assert.True(t, ok)
	assert.InDelta(t, 10.0, usage.CPUPercent, 0.1)
	assert.InDelta(t, 100*1024*1024, float64(usage.MemoryBytes), 1024)
	
	// Remove the component
	monitor.RemoveComponent("test-component")
	
	// Verify it's no longer available
	_, ok = monitor.GetResourceUsage("test-component")
	assert.False(t, ok)
}

func TestMonitorStartStop(t *testing.T) {
	config := watchdog.Config{
		MonitoringInterval: 10 * time.Millisecond,
		ComponentConfigs: map[string]watchdog.ComponentConfig{
			"test-component": {
				Enabled: true,
				MaxCPUPercent: 80.0,
				MaxMemoryMB: 200,
				MaxFileDescriptors: 1000,
				MaxGoroutines: 100,
			},
		},
	}
	
	monitor := watchdog.NewResourceMonitor(config)
	
	// Start the monitor
	err := monitor.Start()
	assert.NoError(t, err)
	
	// Stop the monitor
	err = monitor.Stop()
	assert.NoError(t, err)
}

func TestThresholdHandlers(t *testing.T) {
	config := watchdog.Config{
		MonitoringInterval: 10 * time.Millisecond,
		ComponentConfigs: map[string]watchdog.ComponentConfig{
			"test-component": {
				Enabled: true,
				MaxCPUPercent: 80.0,
				MaxMemoryMB: 200,
				MaxFileDescriptors: 1000,
				MaxGoroutines: 100,
			},
		},
	}
	
	monitor := watchdog.NewResourceMonitor(config)
	
	// Create a component with high resource usage
	component := NewMockMonitorableComponent("test-component")
	component.SetResourceUsage(watchdog.ResourceUsage{
		CPUPercent: 90.0, // > 80.0 threshold
		MemoryBytes: 300 * 1024 * 1024, // > 200 MB threshold
		FileDescriptors: 50,
		Goroutines: 50,
		LastUpdated: time.Now(),
	})
	
	// Add the component
	err := monitor.AddComponent(component)
	assert.NoError(t, err)
	
	// Create a channel to receive threshold events
	eventCh := make(chan watchdog.ThresholdExceededEvent, 10)
	
	// Add a threshold handler
	monitor.AddThresholdHandler(func(event watchdog.ThresholdExceededEvent) {
		eventCh <- event
	})
	
	// Start the monitor
	err = monitor.Start()
	assert.NoError(t, err)
	
	// Wait for events
	var cpuEvent, memoryEvent watchdog.ThresholdExceededEvent
	timeout := time.After(200 * time.Millisecond)
	eventCount := 0
	
eventLoop:
	for {
		select {
		case event := <-eventCh:
			if event.ResourceType == "CPU" {
				cpuEvent = event
			} else if event.ResourceType == "Memory" {
				memoryEvent = event
			}
			eventCount++
			if eventCount >= 2 {
				break eventLoop
			}
		case <-timeout:
			break eventLoop
		}
	}
	
	// Stop the monitor
	err = monitor.Stop()
	assert.NoError(t, err)
	
	// Verify the CPU event
	assert.Equal(t, "test-component", cpuEvent.ComponentName)
	assert.Equal(t, "CPU", cpuEvent.ResourceType)
	assert.InDelta(t, 90.0, cpuEvent.CurrentValue, 0.1)
	assert.InDelta(t, 80.0, cpuEvent.ThresholdValue, 0.1)
	
	// Verify the memory event
	assert.Equal(t, "test-component", memoryEvent.ComponentName)
	assert.Equal(t, "Memory", memoryEvent.ResourceType)
	assert.InDelta(t, 300.0, memoryEvent.CurrentValue, 1.0) // MB
	assert.InDelta(t, 200.0, memoryEvent.ThresholdValue, 0.1)
}

func TestResourceHistory(t *testing.T) {
	config := watchdog.Config{
		MonitoringInterval: 10 * time.Millisecond,
		ComponentConfigs: map[string]watchdog.ComponentConfig{
			"test-component": {
				Enabled: true,
				MaxCPUPercent: 80.0,
				MaxMemoryMB: 200,
				MaxFileDescriptors: 1000,
				MaxGoroutines: 100,
			},
		},
	}
	
	monitor := watchdog.NewResourceMonitor(config)
	
	// Create a component
	component := NewMockMonitorableComponent("test-component")
	
	// Add the component
	err := monitor.AddComponent(component)
	assert.NoError(t, err)
	
	// Start the monitor
	err = monitor.Start()
	assert.NoError(t, err)
	
	// Wait for some history to accumulate
	time.Sleep(50 * time.Millisecond)
	
	// Change resource usage multiple times
	for i := 0; i < 5; i++ {
		component.SetResourceUsage(watchdog.ResourceUsage{
			CPUPercent: 10.0 + float64(i*10),
			MemoryBytes: (100 + uint64(i*50)) * 1024 * 1024,
			FileDescriptors: 10 + i*5,
			Goroutines: 5 + i*2,
			LastUpdated: time.Now(),
		})
		time.Sleep(15 * time.Millisecond)
	}
	
	// Get resource history
	history, ok := monitor.GetResourceHistory("test-component")
	assert.True(t, ok)
	assert.NotEmpty(t, history)
	
	// Stop the monitor
	err = monitor.Stop()
	assert.NoError(t, err)
}

func TestDegradationLevels(t *testing.T) {
	config := watchdog.Config{
		MonitoringInterval: 10 * time.Millisecond,
		ComponentConfigs: map[string]watchdog.ComponentConfig{
			"test-component": {
				Enabled: true,
				MaxCPUPercent: 80.0,
				MaxMemoryMB: 200,
				MaxFileDescriptors: 1000,
				MaxGoroutines: 100,
				DegradationLevels: []watchdog.DegradationLevel{
					{
						Name: "warning",
						CPUThresholdPercent: 60.0,
						MemoryThresholdMB: 150,
						Actions: []string{"reduce_frequency"},
						Description: "Warning level",
					},
					{
						Name: "critical",
						CPUThresholdPercent: 70.0,
						MemoryThresholdMB: 180,
						Actions: []string{"reduce_frequency", "disable_features"},
						Description: "Critical level",
					},
				},
			},
		},
	}
	
	monitor := watchdog.NewResourceMonitor(config)
	
	// Create a component
	component := NewMockMonitorableComponent("test-component")
	
	// Add the component
	err := monitor.AddComponent(component)
	assert.NoError(t, err)
	
	// Start the monitor
	err = monitor.Start()
	assert.NoError(t, err)
	
	// Set resource usage to warning level
	component.SetResourceUsage(watchdog.ResourceUsage{
		CPUPercent: 65.0, // > 60.0 warning threshold
		MemoryBytes: 160 * 1024 * 1024, // > 150 MB warning threshold
		FileDescriptors: 50,
		Goroutines: 50,
		LastUpdated: time.Now(),
	})
	
	// Wait for degradation to be detected
	time.Sleep(50 * time.Millisecond)
	
	// Get degradation level
	level, ok := monitor.GetDegradationLevel("test-component")
	assert.True(t, ok)
	assert.Equal(t, "warning", level)
	
	// Increase to critical level
	component.SetResourceUsage(watchdog.ResourceUsage{
		CPUPercent: 75.0, // > 70.0 critical threshold
		MemoryBytes: 190 * 1024 * 1024, // > 180 MB critical threshold
		FileDescriptors: 50,
		Goroutines: 50,
		LastUpdated: time.Now(),
	})
	
	// Wait for degradation to be updated
	time.Sleep(50 * time.Millisecond)
	
	// Get new degradation level
	level, ok = monitor.GetDegradationLevel("test-component")
	assert.True(t, ok)
	assert.Equal(t, "critical", level)
	
	// Reduce to normal level
	component.SetResourceUsage(watchdog.ResourceUsage{
		CPUPercent: 50.0, // Below warning threshold
		MemoryBytes: 100 * 1024 * 1024, // Below warning threshold
		FileDescriptors: 50,
		Goroutines: 50,
		LastUpdated: time.Now(),
	})
	
	// Wait for degradation to be updated
	time.Sleep(50 * time.Millisecond)
	
	// Get new degradation level (should be empty = no degradation)
	level, ok = monitor.GetDegradationLevel("test-component")
	assert.True(t, ok)
	assert.Equal(t, "", level)
	
	// Stop the monitor
	err = monitor.Stop()
	assert.NoError(t, err)
}

func TestTotalResourceUsage(t *testing.T) {
	config := watchdog.Config{
		MonitoringInterval: 10 * time.Millisecond,
	}
	
	monitor := watchdog.NewResourceMonitor(config)
	
	// Get total resource usage
	usage := monitor.GetTotalResourceUsage()
	
	// Basic validation of returned data
	assert.True(t, usage.MemoryBytes > 0)
	assert.True(t, usage.Goroutines > 0)
	assert.False(t, usage.LastUpdated.IsZero())
}
