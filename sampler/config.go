package sampler

import (
	"fmt"
	"time"
)

// Config holds configuration parameters for samplers
type Config struct {
	// SamplerType specifies which sampler implementation to use
	SamplerType string `yaml:"samplerType"`

	// SampleInterval specifies how often to sample processes
	SampleInterval time.Duration `yaml:"sampleInterval"`

	// MaxSamplerCPU is the maximum allowed CPU percentage for the sampler
	MaxSamplerCPU float64 `yaml:"maxSamplerCPU"`

	// TopN specific configuration
	TopN TopNConfig `yaml:"topN"`
}

// TopNConfig holds configuration for the Top-N sampler
type TopNConfig struct {
	// MaxProcesses is the maximum number of processes to track
	MaxProcesses int `yaml:"maxProcesses"`

	// CPUWeight is the weight given to CPU usage in scoring
	CPUWeight float64 `yaml:"cpuWeight"`

	// RSSWeight is the weight given to memory usage in scoring
	RSSWeight float64 `yaml:"rssWeight"`

	// MinScore is the minimum score a process must have to be tracked
	MinScore float64 `yaml:"minScore"`

	// StabilityFactor affects how quickly scores change (0-1)
	StabilityFactor float64 `yaml:"stabilityFactor"`

	// ChurnHandlingEnabled enables optimizations for high PID churn
	ChurnHandlingEnabled bool `yaml:"churnHandlingEnabled"`

	// ChurnThreshold is the PID churn rate that activates optimizations
	ChurnThreshold int `yaml:"churnThreshold"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		SamplerType:     "topn",
		SampleInterval:  time.Second * 15,
		MaxSamplerCPU:   0.5, // 0.5% maximum CPU usage
		TopN: TopNConfig{
			MaxProcesses:        500,
			CPUWeight:           0.7,
			RSSWeight:           0.3,
			MinScore:            0.001,
			StabilityFactor:     0.8,
			ChurnHandlingEnabled: true,
			ChurnThreshold:      2000, // 2000 PIDs/s
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.SamplerType == "" {
		return fmt.Errorf("sampler type cannot be empty")
	}

	if c.SampleInterval < time.Second {
		return fmt.Errorf("sample interval cannot be less than 1 second")
	}

	if c.MaxSamplerCPU <= 0 || c.MaxSamplerCPU > 5 {
		return fmt.Errorf("max sampler CPU must be between 0 and 5 percent")
	}

	// Validate TopN config
	if c.TopN.MaxProcesses <= 0 {
		return fmt.Errorf("max processes must be positive")
	}

	if c.TopN.CPUWeight < 0 || c.TopN.RSSWeight < 0 {
		return fmt.Errorf("weights cannot be negative")
	}

	if c.TopN.CPUWeight+c.TopN.RSSWeight == 0 {
		return fmt.Errorf("at least one weight must be positive")
	}

	if c.TopN.StabilityFactor < 0 || c.TopN.StabilityFactor > 1 {
		return fmt.Errorf("stability factor must be between 0 and 1")
	}

	return nil
}
