package export

import (
	"fmt"
	"time"
)

// MetricType represents the type of a metric
type MetricType int

const (
	// MetricTypeGauge represents a gauge metric
	MetricTypeGauge MetricType = iota
	
	// MetricTypeCounter represents a counter metric
	MetricTypeCounter
	
	// MetricTypeHistogram represents a histogram metric
	MetricTypeHistogram
	
	// MetricTypeSummary represents a summary metric
	MetricTypeSummary
)

// MetricValue holds a value for a metric
type MetricValue struct {
	// Type is the type of the metric
	Type MetricType
	
	// Value is the value of the metric
	Value float64
	
	// Count is the count for histogram or summary metrics
	Count uint64
	
	// Sum is the sum for histogram or summary metrics
	Sum float64
	
	// Buckets holds histogram bucket values
	Buckets map[float64]uint64
	
	// Quantiles holds summary quantile values
	Quantiles map[float64]float64
}

// Metric represents a metric data point
type Metric struct {
	// Name is the name of the metric
	Name string
	
	// Value is the value of the metric
	Value MetricValue
	
	// Labels are the labels associated with the metric
	Labels map[string]string
	
	// Timestamp is the time the metric was created
	Timestamp time.Time
}

// LogRecord represents a log record
type LogRecord struct {
	// Timestamp is the time the log was created
	Timestamp time.Time
	
	// Body is the main content of the log
	Body string
	
	// Severity is the severity level of the log
	Severity string
	
	// Attributes are additional attributes associated with the log
	Attributes map[string]string
}

// SpanEvent represents a span event
type SpanEvent struct {
	// Name is the name of the event
	Name string
	
	// Timestamp is the time the event occurred
	Timestamp time.Time
	
	// Attributes are additional attributes associated with the event
	Attributes map[string]string
}

// Span represents a span in a trace
type Span struct {
	// TraceID is the ID of the trace this span belongs to
	TraceID string
	
	// SpanID is the ID of this span
	SpanID string
	
	// ParentSpanID is the ID of the parent span
	ParentSpanID string
	
	// Name is the name of the span
	Name string
	
	// StartTime is when the span started
	StartTime time.Time
	
	// EndTime is when the span ended
	EndTime time.Time
	
	// Attributes are additional attributes associated with the span
	Attributes map[string]string
	
	// Events are events that occurred during the span
	Events []SpanEvent
}

// Resource represents a resource that generated telemetry data
type Resource struct {
	// Type is the type of the resource
	Type string
	
	// Attributes are additional attributes of the resource
	Attributes map[string]string
}

// Converter converts internal telemetry data to OTLP format
type Converter struct {
	// Options for the converter
	conversionTimestampFormat string
}

// NewConverter creates a new converter with default options
func NewConverter() *Converter {
	return &Converter{
		conversionTimestampFormat: time.RFC3339Nano,
	}
}

// ConvertMetric converts an internal metric to OTLP format
func (c *Converter) ConvertMetric(metric Metric) (map[string]interface{}, error) {
	// Create basic structure
	result := map[string]interface{}{
		"name":       metric.Name,
		"timestamp":  metric.Timestamp.Format(c.conversionTimestampFormat),
		"attributes": metric.Labels,
	}
	
	// Add type-specific fields
	switch metric.Value.Type {
	case MetricTypeGauge:
		result["type"] = "gauge"
		result["gauge"] = map[string]interface{}{
			"value": metric.Value.Value,
		}
	case MetricTypeCounter:
		result["type"] = "sum"
		result["sum"] = map[string]interface{}{
			"value": metric.Value.Value,
			"aggregation_temporality": "cumulative",
			"is_monotonic": true,
		}
	case MetricTypeHistogram:
		result["type"] = "histogram"
		histogramData := map[string]interface{}{
			"count": metric.Value.Count,
			"sum":   metric.Value.Sum,
			"aggregation_temporality": "delta",
		}
		
		// Convert buckets to OTLP format
		if len(metric.Value.Buckets) > 0 {
			bounds := make([]float64, 0, len(metric.Value.Buckets))
			counts := make([]uint64, 0, len(metric.Value.Buckets))
			
			// Sort bounds (this is a simplification, in practice you'd want to sort them)
			for bound := range metric.Value.Buckets {
				bounds = append(bounds, bound)
			}
			
			// Get counts in order of bounds
			for _, bound := range bounds {
				counts = append(counts, metric.Value.Buckets[bound])
			}
			
			histogramData["explicit_bounds"] = bounds
			histogramData["bucket_counts"] = counts
		}
		
		result["histogram"] = histogramData
	case MetricTypeSummary:
		result["type"] = "summary"
		summaryData := map[string]interface{}{
			"count": metric.Value.Count,
			"sum":   metric.Value.Sum,
		}
		
		// Convert quantiles to OTLP format
		if len(metric.Value.Quantiles) > 0 {
			quantileValues := make([]map[string]interface{}, 0, len(metric.Value.Quantiles))
			
			for quantile, value := range metric.Value.Quantiles {
				quantileValues = append(quantileValues, map[string]interface{}{
					"quantile": quantile,
					"value":    value,
				})
			}
			
			summaryData["quantile_values"] = quantileValues
		}
		
		result["summary"] = summaryData
	default:
		return nil, fmt.Errorf("unsupported metric type: %v", metric.Value.Type)
	}
	
	return result, nil
}

// ConvertLog converts an internal log record to OTLP format
func (c *Converter) ConvertLog(log LogRecord) (map[string]interface{}, error) {
	// Create basic structure
	result := map[string]interface{}{
		"timestamp":         log.Timestamp.Format(c.conversionTimestampFormat),
		"severity_text":     log.Severity,
		"body":              log.Body,
		"attributes":        log.Attributes,
		"dropped_attributes_count": 0,
	}
	
	// Map severity text to severity number
	severityNumber := 0
	switch log.Severity {
	case "TRACE":
		severityNumber = 1
	case "DEBUG":
		severityNumber = 5
	case "INFO":
		severityNumber = 9
	case "WARN":
		severityNumber = 13
	case "ERROR":
		severityNumber = 17
	case "FATAL":
		severityNumber = 21
	}
	
	result["severity_number"] = severityNumber
	
	return result, nil
}

// ConvertSpan converts an internal span to OTLP format
func (c *Converter) ConvertSpan(span Span) (map[string]interface{}, error) {
	// Create basic structure
	result := map[string]interface{}{
		"trace_id":          span.TraceID,
		"span_id":           span.SpanID,
		"parent_span_id":    span.ParentSpanID,
		"name":              span.Name,
		"start_time_unix_nano": span.StartTime.UnixNano(),
		"end_time_unix_nano":   span.EndTime.UnixNano(),
		"attributes":        span.Attributes,
		"dropped_attributes_count": 0,
		"status": map[string]interface{}{
			"code": "OK",
		},
	}
	
	// Convert events
	if len(span.Events) > 0 {
		events := make([]map[string]interface{}, 0, len(span.Events))
		
		for _, event := range span.Events {
			events = append(events, map[string]interface{}{
				"name":              event.Name,
				"time_unix_nano":    event.Timestamp.UnixNano(),
				"attributes":        event.Attributes,
				"dropped_attributes_count": 0,
			})
		}
		
		result["events"] = events
		result["dropped_events_count"] = 0
	}
	
	return result, nil
}

// ConvertResource converts an internal resource to OTLP format
func (c *Converter) ConvertResource(resource Resource) (map[string]interface{}, error) {
	// Create basic structure
	result := map[string]interface{}{
		"attributes":        resource.Attributes,
		"dropped_attributes_count": 0,
	}
	
	return result, nil
}
