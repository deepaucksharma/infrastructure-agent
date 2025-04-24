package sampler

import (
	"sync"
	"time"
)

// MetricsTracker collects and aggregates metrics for the sampler module.
type MetricsTracker struct {
	metrics     map[string]float64
	mu          sync.RWMutex
	lastUpdated time.Time
	window      time.Duration
	history     map[string][]float64
}

// NewMetricsTracker creates a new metrics tracker with the specified window duration.
func NewMetricsTracker(window time.Duration) *MetricsTracker {
	return &MetricsTracker{
		metrics:     make(map[string]float64),
		lastUpdated: time.Now(),
		window:      window,
		history:     make(map[string][]float64),
	}
}

// Set sets the value of a metric.
func (m *MetricsTracker) Set(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics[name] = value
	
	// Track history for selected metrics
	switch name {
	case "topn_capture_ratio", "topn_update_time_seconds", "sampler_cpu_percent":
		if _, ok := m.history[name]; !ok {
			m.history[name] = make([]float64, 0, 100)
		}
		m.history[name] = append(m.history[name], value)
		// Keep only the most recent values within window
		m.pruneHistory(name)
	}
}

// Get returns the value of a metric.
func (m *MetricsTracker) Get(name string) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.metrics[name]
	return value, ok
}

// GetAll returns a copy of all metrics.
func (m *MetricsTracker) GetAll() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make(map[string]float64, len(m.metrics))
	for k, v := range m.metrics {
		metrics[k] = v
	}
	return metrics
}

// Increment increments a metric by the specified amount.
func (m *MetricsTracker) Increment(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics[name] += value
}

// Average returns the average value of a metric over the window.
func (m *MetricsTracker) Average(name string) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	values, ok := m.history[name]
	if !ok || len(values) == 0 {
		return 0, false
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values)), true
}

// Max returns the maximum value of a metric over the window.
func (m *MetricsTracker) Max(name string) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	values, ok := m.history[name]
	if !ok || len(values) == 0 {
		return 0, false
	}

	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max, true
}

// Min returns the minimum value of a metric over the window.
func (m *MetricsTracker) Min(name string) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	values, ok := m.history[name]
	if !ok || len(values) == 0 {
		return 0, false
	}

	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min, true
}

// pruneHistory removes old values from the history.
func (m *MetricsTracker) pruneHistory(name string) {
	now := time.Now()
	if now.Sub(m.lastUpdated) < m.window/10 {
		// Don't prune too often
		return
	}
	
	// Update last pruned time
	m.lastUpdated = now
	
	// Cap history size to prevent unbounded growth
	maxHistorySize := 1000
	if values, ok := m.history[name]; ok && len(values) > maxHistorySize {
		m.history[name] = values[len(values)-maxHistorySize:]
	}
}
