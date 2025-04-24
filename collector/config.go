package collector

import (
	"fmt"
	"time"
)

// Config holds configuration parameters for collectors
type Config struct {
	// CollectorType specifies which collector implementation to use
	CollectorType string `yaml:"collectorType"`
	
	// CollectionInterval specifies how often to collect data
	CollectionInterval time.Duration `yaml:"collectionInterval"`
	
	// MaxCPUUsage is the maximum allowed CPU percentage for the collector
	MaxCPUUsage float64 `yaml:"maxCPUUsage"`
	
	// ProcessScanner specific configuration
	ProcessScanner ProcessScannerConfig `yaml:"processScanner"`
}

// ProcessScannerConfig holds configuration for the process scanner
type ProcessScannerConfig struct {
	// Enabled determines whether process scanning is enabled
	Enabled bool `yaml:"enabled"`
	
	// ScanInterval specifies how often to scan for processes
	ScanInterval time.Duration `yaml:"scanInterval"`
	
	// MaxProcesses is the maximum number of processes to track
	MaxProcesses int `yaml:"maxProcesses"`
	
	// ExcludePatterns are regex patterns for processes to exclude
	ExcludePatterns []string `yaml:"excludePatterns"`
	
	// IncludePatterns are regex patterns for processes to include
	IncludePatterns []string `yaml:"includePatterns"`
	
	// ProcFSPath is the path to procfs (Linux only)
	ProcFSPath string `yaml:"procFSPath"`
	
	// RefreshCPUStats determines whether to refresh CPU stats
	RefreshCPUStats bool `yaml:"refreshCPUStats"`
	
	// EventBatchSize is the maximum number of events to send in one batch
	EventBatchSize int `yaml:"eventBatchSize"`
	
	// EventChannelSize is the size of the event channel buffer
	EventChannelSize int `yaml:"eventChannelSize"`
	
	// RetryInterval is the time to wait before retrying after a failure
	RetryInterval time.Duration `yaml:"retryInterval"`
	
	// AdaptiveSampling enables adaptive sampling based on system load
	AdaptiveSampling bool `yaml:"adaptiveSampling"`
	
	// MaxScanTime is the maximum time allowed for a full scan
	MaxScanTime time.Duration `yaml:"maxScanTime"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		CollectorType:     "process_scanner",
		CollectionInterval: time.Second * 15,
		MaxCPUUsage:       0.75, // 0.75% maximum CPU usage
		ProcessScanner: ProcessScannerConfig{
			Enabled:         true,
			ScanInterval:    time.Second * 10,
			MaxProcesses:    3000,
			ExcludePatterns: []string{},
			IncludePatterns: []string{},
			ProcFSPath:      "/proc",
			RefreshCPUStats: true,
			EventBatchSize:  100,
			EventChannelSize: 1000,
			RetryInterval:   time.Second * 5,
			AdaptiveSampling: true,
			MaxScanTime:     time.Millisecond * 200,
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.CollectorType == "" {
		return fmt.Errorf("collector type cannot be empty")
	}
	
	if c.CollectionInterval < time.Second {
		return fmt.Errorf("collection interval cannot be less than 1 second")
	}
	
	if c.MaxCPUUsage <= 0 || c.MaxCPUUsage > 5 {
		return fmt.Errorf("max CPU usage must be between 0 and 5 percent")
	}
	
	// Validate process scanner config
	if c.ProcessScanner.Enabled {
		if c.ProcessScanner.ScanInterval < time.Second {
			return fmt.Errorf("scan interval cannot be less than 1 second")
		}
		
		if c.ProcessScanner.MaxProcesses <= 0 {
			return fmt.Errorf("max processes must be positive")
		}
		
		if c.ProcessScanner.EventBatchSize <= 0 {
			return fmt.Errorf("event batch size must be positive")
		}
		
		if c.ProcessScanner.EventChannelSize <= 0 {
			return fmt.Errorf("event channel size must be positive")
		}
		
		if c.ProcessScanner.RetryInterval < time.Second {
			return fmt.Errorf("retry interval cannot be less than 1 second")
		}
		
		if c.ProcessScanner.MaxScanTime < time.Millisecond*10 {
			return fmt.Errorf("max scan time cannot be less than 10 milliseconds")
		}
	}
	
	return nil
}
