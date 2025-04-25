package watchdog

import (
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// CircuitClosed indicates the circuit is closed and requests pass through
	CircuitClosed CircuitState = iota
	
	// CircuitOpen indicates the circuit is open and requests are rejected
	CircuitOpen
	
	// CircuitHalfOpen indicates the circuit is half-open and some requests are allowed through
	CircuitHalfOpen
)

// String returns a string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "Closed"
	case CircuitOpen:
		return "Open"
	case CircuitHalfOpen:
		return "HalfOpen"
	default:
		return "Unknown"
	}
}

// CircuitBreakerStatus holds status information about a circuit breaker
type CircuitBreakerStatus struct {
	// State is the current state of the circuit breaker
	State CircuitState
	
	// Failures is the current failure count
	Failures int
	
	// SuccessesInHalfOpen is the current success count in half-open state
	SuccessesInHalfOpen int
	
	// LastStateChangeTime is the time of the last state change
	LastStateChangeTime time.Time
	
	// OpenUntil is the time the circuit will remain open (if in open state)
	OpenUntil time.Time
}

// StateChangeListener is a function that is called when the circuit state changes
type StateChangeListener func(name string, oldState, newState CircuitState)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name                  string
	config                CircuitBreakerConfig
	state                 CircuitState
	failures              int
	successesInHalfOpen   int
	lastStateChangeTime   time.Time
	openUntil             time.Time
	listeners             []StateChangeListener
	mu                    sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:                name,
		config:              config,
		state:               CircuitClosed,
		failures:            0,
		successesInHalfOpen: 0,
		lastStateChangeTime: time.Now(),
		listeners:           make([]StateChangeListener, 0),
	}
}

// AddStateChangeListener adds a listener for state changes
func (cb *CircuitBreaker) AddStateChangeListener(listener StateChangeListener) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.listeners = append(cb.listeners, listener)
}

// Name returns the name of the circuit breaker
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// AllowOperation returns true if the operation is allowed
func (cb *CircuitBreaker) AllowOperation() bool {
	if !cb.config.Enabled {
		return true
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		now := time.Now()
		if now.After(cb.openUntil) {
			cb.toHalfOpen()
			return true
		}
		return false
	case CircuitHalfOpen:
		// In half-open state, we only allow one request at a time
		return cb.successesInHalfOpen == 0
	default:
		return true
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	if !cb.config.Enabled {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failures = 0
	case CircuitHalfOpen:
		cb.successesInHalfOpen++
		if cb.successesInHalfOpen >= cb.config.HalfOpenSuccessThreshold {
			cb.toClosed()
		}
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	if !cb.config.Enabled {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			cb.toOpen()
		}
	case CircuitHalfOpen:
		cb.toOpen()
	}
}

// Status returns the current status of the circuit breaker
func (cb *CircuitBreaker) Status() CircuitBreakerStatus {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return CircuitBreakerStatus{
		State:               cb.state,
		Failures:            cb.failures,
		SuccessesInHalfOpen: cb.successesInHalfOpen,
		LastStateChangeTime: cb.lastStateChangeTime,
		OpenUntil:           cb.openUntil,
	}
}

// Reset resets the circuit breaker to its initial state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.toClosed()
}

// toOpen transitions the circuit breaker to the open state
func (cb *CircuitBreaker) toOpen() {
	oldState := cb.state
	if cb.state != CircuitOpen {
		cb.state = CircuitOpen
		cb.openUntil = time.Now().Add(cb.config.ResetTimeout)
		cb.lastStateChangeTime = time.Now()
		cb.notifyStateChange(oldState, CircuitOpen)
	}
}

// toHalfOpen transitions the circuit breaker to the half-open state
func (cb *CircuitBreaker) toHalfOpen() {
	oldState := cb.state
	if cb.state != CircuitHalfOpen {
		cb.state = CircuitHalfOpen
		cb.successesInHalfOpen = 0
		cb.lastStateChangeTime = time.Now()
		cb.notifyStateChange(oldState, CircuitHalfOpen)
	}
}

// toClosed transitions the circuit breaker to the closed state
func (cb *CircuitBreaker) toClosed() {
	oldState := cb.state
	if cb.state != CircuitClosed {
		cb.state = CircuitClosed
		cb.failures = 0
		cb.successesInHalfOpen = 0
		cb.lastStateChangeTime = time.Now()
		cb.notifyStateChange(oldState, CircuitClosed)
	}
}

// notifyStateChange notifies all listeners of a state change
func (cb *CircuitBreaker) notifyStateChange(oldState, newState CircuitState) {
	for _, listener := range cb.listeners {
		go listener(cb.name, oldState, newState)
	}
}
