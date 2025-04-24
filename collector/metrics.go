package collector

import (
	"sync"
	"time"
)

// MetricsTracker collects and aggregates metrics for the collector module
type MetricsTracker struct {
	metrics     map[string]float64
	counters    map[string]int64
	timers      map[string]time.Duration
	mutex       sync.RWMutex
	startTime   time.Time
}

// NewMetricsTracker creates a new metrics tracker
func NewMetricsTracker() *MetricsTracker {
	return &MetricsTracker{
		metrics:   make(map[string]float64),
		counters:  make(map[string]int64),
		timers:    make(map[string]time.Duration),
		startTime: time.Now(),
	}
}

// SetGauge sets a gauge metric value
func (m *MetricsTracker) SetGauge(name string, value float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.metrics[name] = value
}

// GetGauge gets a gauge metric value
func (m *MetricsTracker) GetGauge(name string) float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.metrics[name]
}

// IncrementCounter increments a counter by the specified amount
func (m *MetricsTracker) IncrementCounter(name string, value int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.counters[name] += value
}

// GetCounter gets a counter value
func (m *MetricsTracker) GetCounter(name string) int64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.counters[name]
}

// RecordDuration records a duration for a timer
func (m *MetricsTracker) RecordDuration(name string, duration time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.timers[name] = duration
}

// GetDuration gets a timer duration
func (m *MetricsTracker) GetDuration(name string) time.Duration {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.timers[name]
}

// StartTimer starts a timer and returns a function to stop it
func (m *MetricsTracker) StartTimer(name string) func() {
	start := time.Now()
	
	return func() {
		duration := time.Since(start)
		m.RecordDuration(name, duration)
	}
}

// GetAllMetrics returns all metrics as a map
func (m *MetricsTracker) GetAllMetrics() map[string]float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	result := make(map[string]float64, len(m.metrics)+len(m.counters)+len(m.timers)+1)
	
	// Copy gauge metrics
	for k, v := range m.metrics {
		result[k] = v
	}
	
	// Convert counters to float64
	for k, v := range m.counters {
		result[k] = float64(v)
	}
	
	// Convert timers to milliseconds
	for k, v := range m.timers {
		result[k+"_ms"] = float64(v.Milliseconds())
	}
	
	// Add uptime
	result["uptime_seconds"] = float64(time.Since(m.startTime).Seconds())
	
	return result
}

// Reset clears all metrics
func (m *MetricsTracker) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.metrics = make(map[string]float64)
	m.counters = make(map[string]int64)
	m.timers = make(map[string]time.Duration)
	m.startTime = time.Now()
}

// ProcessScannerMetrics defines the standard metrics for the process scanner
const (
	// Performance metrics
	MetricScanDuration         = "scan_duration_ms"
	MetricCPUUsage             = "cpu_usage_percent"
	MetricMemoryUsage          = "memory_usage_bytes"
	
	// Process metrics
	MetricProcessCount         = "process_count"
	MetricProcessCreated       = "process_created_total"
	MetricProcessUpdated       = "process_updated_total"
	MetricProcessTerminated    = "process_terminated_total"
	
	// Error metrics
	MetricScanErrors           = "scan_errors_total"
	MetricLimitBreaches        = "limit_breaches_total"
	MetricNotificationErrors   = "notification_errors_total"
	
	// Resource tracking
	MetricScanIntervalActual   = "scan_interval_actual_ms"
	MetricAdaptiveRateChanges  = "adaptive_rate_changes_total"
	MetricEventQueueSize       = "event_queue_size"
	MetricConsumerCount        = "consumer_count"
)
