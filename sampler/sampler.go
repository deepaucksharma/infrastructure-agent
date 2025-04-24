// Package sampler provides interfaces and implementations for process sampling strategies.
package sampler

import (
	"context"
	"time"
)

// ProcessInfo represents a process with its resource usage metrics.
type ProcessInfo struct {
	PID       int       `json:"pid"`
	Name      string    `json:"name"`
	Command   string    `json:"command"`
	CPU       float64   `json:"cpu"`      // CPU usage percentage
	RSS       int64     `json:"rss"`      // Resident Set Size in bytes
	StartTime time.Time `json:"startTime"`
	Score     float64   `json:"score"`    // Computed score for ranking
}

// Sampler defines the interface that all samplers must implement.
type Sampler interface {
	// Init initializes the sampler with a context
	Init(ctx context.Context) error

	// Update updates the internal state with new process information
	Update(processes []*ProcessInfo) error

	// GetTopN returns the top N processes according to the sampling strategy
	GetTopN(n int) []*ProcessInfo

	// Metrics returns performance metrics for the sampler
	Metrics() map[string]float64

	// Resources returns resource usage of the sampler itself
	Resources() map[string]float64

	// Shutdown gracefully shuts down the sampler
	Shutdown() error
}

// SamplerFactory creates a new sampler instance
type SamplerFactory func() Sampler

// RegisterSampler registers a sampler factory with a name
func RegisterSampler(name string, factory SamplerFactory) {
	samplerRegistry[name] = factory
}

// samplerRegistry holds all registered samplers
var samplerRegistry = make(map[string]SamplerFactory)

// GetSampler returns a sampler factory by name
func GetSampler(name string) (SamplerFactory, bool) {
	factory, exists := samplerRegistry[name]
	return factory, exists
}

// GetSamplerNames returns all registered sampler names
func GetSamplerNames() []string {
	names := make([]string, 0, len(samplerRegistry))
	for name := range samplerRegistry {
		names = append(names, name)
	}
	return names
}
