package export

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"time"
)

// GRPCClient is a client for the OTLP gRPC service
type GRPCClient struct {
	config Config
	// In a real implementation, this would contain gRPC client objects
}

// NewGRPCClient creates a new gRPC client with the given configuration
func NewGRPCClient(config Config) (*GRPCClient, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Check protocol
	if config.Protocol != ProtocolGRPC {
		return nil, fmt.Errorf("expected gRPC protocol, got %s", config.Protocol)
	}
	
	return &GRPCClient{
		config: config,
	}, nil
}

// SendMetrics sends metrics to the OTLP service
func (c *GRPCClient) SendMetrics(ctx context.Context, metrics []map[string]interface{}, resource map[string]interface{}) error {
	// In a real implementation, this would convert the metrics to protobuf and send them using gRPC
	// For now, we'll just simulate the process
	
	log.Printf("Sending %d metrics to %s", len(metrics), c.config.Endpoints[0])
	
	// Simulate network delay
	time.Sleep(10 * time.Millisecond)
	
	return nil
}

// SendLogs sends logs to the OTLP service
func (c *GRPCClient) SendLogs(ctx context.Context, logs []map[string]interface{}, resource map[string]interface{}) error {
	// In a real implementation, this would convert the logs to protobuf and send them using gRPC
	// For now, we'll just simulate the process
	
	log.Printf("Sending %d logs to %s", len(logs), c.config.Endpoints[0])
	
	// Simulate network delay
	time.Sleep(10 * time.Millisecond)
	
	return nil
}

// SendTraces sends traces to the OTLP service
func (c *GRPCClient) SendTraces(ctx context.Context, spans []map[string]interface{}, resource map[string]interface{}) error {
	// In a real implementation, this would convert the traces to protobuf and send them using gRPC
	// For now, we'll just simulate the process
	
	log.Printf("Sending %d spans to %s", len(spans), c.config.Endpoints[0])
	
	// Simulate network delay
	time.Sleep(10 * time.Millisecond)
	
	return nil
}

// Shutdown shuts down the gRPC client
func (c *GRPCClient) Shutdown(ctx context.Context) error {
	// In a real implementation, this would close the gRPC connection
	// For now, we'll just simulate the process
	
	log.Printf("Shutting down gRPC client")
	
	return nil
}

// createTLSConfig creates a TLS configuration from the exporter TLS configuration
func createTLSConfig(config TLSConfig) (*tls.Config, error) {
	if !config.Enabled {
		return nil, nil
	}
	
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify,
	}
	
	// Load client certificate if provided
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	
	// Load CA certificate if provided
	if config.CAFile != "" {
		caCert, err := ioutil.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to add CA certificate to pool")
		}
		
		tlsConfig.RootCAs = caPool
	}
	
	return tlsConfig, nil
}
