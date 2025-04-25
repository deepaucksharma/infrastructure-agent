package tests

import (
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/watchdog"
	"github.com/stretchr/testify/assert"
)

// TestComponentMonitorCreation tests creating a component monitor
func TestComponentMonitorCreation(t *testing.T) {
	circuitBreaker := watchdog.NewCircuitBreaker("test-component", watchdog.CircuitBreakerConfig{
		Enabled:                 true,
		FailureThreshold:        3,
		ResetTimeout:            5 * time.Second,
		HalfOpenSuccessThreshold: 2,
	})
	
	thresholdConfig := watchdog.ThresholdConfig{
		CPU: watchdog.CPUThreshold{
			Component: 80.0,
		},
		Memory: watchdog.MemoryThreshold{
			Component: 200.0,
		},
	}
	
	monitor := watchdog.NewComponentMonitor("test-component", *circuitBreaker, thresholdConfig)
	assert.NotNil(t, monitor)
	assert.Equal(t, "test-component", monitor.ID)
}

// TestUpdateResourceUsage tests updating resource usage
func TestUpdateResourceUsage(t *testing.T) {
	circuitBreaker := watchdog.NewCircuitBreaker("test-component", watchdog.CircuitBreakerConfig{
		Enabled:                 true,
		FailureThreshold:        3,
		ResetTimeout:            5 * time.Second,
		HalfOpenSuccessThreshold: 2,
	})
	
	thresholdConfig := watchdog.ThresholdConfig{
		CPU: watchdog.CPUThreshold{
			Component: 80.0,
		},
		Memory: watchdog.MemoryThreshold{
			Component: 200.0,
		},
	}
	
	monitor := watchdog.NewComponentMonitor("test-component", *circuitBreaker, thresholdConfig)
	
	// Update resource usage below thresholds
	now := time.Now()
	usage := watchdog.ResourceUsage{
		CPU:          50.0,
		Memory:       100.0,
		Threads:      10,
		IOReadBytes:  1024,
		IOWriteBytes: 2048,
		Timestamp:    now,
	}
	
	monitor.UpdateResourceUsage(usage)
	
	// Get current resource usage
	currentUsage := monitor.GetResourceUsage()
	assert.Equal(t, usage, currentUsage)
	
	// Verify no threshold violations
	assert.False(t, monitor.IsThresholdViolated(watchdog.ResourceCPU, 0))
	assert.False(t, monitor.IsThresholdViolated(watchdog.ResourceMemory, 0))
	
	// Update resource usage above thresholds
	usage = watchdog.ResourceUsage{
		CPU:          90.0, // > 80.0 threshold
		Memory:       250.0, // > 200.0 threshold
		Threads:      20,
		IOReadBytes:  2048,
		IOWriteBytes: 4096,
		Timestamp:    now.Add(1 * time.Second),
	}
	
	monitor.UpdateResourceUsage(usage)
	
	// Verify threshold violations
	assert.True(t, monitor.IsThresholdViolated(watchdog.ResourceCPU, 0))
	assert.True(t, monitor.IsThresholdViolated(watchdog.ResourceMemory, 0))
}

// TestHeartbeat tests heartbeat functionality
func TestHeartbeat(t *testing.T) {
	circuitBreaker := watchdog.NewCircuitBreaker("test-component", watchdog.CircuitBreakerConfig{
		Enabled:                 true,
		FailureThreshold:        3,
		ResetTimeout:            5 * time.Second,
		HalfOpenSuccessThreshold: 2,
	})
	
	thresholdConfig := watchdog.ThresholdConfig{
		CPU: watchdog.CPUThreshold{
			Component: 80.0,
		},
		Memory: watchdog.MemoryThreshold{
			Component: 200.0,
		},
	}
	
	monitor := watchdog.NewComponentMonitor("test-component", *circuitBreaker, thresholdConfig)
	
	// Initial heartbeat age should be very small
	initialAge := monitor.GetHeartbeatAge()
	assert.True(t, initialAge < 1*time.Second)
	
	// Wait some time
	time.Sleep(100 * time.Millisecond)
	
	// Age should have increased
	midAge := monitor.GetHeartbeatAge()
	assert.True(t, midAge >= 100*time.Millisecond)
	
	// Update heartbeat
	monitor.UpdateHeartbeat()
	
	// Age should be reset to a small value
	newAge := monitor.GetHeartbeatAge()
	assert.True(t, newAge < midAge)
}

// TestAverageUsage tests calculating average resource usage
func TestAverageUsage(t *testing.T) {
	circuitBreaker := watchdog.NewCircuitBreaker("test-component", watchdog.CircuitBreakerConfig{
		Enabled:                 true,
		FailureThreshold:        3,
		ResetTimeout:            5 * time.Second,
		HalfOpenSuccessThreshold: 2,
	})
	
	thresholdConfig := watchdog.ThresholdConfig{
		CPU: watchdog.CPUThreshold{
			Component: 80.0,
		},
		Memory: watchdog.MemoryThreshold{
			Component: 200.0,
		},
	}
	
	monitor := watchdog.NewComponentMonitor("test-component", *circuitBreaker, thresholdConfig)
	
	// Record multiple usage samples
	now := time.Now()
	
	// First sample at t=0
	monitor.UpdateResourceUsage(watchdog.ResourceUsage{
		CPU:          10.0,
		Memory:       50.0,
		Threads:      5,
		IOReadBytes:  1000,
		IOWriteBytes: 2000,
		Timestamp:    now,
	})
	
	// Second sample at t=1s
	monitor.UpdateResourceUsage(watchdog.ResourceUsage{
		CPU:          20.0,
		Memory:       60.0,
		Threads:      6,
		IOReadBytes:  1200,
		IOWriteBytes: 2200,
		Timestamp:    now.Add(1 * time.Second),
	})
	
	// Third sample at t=2s
	monitor.UpdateResourceUsage(watchdog.ResourceUsage{
		CPU:          30.0,
		Memory:       70.0,
		Threads:      7,
		IOReadBytes:  1400,
		IOWriteBytes: 2400,
		Timestamp:    now.Add(2 * time.Second),
	})
	
	// Fourth sample (current) at t=3s
	currentUsage := watchdog.ResourceUsage{
		CPU:          40.0,
		Memory:       80.0,
		Threads:      8,
		IOReadBytes:  1600,
		IOWriteBytes: 2600,
		Timestamp:    now.Add(3 * time.Second),
	}
	monitor.UpdateResourceUsage(currentUsage)
	
	// Get average usage over 2 seconds
	avgUsage := monitor.GetAverageUsage(2 * time.Second)
	
	// Average should be weighted toward the recent values
	// Should be approximately average of the last 2 samples plus current
	assert.InDelta(t, 35.0, avgUsage.CPU, 5.0)          // ~(30+40)/2
	assert.InDelta(t, 75.0, avgUsage.Memory, 5.0)       // ~(70+80)/2
	assert.InDelta(t, 7.5, float64(avgUsage.Threads), 1.0) // ~(7+8)/2
}

// TestThresholdViolationDuration tests measuring the duration of threshold violations
func TestThresholdViolationDuration(t *testing.T) {
	circuitBreaker := watchdog.NewCircuitBreaker("test-component", watchdog.CircuitBreakerConfig{
		Enabled:                 true,
		FailureThreshold:        3,
		ResetTimeout:            5 * time.Second,
		HalfOpenSuccessThreshold: 2,
	})
	
	thresholdConfig := watchdog.ThresholdConfig{
		CPU: watchdog.CPUThreshold{
			Component: 80.0,
		},
		Memory: watchdog.MemoryThreshold{
			Component: 200.0,
		},
	}
	
	monitor := watchdog.NewComponentMonitor("test-component", *circuitBreaker, thresholdConfig)
	
	// Initial state - no violations
	assert.False(t, monitor.IsThresholdViolated(watchdog.ResourceCPU, 0))
	assert.Equal(t, time.Duration(0), monitor.GetThresholdViolationDuration(watchdog.ResourceCPU))
	
	// Update with a threshold violation
	now := time.Now()
	monitor.UpdateResourceUsage(watchdog.ResourceUsage{
		CPU:          90.0, // > 80.0 threshold
		Memory:       100.0,
		Threads:      10,
		Timestamp:    now,
	})
	
	// Should be violated with zero duration
	assert.True(t, monitor.IsThresholdViolated(watchdog.ResourceCPU, 0))
	assert.Equal(t, time.Duration(0), monitor.GetThresholdViolationDuration(watchdog.ResourceCPU))
	
	// Wait some time
	time.Sleep(100 * time.Millisecond)
	
	// Update again with violation still occurring
	monitor.UpdateResourceUsage(watchdog.ResourceUsage{
		CPU:          95.0, // still > 80.0 threshold
		Memory:       100.0,
		Threads:      10,
		Timestamp:    now.Add(100 * time.Millisecond),
	})
	
	// Duration should be tracked
	assert.True(t, monitor.IsThresholdViolated(watchdog.ResourceCPU, 0))
	
	duration := monitor.GetThresholdViolationDuration(watchdog.ResourceCPU)
	assert.True(t, duration >= 100*time.Millisecond)
	
	// Check with a minimum duration requirement
	assert.True(t, monitor.IsThresholdViolated(watchdog.ResourceCPU, 50*time.Millisecond))
	assert.False(t, monitor.IsThresholdViolated(watchdog.ResourceCPU, 200*time.Millisecond))
	
	// Update with values below threshold
	monitor.UpdateResourceUsage(watchdog.ResourceUsage{
		CPU:          70.0, // < 80.0 threshold
		Memory:       100.0,
		Threads:      10,
		Timestamp:    now.Add(200 * time.Millisecond),
	})
	
	// Violation should be cleared
	assert.False(t, monitor.IsThresholdViolated(watchdog.ResourceCPU, 0))
	assert.Equal(t, time.Duration(0), monitor.GetThresholdViolationDuration(watchdog.ResourceCPU))
}

// TestMultipleResourceThresholds tests monitoring multiple resource thresholds
func TestMultipleResourceThresholds(t *testing.T) {
	circuitBreaker := watchdog.NewCircuitBreaker("test-component", watchdog.CircuitBreakerConfig{
		Enabled:                 true,
		FailureThreshold:        3,
		ResetTimeout:            5 * time.Second,
		HalfOpenSuccessThreshold: 2,
	})
	
	thresholdConfig := watchdog.ThresholdConfig{
		CPU: watchdog.CPUThreshold{
			Component: 80.0,
		},
		Memory: watchdog.MemoryThreshold{
			Component: 200.0,
		},
	}
	
	monitor := watchdog.NewComponentMonitor("test-component", *circuitBreaker, thresholdConfig)
	
	// Update with multiple threshold violations
	now := time.Now()
	monitor.UpdateResourceUsage(watchdog.ResourceUsage{
		CPU:          90.0, // > 80.0 CPU threshold
		Memory:       250.0, // > 200.0 Memory threshold
		Threads:      10,
		Timestamp:    now,
	})
	
	// Both thresholds should be violated
	assert.True(t, monitor.IsThresholdViolated(watchdog.ResourceCPU, 0))
	assert.True(t, monitor.IsThresholdViolated(watchdog.ResourceMemory, 0))
	
	// Fix just the CPU threshold
	monitor.UpdateResourceUsage(watchdog.ResourceUsage{
		CPU:          70.0, // < 80.0 CPU threshold
		Memory:       250.0, // still > 200.0 Memory threshold
		Threads:      10,
		Timestamp:    now.Add(100 * time.Millisecond),
	})
	
	// Only memory should be violated now
	assert.False(t, monitor.IsThresholdViolated(watchdog.ResourceCPU, 0))
	assert.True(t, monitor.IsThresholdViolated(watchdog.ResourceMemory, 0))
	
	// Fix the memory threshold
	monitor.UpdateResourceUsage(watchdog.ResourceUsage{
		CPU:          70.0,
		Memory:       150.0, // < 200.0 Memory threshold
		Threads:      10,
		Timestamp:    now.Add(200 * time.Millisecond),
	})
	
	// No thresholds should be violated
	assert.False(t, monitor.IsThresholdViolated(watchdog.ResourceCPU, 0))
	assert.False(t, monitor.IsThresholdViolated(watchdog.ResourceMemory, 0))
}
