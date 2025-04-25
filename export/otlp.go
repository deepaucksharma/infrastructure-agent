package export

import (
	"context"
	"fmt"
	"log"
)

// OTLPExporter is the main implementation of the OTLP exporter
type OTLPExporter struct {
	*BaseExporter
	converter *Converter
}

// OTLPExporterFactory creates new OTLP exporters
type OTLPExporterFactory struct{}

// Create returns a new OTLP exporter with the provided configuration
func (f *OTLPExporterFactory) Create(config Config) (Exporter, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Create converter
	converter := NewConverter()
	
	// Create processor function
	processor := func(ctx context.Context, batch *Batch) error {
		// Process based on protocol
		switch config.Protocol {
		case ProtocolGRPC:
			return processGRPCBatch(ctx, batch, converter, config)
		case ProtocolHTTP:
			return processHTTPBatch(ctx, batch, converter, config)
		default:
			return fmt.Errorf("unsupported protocol: %s", config.Protocol)
		}
	}
	
	// Create base exporter
	baseExporter := NewBaseExporter(config, processor)
	
	// Create OTLP exporter
	exporter := &OTLPExporter{
		BaseExporter: baseExporter,
		converter:    converter,
	}
	
	return exporter, nil
}

// processGRPCBatch processes a batch of telemetry data using gRPC
func processGRPCBatch(ctx context.Context, batch *Batch, converter *Converter, config Config) error {
	// In a real implementation, this would convert the batch to OTLP format and send it using gRPC
	// For now, we'll just simulate the process
	
	// Convert batch to OTLP format
	metrics := make([]map[string]interface{}, 0)
	logs := make([]map[string]interface{}, 0)
	spans := make([]map[string]interface{}, 0)
	
	// Process each item in the batch
	for _, item := range batch.Items {
		switch data := item.(type) {
		case Metric:
			otlpMetric, err := converter.ConvertMetric(data)
			if err != nil {
				log.Printf("Error converting metric: %v", err)
				continue
			}
			metrics = append(metrics, otlpMetric)
		case LogRecord:
			otlpLog, err := converter.ConvertLog(data)
			if err != nil {
				log.Printf("Error converting log: %v", err)
				continue
			}
			logs = append(logs, otlpLog)
		case Span:
			otlpSpan, err := converter.ConvertSpan(data)
			if err != nil {
				log.Printf("Error converting span: %v", err)
				continue
			}
			spans = append(spans, otlpSpan)
		default:
			log.Printf("Unsupported telemetry data type: %T", data)
		}
	}
	
	// Create resource information
	resource := map[string]interface{}{
		"attributes": map[string]string{
			"service.name":        "infrastructure-agent",
			"service.version":     "1.0.0",
			"telemetry.sdk.name":  "infrastructure-agent",
			"telemetry.sdk.language": "go",
		},
	}
	
	// Create payload
	payload := map[string]interface{}{
		"resource_metrics": []map[string]interface{}{
			{
				"resource": resource,
				"scope_metrics": []map[string]interface{}{
					{
						"scope": map[string]interface{}{
							"name":    "infrastructure-agent",
							"version": "1.0.0",
						},
						"metrics": metrics,
					},
				},
			},
		},
		"resource_logs": []map[string]interface{}{
			{
				"resource": resource,
				"scope_logs": []map[string]interface{}{
					{
						"scope": map[string]interface{}{
							"name":    "infrastructure-agent",
							"version": "1.0.0",
						},
						"log_records": logs,
					},
				},
			},
		},
		"resource_spans": []map[string]interface{}{
			{
				"resource": resource,
				"scope_spans": []map[string]interface{}{
					{
						"scope": map[string]interface{}{
							"name":    "infrastructure-agent",
							"version": "1.0.0",
						},
						"spans": spans,
					},
				},
			},
		},
	}
	
	// In a real implementation, this would send the payload using gRPC
	// For now, we'll just log a message
	log.Printf("Processed batch with %d metrics, %d logs, and %d spans using gRPC", 
		len(metrics), len(logs), len(spans))
	
	return nil
}

// processHTTPBatch processes a batch of telemetry data using HTTP
func processHTTPBatch(ctx context.Context, batch *Batch, converter *Converter, config Config) error {
	// In a real implementation, this would convert the batch to OTLP format and send it using HTTP
	// For now, we'll just simulate the process
	
	// Convert batch to OTLP format (same as gRPC for simplicity)
	metrics := make([]map[string]interface{}, 0)
	logs := make([]map[string]interface{}, 0)
	spans := make([]map[string]interface{}, 0)
	
	// Process each item in the batch
	for _, item := range batch.Items {
		switch data := item.(type) {
		case Metric:
			otlpMetric, err := converter.ConvertMetric(data)
			if err != nil {
				log.Printf("Error converting metric: %v", err)
				continue
			}
			metrics = append(metrics, otlpMetric)
		case LogRecord:
			otlpLog, err := converter.ConvertLog(data)
			if err != nil {
				log.Printf("Error converting log: %v", err)
				continue
			}
			logs = append(logs, otlpLog)
		case Span:
			otlpSpan, err := converter.ConvertSpan(data)
			if err != nil {
				log.Printf("Error converting span: %v", err)
				continue
			}
			spans = append(spans, otlpSpan)
		default:
			log.Printf("Unsupported telemetry data type: %T", data)
		}
	}
	
	// In a real implementation, this would send the payload using HTTP
	// For now, we'll just log a message
	log.Printf("Processed batch with %d metrics, %d logs, and %d spans using HTTP", 
		len(metrics), len(logs), len(spans))
	
	return nil
}

// NewOTLPExporterFactory creates a new OTLP exporter factory
func NewOTLPExporterFactory() ExporterFactory {
	return &OTLPExporterFactory{}
}
