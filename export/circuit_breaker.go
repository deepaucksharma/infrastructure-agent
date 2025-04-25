package export

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

// CircuitBreaker implements the circuit breaker pattern to prevent sending requests
// to a service that is likely to fail.
type CircuitBreaker struct {
	mu                     sync.RWMutex
	state                  CircuitState
	config                 CircuitConfig
	failures               int
	successesInHalfOpen    int
	lastStateChangeTime    time.Time
	openUntil              time.Time
	stateChangeSubscribers []func(CircuitState)
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(config CircuitConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state:                  CircuitClosed,
		config:                 config,
		failures:               0,
		successesInHalfOpen:    0,
		lastStateChangeTime:    time.Now(),
		stateChangeSubscribers: make([]func(CircuitState), 0),
	}
}

// OnStateChange registers a function to be called when the circuit state changes.
func (cb *CircuitBreaker) OnStateChange(fn func(CircuitState)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.stateChangeSubscribers = append(cb.stateChangeSubscribers, fn)
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// AllowRequest returns true if the circuit breaker will allow a request to proceed.
func (cb *CircuitBreaker) AllowRequest() bool {
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

// RecordSuccess records a successful request.
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

// RecordFailure records a failed request.
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

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.toClosed()
}

// toOpen transitions the circuit breaker to the open state
func (cb *CircuitBreaker) toOpen() {
	if cb.state != CircuitOpen {
		cb.state = CircuitOpen
		cb.openUntil = time.Now().Add(cb.config.ResetTimeout)
		cb.lastStateChangeTime = time.Now()
		cb.notifySubscribers()
	}
}

// toHalfOpen transitions the circuit breaker to the half-open state
func (cb *CircuitBreaker) toHalfOpen() {
	if cb.state != CircuitHalfOpen {
		cb.state = CircuitHalfOpen
		cb.successesInHalfOpen = 0
		cb.lastStateChangeTime = time.Now()
		cb.notifySubscribers()
	}
}

// toClosed transitions the circuit breaker to the closed state
func (cb *CircuitBreaker) toClosed() {
	if cb.state != CircuitClosed {
		cb.state = CircuitClosed
		cb.failures = 0
		cb.successesInHalfOpen = 0
		cb.lastStateChangeTime = time.Now()
		cb.notifySubscribers()
	}
}

// notifySubscribers notifies all subscribers of a state change
func (cb *CircuitBreaker) notifySubscribers() {
	state := cb.state
	for _, fn := range cb.stateChangeSubscribers {
		go fn(state)
	}
}
