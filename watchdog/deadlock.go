package watchdog

import (
	"runtime"
	"sync"
	"time"
)

// DeadlockInfo contains information about a detected deadlock.
type DeadlockInfo struct {
	// ComponentName is the name of the deadlocked component.
	ComponentName string
	
	// DetectedAt is when the deadlock was detected.
	DetectedAt time.Time
	
	// LastResponseTime is the time since the last response.
	LastResponseTime time.Duration
	
	// GoroutineStacks contains stack traces of the deadlocked goroutines.
	GoroutineStacks string
	
	// AdditionalInfo contains additional information about the deadlock.
	AdditionalInfo map[string]string
}

// DeadlockDetector is responsible for detecting deadlocks in components.
type DeadlockDetector struct {
	config              Config
	componentMonitor    *ComponentMonitor
	detectedDeadlocks   map[string]DeadlockInfo
	mu                  sync.RWMutex
}

// NewDeadlockDetector creates a new deadlock detector.
func NewDeadlockDetector(config Config, monitor *ComponentMonitor) *DeadlockDetector {
	detector := &DeadlockDetector{
		config:             config,
		componentMonitor:   monitor,
		detectedDeadlocks:  make(map[string]DeadlockInfo),
	}
	
	// Register for deadlock events from the monitor
	monitor.AddDeadlockDetectedHandler(detector.handleDeadlockDetected)
	
	return detector
}

// handleDeadlockDetected is called when a deadlock is detected.
func (d *DeadlockDetector) handleDeadlockDetected(componentName string, metrics ComponentMetrics) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Check if we already have a record for this deadlock
	if _, exists := d.detectedDeadlocks[componentName]; exists {
		return
	}
	
	// Create deadlock info
	deadlockInfo := DeadlockInfo{
		ComponentName:    componentName,
		DetectedAt:       time.Now(),
		LastResponseTime: metrics.LastResponseTime,
		GoroutineStacks:  d.captureGoroutineStacks(),
		AdditionalInfo:   make(map[string]string),
	}
	
	// Add any component-specific information
	deadlockInfo.AdditionalInfo["state"] = metrics.State
	deadlockInfo.AdditionalInfo["health"] = metrics.HealthStatus
	deadlockInfo.AdditionalInfo["goroutines"] = string(metrics.GoroutineCount)
	
	// Store the deadlock info
	d.detectedDeadlocks[componentName] = deadlockInfo
}

// captureGoroutineStacks returns stack traces for all goroutines.
func (d *DeadlockDetector) captureGoroutineStacks() string {
	buf := make([]byte, 1<<20) // 1MB buffer
	stackLen := runtime.Stack(buf, true)
	return string(buf[:stackLen])
}

// GetDetectedDeadlocks returns information about detected deadlocks.
func (d *DeadlockDetector) GetDetectedDeadlocks() map[string]DeadlockInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// Create a copy to avoid external modification
	deadlocks := make(map[string]DeadlockInfo, len(d.detectedDeadlocks))
	for k, v := range d.detectedDeadlocks {
		deadlocks[k] = v
	}
	
	return deadlocks
}

// ClearDeadlock clears the deadlock record for a component.
func (d *DeadlockDetector) ClearDeadlock(componentName string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	delete(d.detectedDeadlocks, componentName)
}

// HasDeadlock returns true if the component has a detected deadlock.
func (d *DeadlockDetector) HasDeadlock(componentName string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	_, exists := d.detectedDeadlocks[componentName]
	return exists
}
