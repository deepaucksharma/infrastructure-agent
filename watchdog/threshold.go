package watchdog

import (
	"time"
)

// CPUThreshold represents CPU usage thresholds
type CPUThreshold struct {
	// Global is the global CPU threshold percentage
	Global float64
	
	// Component is the per-component CPU threshold percentage
	Component float64
}

// MemoryThreshold represents memory usage thresholds
type MemoryThreshold struct {
	// Global is the global memory threshold in MB
	Global float64
	
	// Component is the per-component memory threshold in MB
	Component float64
}

// ResourceThresholds represents resource usage thresholds
type ResourceThresholds struct {
	// MaxCPUPercent is the maximum CPU usage percentage
	MaxCPUPercent float64
	
	// MaxMemoryMB is the maximum memory usage in MB
	MaxMemoryMB int
	
	// MaxGoroutines is the maximum number of goroutines
	MaxGoroutines int
	
	// MaxFileHandles is the maximum number of file handles
	MaxFileHandles int
	
	// MaxGCPercent is the maximum percentage of time spent in GC
	MaxGCPercent float64
}

// ThresholdConfig represents the configuration for resource thresholds
type ThresholdConfig struct {
	// CPU contains CPU threshold configuration
	CPU CPUThreshold
	
	// Memory contains memory threshold configuration
	Memory MemoryThreshold
	
	// Goroutines is the maximum number of goroutines
	Goroutines int
	
	// FileDescriptors is the maximum number of file descriptors
	FileDescriptors int
	
	// GCPercent is the maximum percentage of time spent in GC
	GCPercent float64
}

// NewThresholdConfig creates a new threshold configuration with default values
func NewThresholdConfig() ThresholdConfig {
	return ThresholdConfig{
		CPU: CPUThreshold{
			Global:    80.0,
			Component: 70.0,
		},
		Memory: MemoryThreshold{
			Global:    1000.0,
			Component: 200.0,
		},
		Goroutines:      1000,
		FileDescriptors: 1000,
		GCPercent:       10.0,
	}
}

// DefaultResourceThresholds returns default resource thresholds
func DefaultResourceThresholds() ResourceThresholds {
	return ResourceThresholds{
		MaxCPUPercent:  70.0,
		MaxMemoryMB:    200,
		MaxGoroutines:  1000,
		MaxFileHandles: 1000,
		MaxGCPercent:   10.0,
	}
}

// ComponentMetrics contains metrics for a component
type ComponentMetrics struct {
	// LastResponseTime is the time since the last response
	LastResponseTime time.Duration
	
	// GoroutineCount is the number of goroutines
	GoroutineCount int
	
	// State is the current state of the component
	State string
	
	// HealthStatus is the health status of the component
	HealthStatus string
}
