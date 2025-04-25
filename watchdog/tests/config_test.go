package tests

import (
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/watchdog"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := watchdog.DefaultConfig()
	
	// Verify default config is enabled
	assert.True(t, config.Enabled)
	
	// Verify default monitoring interval
	assert.Equal(t, 15*time.Second, config.MonitoringInterval)
	
	// Verify component configs exist
	assert.NotEmpty(t, config.ComponentConfigs)
	
	// Check some specific component configs
	collector, exists := config.ComponentConfigs["collector"]
	assert.True(t, exists)
	assert.True(t, collector.Enabled)
	assert.Equal(t, 0.75, collector.MaxCPUPercent)
	assert.Equal(t, 100, collector.MaxMemoryMB)
	
	sampler, exists := config.ComponentConfigs["sampler"]
	assert.True(t, exists)
	assert.True(t, sampler.Enabled)
	assert.Equal(t, 0.5, sampler.MaxCPUPercent)
	assert.Equal(t, 50, sampler.MaxMemoryMB)
	
	// Verify deadlock detection config
	assert.True(t, config.DeadlockDetection.Enabled)
	assert.Equal(t, 5*time.Second, config.DeadlockDetection.HeartbeatInterval)
	assert.Equal(t, 3, config.DeadlockDetection.HeartbeatMissThreshold)
	
	// Verify restart policy config
	assert.True(t, config.RestartPolicy.Enabled)
	assert.Equal(t, 5*time.Second, config.RestartPolicy.GracefulShutdownTimeout)
	assert.Equal(t, 5, config.RestartPolicy.MaxRestartAttempts)
	
	// Verify diagnostic collection config
	assert.Equal(t, "normal", config.DiagnosticCollection.DetailLevel)
	assert.Equal(t, 100, config.DiagnosticCollection.MaxEvents)
	assert.True(t, config.DiagnosticCollection.IncludeStackTraces)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		modifyConfig func(*watchdog.Config)
		shouldFail  bool
	}{
		{
			name: "valid default config",
			modifyConfig: func(c *watchdog.Config) {},
			shouldFail: false,
		},
		{
			name: "disabled config is valid",
			modifyConfig: func(c *watchdog.Config) {
				c.Enabled = false
			},
			shouldFail: false,
		},
		{
			name: "invalid monitoring interval",
			modifyConfig: func(c *watchdog.Config) {
				c.MonitoringInterval = -1 * time.Second
			},
			shouldFail: true,
		},
		{
			name: "no component configs",
			modifyConfig: func(c *watchdog.Config) {
				c.ComponentConfigs = map[string]watchdog.ComponentConfig{}
			},
			shouldFail: true,
		},
		{
			name: "invalid CPU percentage",
			modifyConfig: func(c *watchdog.Config) {
				config := c.ComponentConfigs["collector"]
				config.MaxCPUPercent = 101
				c.ComponentConfigs["collector"] = config
			},
			shouldFail: true,
		},
		{
			name: "zero memory limit",
			modifyConfig: func(c *watchdog.Config) {
				config := c.ComponentConfigs["collector"]
				config.MaxMemoryMB = 0
				c.ComponentConfigs["collector"] = config
			},
			shouldFail: true,
		},
		{
			name: "invalid circuit breaker failure threshold",
			modifyConfig: func(c *watchdog.Config) {
				config := c.ComponentConfigs["collector"]
				config.CircuitBreaker.FailureThreshold = 0
				c.ComponentConfigs["collector"] = config
			},
			shouldFail: true,
		},
		{
			name: "invalid circuit breaker reset timeout",
			modifyConfig: func(c *watchdog.Config) {
				config := c.ComponentConfigs["collector"]
				config.CircuitBreaker.ResetTimeout = -1 * time.Second
				c.ComponentConfigs["collector"] = config
			},
			shouldFail: true,
		},
		{
			name: "invalid heartbeat interval",
			modifyConfig: func(c *watchdog.Config) {
				c.DeadlockDetection.HeartbeatInterval = 0
			},
			shouldFail: true,
		},
		{
			name: "invalid heartbeat miss threshold",
			modifyConfig: func(c *watchdog.Config) {
				c.DeadlockDetection.HeartbeatMissThreshold = 0
			},
			shouldFail: true,
		},
		{
			name: "invalid max operation time",
			modifyConfig: func(c *watchdog.Config) {
				c.DeadlockDetection.MaxOperationTime = -1 * time.Second
			},
			shouldFail: true,
		},
		{
			name: "invalid graceful shutdown timeout",
			modifyConfig: func(c *watchdog.Config) {
				c.RestartPolicy.GracefulShutdownTimeout = 0
			},
			shouldFail: true,
		},
		{
			name: "invalid max restart attempts",
			modifyConfig: func(c *watchdog.Config) {
				c.RestartPolicy.MaxRestartAttempts = 0
			},
			shouldFail: true,
		},
		{
			name: "invalid restart backoff factor",
			modifyConfig: func(c *watchdog.Config) {
				c.RestartPolicy.RestartBackoffFactor = 0.5
			},
			shouldFail: true,
		},
		{
			name: "invalid max events",
			modifyConfig: func(c *watchdog.Config) {
				c.DiagnosticCollection.MaxEvents = 0
			},
			shouldFail: true,
		},
		{
			name: "valid custom config",
			modifyConfig: func(c *watchdog.Config) {
				c.MonitoringInterval = 30 * time.Second
				config := c.ComponentConfigs["collector"]
				config.MaxCPUPercent = 50
				config.MaxMemoryMB = 200
				c.ComponentConfigs["collector"] = config
			},
			shouldFail: false,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := watchdog.DefaultConfig()
			test.modifyConfig(&config)
			
			err := config.Validate()
			
			if test.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDegradationLevelValidation(t *testing.T) {
	config := watchdog.DefaultConfig()
	
	// Get a component config with degradation levels
	componentConfig := config.ComponentConfigs["collector"]
	
	// Validate a valid degradation level
	validLevel := componentConfig.DegradationLevels[0]
	assert.Equal(t, "warning", validLevel.Name)
	assert.True(t, validLevel.CPUThresholdPercent > 0)
	assert.True(t, validLevel.CPUThresholdPercent <= 100)
	assert.True(t, validLevel.MemoryThresholdMB > 0)
	assert.NotEmpty(t, validLevel.Actions)
	
	// Create an invalid degradation level
	componentConfig.DegradationLevels = append(componentConfig.DegradationLevels, watchdog.DegradationLevel{
		Name:                "invalid",
		CPUThresholdPercent: -10, // Invalid
		MemoryThresholdMB:   0,   // Invalid
		Actions:             []string{}, // Invalid
	})
	
	config.ComponentConfigs["collector"] = componentConfig
	
	// Validation should fail
	err := config.Validate()
	assert.Error(t, err)
}

func TestRestartPolicyValidation(t *testing.T) {
	config := watchdog.DefaultConfig()
	
	// Test valid restart policy
	validPolicy := config.RestartPolicy
	assert.True(t, validPolicy.Enabled)
	assert.True(t, validPolicy.GracefulShutdownTimeout > 0)
	assert.True(t, validPolicy.MaxRestartAttempts > 0)
	assert.True(t, validPolicy.RestartBackoffInitial > 0)
	assert.True(t, validPolicy.RestartBackoffMax > 0)
	assert.True(t, validPolicy.RestartBackoffFactor > 1.0)
	
	// Test invalid restart policy
	config.RestartPolicy.RestartBackoffFactor = 0.5 // Invalid
	err := config.Validate()
	assert.Error(t, err)
	
	// Disabled restart policy should bypass validation
	config.RestartPolicy.Enabled = false
	err = config.Validate()
	assert.NoError(t, err)
}
