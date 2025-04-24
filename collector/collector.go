// Package collector provides functionality for collecting system and process metrics.
package collector

import (
	"context"
	"time"
)

// Collector defines the interface that all collectors must implement.
type Collector interface {
	// Init initializes the collector with a context
	Init(ctx context.Context) error
	
	// Start begins the collection process
	Start() error
	
	// Stop halts the collection process
	Stop() error
	
	// Status returns the current status of the collector
	Status() Status
	
	// Metrics returns performance metrics for the collector
	Metrics() map[string]float64
	
	// Resources returns resource usage of the collector itself
	Resources() map[string]float64
	
	// Shutdown gracefully shuts down the collector
	Shutdown() error
}

// Status represents the current state of a collector
type Status string

const (
	// StatusInitialized means the collector is initialized but not started
	StatusInitialized Status = "initialized"
	
	// StatusRunning means the collector is actively collecting data
	StatusRunning Status = "running"
	
	// StatusPaused means the collector is temporarily paused
	StatusPaused Status = "paused"
	
	// StatusStopped means the collector has been stopped
	StatusStopped Status = "stopped"
	
	// StatusError means the collector encountered an error
	StatusError Status = "error"
)

// CollectorFactory creates a new collector instance
type CollectorFactory func() Collector

// RegisterCollector registers a collector factory with a name
func RegisterCollector(name string, factory CollectorFactory) {
	collectorRegistry[name] = factory
}

// collectorRegistry holds all registered collectors
var collectorRegistry = make(map[string]CollectorFactory)

// GetCollector returns a collector factory by name
func GetCollector(name string) (CollectorFactory, bool) {
	factory, exists := collectorRegistry[name]
	return factory, exists
}

// GetCollectorNames returns all registered collector names
func GetCollectorNames() []string {
	names := make([]string, 0, len(collectorRegistry))
	for name := range collectorRegistry {
		names = append(names, name)
	}
	return names
}

// ProcessEvent represents a process lifecycle event
type ProcessEvent struct {
	// Type of event (created, updated, terminated)
	Type ProcessEventType
	
	// Process information
	Process *ProcessInfo
	
	// Timestamp of the event
	Timestamp time.Time
}

// ProcessEventType defines the type of process event
type ProcessEventType string

const (
	// ProcessCreated indicates a new process was created
	ProcessCreated ProcessEventType = "created"
	
	// ProcessUpdated indicates an existing process was updated
	ProcessUpdated ProcessEventType = "updated"
	
	// ProcessTerminated indicates a process was terminated
	ProcessTerminated ProcessEventType = "terminated"
)

// ProcessConsumer defines the interface for components that consume process information
type ProcessConsumer interface {
	// HandleProcessEvent handles a process event
	HandleProcessEvent(event ProcessEvent) error
}
