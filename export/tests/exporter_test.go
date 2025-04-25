package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/deepaucksharma/infrastructure-agent/export"
)

func TestOTLPExporterCreation(t *testing.T) {
	// Create factory
	factory := export.NewOTLPExporterFactory()
	
	// Create config
	config := export.DefaultConfig()
	
	// Create exporter
	exporter, err := factory.Create(config)
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}
	
	// Verify exporter is not nil
	if exporter == nil {
		t.Fatal("Exporter is nil")
	}
	
	// Verify exporter status
	status := exporter.Status()
	if status.State != export.ExporterRunning {
		t.Fatalf("Expected exporter state %v, got %v", export.ExporterRunning, status.State)
	}
	
	// Shutdown exporter
	err = exporter.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Failed to shutdown exporter: %v", err)
	}
	
	// Verify exporter is stopped
	status = exporter.Status()
	if status.State != export.ExporterStopped {
		t.Fatalf("Expected exporter state %v, got %v", export.ExporterStopped, status.State)
	}
}

func TestOTLPExporterExport(t *testing.T) {
	// Create factory
	factory := export.NewOTLPExporterFactory()
	
	// Create config
	config := export.DefaultConfig()
	
	// Set smaller batch size for testing
	config.Batch.Size = 1
	
	// Create exporter
	exporter, err := factory.Create(config)
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}
	
	// Create a metric
	metric := export.Metric{
		Name: "test_metric",
		Value: export.MetricValue{
			Type:  export.MetricTypeGauge,
			Value: 42.0,
		},
		Labels: map[string]string{
			"service": "test",
			"host":    "localhost",
		},
		Timestamp: time.Now(),
	}
	
	// Export the metric
	err = exporter.Export(context.Background(), metric)
	if err != nil {
		t.Fatalf("Failed to export metric: %v", err)
	}
	
	// Verify exporter status
	status := exporter.Status()
	if status.TotalExports != 1 {
		t.Fatalf("Expected 1 export, got %d", status.TotalExports)
	}
	
	// Shutdown exporter
	err = exporter.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Failed to shutdown exporter: %v", err)
	}
}

func TestCircuitBreaker(t *testing.T) {
	// Create a circuit breaker
	config := export.CircuitConfig{
		Enabled:                 true,
		FailureThreshold:        2,
		ResetTimeout:            1 * time.Second,
		HalfOpenSuccessThreshold: 1,
	}
	
	cb := export.NewCircuitBreaker(config)
	
	// Verify initial state
	if cb.State() != export.CircuitClosed {
		t.Fatalf("Expected circuit state %v, got %v", export.CircuitClosed, cb.State())
	}
	
	// Record failures to open the circuit
	cb.RecordFailure()
	if cb.State() != export.CircuitClosed {
		t.Fatalf("Expected circuit state %v, got %v", export.CircuitClosed, cb.State())
	}
	
	cb.RecordFailure()
	if cb.State() != export.CircuitOpen {
		t.Fatalf("Expected circuit state %v, got %v", export.CircuitOpen, cb.State())
	}
	
	// Verify request is not allowed
	if cb.AllowRequest() {
		t.Fatal("Request should not be allowed when circuit is open")
	}
	
	// Wait for reset timeout
	time.Sleep(1100 * time.Millisecond)
	
	// Verify circuit is half-open
	if !cb.AllowRequest() {
		t.Fatal("Request should be allowed when circuit is half-open")
	}
	
	// Record success to close the circuit
	cb.RecordSuccess()
	if cb.State() != export.CircuitClosed {
		t.Fatalf("Expected circuit state %v, got %v", export.CircuitClosed, cb.State())
	}
}

func TestBackoffRetrier(t *testing.T) {
	// Create a retrier
	config := export.RetryConfig{
		Enabled:         true,
		MaxAttempts:     3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
	}
	
	retrier := export.NewBackoffRetrier(config)
	
	// Test successful execution
	attemptCount := 0
	err := retrier.Do(context.Background(), func() error {
		attemptCount++
		return nil
	})
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if attemptCount != 1 {
		t.Fatalf("Expected 1 attempt, got %d", attemptCount)
	}
	
	// Test failed execution with retries
	attemptCount = 0
	startTime := time.Now()
	
	err = retrier.Do(context.Background(), func() error {
		attemptCount++
		return fmt.Errorf("test error")
	})
	
	duration := time.Since(startTime)
	
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	
	if attemptCount != 3 {
		t.Fatalf("Expected 3 attempts, got %d", attemptCount)
	}
	
	// Verify minimum duration (initial + initial*multiplier)
	// We can't be too precise due to jitter
	minDuration := 100*time.Millisecond + 200*time.Millisecond
	if duration < minDuration {
		t.Fatalf("Expected duration >= %v, got %v", minDuration, duration)
	}
}
