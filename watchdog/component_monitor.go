package watchdog

import (
	"sync"
	"time"
)

// ResourceType represents a type of resource to monitor
type ResourceType string

const (
	// ResourceCPU represents CPU usage
	ResourceCPU ResourceType = "CPU"
	
	// ResourceMemory represents memory usage
	ResourceMemory ResourceType = "Memory"
	
	// ResourceThreads represents thread count
	ResourceThreads ResourceType = "Threads"
	
	// ResourceIO represents I/O operations
	ResourceIO ResourceType = "IO"
)

// ResourceUsage represents the resource usage of a component
type ResourceUsage struct {
	// CPU is the CPU usage percentage
	CPU float64
	
	// Memory is the memory usage in MB
	Memory float64
	
	// Threads is the thread count
	Threads int
	
	// IOReadBytes is the bytes read
	IOReadBytes int64
	
	// IOWriteBytes is the bytes written
	IOWriteBytes int64
	
	// Timestamp is when the resource usage was collected
	Timestamp time.Time
}

// ResourceSample represents a sample of resource usage over time
type ResourceSample struct {
	// Usage is the resource usage
	Usage ResourceUsage
	
	// Duration is the duration this sample represents
	Duration time.Duration
}

// ComponentMonitor monitors resource usage of a component
type ComponentMonitor struct {
	// ID is the component ID
	ID string
	
	// CircuitBreaker is the circuit breaker for this component
	CircuitBreaker CircuitBreaker
	
	// CurrentUsage is the current resource usage
	CurrentUsage ResourceUsage
	
	// UsageHistory stores historical resource usage
	UsageHistory []ResourceSample
	
	// LastHeartbeatTime is the time of the last heartbeat
	LastHeartbeatTime time.Time
	
	// LastThresholdViolationTime is the time of the last threshold violation
	LastThresholdViolationTime map[ResourceType]time.Time
	
	// ThresholdViolationDuration is the duration of the current threshold violation
	ThresholdViolationDuration map[ResourceType]time.Duration
	
	// Thresholds are the resource thresholds for this component
	Thresholds map[ResourceType]float64
	
	// Lock protects the component monitor
	Lock sync.RWMutex
}

// NewComponentMonitor creates a new component monitor
func NewComponentMonitor(id string, circuitBreaker CircuitBreaker, config ThresholdConfig) *ComponentMonitor {
	return &ComponentMonitor{
		ID:                        id,
		CircuitBreaker:            circuitBreaker,
		UsageHistory:              make([]ResourceSample, 0, 60), // 10 minutes at 10s interval
		LastHeartbeatTime:         time.Now(),
		LastThresholdViolationTime: make(map[ResourceType]time.Time),
		ThresholdViolationDuration: make(map[ResourceType]time.Duration),
		Thresholds: map[ResourceType]float64{
			ResourceCPU:    config.CPU.Component,
			ResourceMemory: config.Memory.Component,
		},
	}
}

// UpdateResourceUsage updates the current resource usage
func (cm *ComponentMonitor) UpdateResourceUsage(usage ResourceUsage) {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()
	
	// Store previous usage as sample
	if cm.CurrentUsage.Timestamp.Unix() > 0 {
		sample := ResourceSample{
			Usage:    cm.CurrentUsage,
			Duration: usage.Timestamp.Sub(cm.CurrentUsage.Timestamp),
		}
		
		cm.UsageHistory = append(cm.UsageHistory, sample)
		
		// Keep only the last 60 samples
		if len(cm.UsageHistory) > 60 {
			cm.UsageHistory = cm.UsageHistory[1:]
		}
	}
	
	// Update current usage
	cm.CurrentUsage = usage
	
	// Check thresholds
	cm.checkThresholds()
}

// UpdateHeartbeat updates the last heartbeat time
func (cm *ComponentMonitor) UpdateHeartbeat() {
	cm.Lock.Lock()
	defer cm.Lock.Unlock()
	
	cm.LastHeartbeatTime = time.Now()
}

// GetResourceUsage returns the current resource usage
func (cm *ComponentMonitor) GetResourceUsage() ResourceUsage {
	cm.Lock.RLock()
	defer cm.Lock.RUnlock()
	
	return cm.CurrentUsage
}

// GetHeartbeatAge returns how long since the last heartbeat
func (cm *ComponentMonitor) GetHeartbeatAge() time.Duration {
	cm.Lock.RLock()
	defer cm.Lock.RUnlock()
	
	return time.Since(cm.LastHeartbeatTime)
}

// GetAverageUsage returns the average resource usage over the given duration
func (cm *ComponentMonitor) GetAverageUsage(duration time.Duration) ResourceUsage {
	cm.Lock.RLock()
	defer cm.Lock.RUnlock()
	
	if len(cm.UsageHistory) == 0 {
		return cm.CurrentUsage
	}
	
	// Find samples within the duration
	now := time.Now()
	startTime := now.Add(-duration)
	
	var totalCPU, totalMemory float64
	var totalThreads int
	var totalIORead, totalIOWrite int64
	var totalDuration time.Duration
	
	// Start with current usage
	totalCPU = cm.CurrentUsage.CPU
	totalMemory = cm.CurrentUsage.Memory
	totalThreads = cm.CurrentUsage.Threads
	totalIORead = cm.CurrentUsage.IOReadBytes
	totalIOWrite = cm.CurrentUsage.IOWriteBytes
	
	// Estimate current sample duration (from last sample to now)
	currentDuration := now.Sub(cm.CurrentUsage.Timestamp)
	if currentDuration > duration {
		currentDuration = duration
	}
	totalDuration = currentDuration
	
	// Add historical samples within duration
	for i := len(cm.UsageHistory) - 1; i >= 0; i-- {
		sample := cm.UsageHistory[i]
		
		// Stop if we've gone beyond our duration
		if sample.Usage.Timestamp.Before(startTime) {
			break
		}
		
		// Calculate sample duration that falls within our window
		sampleDuration := sample.Duration
		if sample.Usage.Timestamp.Add(sampleDuration).After(now) {
			endTime := now
			if endTime.After(sample.Usage.Timestamp.Add(sampleDuration)) {
				endTime = sample.Usage.Timestamp.Add(sampleDuration)
			}
			sampleDuration = endTime.Sub(sample.Usage.Timestamp)
		}
		
		if sample.Usage.Timestamp.Before(startTime) {
			startOffset := startTime.Sub(sample.Usage.Timestamp)
			if startOffset >= sampleDuration {
				continue
			}
			sampleDuration -= startOffset
		}
		
		// Skip zero duration samples
		if sampleDuration <= 0 {
			continue
		}
		
		// Add weighted sample to totals
		totalCPU += sample.Usage.CPU * float64(sampleDuration) / float64(duration)
		totalMemory += sample.Usage.Memory * float64(sampleDuration) / float64(duration)
		totalThreads += sample.Usage.Threads * int(sampleDuration) / int(duration)
		totalIORead += sample.Usage.IOReadBytes * int64(sampleDuration) / int64(duration)
		totalIOWrite += sample.Usage.IOWriteBytes * int64(sampleDuration) / int64(duration)
		
		totalDuration += sampleDuration
		
		// Stop if we've covered the full duration
		if totalDuration >= duration {
			break
		}
	}
	
	// Calculate weighted average based on actual duration covered
	if totalDuration < duration {
		weightFactor := float64(duration) / float64(totalDuration)
		totalCPU *= weightFactor
		totalMemory *= weightFactor
		totalThreads = int(float64(totalThreads) * weightFactor)
		totalIORead = int64(float64(totalIORead) * weightFactor)
		totalIOWrite = int64(float64(totalIOWrite) * weightFactor)
	}
	
	return ResourceUsage{
		CPU:          totalCPU,
		Memory:       totalMemory,
		Threads:      totalThreads,
		IOReadBytes:  totalIORead,
		IOWriteBytes: totalIOWrite,
		Timestamp:    now,
	}
}

// checkThresholds checks if any resource thresholds are violated
func (cm *ComponentMonitor) checkThresholds() {
	now := time.Now()
	
	// Check CPU threshold
	if cm.CurrentUsage.CPU > cm.Thresholds[ResourceCPU] {
		lastViolation, exists := cm.LastThresholdViolationTime[ResourceCPU]
		if !exists {
			cm.LastThresholdViolationTime[ResourceCPU] = now
			cm.ThresholdViolationDuration[ResourceCPU] = 0
		} else {
			cm.ThresholdViolationDuration[ResourceCPU] = now.Sub(lastViolation)
		}
	} else {
		delete(cm.LastThresholdViolationTime, ResourceCPU)
		delete(cm.ThresholdViolationDuration, ResourceCPU)
	}
	
	// Check Memory threshold
	if cm.CurrentUsage.Memory > cm.Thresholds[ResourceMemory] {
		lastViolation, exists := cm.LastThresholdViolationTime[ResourceMemory]
		if !exists {
			cm.LastThresholdViolationTime[ResourceMemory] = now
			cm.ThresholdViolationDuration[ResourceMemory] = 0
		} else {
			cm.ThresholdViolationDuration[ResourceMemory] = now.Sub(lastViolation)
		}
	} else {
		delete(cm.LastThresholdViolationTime, ResourceMemory)
		delete(cm.ThresholdViolationDuration, ResourceMemory)
	}
}

// IsThresholdViolated checks if a resource threshold is currently violated
func (cm *ComponentMonitor) IsThresholdViolated(resource ResourceType, minDuration time.Duration) bool {
	cm.Lock.RLock()
	defer cm.Lock.RUnlock()
	
	duration, exists := cm.ThresholdViolationDuration[resource]
	if !exists {
		return false
	}
	
	return duration >= minDuration
}

// GetThresholdViolationDuration returns the duration of a threshold violation
func (cm *ComponentMonitor) GetThresholdViolationDuration(resource ResourceType) time.Duration {
	cm.Lock.RLock()
	defer cm.Lock.RUnlock()
	
	duration, exists := cm.ThresholdViolationDuration[resource]
	if !exists {
		return 0
	}
	
	return duration
}
