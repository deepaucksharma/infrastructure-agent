package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/watchdog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRestartableComponent implements the Restartable interface for testing
type MockRestartableComponent struct {
	mock.Mock
	running bool
}

// Shutdown implements the Restartable interface
func (m *MockRestartableComponent) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	
	if args.Error(0) == nil {
		m.running = false
	}
	
	return args.Error(0)
}

// Start implements the Restartable interface
func (m *MockRestartableComponent) Start(ctx context.Context) error {
	args := m.Called(ctx)
	
	if args.Error(0) == nil {
		m.running = true
	}
	
	return args.Error(0)
}

// IsRunning implements the Restartable interface
func (m *MockRestartableComponent) IsRunning() bool {
	args := m.Called()
	return args.Bool(0)
}

// TestRestartManagerCreation tests creating a restart manager
func TestRestartManagerCreation(t *testing.T) {
	config := watchdog.RestartConfig{
		Enabled:                true,
		GracefulShutdownTimeout: 5 * time.Second,
		MaxRestartAttempts:     3,
		RestartBackoffInitial:  1 * time.Second,
		RestartBackoffMax:      30 * time.Second,
		RestartBackoffFactor:   2.0,
	}
	
	component := new(MockRestartableComponent)
	
	manager := watchdog.NewRestartManager(config, component)
	assert.NotNil(t, manager)
	
	// Initial state
	assert.Equal(t, 0, manager.GetRestartAttempts())
	assert.True(t, manager.GetLastRestartTime().IsZero())
}

// TestSuccessfulRestart tests a successful restart
func TestSuccessfulRestart(t *testing.T) {
	config := watchdog.RestartConfig{
		Enabled:                true,
		GracefulShutdownTimeout: 1 * time.Second,
		MaxRestartAttempts:     3,
		RestartBackoffInitial:  1 * time.Second,
		RestartBackoffMax:      30 * time.Second,
		RestartBackoffFactor:   2.0,
	}
	
	component := new(MockRestartableComponent)
	
	// Set up the component to not be running
	component.On("IsRunning").Return(false)
	component.On("Shutdown", mock.Anything).Return(nil)
	component.On("Start", mock.Anything).Return(nil)
	
	manager := watchdog.NewRestartManager(config, component)
	
	// Attempt restart
	success, err := manager.AttemptRestart(context.Background())
	assert.True(t, success)
	assert.NoError(t, err)
	
	// Check that restart was recorded
	assert.Equal(t, 0, manager.GetRestartAttempts())
	assert.False(t, manager.GetLastRestartTime().IsZero())
	
	// Verify expectations
	component.AssertExpectations(t)
}

// TestAlreadyRunning tests attempting to restart an already running component
func TestAlreadyRunning(t *testing.T) {
	config := watchdog.RestartConfig{
		Enabled:                true,
		GracefulShutdownTimeout: 1 * time.Second,
		MaxRestartAttempts:     3,
		RestartBackoffInitial:  1 * time.Second,
		RestartBackoffMax:      30 * time.Second,
		RestartBackoffFactor:   2.0,
	}
	
	component := new(MockRestartableComponent)
	
	// Set up the component to be running
	component.On("IsRunning").Return(true)
	
	manager := watchdog.NewRestartManager(config, component)
	
	// Attempt restart
	success, err := manager.AttemptRestart(context.Background())
	assert.True(t, success)
	assert.NoError(t, err)
	
	// Verify restart functions were not called
	component.AssertNotCalled(t, "Shutdown")
	component.AssertNotCalled(t, "Start")
}

// TestFailedRestart tests a failed restart
func TestFailedRestart(t *testing.T) {
	config := watchdog.RestartConfig{
		Enabled:                true,
		GracefulShutdownTimeout: 1 * time.Second,
		MaxRestartAttempts:     3,
		RestartBackoffInitial:  1 * time.Second,
		RestartBackoffMax:      30 * time.Second,
		RestartBackoffFactor:   2.0,
	}
	
	component := new(MockRestartableComponent)
	
	// Set up the component to not be running, but fail to start
	component.On("IsRunning").Return(false)
	component.On("Shutdown", mock.Anything).Return(nil)
	component.On("Start", mock.Anything).Return(errors.New("start failed"))
	
	manager := watchdog.NewRestartManager(config, component)
	
	// Attempt restart
	success, err := manager.AttemptRestart(context.Background())
	assert.False(t, success)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start failed")
	
	// Check that restart attempt was recorded
	assert.Equal(t, 1, manager.GetRestartAttempts())
	assert.False(t, manager.GetLastRestartTime().IsZero())
	
	// Verify expectations
	component.AssertExpectations(t)
}

// TestMaxRestartAttempts tests reaching the maximum restart attempts
func TestMaxRestartAttempts(t *testing.T) {
	config := watchdog.RestartConfig{
		Enabled:                true,
		GracefulShutdownTimeout: 1 * time.Second,
		MaxRestartAttempts:     2,
		RestartBackoffInitial:  10 * time.Millisecond,
		RestartBackoffMax:      30 * time.Second,
		RestartBackoffFactor:   2.0,
	}
	
	component := new(MockRestartableComponent)
	
	// Set up the component to not be running, but fail to start
	component.On("IsRunning").Return(false)
	component.On("Shutdown", mock.Anything).Return(nil)
	component.On("Start", mock.Anything).Return(errors.New("start failed"))
	
	manager := watchdog.NewRestartManager(config, component)
	
	// First attempt
	success, err := manager.AttemptRestart(context.Background())
	assert.False(t, success)
	assert.Error(t, err)
	assert.Equal(t, 1, manager.GetRestartAttempts())
	
	// Wait for backoff
	time.Sleep(20 * time.Millisecond)
	
	// Second attempt
	success, err = manager.AttemptRestart(context.Background())
	assert.False(t, success)
	assert.Error(t, err)
	assert.Equal(t, 2, manager.GetRestartAttempts())
	
	// Wait for backoff
	time.Sleep(40 * time.Millisecond)
	
	// Third attempt (should fail due to max attempts)
	success, err = manager.AttemptRestart(context.Background())
	assert.False(t, success)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum restart attempts reached")
	assert.Equal(t, 2, manager.GetRestartAttempts())
	
	// Verify expectations (Start should only be called twice)
	component.AssertNumberOfCalls(t, "Start", 2)
}

// TestBackoff tests the backoff mechanism
func TestBackoff(t *testing.T) {
	config := watchdog.RestartConfig{
		Enabled:                true,
		GracefulShutdownTimeout: 1 * time.Second,
		MaxRestartAttempts:     10,
		RestartBackoffInitial:  100 * time.Millisecond,
		RestartBackoffMax:      500 * time.Millisecond,
		RestartBackoffFactor:   2.0,
	}
	
	component := new(MockRestartableComponent)
	
	// Set up the component to not be running, but fail to start
	component.On("IsRunning").Return(false)
	component.On("Shutdown", mock.Anything).Return(nil)
	component.On("Start", mock.Anything).Return(errors.New("start failed"))
	
	manager := watchdog.NewRestartManager(config, component)
	
	// First attempt
	success, err := manager.AttemptRestart(context.Background())
	assert.False(t, success)
	assert.Error(t, err)
	
	// Try again immediately (should fail due to backoff)
	success, err = manager.AttemptRestart(context.Background())
	assert.False(t, success)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backoff in progress")
	
	// Wait for first backoff to complete
	time.Sleep(110 * time.Millisecond)
	
	// Second attempt
	success, err = manager.AttemptRestart(context.Background())
	assert.False(t, success)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to restart component")
	
	// Try again immediately (should fail due to increased backoff)
	success, err = manager.AttemptRestart(context.Background())
	assert.False(t, success)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backoff in progress")
	
	// Verify expectations (Start should only be called twice)
	component.AssertNumberOfCalls(t, "Start", 2)
}

// TestShutdownFailure tests handling a failed shutdown
func TestShutdownFailure(t *testing.T) {
	config := watchdog.RestartConfig{
		Enabled:                true,
		GracefulShutdownTimeout: 1 * time.Second,
		MaxRestartAttempts:     3,
		RestartBackoffInitial:  1 * time.Second,
		RestartBackoffMax:      30 * time.Second,
		RestartBackoffFactor:   2.0,
	}
	
	component := new(MockRestartableComponent)
	
	// Set up the component to not be running, fail shutdown but succeed start
	component.On("IsRunning").Return(false)
	component.On("Shutdown", mock.Anything).Return(errors.New("shutdown failed"))
	component.On("Start", mock.Anything).Return(nil)
	
	manager := watchdog.NewRestartManager(config, component)
	
	// Attempt restart
	success, err := manager.AttemptRestart(context.Background())
	assert.True(t, success)
	assert.NoError(t, err)
	
	// Verify expectations (should still try to start even after shutdown fails)
	component.AssertExpectations(t)
}

// TestResetRestartAttempts tests resetting restart attempts
func TestResetRestartAttempts(t *testing.T) {
	config := watchdog.RestartConfig{
		Enabled:                true,
		GracefulShutdownTimeout: 1 * time.Second,
		MaxRestartAttempts:     3,
		RestartBackoffInitial:  10 * time.Millisecond,
		RestartBackoffMax:      30 * time.Second,
		RestartBackoffFactor:   2.0,
	}
	
	component := new(MockRestartableComponent)
	
	// Set up the component to not be running, but fail to start
	component.On("IsRunning").Return(false)
	component.On("Shutdown", mock.Anything).Return(nil)
	component.On("Start", mock.Anything).Return(errors.New("start failed"))
	
	manager := watchdog.NewRestartManager(config, component)
	
	// First attempt
	success, err := manager.AttemptRestart(context.Background())
	assert.False(t, success)
	assert.Error(t, err)
	assert.Equal(t, 1, manager.GetRestartAttempts())
	
	// Wait for backoff
	time.Sleep(20 * time.Millisecond)
	
	// Second attempt
	success, err = manager.AttemptRestart(context.Background())
	assert.False(t, success)
	assert.Error(t, err)
	assert.Equal(t, 2, manager.GetRestartAttempts())
	
	// Reset restart attempts
	manager.ResetRestartAttempts()
	assert.Equal(t, 0, manager.GetRestartAttempts())
	
	// Try again, should work even without waiting for backoff
	success, err = manager.AttemptRestart(context.Background())
	assert.False(t, success)
	assert.Error(t, err)
	assert.Equal(t, 1, manager.GetRestartAttempts())
	
	// Verify expectations
	component.AssertNumberOfCalls(t, "Start", 3)
}

// TestDisabledRestart tests behavior when restart is disabled
func TestDisabledRestart(t *testing.T) {
	config := watchdog.RestartConfig{
		Enabled:                false,
		GracefulShutdownTimeout: 1 * time.Second,
		MaxRestartAttempts:     3,
		RestartBackoffInitial:  1 * time.Second,
		RestartBackoffMax:      30 * time.Second,
		RestartBackoffFactor:   2.0,
	}
	
	component := new(MockRestartableComponent)
	
	// Set up the component to not be running
	component.On("IsRunning").Return(false)
	
	manager := watchdog.NewRestartManager(config, component)
	
	// Attempt restart
	success, err := manager.AttemptRestart(context.Background())
	assert.False(t, success)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "restart is disabled")
	
	// Verify expectations (should not try to restart)
	component.AssertNotCalled(t, "Shutdown")
	component.AssertNotCalled(t, "Start")
}
