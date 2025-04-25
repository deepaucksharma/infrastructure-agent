package tests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/watchdog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockComponent implements the required interfaces for testing the watchdog
type MockComponent struct {
	mock.Mock
	healthStatus watchdog.HealthStatus
	mutex        sync.RWMutex
	running      bool
	degradLevel  int
}

// GetResourceUsage implements the Monitorable interface
func (m *MockComponent) GetResourceUsage() watchdog.ResourceUsage {
	args := m.Called()
	return args.Get(0).(watchdog.ResourceUsage)
}

// GetHealth implements the Monitorable interface
func (m *MockComponent) GetHealth() watchdog.HealthStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.healthStatus
}

// SetHealth sets the health status for testing
func (m *MockComponent) SetHealth(health watchdog.HealthStatus) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.healthStatus = health
}

// SetResourceUsage sets up the mock to return a specific resource usage
func (m *MockComponent) SetResourceUsage(usage watchdog.ResourceUsage) {
	m.On("GetResourceUsage").Return(usage)
}

// Shutdown implements the Restartable interface
func (m *MockComponent) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	
	m.mutex.Lock()
	m.running = false
	m.mutex.Unlock()
	
	return args.Error(0)
}

// Start implements the Restartable interface
func (m *MockComponent) Start(ctx context.Context) error {
	args := m.Called(ctx)
	
	m.mutex.Lock()
	m.running = true
	m.mutex.Unlock()
	
	return args.Error(0)
}

// IsRunning implements the Restartable interface
func (m *MockComponent) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.running
}

// SetDegradationLevel implements the Degradable interface
func (m *MockComponent) SetDegradationLevel(level int) error {
	args := m.Called(level)
	
	if args.Error(0) == nil {
		m.mutex.Lock()
		m.degradLevel = level
		m.mutex.Unlock()
	}
	
	return args.Error(0)
}

// GetDegradationLevel implements the Degradable interface
func (m *MockComponent) GetDegradationLevel() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.degradLevel
}

// NewMockComponent creates a new mock component for testing
func NewMockComponent() *MockComponent {
	mock := &MockComponent{
		healthStatus: watchdog.HealthOK,
		running:      true,
		degradLevel:  0,
	}
	
	// Setup default behavior
	mock.On("Shutdown", mock.Anything).Return(nil)
	mock.On("Start", mock.Anything).Return(nil)
	mock.On("SetDegradationLevel", mock.Anything).Return(nil)
	
	// Setup default resource usage
	defaultUsage := watchdog.ResourceUsage{
		CPUPercent:  1.0,
		MemoryMB:    10.0,
		Goroutines:  10,
		FileHandles: 5,
		GCPercent:   0.5,
		Timestamp:   time.Now(),
	}
	mock.SetResourceUsage(defaultUsage)
	
	return mock
}

func TestWatchdogStartStop(t *testing.T) {
	config := watchdog.Config{
		MonitorInterval: 10 * time.Millisecond,
		GlobalThresholds: watchdog.ResourceThresholds{
			MaxCPUPercent:  90.0,
			MaxMemoryMB:    1000,
			MaxGoroutines:  1000,
			MaxFileHandles: 1000,
			MaxGCPercent:   10.0,
		},
		DeadlockDetection: watchdog.DeadlockConfig{
			Enabled:             true,
			CheckInterval:       50 * time.Millisecond,
			GoroutineThreshold:  1000,
		},
		DegradationEnabled:  true,
		DegradationLevels:   3,
		EventsEnabled:       true,
	}
	
	wd, err := watchdog.NewWatchdog(config)
	assert.NoError(t, err)
	assert.NotNil(t, wd)
	
	// Start the watchdog
	err = wd.Start()
	assert.NoError(t, err)
	
	// Stop the watchdog
	err = wd.Stop()
	assert.NoError(t, err)
}

func TestRegisterComponent(t *testing.T) {
	config := watchdog.Config{
		MonitorInterval: 10 * time.Millisecond,
		GlobalThresholds: watchdog.ResourceThresholds{
			MaxCPUPercent:  90.0,
			MaxMemoryMB:    1000,
			MaxGoroutines:  1000,
			MaxFileHandles: 1000,
			MaxGCPercent:   10.0,
		},
	}
	
	wd, err := watchdog.NewWatchdog(config)
	assert.NoError(t, err)
	
	// Create a mock component
	mockComponent := NewMockComponent()
	
	// Register the component
	err = wd.RegisterComponent("test-component", mockComponent)
	assert.NoError(t, err)
	
	// Get component status
	status, err := wd.GetComponentStatus("test-component")
	assert.NoError(t, err)
	assert.Equal(t, "test-component", status.Name)
	assert.Equal(t, watchdog.HealthUnknown, status.Health)
	assert.Equal(t, watchdog.CircuitClosed, status.CircuitState)
	assert.Equal(t, 0, status.RestartCount)
	assert.Equal(t, 0, status.DegradationLevel)
	
	// Try to register the same component again
	err = wd.RegisterComponent("test-component", mockComponent)
	assert.Error(t, err)
	
	// Register a component that doesn't implement Monitorable
	err = wd.RegisterComponent("invalid-component", &struct{}{})
	assert.Error(t, err)
}

func TestUnregisterComponent(t *testing.T) {
	config := watchdog.Config{
		MonitorInterval: 10 * time.Millisecond,
		GlobalThresholds: watchdog.ResourceThresholds{
			MaxCPUPercent:  90.0,
			MaxMemoryMB:    1000,
			MaxGoroutines:  1000,
			MaxFileHandles: 1000,
			MaxGCPercent:   10.0,
		},
	}
	
	wd, err := watchdog.NewWatchdog(config)
	assert.NoError(t, err)
	
	// Create a mock component
	mockComponent := NewMockComponent()
	
	// Register the component
	err = wd.RegisterComponent("test-component", mockComponent)
	assert.NoError(t, err)
	
	// Unregister the component
	err = wd.UnregisterComponent("test-component")
	assert.NoError(t, err)
	
	// Check that the component is no longer registered
	_, err = wd.GetComponentStatus("test-component")
	assert.Error(t, err)
	
	// Try to unregister a non-registered component
	err = wd.UnregisterComponent("non-existent")
	assert.Error(t, err)
}

func TestGetAllComponentStatuses(t *testing.T) {
	config := watchdog.Config{
		MonitorInterval: 10 * time.Millisecond,
		GlobalThresholds: watchdog.ResourceThresholds{
			MaxCPUPercent:  90.0,
			MaxMemoryMB:    1000,
			MaxGoroutines:  1000,
			MaxFileHandles: 1000,
			MaxGCPercent:   10.0,
		},
	}
	
	wd, err := watchdog.NewWatchdog(config)
	assert.NoError(t, err)
	
	// Create mock components
	component1 := NewMockComponent()
	component2 := NewMockComponent()
	
	// Register the components
	err = wd.RegisterComponent("component1", component1)
	assert.NoError(t, err)
	
	err = wd.RegisterComponent("component2", component2)
	assert.NoError(t, err)
	
	// Get all component statuses
	statuses := wd.GetAllComponentStatuses()
	assert.Len(t, statuses, 2)
	assert.Contains(t, statuses, "component1")
	assert.Contains(t, statuses, "component2")
}

func TestSetThresholds(t *testing.T) {
	config := watchdog.Config{
		MonitorInterval: 10 * time.Millisecond,
		GlobalThresholds: watchdog.ResourceThresholds{
			MaxCPUPercent:  90.0,
			MaxMemoryMB:    1000,
			MaxGoroutines:  1000,
			MaxFileHandles: 1000,
			MaxGCPercent:   10.0,
		},
	}
	
	wd, err := watchdog.NewWatchdog(config)
	assert.NoError(t, err)
	
	// Create a mock component
	mockComponent := NewMockComponent()
	
	// Register the component
	err = wd.RegisterComponent("test-component", mockComponent)
	assert.NoError(t, err)
	
	// Set new thresholds
	newThresholds := watchdog.ResourceThresholds{
		MaxCPUPercent:  50.0,
		MaxMemoryMB:    500,
		MaxGoroutines:  500,
		MaxFileHandles: 500,
		MaxGCPercent:   5.0,
	}
	
	err = wd.SetThresholds("test-component", newThresholds)
	assert.NoError(t, err)
	
	// Try to set thresholds for a non-registered component
	err = wd.SetThresholds("non-existent", newThresholds)
	assert.Error(t, err)
}

func TestComponentMonitoring(t *testing.T) {
	config := watchdog.Config{
		MonitorInterval: 10 * time.Millisecond,
		GlobalThresholds: watchdog.ResourceThresholds{
			MaxCPUPercent:  90.0,
			MaxMemoryMB:    1000,
			MaxGoroutines:  1000,
			MaxFileHandles: 1000,
			MaxGCPercent:   10.0,
		},
	}
	
	wd, err := watchdog.NewWatchdog(config)
	assert.NoError(t, err)
	
	// Create a mock component
	mockComponent := NewMockComponent()
	
	// Set resource usage that exceeds thresholds
	highUsage := watchdog.ResourceUsage{
		CPUPercent:  95.0,
		MemoryMB:    1500.0,
		Goroutines:  1500,
		FileHandles: 1500,
		GCPercent:   15.0,
		Timestamp:   time.Now(),
	}
	
	mockComponent.SetResourceUsage(highUsage)
	mockComponent.SetHealth(watchdog.HealthDegraded)
	
	// Register the component
	err = wd.RegisterComponent("test-component", mockComponent)
	assert.NoError(t, err)
	
	// Set component-specific thresholds
	componentThresholds := watchdog.ResourceThresholds{
		MaxCPUPercent:  50.0,
		MaxMemoryMB:    500,
		MaxGoroutines:  500,
		MaxFileHandles: 500,
		MaxGCPercent:   5.0,
	}
	
	err = wd.SetThresholds("test-component", componentThresholds)
	assert.NoError(t, err)
	
	// Start the watchdog
	err = wd.Start()
	assert.NoError(t, err)
	
	// Wait for monitoring to trigger
	time.Sleep(50 * time.Millisecond)
	
	// Get component status
	status, err := wd.GetComponentStatus("test-component")
	assert.NoError(t, err)
	
	// Check that health was updated
	assert.Equal(t, watchdog.HealthDegraded, status.Health)
	
	// Check that resource usage was updated
	assert.InDelta(t, 95.0, status.ResourceUsage.CPUPercent, 0.1)
	assert.InDelta(t, 1500.0, status.ResourceUsage.MemoryMB, 0.1)
	
	// Check that circuit breaker was updated (should be open due to threshold violations)
	assert.Equal(t, watchdog.CircuitOpen, status.CircuitState)
	
	// Check that incidents were recorded
	assert.Greater(t, len(status.Incidents), 0)
	
	// Stop the watchdog
	err = wd.Stop()
	assert.NoError(t, err)
}

func TestRestartableComponent(t *testing.T) {
	config := watchdog.Config{
		MonitorInterval: 10 * time.Millisecond,
		GlobalThresholds: watchdog.ResourceThresholds{
			MaxCPUPercent:  90.0,
			MaxMemoryMB:    1000,
			MaxGoroutines:  1000,
			MaxFileHandles: 1000,
			MaxGCPercent:   10.0,
		},
	}
	
	wd, err := watchdog.NewWatchdog(config)
	assert.NoError(t, err)
	
	// Create a mock component that will exceed thresholds
	mockComponent := NewMockComponent()
	
	// Set resource usage that exceeds thresholds
	highUsage := watchdog.ResourceUsage{
		CPUPercent:  95.0,
		MemoryMB:    1500.0,
		Goroutines:  1500,
		FileHandles: 1500,
		GCPercent:   15.0,
		Timestamp:   time.Now(),
	}
	
	mockComponent.SetResourceUsage(highUsage)
	mockComponent.SetHealth(watchdog.HealthCritical)
	
	// Register the component
	err = wd.RegisterComponent("test-component", mockComponent)
	assert.NoError(t, err)
	
	// Start the watchdog
	err = wd.Start()
	assert.NoError(t, err)
	
	// Wait for restart to happen
	time.Sleep(100 * time.Millisecond)
	
	// Get component status
	status, err := wd.GetComponentStatus("test-component")
	assert.NoError(t, err)
	
	// Check that the component was restarted
	assert.GreaterOrEqual(t, status.RestartCount, 1)
	assert.False(t, status.LastRestart.IsZero())
	
	// Stop the watchdog
	err = wd.Stop()
	assert.NoError(t, err)
}

func TestFailedRestartComponent(t *testing.T) {
	config := watchdog.Config{
		MonitorInterval: 10 * time.Millisecond,
		GlobalThresholds: watchdog.ResourceThresholds{
			MaxCPUPercent:  90.0,
			MaxMemoryMB:    1000,
			MaxGoroutines:  1000,
			MaxFileHandles: 1000,
			MaxGCPercent:   10.0,
		},
	}
	
	wd, err := watchdog.NewWatchdog(config)
	assert.NoError(t, err)
	
	// Create a mock component that will exceed thresholds
	mockComponent := NewMockComponent()
	
	// Set resource usage that exceeds thresholds
	highUsage := watchdog.ResourceUsage{
		CPUPercent:  95.0,
		MemoryMB:    1500.0,
		Goroutines:  1500,
		FileHandles: 1500,
		GCPercent:   15.0,
		Timestamp:   time.Now(),
	}
	
	mockComponent.SetResourceUsage(highUsage)
	mockComponent.SetHealth(watchdog.HealthCritical)
	
	// Make restart fail
	mockComponent.On("Start", mock.Anything).Return(errors.New("failed to start"))
	
	// Register the component
	err = wd.RegisterComponent("test-component", mockComponent)
	assert.NoError(t, err)
	
	// Start the watchdog
	err = wd.Start()
	assert.NoError(t, err)
	
	// Wait for restart attempts
	time.Sleep(100 * time.Millisecond)
	
	// Get component status
	status, err := wd.GetComponentStatus("test-component")
	assert.NoError(t, err)
	
	// Check that incidents include restart failures
	var hasRestartFailure bool
	for _, incident := range status.Incidents {
		if incident.Type == watchdog.IncidentRestartFailed {
			hasRestartFailure = true
			break
		}
	}
	assert.True(t, hasRestartFailure)
	
	// Stop the watchdog
	err = wd.Stop()
	assert.NoError(t, err)
}

func TestDegradableComponent(t *testing.T) {
	config := watchdog.Config{
		MonitorInterval:    10 * time.Millisecond,
		DegradationEnabled: true,
		DegradationLevels:  3,
		GlobalThresholds: watchdog.ResourceThresholds{
			MaxCPUPercent:  90.0,
			MaxMemoryMB:    1000,
			MaxGoroutines:  1000,
			MaxFileHandles: 1000,
			MaxGCPercent:   10.0,
		},
	}
	
	wd, err := watchdog.NewWatchdog(config)
	assert.NoError(t, err)
	
	// Create a mock component
	mockComponent := NewMockComponent()
	
	// Set resource usage that exceeds thresholds
	highUsage := watchdog.ResourceUsage{
		CPUPercent:  95.0,
		MemoryMB:    1500.0,
		Goroutines:  1500,
		FileHandles: 1500,
		GCPercent:   15.0,
		Timestamp:   time.Now(),
	}
	
	mockComponent.SetResourceUsage(highUsage)
	
	// First set health as degraded
	mockComponent.SetHealth(watchdog.HealthDegraded)
	
	// Register the component
	err = wd.RegisterComponent("test-component", mockComponent)
	assert.NoError(t, err)
	
	// Start the watchdog
	err = wd.Start()
	assert.NoError(t, err)
	
	// Wait for degradation to happen
	time.Sleep(50 * time.Millisecond)
	
	// Get component status
	status, err := wd.GetComponentStatus("test-component")
	assert.NoError(t, err)
	
	// Check that degradation level was set
	assert.Greater(t, status.DegradationLevel, 0)
	
	// Now change health to critical
	mockComponent.SetHealth(watchdog.HealthCritical)
	
	// Wait for degradation to increase
	time.Sleep(50 * time.Millisecond)
	
	// Get updated status
	newStatus, err := wd.GetComponentStatus("test-component")
	assert.NoError(t, err)
	
	// Degradation level should be higher or at max
	assert.GreaterOrEqual(t, newStatus.DegradationLevel, status.DegradationLevel)
	assert.LessOrEqual(t, newStatus.DegradationLevel, config.DegradationLevels)
	
	// Stop the watchdog
	err = wd.Stop()
	assert.NoError(t, err)
}
