package export

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ExporterState represents the state of an exporter
type ExporterState int

const (
	// ExporterRunning indicates the exporter is running
	ExporterRunning ExporterState = iota
	
	// ExporterShuttingDown indicates the exporter is shutting down
	ExporterShuttingDown
	
	// ExporterStopped indicates the exporter is stopped
	ExporterStopped
)

// ExporterStatus holds status information about an exporter
type ExporterStatus struct {
	// State is the current state of the exporter
	State ExporterState
	
	// LastExportTime is the time of the last successful export
	LastExportTime time.Time
	
	// LastExportError is the last error encountered during export
	LastExportError error
	
	// TotalExports is the total number of successful exports
	TotalExports int64
	
	// TotalErrors is the total number of export errors
	TotalErrors int64
	
	// CircuitState is the current state of the circuit breaker
	CircuitState CircuitState
}

// TelemetryData represents a generic piece of telemetry data
type TelemetryData interface {
	// Type returns the type of telemetry data
	Type() string
}

// Exporter defines the interface for telemetry exporters
type Exporter interface {
	// Export exports the provided telemetry data
	Export(ctx context.Context, data TelemetryData) error
	
	// Shutdown performs a graceful shutdown of the exporter
	Shutdown(ctx context.Context) error
	
	// Status returns the current status of the exporter
	Status() ExporterStatus
}

// ExporterFactory creates and configures new exporters
type ExporterFactory interface {
	// Create returns a new exporter with the provided configuration
	Create(config Config) (Exporter, error)
}

// BaseExporter provides common functionality for all exporters
type BaseExporter struct {
	config           Config
	circuitBreaker   *CircuitBreaker
	retrier          *BackoffRetrier
	batchProcessor   *BatchProcessor
	state            ExporterState
	lastExportTime   time.Time
	lastExportError  error
	totalExports     int64
	totalErrors      int64
	mu               sync.RWMutex
	exporterSpecific interface{} // Holds protocol-specific exporters (gRPC or HTTP)
}

// NewBaseExporter creates a new base exporter with the given configuration
func NewBaseExporter(config Config, processor func(context.Context, *Batch) error) *BaseExporter {
	circuitBreaker := NewCircuitBreaker(config.CircuitBreaker)
	retrier := NewBackoffRetrier(config.Retry)
	batchProcessor := NewBatchProcessor(config.Batch, processor)
	
	return &BaseExporter{
		config:         config,
		circuitBreaker: circuitBreaker,
		retrier:        retrier,
		batchProcessor: batchProcessor,
		state:          ExporterRunning,
	}
}

// Export exports the provided telemetry data
func (e *BaseExporter) Export(ctx context.Context, data TelemetryData) error {
	// Check if the exporter is running
	if e.state != ExporterRunning {
		return fmt.Errorf("exporter is not running")
	}
	
	// Check if the circuit breaker allows the request
	if !e.circuitBreaker.AllowRequest() {
		e.recordError(fmt.Errorf("circuit breaker is open"))
		return fmt.Errorf("circuit breaker is open")
	}
	
	// Add the data to the batch processor
	err := e.batchProcessor.Add(ctx, data)
	if err != nil {
		// Record the error and notify the circuit breaker
		e.recordError(err)
		e.circuitBreaker.RecordFailure()
		return err
	}
	
	// Record a successful export
	e.recordSuccess()
	e.circuitBreaker.RecordSuccess()
	return nil
}

// Shutdown performs a graceful shutdown of the exporter
func (e *BaseExporter) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	if e.state == ExporterRunning {
		e.state = ExporterShuttingDown
	}
	e.mu.Unlock()
	
	// Shutdown the batch processor
	err := e.batchProcessor.Shutdown(ctx)
	
	e.mu.Lock()
	e.state = ExporterStopped
	e.mu.Unlock()
	
	return err
}

// Status returns the current status of the exporter
func (e *BaseExporter) Status() ExporterStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	return ExporterStatus{
		State:          e.state,
		LastExportTime: e.lastExportTime,
		LastExportError: e.lastExportError,
		TotalExports:   e.totalExports,
		TotalErrors:    e.totalErrors,
		CircuitState:   e.circuitBreaker.State(),
	}
}

// recordSuccess records a successful export
func (e *BaseExporter) recordSuccess() {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.lastExportTime = time.Now()
	e.totalExports++
}

// recordError records an export error
func (e *BaseExporter) recordError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.lastExportError = err
	e.totalErrors++
}

// SetExporterSpecific sets the protocol-specific exporter
func (e *BaseExporter) SetExporterSpecific(exporterSpecific interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.exporterSpecific = exporterSpecific
}

// GetExporterSpecific returns the protocol-specific exporter
func (e *BaseExporter) GetExporterSpecific() interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	return e.exporterSpecific
}
