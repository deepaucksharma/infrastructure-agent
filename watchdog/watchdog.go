package watchdog

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// HealthStatus represents the health of a component
type HealthStatus string

const (
	// HealthOK indicates the component is healthy
	HealthOK HealthStatus = "ok"
	
	// HealthDegraded indicates the component has degraded functionality
	HealthDegraded HealthStatus = "degraded"
	
	// HealthCritical indicates the component is in a critical state
	HealthCritical HealthStatus = "critical"
	
	// HealthUnknown indicates the component's health is unknown
	HealthUnknown HealthStatus = "unknown"
)

// CircuitState represents the state of a circuit breaker
type CircuitState string

const (
	// CircuitClosed indicates normal operation
	CircuitClosed CircuitState = "closed"
	
	// CircuitOpen indicates the component is disabled
	CircuitOpen CircuitState = "open"
	
	// CircuitHalfOpen indicates the component is being tested
	CircuitHalfOpen CircuitState = "half-open"
)

// ResourceUsage captures resource usage metrics for a component
type ResourceUsage struct {
	// CPUPercent is the CPU usage percentage
	CPUPercent float64
	
	// MemoryMB is the memory usage in MB
	MemoryMB float64
	
	// Goroutines is the number of goroutines
	Goroutines int
	
	// FileHandles is the number of open file handles
	FileHandles int
	
	// GCPercent is the percentage of time spent in GC
	GCPercent float64
	
	// Timestamp is when the measurement was taken
	Timestamp time.Time
	
	// Measurements are historical measurements
	Measurements []TimestampedMeasurement
}

// TimestampedMeasurement is a measurement with a timestamp
type TimestampedMeasurement struct {
	// Timestamp is when the measurement was taken
	Timestamp time.Time
	
	// CPUPercent is the CPU usage percentage
	CPUPercent float64
	
	// MemoryMB is the memory usage in MB
	MemoryMB float64
	
	// Goroutines is the number of goroutines
	Goroutines int
	
	// FileHandles is the number of open file handles
	FileHandles int
	
	// GCPercent is the percentage of time spent in GC
	GCPercent float64
}

// IncidentType represents the type of incident
type IncidentType string

const (
	// IncidentResourceExceeded indicates a resource threshold was exceeded
	IncidentResourceExceeded IncidentType = "resource_exceeded"
	
	// IncidentDeadlockDetected indicates a deadlock was detected
	IncidentDeadlockDetected IncidentType = "deadlock_detected"
	
	// IncidentRestartFailed indicates a component restart failed
	IncidentRestartFailed IncidentType = "restart_failed"
	
	// IncidentCrash indicates a component crashed
	IncidentCrash IncidentType = "crash"
)

// Incident represents a detected problem
type Incident struct {
	// ID is a unique identifier for the incident
	ID string
	
	// Timestamp is when the incident occurred
	Timestamp time.Time
	
	// Type is the type of incident
	Type IncidentType
	
	// Description is a human-readable description of the incident
	Description string
	
	// ResourceUsage is the resource usage at the time of the incident
	ResourceUsage ResourceUsage
	
	// Remediation is a suggested remediation action
	Remediation string
}

// ComponentStatus represents the status of a monitored component
type ComponentStatus struct {
	// Name is the name of the component
	Name string
	
	// Health is the health status of the component
	Health HealthStatus
	
	// CircuitState is the state of the circuit breaker
	CircuitState CircuitState
	
	// ResourceUsage is the current resource usage
	ResourceUsage ResourceUsage
	
	// LastRestart is when the component was last restarted
	LastRestart time.Time
	
	// RestartCount is the number of times the component has been restarted
	RestartCount int
	
	// Incidents are recent incidents for the component
	Incidents []Incident
	
	// DegradationLevel is the current degradation level (0 = none)
	DegradationLevel int
}

// Monitorable defines the interface for components that can be monitored
type Monitorable interface {
	// GetResourceUsage returns the resource usage for the component
	GetResourceUsage() ResourceUsage
	
	// GetHealth returns the health status of the component
	GetHealth() HealthStatus
}

// Restartable defines the interface for components that can be restarted
type Restartable interface {
	// Shutdown performs a graceful shutdown of the component
	Shutdown(ctx context.Context) error
	
	// Start starts the component
	Start(ctx context.Context) error
	
	// IsRunning returns whether the component is running
	IsRunning() bool
}

// Degradable defines the interface for components that support degradation
type Degradable interface {
	// SetDegradationLevel sets the degradation level for the component
	SetDegradationLevel(level int) error
	
	// GetDegradationLevel returns the current degradation level
	GetDegradationLevel() int
}

// Watchdog defines the interface for the watchdog component
type Watchdog interface {
	// Start starts the watchdog monitoring
	Start() error
	
	// Stop stops the watchdog monitoring
	Stop() error
	
	// RegisterComponent registers a component for monitoring
	RegisterComponent(name string, component interface{}) error
	
	// UnregisterComponent removes a component from monitoring
	UnregisterComponent(name string) error
	
	// GetComponentStatus returns the status of a monitored component
	GetComponentStatus(name string) (ComponentStatus, error)
	
	// GetAllComponentStatuses returns the status of all monitored components
	GetAllComponentStatuses() map[string]ComponentStatus
	
	// SetThresholds updates the thresholds for a component
	SetThresholds(name string, thresholds ResourceThresholds) error
}

// watchdogImpl is the implementation of the Watchdog interface
type watchdogImpl struct {
	config Config
	
	// components are the monitored components
	components map[string]interface{}
	
	// componentConfigs are the configurations for monitored components
	componentConfigs map[string]ComponentConfig
	
	// componentStatuses are the current statuses of monitored components
	componentStatuses map[string]ComponentStatus
	
	// circuitBreakers are the circuit breakers for monitored components
	circuitBreakers map[string]*CircuitBreaker
	
	// restartManagers are the restart managers for restartable components
	restartManagers map[string]*RestartManager
	
	// monitor is the resource monitor
	monitor *Monitor
	
	// deadlockDetector is the deadlock detector
	deadlockDetector *DeadlockDetector
	
	// degradationController is the degradation controller
	degradationController *DegradationController
	
	// diagnostics is the diagnostics provider
	diagnostics *DiagnosticsProvider
	
	// mutex protects the watchdog state
	mutex sync.RWMutex
	
	// running indicates whether the watchdog is running
	running bool
	
	// monitorContext is the context for the monitoring loop
	monitorContext context.Context
	
	// monitorCancel is the cancel function for the monitoring loop
	monitorCancel context.CancelFunc
	
	// monitorWg is a wait group for the monitoring goroutines
	monitorWg sync.WaitGroup
	
	// startTime is when the watchdog was started
	startTime time.Time
}

// NewWatchdog creates a new watchdog with the given configuration
func NewWatchdog(config Config) (Watchdog, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid watchdog configuration: %w", err)
	}
	
	w := &watchdogImpl{
		config:            config,
		components:        make(map[string]interface{}),
		componentConfigs:  make(map[string]ComponentConfig),
		componentStatuses: make(map[string]ComponentStatus),
		circuitBreakers:   make(map[string]*CircuitBreaker),
		restartManagers:   make(map[string]*RestartManager),
	}
	
	// Create monitor with the global thresholds
	monitor, err := NewMonitor(config.GlobalThresholds)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource monitor: %w", err)
	}
	w.monitor = monitor
	
	// Create deadlock detector if enabled
	if config.DeadlockDetection.Enabled {
		detector, err := NewDeadlockDetector(config.DeadlockDetection)
		if err != nil {
			return nil, fmt.Errorf("failed to create deadlock detector: %w", err)
		}
		w.deadlockDetector = detector
	}
	
	// Create degradation controller if enabled
	if config.DegradationEnabled {
		controller, err := NewDegradationController(config.DegradationLevels)
		if err != nil {
			return nil, fmt.Errorf("failed to create degradation controller: %w", err)
		}
		w.degradationController = controller
	}
	
	// Create diagnostics provider if events are enabled
	if config.EventsEnabled {
		w.diagnostics = NewDiagnosticsProvider()
	}
	
	return w, nil
}

// Start starts the watchdog monitoring
func (w *watchdogImpl) Start() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	if w.running {
		return nil // Already running
	}
	
	// Create a context for the monitoring loop
	w.monitorContext, w.monitorCancel = context.WithCancel(context.Background())
	
	// Start the monitoring loop
	w.monitorWg.Add(1)
	go w.monitorLoop()
	
	// Start the deadlock detector if enabled
	if w.deadlockDetector != nil {
		w.monitorWg.Add(1)
		go w.deadlockDetectionLoop()
	}
	
	w.running = true
	w.startTime = time.Now()
	
	log.Printf("Watchdog started with %d configured components", len(w.componentConfigs))
	
	return nil
}

// Stop stops the watchdog monitoring
func (w *watchdogImpl) Stop() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	if !w.running {
		return nil // Not running
	}
	
	// Stop the monitoring loop
	if w.monitorCancel != nil {
		w.monitorCancel()
	}
	
	// Wait for monitoring goroutines to finish
	w.monitorWg.Wait()
	
	w.running = false
	
	log.Println("Watchdog stopped")
	
	return nil
}

// RegisterComponent registers a component for monitoring
func (w *watchdogImpl) RegisterComponent(name string, component interface{}) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	// Check if the component is already registered
	if _, exists := w.components[name]; exists {
		return fmt.Errorf("component already registered: %s", name)
	}
	
	// Check if the component implements the required interfaces
	_, monitorable := component.(Monitorable)
	if !monitorable {
		return fmt.Errorf("component does not implement Monitorable interface: %s", name)
	}
	
	// Get or create component configuration
	config, exists := w.componentConfigs[name]
	if !exists {
		// Create default configuration
		config = DefaultComponentConfig(name)
		w.componentConfigs[name] = config
	}
	
	// Create circuit breaker
	circuitBreaker := NewCircuitBreaker(config.Name)
	w.circuitBreakers[name] = circuitBreaker
	
	// Create restart manager if component is restartable
	if restartable, ok := component.(Restartable); ok {
		restartManager := NewRestartManager(config.Restart, restartable)
		w.restartManagers[name] = restartManager
	}
	
	// Store the component
	w.components[name] = component
	
	// Initialize component status
	w.componentStatuses[name] = ComponentStatus{
		Name:            name,
		Health:          HealthUnknown,
		CircuitState:    CircuitClosed,
		LastRestart:     time.Time{},
		RestartCount:    0,
		Incidents:       []Incident{},
		DegradationLevel: 0,
	}
	
	log.Printf("Component registered for monitoring: %s", name)
	
	return nil
}

// UnregisterComponent removes a component from monitoring
func (w *watchdogImpl) UnregisterComponent(name string) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	// Check if the component is registered
	if _, exists := w.components[name]; !exists {
		return fmt.Errorf("component not registered: %s", name)
	}
	
	// Remove the component
	delete(w.components, name)
	delete(w.componentConfigs, name)
	delete(w.componentStatuses, name)
	delete(w.circuitBreakers, name)
	delete(w.restartManagers, name)
	
	log.Printf("Component unregistered from monitoring: %s", name)
	
	return nil
}

// GetComponentStatus returns the status of a monitored component
func (w *watchdogImpl) GetComponentStatus(name string) (ComponentStatus, error) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	
	// Check if the component is registered
	status, exists := w.componentStatuses[name]
	if !exists {
		return ComponentStatus{}, fmt.Errorf("component not registered: %s", name)
	}
	
	return status, nil
}

// GetAllComponentStatuses returns the status of all monitored components
func (w *watchdogImpl) GetAllComponentStatuses() map[string]ComponentStatus {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	
	// Create a copy of the component statuses
	statuses := make(map[string]ComponentStatus, len(w.componentStatuses))
	for name, status := range w.componentStatuses {
		statuses[name] = status
	}
	
	return statuses
}

// SetThresholds updates the thresholds for a component
func (w *watchdogImpl) SetThresholds(name string, thresholds ResourceThresholds) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	// Check if the component is registered
	config, exists := w.componentConfigs[name]
	if !exists {
		return fmt.Errorf("component not registered: %s", name)
	}
	
	// Update the thresholds
	config.Thresholds = thresholds
	w.componentConfigs[name] = config
	
	log.Printf("Thresholds updated for component: %s", name)
	
	return nil
}

// monitorLoop is the main monitoring loop
func (w *watchdogImpl) monitorLoop() {
	defer w.monitorWg.Done()
	
	ticker := time.NewTicker(w.config.MonitorInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-w.monitorContext.Done():
			return
		case <-ticker.C:
			w.monitor.CollectGlobalMetrics()
			w.monitorComponents()
		}
	}
}

// deadlockDetectionLoop runs the deadlock detection loop
func (w *watchdogImpl) deadlockDetectionLoop() {
	defer w.monitorWg.Done()
	
	ticker := time.NewTicker(w.config.DeadlockDetection.CheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-w.monitorContext.Done():
			return
		case <-ticker.C:
			if w.deadlockDetector != nil {
				w.detectDeadlocks()
			}
		}
	}
}

// monitorComponents monitors all registered components
func (w *watchdogImpl) monitorComponents() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	for name, component := range w.components {
		monitorable, ok := component.(Monitorable)
		if !ok {
			continue
		}
		
		// Get component configuration
		config := w.componentConfigs[name]
		
		// Get current component status
		status := w.componentStatuses[name]
		
		// Get resource usage
		resourceUsage := monitorable.GetResourceUsage()
		status.ResourceUsage = resourceUsage
		
		// Get health status
		health := monitorable.GetHealth()
		status.Health = health
		
		// Check thresholds
		exceeded, resource := w.checkThresholds(name, resourceUsage, config.Thresholds)
		if exceeded {
			// Create an incident
			incident := w.createResourceIncident(name, resource, resourceUsage, config.Thresholds)
			status.Incidents = append(status.Incidents, incident)
			
			// Limit the number of incidents
			if len(status.Incidents) > 10 {
				status.Incidents = status.Incidents[len(status.Incidents)-10:]
			}
			
			// Update circuit breaker
			circuitBreaker := w.circuitBreakers[name]
			if circuitBreaker != nil {
				circuitBreaker.RecordFailure()
				status.CircuitState = circuitBreaker.State()
			}
			
			// Handle degradation if component supports it
			if w.config.DegradationEnabled && w.degradationController != nil {
				if degradable, ok := component.(Degradable); ok {
					w.handleDegradation(name, degradable, &status)
				}
			}
			
			// Handle restart if component supports it and circuit is open
			if restartManager, exists := w.restartManagers[name]; exists && 
				status.CircuitState == CircuitOpen && 
				config.Restart.Enabled {
				w.handleRestart(name, restartManager, &status)
			}
		} else {
			// Update circuit breaker with success
			circuitBreaker := w.circuitBreakers[name]
			if circuitBreaker != nil {
				circuitBreaker.RecordSuccess()
				status.CircuitState = circuitBreaker.State()
			}
			
			// If circuit is closed, reset degradation if applicable
			if status.CircuitState == CircuitClosed && 
				w.config.DegradationEnabled && 
				w.degradationController != nil {
				if degradable, ok := component.(Degradable); ok && status.DegradationLevel > 0 {
					if err := degradable.SetDegradationLevel(0); err == nil {
						status.DegradationLevel = 0
					}
				}
			}
		}
		
		// Update component status
		w.componentStatuses[name] = status
	}
}

// checkThresholds checks if any resource thresholds are exceeded
func (w *watchdogImpl) checkThresholds(name string, usage ResourceUsage, thresholds ResourceThresholds) (bool, string) {
	if usage.CPUPercent > thresholds.MaxCPUPercent {
		return true, "CPU"
	}
	
	if usage.MemoryMB > float64(thresholds.MaxMemoryMB) {
		return true, "Memory"
	}
	
	if usage.Goroutines > thresholds.MaxGoroutines {
		return true, "Goroutines"
	}
	
	if usage.FileHandles > thresholds.MaxFileHandles {
		return true, "FileHandles"
	}
	
	if usage.GCPercent > thresholds.MaxGCPercent {
		return true, "GC"
	}
	
	return false, ""
}

// createResourceIncident creates a resource incident
func (w *watchdogImpl) createResourceIncident(
	name string, 
	resource string, 
	usage ResourceUsage, 
	thresholds ResourceThresholds,
) Incident {
	var value float64
	var threshold float64
	var unit string
	
	switch resource {
	case "CPU":
		value = usage.CPUPercent
		threshold = thresholds.MaxCPUPercent
		unit = "%"
	case "Memory":
		value = usage.MemoryMB
		threshold = float64(thresholds.MaxMemoryMB)
		unit = "MB"
	case "Goroutines":
		value = float64(usage.Goroutines)
		threshold = float64(thresholds.MaxGoroutines)
		unit = ""
	case "FileHandles":
		value = float64(usage.FileHandles)
		threshold = float64(thresholds.MaxFileHandles)
		unit = ""
	case "GC":
		value = usage.GCPercent
		threshold = thresholds.MaxGCPercent
		unit = "%"
	}
	
	description := fmt.Sprintf(
		"%s usage exceeded for component %s: %.2f%s > %.2f%s",
		resource, name, value, unit, threshold, unit,
	)
	
	remediation := fmt.Sprintf(
		"Consider increasing %s threshold or optimizing %s usage in component %s.",
		resource, resource, name,
	)
	
	// Create an incident
	incident := Incident{
		ID:            fmt.Sprintf("%s-%s-%d", name, resource, time.Now().UnixNano()),
		Timestamp:     time.Now(),
		Type:          IncidentResourceExceeded,
		Description:   description,
		ResourceUsage: usage,
		Remediation:   remediation,
	}
	
	// Log the incident
	log.Printf("Incident detected: %s", description)
	
	// Emit a diagnostic event if enabled
	if w.config.EventsEnabled && w.diagnostics != nil {
		w.diagnostics.EmitAgentDiagEvent(incident)
	}
	
	return incident
}

// handleDegradation handles degradation for a component
func (w *watchdogImpl) handleDegradation(
	name string, 
	degradable Degradable, 
	status *ComponentStatus,
) {
	currentLevel := status.DegradationLevel
	
	// Calculate new degradation level based on severity
	var newLevel int
	switch status.Health {
	case HealthCritical:
		newLevel = w.config.DegradationLevels // Max degradation
	case HealthDegraded:
		newLevel = currentLevel + 1
		if newLevel > w.config.DegradationLevels {
			newLevel = w.config.DegradationLevels
		}
	default:
		newLevel = currentLevel
	}
	
	// Apply new degradation level if it has changed
	if newLevel != currentLevel {
		if err := degradable.SetDegradationLevel(newLevel); err == nil {
			status.DegradationLevel = newLevel
			log.Printf("Component %s degraded to level %d", name, newLevel)
		}
	}
}

// handleRestart handles restart for a component
func (w *watchdogImpl) handleRestart(
	name string, 
	restartManager *RestartManager, 
	status *ComponentStatus,
) {
	// Attempt to restart the component
	success, err := restartManager.AttemptRestart(w.monitorContext)
	
	if success {
		// Update restart metrics
		status.LastRestart = time.Now()
		status.RestartCount++
		log.Printf("Component %s restarted successfully", name)
	} else {
		// Create a restart failure incident
		incident := Incident{
			ID:          fmt.Sprintf("%s-restart-failure-%d", name, time.Now().UnixNano()),
			Timestamp:   time.Now(),
			Type:        IncidentRestartFailed,
			Description: fmt.Sprintf("Failed to restart component %s: %v", name, err),
			Remediation: "Check component implementation and logs for errors.",
		}
		status.Incidents = append(status.Incidents, incident)
		
		// Log the incident
		log.Printf("Restart failed: %s", incident.Description)
		
		// Emit a diagnostic event if enabled
		if w.config.EventsEnabled && w.diagnostics != nil {
			w.diagnostics.EmitAgentDiagEvent(incident)
		}
	}
}

// detectDeadlocks checks for deadlocks in all components
func (w *watchdogImpl) detectDeadlocks() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	if w.deadlockDetector == nil {
		return
	}
	
	// Get global goroutine count
	globalMetrics := w.monitor.GetGlobalMetrics()
	
	// Skip detailed analysis if below threshold
	if globalMetrics.Goroutines < w.config.DeadlockDetection.GoroutineThreshold {
		return
	}
	
	// Detect deadlocks
	deadlocks := w.deadlockDetector.DetectDeadlocks()
	
	// Handle detected deadlocks
	for _, deadlock := range deadlocks {
		// Find the affected component
		componentName := deadlock.ComponentName
		
		// Skip if component not registered
		status, exists := w.componentStatuses[componentName]
		if !exists {
			continue
		}
		
		// Create a deadlock incident
		incident := Incident{
			ID:          fmt.Sprintf("%s-deadlock-%d", componentName, time.Now().UnixNano()),
			Timestamp:   time.Now(),
			Type:        IncidentDeadlockDetected,
			Description: fmt.Sprintf("Deadlock detected in component %s: %s", componentName, deadlock.Description),
			Remediation: deadlock.Remediation,
		}
		status.Incidents = append(status.Incidents, incident)
		
		// Update circuit breaker
		circuitBreaker := w.circuitBreakers[componentName]
		if circuitBreaker != nil {
			circuitBreaker.RecordFailure()
			status.CircuitState = circuitBreaker.State()
		}
		
		// Log the incident
		log.Printf("Deadlock detected: %s", incident.Description)
		
		// Emit a diagnostic event if enabled
		if w.config.EventsEnabled && w.diagnostics != nil {
			w.diagnostics.EmitAgentDiagEvent(incident)
		}
		
		// Update component status
		w.componentStatuses[componentName] = status
	}
}
