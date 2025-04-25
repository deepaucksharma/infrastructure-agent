package watchdog

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RestartManager handles restarting components
type RestartManager struct {
	// config is the restart configuration
	config RestartConfig
	
	// component is the component to restart
	component Restartable
	
	// restartAttempts is the number of restart attempts
	restartAttempts int
	
	// lastRestartTime is when the component was last restarted
	lastRestartTime time.Time
	
	// currentBackoff is the current backoff duration
	currentBackoff time.Duration
	
	// mutex protects the manager state
	mutex sync.RWMutex
}

// NewRestartManager creates a new restart manager
func NewRestartManager(config RestartConfig, component Restartable) *RestartManager {
	return &RestartManager{
		config:          config,
		component:       component,
		restartAttempts: 0,
		currentBackoff:  config.RestartBackoffInitial,
	}
}

// AttemptRestart attempts to restart the component
func (rm *RestartManager) AttemptRestart(ctx context.Context) (bool, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	
	// Check if restart is enabled
	if !rm.config.Enabled {
		return false, fmt.Errorf("restart is disabled")
	}
	
	// Check if maximum restart attempts reached
	if rm.restartAttempts >= rm.config.MaxRestartAttempts {
		return false, fmt.Errorf("maximum restart attempts reached (%d)", rm.config.MaxRestartAttempts)
	}
	
	// Check if component is already running
	if rm.component.IsRunning() {
		return true, nil
	}
	
	// Check if we need to wait for backoff
	if !rm.lastRestartTime.IsZero() {
		timeElapsed := time.Since(rm.lastRestartTime)
		if timeElapsed < rm.currentBackoff {
			return false, fmt.Errorf("backoff in progress, %s remaining", rm.currentBackoff-timeElapsed)
		}
	}
	
	// Create a context with timeout for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, rm.config.GracefulShutdownTimeout)
	defer shutdownCancel()
	
	// Attempt to shutdown gracefully
	err := rm.component.Shutdown(shutdownCtx)
	if err != nil {
		// Log but continue with restart
		fmt.Printf("Warning: graceful shutdown failed: %v", err)
	}
	
	// Attempt to start the component
	err = rm.component.Start(ctx)
	if err != nil {
		// Increment restart attempts
		rm.restartAttempts++
		rm.lastRestartTime = time.Now()
		
		// Increase backoff duration
		rm.currentBackoff = time.Duration(float64(rm.currentBackoff) * rm.config.RestartBackoffFactor)
		if rm.currentBackoff > rm.config.RestartBackoffMax {
			rm.currentBackoff = rm.config.RestartBackoffMax
		}
		
		return false, fmt.Errorf("failed to restart component: %w", err)
	}
	
	// Reset backoff on successful restart
	rm.restartAttempts = 0
	rm.lastRestartTime = time.Now()
	rm.currentBackoff = rm.config.RestartBackoffInitial
	
	return true, nil
}

// GetRestartAttempts returns the number of restart attempts
func (rm *RestartManager) GetRestartAttempts() int {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	return rm.restartAttempts
}

// GetLastRestartTime returns when the component was last restarted
func (rm *RestartManager) GetLastRestartTime() time.Time {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	return rm.lastRestartTime
}

// ResetRestartAttempts resets the restart attempts counter
func (rm *RestartManager) ResetRestartAttempts() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	
	rm.restartAttempts = 0
	rm.currentBackoff = rm.config.RestartBackoffInitial
}
