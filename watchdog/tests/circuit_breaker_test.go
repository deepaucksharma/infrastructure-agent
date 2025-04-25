package tests

import (
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/watchdog"
	"github.com/stretchr/testify/assert"
)

func TestCircuitBreakerTransitions(t *testing.T) {
	config := watchdog.CircuitBreakerConfig{
		Enabled:                 true,
		FailureThreshold:        3,
		ResetTimeout:            100 * time.Millisecond,
		HalfOpenSuccessThreshold: 2,
	}
	
	cb := watchdog.NewCircuitBreaker("test-component", config)
	
	// Should start closed
	assert.Equal(t, watchdog.CircuitClosed, cb.State())
	assert.True(t, cb.AllowOperation())
	
	// Record failures to trigger threshold
	cb.RecordFailure()
	assert.Equal(t, watchdog.CircuitClosed, cb.State())
	assert.True(t, cb.AllowOperation())
	
	cb.RecordFailure()
	assert.Equal(t, watchdog.CircuitClosed, cb.State())
	assert.True(t, cb.AllowOperation())
	
	cb.RecordFailure()
	assert.Equal(t, watchdog.CircuitOpen, cb.State())
	assert.False(t, cb.AllowOperation())
	
	// Wait for reset timeout
	time.Sleep(110 * time.Millisecond)
	
	// Should now be half-open
	assert.Equal(t, watchdog.CircuitHalfOpen, cb.State())
	assert.True(t, cb.AllowOperation())
	
	// Record one success in half-open state
	cb.RecordSuccess()
	assert.Equal(t, watchdog.CircuitHalfOpen, cb.State())
	assert.True(t, cb.AllowOperation())
	
	// Record another success to meet half-open success threshold
	cb.RecordSuccess() 
	assert.Equal(t, watchdog.CircuitClosed, cb.State())
	assert.True(t, cb.AllowOperation())
}

func TestCircuitBreakerReset(t *testing.T) {
	config := watchdog.CircuitBreakerConfig{
		Enabled:                 true,
		FailureThreshold:        2,
		ResetTimeout:            1 * time.Second,
		HalfOpenSuccessThreshold: 1,
	}
	
	cb := watchdog.NewCircuitBreaker("test-component", config)
	
	// Transition to open
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, watchdog.CircuitOpen, cb.State())
	
	// Manually reset
	cb.Reset()
	assert.Equal(t, watchdog.CircuitClosed, cb.State())
	assert.True(t, cb.AllowOperation())
}

func TestCircuitBreakerStatus(t *testing.T) {
	config := watchdog.CircuitBreakerConfig{
		Enabled:                 true,
		FailureThreshold:        2,
		ResetTimeout:            1 * time.Second,
		HalfOpenSuccessThreshold: 1,
	}
	
	cb := watchdog.NewCircuitBreaker("test-component", config)
	
	// Get initial status
	status := cb.Status()
	assert.Equal(t, watchdog.CircuitClosed, status.State)
	assert.Equal(t, 0, status.Failures)
	assert.Equal(t, 0, status.SuccessesInHalfOpen)
	
	// Update state
	cb.RecordFailure()
	
	status = cb.Status()
	assert.Equal(t, watchdog.CircuitClosed, status.State)
	assert.Equal(t, 1, status.Failures)
}

func TestCircuitBreakerStateChangeListener(t *testing.T) {
	config := watchdog.CircuitBreakerConfig{
		Enabled:                 true,
		FailureThreshold:        1,
		ResetTimeout:            100 * time.Millisecond,
		HalfOpenSuccessThreshold: 1,
	}
	
	cb := watchdog.NewCircuitBreaker("test-component", config)
	
	stateChanges := make([]watchdog.CircuitState, 0)
	
	// Add a listener
	cb.AddStateChangeListener(func(name string, oldState, newState watchdog.CircuitState) {
		assert.Equal(t, "test-component", name)
		stateChanges = append(stateChanges, newState)
	})
	
	// Trigger state changes
	cb.RecordFailure() // Closed -> Open
	
	// Wait for reset timeout
	time.Sleep(110 * time.Millisecond)
	
	// Now half-open
	assert.Equal(t, watchdog.CircuitHalfOpen, cb.State())
	
	cb.RecordSuccess() // Half-open -> Closed
	
	// Verify listener was called correctly
	assert.Equal(t, 3, len(stateChanges))
	assert.Equal(t, watchdog.CircuitOpen, stateChanges[0])
	assert.Equal(t, watchdog.CircuitHalfOpen, stateChanges[1])
	assert.Equal(t, watchdog.CircuitClosed, stateChanges[2])
}

func TestCircuitBreakerDisabled(t *testing.T) {
	config := watchdog.CircuitBreakerConfig{
		Enabled:                 false,
		FailureThreshold:        1,
		ResetTimeout:            1 * time.Second,
		HalfOpenSuccessThreshold: 1,
	}
	
	cb := watchdog.NewCircuitBreaker("test-component", config)
	
	// Should allow operation even with failures
	assert.True(t, cb.AllowOperation())
	
	cb.RecordFailure()
	assert.True(t, cb.AllowOperation())
	
	cb.RecordFailure()
	assert.True(t, cb.AllowOperation())
	
	// State should still be closed
	assert.Equal(t, watchdog.CircuitClosed, cb.State())
}
