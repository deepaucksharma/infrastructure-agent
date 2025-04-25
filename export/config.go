package export

import (
	"errors"
	"time"
)

// ProtocolType defines the supported OTLP export protocols
type ProtocolType string

const (
	// ProtocolGRPC indicates the gRPC protocol for OTLP
	ProtocolGRPC ProtocolType = "grpc"
	
	// ProtocolHTTP indicates the HTTP protocol for OTLP
	ProtocolHTTP ProtocolType = "http"
)

// TLSConfig holds the TLS configuration for the exporter
type TLSConfig struct {
	// Enabled indicates whether TLS is enabled
	Enabled bool `yaml:"enabled"`
	
	// CertFile is the path to the client certificate file
	CertFile string `yaml:"cert_file"`
	
	// KeyFile is the path to the client key file
	KeyFile string `yaml:"key_file"`
	
	// CAFile is the path to the CA certificate file
	CAFile string `yaml:"ca_file"`
	
	// InsecureSkipVerify indicates whether to skip server certificate verification
	InsecureSkipVerify bool `yaml:"insecure_skip_verify"`
}

// AuthConfig holds the authentication configuration for the exporter
type AuthConfig struct {
	// Type is the authentication type (none, basic, bearer, etc.)
	Type string `yaml:"type"`
	
	// Username is used for basic authentication
	Username string `yaml:"username"`
	
	// Password is used for basic authentication
	Password string `yaml:"password"`
	
	// Token is used for bearer token authentication
	Token string `yaml:"token"`
}

// BatchConfig holds the batching configuration for the exporter
type BatchConfig struct {
	// Size is the maximum number of items in a batch
	Size int `yaml:"size"`
	
	// Timeout is the maximum time to wait before sending a batch
	Timeout time.Duration `yaml:"timeout"`
}

// RetryConfig holds the retry configuration for the exporter
type RetryConfig struct {
	// Enabled indicates whether retries are enabled
	Enabled bool `yaml:"enabled"`
	
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int `yaml:"max_attempts"`
	
	// InitialInterval is the initial retry interval
	InitialInterval time.Duration `yaml:"initial_interval"`
	
	// MaxInterval is the maximum retry interval
	MaxInterval time.Duration `yaml:"max_interval"`
	
	// Multiplier is the factor by which the retry interval increases
	Multiplier float64 `yaml:"multiplier"`
}

// CircuitConfig holds the circuit breaker configuration for the exporter
type CircuitConfig struct {
	// Enabled indicates whether the circuit breaker is enabled
	Enabled bool `yaml:"enabled"`
	
	// FailureThreshold is the number of consecutive failures before opening the circuit
	FailureThreshold int `yaml:"failure_threshold"`
	
	// ResetTimeout is the time to wait before attempting to close the circuit
	ResetTimeout time.Duration `yaml:"reset_timeout"`
	
	// HalfOpenSuccessThreshold is the number of consecutive successes in half-open state before closing the circuit
	HalfOpenSuccessThreshold int `yaml:"half_open_success_threshold"`
}

// Config holds the configuration for the OTLP exporter
type Config struct {
	// Enabled indicates whether the OTLP exporter is enabled
	Enabled bool `yaml:"enabled"`
	
	// Endpoints are the OTLP endpoints to export to
	Endpoints []string `yaml:"endpoints"`
	
	// Protocol is the OTLP protocol to use (grpc or http)
	Protocol ProtocolType `yaml:"protocol"`
	
	// Headers are custom headers to include in HTTP requests
	Headers map[string]string `yaml:"headers"`
	
	// Compression is the compression type to use (none, gzip)
	Compression string `yaml:"compression"`
	
	// TLS holds the TLS configuration
	TLS TLSConfig `yaml:"tls"`
	
	// Auth holds the authentication configuration
	Auth AuthConfig `yaml:"auth"`
	
	// Batch holds the batching configuration
	Batch BatchConfig `yaml:"batch"`
	
	// Retry holds the retry configuration
	Retry RetryConfig `yaml:"retry"`
	
	// CircuitBreaker holds the circuit breaker configuration
	CircuitBreaker CircuitConfig `yaml:"circuit_breaker"`
}

// DefaultConfig returns a new Config with default values.
func DefaultConfig() Config {
	return Config{
		Enabled:    true,
		Endpoints:  []string{"localhost:4317"},
		Protocol:   ProtocolGRPC,
		Headers:    map[string]string{},
		Compression: "none",
		TLS: TLSConfig{
			Enabled:            false,
			InsecureSkipVerify: false,
		},
		Auth: AuthConfig{
			Type: "none",
		},
		Batch: BatchConfig{
			Size:    1000,
			Timeout: 1 * time.Second,
		},
		Retry: RetryConfig{
			Enabled:         true,
			MaxAttempts:     5,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      1.5,
		},
		CircuitBreaker: CircuitConfig{
			Enabled:                 true,
			FailureThreshold:        5,
			ResetTimeout:            60 * time.Second,
			HalfOpenSuccessThreshold: 2,
		},
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	
	if len(c.Endpoints) == 0 {
		return errors.New("at least one endpoint must be specified")
	}
	
	if c.Protocol != ProtocolGRPC && c.Protocol != ProtocolHTTP {
		return errors.New("protocol must be either 'grpc' or 'http'")
	}
	
	if c.Batch.Size <= 0 {
		return errors.New("batch size must be positive")
	}
	
	if c.Batch.Timeout <= 0 {
		return errors.New("batch timeout must be positive")
	}
	
	if c.Retry.Enabled {
		if c.Retry.MaxAttempts <= 0 {
			return errors.New("max retry attempts must be positive")
		}
		
		if c.Retry.InitialInterval <= 0 {
			return errors.New("initial retry interval must be positive")
		}
		
		if c.Retry.MaxInterval <= 0 {
			return errors.New("max retry interval must be positive")
		}
		
		if c.Retry.Multiplier <= 1.0 {
			return errors.New("retry multiplier must be greater than 1.0")
		}
	}
	
	if c.CircuitBreaker.Enabled {
		if c.CircuitBreaker.FailureThreshold <= 0 {
			return errors.New("circuit breaker failure threshold must be positive")
		}
		
		if c.CircuitBreaker.ResetTimeout <= 0 {
			return errors.New("circuit breaker reset timeout must be positive")
		}
		
		if c.CircuitBreaker.HalfOpenSuccessThreshold <= 0 {
			return errors.New("circuit breaker half-open success threshold must be positive")
		}
	}
	
	return nil
}
