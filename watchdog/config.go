package watchdog

import (
	"errors"
	"fmt"
	"time"
)

// DegradationLevel represents a resource usage threshold that triggers a specific degradation response
type DegradationLevel struct {
	// Name is the name of the degradation level
	Name string `yaml:"name"`
	
	// CPUThresholdPercent is the CPU threshold that triggers this degradation level
	CPUThresholdPercent float64 `yaml:"cpu_threshold_percent"`
	
	// MemoryThresholdMB is the memory threshold that triggers this degradation level
	MemoryThresholdMB int `yaml:"memory_threshold_mb"`
	
	// Actions are the degradation actions to take at this level
	Actions []string `yaml:"actions"`
	
	// Description is a human-readable description of this degradation level
	Description string `yaml:"description"`
}

// CircuitBreakerConfig holds configuration for a circuit breaker
type CircuitBreakerConfig struct {
	// Enabled indicates whether the circuit breaker is enabled
	Enabled bool `yaml:"enabled"`
	
	// FailureThreshold is the number of consecutive failures before opening the circuit
	FailureThreshold int `yaml:"failure_threshold"`
	
	// ResetTimeout is the time to wait before attempting to close the circuit
	ResetTimeout time.Duration `yaml:"reset_timeout"`
	
	// HalfOpenSuccessThreshold is the number of consecutive successes in half-open state before closing the circuit
	HalfOpenSuccessThreshold int `yaml:"half_open_success_threshold"`
}

// DeadlockConfig holds configuration for deadlock detection
type DeadlockConfig struct {
	// Enabled indicates whether deadlock detection is enabled
	Enabled bool `yaml:"enabled"`
	
	// HeartbeatInterval is how often components should send heartbeats
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	
	// HeartbeatMissThreshold is how many missed heartbeats before triggering an alert
	HeartbeatMissThreshold int `yaml:"heartbeat_miss_threshold"`
	
	// StackTraceEnabled indicates whether to capture stack traces on suspected deadlocks
	StackTraceEnabled bool `yaml:"stack_trace_enabled"`
	
	// MaxOperationTime is the maximum allowed time for operations
	MaxOperationTime time.Duration `yaml:"max_operation_time"`
}

// RestartConfig holds configuration for component restart behavior
type RestartConfig struct {
	// Enabled indicates whether automatic restart is enabled
	Enabled bool `yaml:"enabled"`
	
	// GracefulShutdownTimeout is the time to wait for graceful shutdown
	GracefulShutdownTimeout time.Duration `yaml:"graceful_shutdown_timeout"`
	
	// MaxRestartAttempts is the maximum number of restart attempts
	MaxRestartAttempts int `yaml:"max_restart_attempts"`
	
	// RestartBackoffInitial is the initial backoff time for restarts
	RestartBackoffInitial time.Duration `yaml:"restart_backoff_initial"`
	
	// RestartBackoffMax is the maximum backoff time for restarts
	RestartBackoffMax time.Duration `yaml:"restart_backoff_max"`
	
	// RestartBackoffFactor is the factor by which backoff increases
	RestartBackoffFactor float64 `yaml:"restart_backoff_factor"`
}

// DiagnosticConfig holds configuration for diagnostic information collection
type DiagnosticConfig struct {
	// DetailLevel is the level of detail for diagnostic information
	DetailLevel string `yaml:"detail_level"`
	
	// MaxEvents is the maximum number of diagnostic events to retain
	MaxEvents int `yaml:"max_events"`
	
	// IncludeStackTraces indicates whether to include stack traces in diagnostics
	IncludeStackTraces bool `yaml:"include_stack_traces"`
	
	// IncludeSystemMetrics indicates whether to include system metrics in diagnostics
	IncludeSystemMetrics bool `yaml:"include_system_metrics"`
}

// ComponentConfig holds configuration for a specific component
type ComponentConfig struct {
	// Enabled indicates whether the component is monitored
	Enabled bool `yaml:"enabled"`
	
	// MaxCPUPercent is the maximum allowed CPU percentage
	MaxCPUPercent float64 `yaml:"max_cpu_percent"`
	
	// MaxMemoryMB is the maximum allowed memory usage in MB
	MaxMemoryMB int `yaml:"max_memory_mb"`
	
	// MaxFileDescriptors is the maximum allowed file descriptors
	MaxFileDescriptors int `yaml:"max_file_descriptors"`
	
	// MaxGoroutines is the maximum allowed goroutines
	MaxGoroutines int `yaml:"max_goroutines"`
	
	// CircuitBreaker contains circuit breaker configuration
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
	
	// DegradationLevels defines progressive degradation thresholds
	DegradationLevels []DegradationLevel `yaml:"degradation_levels"`
}

// Config holds the configuration for the watchdog module
type Config struct {
	// Enabled indicates whether the watchdog is enabled
	Enabled bool `yaml:"enabled"`
	
	// MonitoringInterval is how often to check resource usage
	MonitoringInterval time.Duration `yaml:"monitoring_interval"`
	
	// ComponentConfigs contains per-component configurations
	ComponentConfigs map[string]ComponentConfig `yaml:"components"`
	
	// DeadlockDetection contains deadlock detection configuration
	DeadlockDetection DeadlockConfig `yaml:"deadlock_detection"`
	
	// RestartPolicy contains component restart configuration
	RestartPolicy RestartConfig `yaml:"restart_policy"`
	
	// DiagnosticCollection contains diagnostic collection configuration
	DiagnosticCollection DiagnosticConfig `yaml:"diagnostic_collection"`
}

// DefaultConfig returns a new Config with default values
func DefaultConfig() Config {
	return Config{
		Enabled:            true,
		MonitoringInterval: 15 * time.Second,
		ComponentConfigs: map[string]ComponentConfig{
			"collector": {
				Enabled:           true,
				MaxCPUPercent:     0.75,
				MaxMemoryMB:       100,
				MaxFileDescriptors: 100,
				MaxGoroutines:     50,
				CircuitBreaker: CircuitBreakerConfig{
					Enabled:                 true,
					FailureThreshold:        3,
					ResetTimeout:            30 * time.Second,
					HalfOpenSuccessThreshold: 2,
				},
				DegradationLevels: []DegradationLevel{
					{
						Name:                "warning",
						CPUThresholdPercent: 0.5,
						MemoryThresholdMB:   75,
						Actions:             []string{"reduce_scan_frequency"},
						Description:         "Reduce scan frequency to conserve resources",
					},
					{
						Name:                "critical",
						CPUThresholdPercent: 0.7,
						MemoryThresholdMB:   90,
						Actions:             []string{"reduce_scan_frequency", "filter_events"},
						Description:         "Severely restrict operations to prevent resource exhaustion",
					},
				},
			},
			"sampler": {
				Enabled:           true,
				MaxCPUPercent:     0.5,
				MaxMemoryMB:       50,
				MaxFileDescriptors: 50,
				MaxGoroutines:     25,
				CircuitBreaker: CircuitBreakerConfig{
					Enabled:                 true,
					FailureThreshold:        3,
					ResetTimeout:            30 * time.Second,
					HalfOpenSuccessThreshold: 2,
				},
				DegradationLevels: []DegradationLevel{
					{
						Name:                "warning",
						CPUThresholdPercent: 0.3,
						MemoryThresholdMB:   35,
						Actions:             []string{"reduce_sample_frequency"},
						Description:         "Reduce sampling frequency to conserve resources",
					},
					{
						Name:                "critical",
						CPUThresholdPercent: 0.45,
						MemoryThresholdMB:   45,
						Actions:             []string{"reduce_sample_frequency", "reduce_tracked_processes"},
						Description:         "Severely restrict sampling to prevent resource exhaustion",
					},
				},
			},
			"sketch": {
				Enabled:           true,
				MaxCPUPercent:     0.25,
				MaxMemoryMB:       30,
				MaxFileDescriptors: 20,
				MaxGoroutines:     10,
				CircuitBreaker: CircuitBreakerConfig{
					Enabled:                 true,
					FailureThreshold:        3,
					ResetTimeout:            30 * time.Second,
					HalfOpenSuccessThreshold: 2,
				},
				DegradationLevels: []DegradationLevel{
					{
						Name:                "warning",
						CPUThresholdPercent: 0.15,
						MemoryThresholdMB:   20,
						Actions:             []string{"switch_to_dense_store"},
						Description:         "Switch to dense store to reduce memory allocations",
					},
					{
						Name:                "critical",
						CPUThresholdPercent: 0.2,
						MemoryThresholdMB:   25,
						Actions:             []string{"switch_to_dense_store", "reduce_accuracy"},
						Description:         "Reduce statistical accuracy to conserve resources",
					},
				},
			},
			"export": {
				Enabled:           true,
				MaxCPUPercent:     0.5,
				MaxMemoryMB:       50,
				MaxFileDescriptors: 100,
				MaxGoroutines:     25,
				CircuitBreaker: CircuitBreakerConfig{
					Enabled:                 true,
					FailureThreshold:        3,
					ResetTimeout:            30 * time.Second,
					HalfOpenSuccessThreshold: 2,
				},
				DegradationLevels: []DegradationLevel{
					{
						Name:                "warning",
						CPUThresholdPercent: 0.3,
						MemoryThresholdMB:   35,
						Actions:             []string{"increase_batch_size"},
						Description:         "Increase batch size to reduce overhead",
					},
					{
						Name:                "critical",
						CPUThresholdPercent: 0.45,
						MemoryThresholdMB:   45,
						Actions:             []string{"increase_batch_size", "reduce_export_frequency"},
						Description:         "Reduce export frequency to conserve resources",
					},
				},
			},
		},
		DeadlockDetection: DeadlockConfig{
			Enabled:               true,
			HeartbeatInterval:     5 * time.Second,
			HeartbeatMissThreshold: 3,
			StackTraceEnabled:     true,
			MaxOperationTime:      30 * time.Second,
		},
		RestartPolicy: RestartConfig{
			Enabled:                true,
			GracefulShutdownTimeout: 5 * time.Second,
			MaxRestartAttempts:     5,
			RestartBackoffInitial:  1 * time.Second,
			RestartBackoffMax:      60 * time.Second,
			RestartBackoffFactor:   2.0,
		},
		DiagnosticCollection: DiagnosticConfig{
			DetailLevel:         "normal",
			MaxEvents:           100,
			IncludeStackTraces:  true,
			IncludeSystemMetrics: true,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	
	if c.MonitoringInterval <= 0 {
		return errors.New("monitoring interval must be positive")
	}
	
	if len(c.ComponentConfigs) == 0 {
		return errors.New("at least one component configuration must be specified")
	}
	
	for name, config := range c.ComponentConfigs {
		if config.MaxCPUPercent <= 0 || config.MaxCPUPercent > 100 {
			return fmt.Errorf("invalid max CPU percentage for component %s: %f", name, config.MaxCPUPercent)
		}
		
		if config.MaxMemoryMB <= 0 {
			return fmt.Errorf("invalid max memory MB for component %s: %d", name, config.MaxMemoryMB)
		}
		
		if config.CircuitBreaker.Enabled {
			if config.CircuitBreaker.FailureThreshold <= 0 {
				return fmt.Errorf("invalid failure threshold for component %s: %d", name, config.CircuitBreaker.FailureThreshold)
			}
			
			if config.CircuitBreaker.ResetTimeout <= 0 {
				return fmt.Errorf("invalid reset timeout for component %s: %v", name, config.CircuitBreaker.ResetTimeout)
			}
			
			if config.CircuitBreaker.HalfOpenSuccessThreshold <= 0 {
				return fmt.Errorf("invalid half-open success threshold for component %s: %d", name, config.CircuitBreaker.HalfOpenSuccessThreshold)
			}
		}
		
		for i, level := range config.DegradationLevels {
			if level.CPUThresholdPercent <= 0 || level.CPUThresholdPercent > 100 {
				return fmt.Errorf("invalid CPU threshold for degradation level %d of component %s: %f", i, name, level.CPUThresholdPercent)
			}
			
			if level.MemoryThresholdMB <= 0 {
				return fmt.Errorf("invalid memory threshold for degradation level %d of component %s: %d", i, name, level.MemoryThresholdMB)
			}
			
			if len(level.Actions) == 0 {
				return fmt.Errorf("no actions specified for degradation level %d of component %s", i, name)
			}
		}
	}
	
	if c.DeadlockDetection.Enabled {
		if c.DeadlockDetection.HeartbeatInterval <= 0 {
			return errors.New("heartbeat interval must be positive")
		}
		
		if c.DeadlockDetection.HeartbeatMissThreshold <= 0 {
			return errors.New("heartbeat miss threshold must be positive")
		}
		
		if c.DeadlockDetection.MaxOperationTime <= 0 {
			return errors.New("max operation time must be positive")
		}
	}
	
	if c.RestartPolicy.Enabled {
		if c.RestartPolicy.GracefulShutdownTimeout <= 0 {
			return errors.New("graceful shutdown timeout must be positive")
		}
		
		if c.RestartPolicy.MaxRestartAttempts <= 0 {
			return errors.New("max restart attempts must be positive")
		}
		
		if c.RestartPolicy.RestartBackoffInitial <= 0 {
			return errors.New("restart backoff initial must be positive")
		}
		
		if c.RestartPolicy.RestartBackoffMax <= 0 {
			return errors.New("restart backoff max must be positive")
		}
		
		if c.RestartPolicy.RestartBackoffFactor <= 1.0 {
			return errors.New("restart backoff factor must be greater than 1.0")
		}
	}
	
	if c.DiagnosticCollection.MaxEvents <= 0 {
		return errors.New("max events must be positive")
	}
	
	return nil
}
