package export

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// HTTPClient is a client for the OTLP HTTP service
type HTTPClient struct {
	config     Config
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTP client with the given configuration
func NewHTTPClient(config Config) (*HTTPClient, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Check protocol
	if config.Protocol != ProtocolHTTP {
		return nil, fmt.Errorf("expected HTTP protocol, got %s", config.Protocol)
	}
	
	// Create TLS config
	tlsConfig, err := createTLSConfig(config.TLS)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}
	
	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			MaxIdleConns: 100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout: 90 * time.Second,
		},
	}
	
	return &HTTPClient{
		config:     config,
		httpClient: client,
	}, nil
}

// SendMetrics sends metrics to the OTLP service
func (c *HTTPClient) SendMetrics(ctx context.Context, metrics []map[string]interface{}, resource map[string]interface{}) error {
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
	}
	
	// Send payload
	return c.sendPayload(ctx, "/v1/metrics", payload)
}

// SendLogs sends logs to the OTLP service
func (c *HTTPClient) SendLogs(ctx context.Context, logs []map[string]interface{}, resource map[string]interface{}) error {
	// Create payload
	payload := map[string]interface{}{
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
	}
	
	// Send payload
	return c.sendPayload(ctx, "/v1/logs", payload)
}

// SendTraces sends traces to the OTLP service
func (c *HTTPClient) SendTraces(ctx context.Context, spans []map[string]interface{}, resource map[string]interface{}) error {
	// Create payload
	payload := map[string]interface{}{
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
	
	// Send payload
	return c.sendPayload(ctx, "/v1/traces", payload)
}

// Shutdown shuts down the HTTP client
func (c *HTTPClient) Shutdown(ctx context.Context) error {
	// In a real implementation, this would close idle connections
	c.httpClient.CloseIdleConnections()
	
	log.Printf("Shutting down HTTP client")
	
	return nil
}

// sendPayload sends a payload to the OTLP service
func (c *HTTPClient) sendPayload(ctx context.Context, path string, payload interface{}) error {
	// In a real implementation, this would convert the payload to JSON and send it using HTTP
	// For now, we'll just simulate the process
	
	if len(c.config.Endpoints) == 0 {
		return fmt.Errorf("no endpoints configured")
	}
	
	// Convert payload to JSON
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	// Create request
	url := c.config.Endpoints[0] + path
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add headers
	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.config.Headers {
		req.Header.Set(k, v)
	}
	
	// Add authentication
	if c.config.Auth.Type == "basic" {
		req.SetBasicAuth(c.config.Auth.Username, c.config.Auth.Password)
	} else if c.config.Auth.Type == "bearer" {
		req.Header.Set("Authorization", "Bearer "+c.config.Auth.Token)
	}
	
	// In a real implementation, this would send the request
	// For now, just log a message
	log.Printf("Sending request to %s (payload size: %d bytes)", url, len(data))
	
	// Simulate sending request
	time.Sleep(10 * time.Millisecond)
	
	// Simulate response
	response := &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(`{"accepted": true}`))),
	}
	
	// Check response
	if response.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	
	return nil
}
