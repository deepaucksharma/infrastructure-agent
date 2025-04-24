// Package sketch provides data structures for efficient statistical analysis.
package sketch

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrInvalidQuantile is returned when an invalid quantile is requested
	ErrInvalidQuantile = errors.New("invalid quantile: must be between 0 and 1")
	
	// ErrEmptySketch is returned when trying to query a sketch with no data
	ErrEmptySketch = errors.New("cannot query empty sketch")
	
	// ErrInvalidParameter is returned when an invalid parameter is provided
	ErrInvalidParameter = errors.New("invalid parameter")
	
	// ErrIncompatibleSketches is returned when trying to merge incompatible sketches
	ErrIncompatibleSketches = errors.New("cannot merge incompatible sketches")
)

// Sketch defines the interface that all sketches must implement.
// A sketch is a data structure that provides approximate answers to
// quantile queries using sublinear space.
type Sketch interface {
	// Add adds a value to the sketch
	Add(value float64) error
	
	// AddWithCount adds a value to the sketch with a specific count
	AddWithCount(value float64, count uint64) error
	
	// GetValueAtQuantile returns the value at the specified quantile
	GetValueAtQuantile(q float64) (float64, error)
	
	// GetQuantileAtValue returns the quantile at which value falls
	GetQuantileAtValue(value float64) (float64, error)
	
	// GetCount returns the total count of values in the sketch
	GetCount() uint64
	
	// GetMin returns the minimum value added to the sketch
	GetMin() (float64, error)
	
	// GetMax returns the maximum value added to the sketch
	GetMax() (float64, error)
	
	// GetSum returns the sum of all values added to the sketch
	GetSum() (float64, error)
	
	// GetAvg returns the average of all values added to the sketch
	GetAvg() (float64, error)
	
	// Merge merges another sketch into this one
	Merge(other Sketch) error
	
	// Copy creates a deep copy of the sketch
	Copy() Sketch
	
	// Reset resets the sketch to an empty state
	Reset()
	
	// Bytes returns a serialized representation of the sketch
	Bytes() ([]byte, error)
	
	// FromBytes populates the sketch from a serialized representation
	FromBytes(data []byte) error
	
	// Resources returns resource usage of the sketch itself
	Resources() map[string]float64
}

// SketchFactory creates a new sketch instance
type SketchFactory func() Sketch

// SketchProvider is a configurable provider of sketches
type SketchProvider interface {
	// Init initializes the provider with a context
	Init(ctx context.Context) error
	
	// GetSketch returns a sketch for the given name
	GetSketch(name string) (Sketch, error)
	
	// ListSketches returns a list of available sketch names
	ListSketches() []string
	
	// Shutdown gracefully shuts down the provider
	Shutdown() error
	
	// Resources returns resource usage of the provider
	Resources() map[string]float64
}

// RegisterSketch registers a sketch factory with a name
func RegisterSketch(name string, factory SketchFactory) {
	sketchRegistry[name] = factory
}

// sketchRegistry holds all registered sketches
var sketchRegistry = make(map[string]SketchFactory)

// GetSketch returns a sketch factory by name
func GetSketch(name string) (SketchFactory, bool) {
	factory, exists := sketchRegistry[name]
	return factory, exists
}

// GetSketchNames returns all registered sketch names
func GetSketchNames() []string {
	names := make([]string, 0, len(sketchRegistry))
	for name := range sketchRegistry {
		names = append(names, name)
	}
	return names
}
