package sketch

import (
	"fmt"
	"math"
)

// Config holds configuration parameters for sketches
type Config struct {
	// SketchType specifies which sketch implementation to use
	SketchType string `yaml:"sketchType"`
	
	// DDSketch specific configuration
	DDSketch DDSketchConfig `yaml:"ddSketch"`
}

// DDSketchConfig holds configuration for the DDSketch
type DDSketchConfig struct {
	// RelativeAccuracy is the gamma parameter (γ) controlling accuracy
	// The relative error is guaranteed to be less than or equal to gamma
	RelativeAccuracy float64 `yaml:"relativeAccuracy"`
	
	// MinValue is the minimum value that can be stored in the sketch
	MinValue float64 `yaml:"minValue"`
	
	// MaxValue is the maximum value that can be stored in the sketch
	MaxValue float64 `yaml:"maxValue"`
	
	// InitialCapacity is the initial capacity for stores
	InitialCapacity int `yaml:"initialCapacity"`
	
	// UseSparseStore determines whether to use a sparse store
	UseSparseStore bool `yaml:"useSparseStore"`
	
	// CollapseThreshold is the threshold for collapsing sparse buckets
	CollapseThreshold uint64 `yaml:"collapseThreshold"`
	
	// AutoSwitch enables automatic switching between sparse and dense stores
	AutoSwitch bool `yaml:"autoSwitch"`
	
	// SwitchThreshold is the density threshold for switching to dense store
	SwitchThreshold float64 `yaml:"switchThreshold"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		SketchType: "ddsketch",
		DDSketch: DDSketchConfig{
			RelativeAccuracy:  0.0075,          // Ensures p95/p99 error ≤ 1%
			MinValue:          1e-9,            // Near-zero positive minimum
			MaxValue:          1e9,             // Large maximum
			InitialCapacity:   128,             // Reasonable initial size
			UseSparseStore:    true,            // Sparse by default for memory efficiency
			CollapseThreshold: 10,              // Collapse buckets with <= 10 counts
			AutoSwitch:        true,            // Enable automatic switching
			SwitchThreshold:   0.5,             // Switch to dense when 50% of buckets are used
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.SketchType == "" {
		return fmt.Errorf("sketch type cannot be empty")
	}
	
	// Validate DDSketch config
	if c.SketchType == "ddsketch" {
		// RelativeAccuracy must be between 0 and 1
		if c.DDSketch.RelativeAccuracy <= 0 || c.DDSketch.RelativeAccuracy >= 1 {
			return fmt.Errorf("relative accuracy must be between 0 and 1")
		}
		
		// MinValue must be positive
		if c.DDSketch.MinValue <= 0 {
			return fmt.Errorf("min value must be positive")
		}
		
		// MaxValue must be greater than MinValue
		if c.DDSketch.MaxValue <= c.DDSketch.MinValue {
			return fmt.Errorf("max value must be greater than min value")
		}
		
		// InitialCapacity must be positive
		if c.DDSketch.InitialCapacity <= 0 {
			return fmt.Errorf("initial capacity must be positive")
		}
		
		// CollapseThreshold must be positive
		if c.DDSketch.CollapseThreshold == 0 {
			return fmt.Errorf("collapse threshold must be positive")
		}
		
		// SwitchThreshold must be between 0 and 1
		if c.DDSketch.AutoSwitch && (c.DDSketch.SwitchThreshold <= 0 || c.DDSketch.SwitchThreshold >= 1) {
			return fmt.Errorf("switch threshold must be between 0 and 1")
		}
	}
	
	return nil
}

// CalculateExpectedError calculates the expected error at the given quantile
// based on the relative accuracy parameter
func (c *DDSketchConfig) CalculateExpectedError(quantile float64) float64 {
	if quantile <= 0 || quantile >= 1 {
		return math.NaN()
	}
	
	// Error formula depends on whether we're looking at upper or lower quantiles
	if quantile >= 0.5 {
		// For upper quantiles (p50, p95, p99, etc.), error is directly related to gamma
		return c.RelativeAccuracy
	} else {
		// For lower quantiles, error is higher due to how DDSketch works
		// The formula is: gamma / (1 - quantile)
		return c.RelativeAccuracy / (1 - quantile)
	}
}

// LogarithmicMapping calculates the mapping parameters used by DDSketch
// based on the relative accuracy parameter
func (c *DDSketchConfig) LogarithmicMapping() (gamma, multiplier, offset float64) {
	gamma = c.RelativeAccuracy
	
	// Calculate logarithmic mapping parameters
	multiplier = 1.0 / math.Log1p(gamma)
	offset = 0
	
	return gamma, multiplier, offset
}
