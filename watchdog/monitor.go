package watchdog

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// ResourceUsage represents the resource usage of a component
type ResourceUsage struct {
	// CPUPercent is the CPU usage percentage
	CPUPercent float64
	
	// MemoryBytes is the memory usage in bytes
	MemoryBytes uint64
	
	// FileDescriptors is the number of open file descriptors
	FileDescriptors int
	
	// Goroutines is the number of active goroutines
	Goroutines int
	
	// LastUpdated is when this resource usage was last updated
	LastUpdated time.Time
}

// ThresholdExceededEvent represents a resource threshold exceeded event
type ThresholdExceededEvent struct {
	// ComponentName is the name of the component that exceeded a threshold
	ComponentName string
	
	// ResourceType is the type of resource that exceeded a threshold (CPU, memory, etc.)
	ResourceType string
	
	// CurrentValue is the current value of the resource
	CurrentValue float64
	
	// ThresholdValue is the threshold value that was exceeded
	ThresholdValue float64
	
	// Timestamp is when the threshold was exceeded
	Timestamp time.Time
}

// ThresholdHandler is a function that is called when a threshold is exceeded
type ThresholdHandler func(event ThresholdExceededEvent)

// Component defines the interface for components that can be monitored
type Component interface {
	// Name returns the name of the component
	Name() string
	
	// ResourceUsage returns the current resource usage of the component
	ResourceUsage() ResourceUsage
	
	// Heartbeat sends a heartbeat to indicate the component is alive
	Heartbeat() error
	
	// Shutdown performs a graceful shutdown of the component
	Shutdown(ctx context.Context) error
	
	// Start starts the component
	Start() error
}

// ResourceMonitor monitors the resource usage of components
type ResourceMonitor struct {
	config        Config
	components    map[string]Component
	usageHistory  map[string][]ResourceUsage
	historyMaxLen int
	handlers      []ThresholdHandler
	degradationState map[string]string // component name -> current degradation level
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex
}

// NewResourceMonitor creates a new resource monitor with the given configuration
func NewResourceMonitor(config Config) *ResourceMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ResourceMonitor{
		config:        config,
		components:    make(map[string]Component),
		usageHistory:  make(map[string][]ResourceUsage),
		historyMaxLen: 20, // Keep last 20 readings
		handlers:      make([]ThresholdHandler, 0),
		degradationState: make(map[string]string),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// AddComponent adds a component to be monitored
func (rm *ResourceMonitor) AddComponent(component Component) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	name := component.Name()
	rm.components[name] = component
	rm.usageHistory[name] = make([]ResourceUsage, 0, rm.historyMaxLen)
	rm.degradationState[name] = ""
	
	return nil
}

// RemoveComponent removes a component from monitoring
func (rm *ResourceMonitor) RemoveComponent(name string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	delete(rm.components, name)
	delete(rm.usageHistory, name)
	delete(rm.degradationState, name)
}

// AddThresholdHandler adds a handler to be called when a threshold is exceeded
func (rm *ResourceMonitor) AddThresholdHandler(handler ThresholdHandler) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.handlers = append(rm.handlers, handler)
}

// Start starts the resource monitoring
func (rm *ResourceMonitor) Start() error {
	rm.wg.Add(1)
	go rm.monitorLoop()
	
	return nil
}

// Stop stops the resource monitoring
func (rm *ResourceMonitor) Stop() error {
	rm.cancel()
	rm.wg.Wait()
	
	return nil
}

// GetResourceUsage returns the current resource usage for a component
func (rm *ResourceMonitor) GetResourceUsage(componentName string) (ResourceUsage, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	component, ok := rm.components[componentName]
	if !ok {
		return ResourceUsage{}, false
	}
	
	return component.ResourceUsage(), true
}

// GetResourceHistory returns the resource usage history for a component
func (rm *ResourceMonitor) GetResourceHistory(componentName string) ([]ResourceUsage, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	history, ok := rm.usageHistory[componentName]
	if !ok {
		return nil, false
	}
	
	// Return a copy to prevent concurrent modification
	result := make([]ResourceUsage, len(history))
	copy(result, history)
	
	return result, true
}

// GetDegradationLevel returns the current degradation level for a component
func (rm *ResourceMonitor) GetDegradationLevel(componentName string) (string, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	level, ok := rm.degradationState[componentName]
	return level, ok
}

// monitorLoop is the main monitoring loop
func (rm *ResourceMonitor) monitorLoop() {
	defer rm.wg.Done()
	
	ticker := time.NewTicker(rm.config.MonitoringInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.checkResources()
		}
	}
}

// checkResources checks the resource usage of all components
func (rm *ResourceMonitor) checkResources() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	now := time.Now()
	
	for name, component := range rm.components {
		usage := component.ResourceUsage()
		usage.LastUpdated = now
		
		// Add to history, maintaining max length
		history := rm.usageHistory[name]
		if len(history) >= rm.historyMaxLen {
			// Shift elements left, dropping oldest
			copy(history, history[1:])
			history = history[:len(history)-1]
		}
		history = append(history, usage)
		rm.usageHistory[name] = history
		
		// Check against thresholds
		rm.checkThresholds(name, usage)
	}
}

// checkThresholds checks resource usage against thresholds
func (rm *ResourceMonitor) checkThresholds(componentName string, usage ResourceUsage) {
	componentConfig, ok := rm.config.ComponentConfigs[componentName]
	if !ok || !componentConfig.Enabled {
		return
	}
	
	// Convert memory from bytes to MB for comparison
	memoryMB := float64(usage.MemoryBytes) / (1024 * 1024)
	
	// Check CPU threshold
	if usage.CPUPercent > componentConfig.MaxCPUPercent {
		event := ThresholdExceededEvent{
			ComponentName: componentName,
			ResourceType:  "CPU",
			CurrentValue:  usage.CPUPercent,
			ThresholdValue: componentConfig.MaxCPUPercent,
			Timestamp:     time.Now(),
		}
		
		rm.notifyThresholdExceeded(event)
	}
	
	// Check memory threshold
	if memoryMB > float64(componentConfig.MaxMemoryMB) {
		event := ThresholdExceededEvent{
			ComponentName: componentName,
			ResourceType:  "Memory",
			CurrentValue:  memoryMB,
			ThresholdValue: float64(componentConfig.MaxMemoryMB),
			Timestamp:     time.Now(),
		}
		
		rm.notifyThresholdExceeded(event)
	}
	
	// Check file descriptor threshold
	if usage.FileDescriptors > componentConfig.MaxFileDescriptors {
		event := ThresholdExceededEvent{
			ComponentName: componentName,
			ResourceType:  "FileDescriptors",
			CurrentValue:  float64(usage.FileDescriptors),
			ThresholdValue: float64(componentConfig.MaxFileDescriptors),
			Timestamp:     time.Now(),
		}
		
		rm.notifyThresholdExceeded(event)
	}
	
	// Check goroutine threshold
	if usage.Goroutines > componentConfig.MaxGoroutines {
		event := ThresholdExceededEvent{
			ComponentName: componentName,
			ResourceType:  "Goroutines",
			CurrentValue:  float64(usage.Goroutines),
			ThresholdValue: float64(componentConfig.MaxGoroutines),
			Timestamp:     time.Now(),
		}
		
		rm.notifyThresholdExceeded(event)
	}
	
	// Check degradation levels
	rm.checkDegradationLevels(componentName, usage)
}

// checkDegradationLevels checks resource usage against degradation levels
func (rm *ResourceMonitor) checkDegradationLevels(componentName string, usage ResourceUsage) {
	componentConfig, ok := rm.config.ComponentConfigs[componentName]
	if !ok || !componentConfig.Enabled {
		return
	}
	
	// Convert memory from bytes to MB for comparison
	memoryMB := float64(usage.MemoryBytes) / (1024 * 1024)
	
	// Find the highest applicable degradation level
	currentLevel := ""
	for _, level := range componentConfig.DegradationLevels {
		if usage.CPUPercent >= level.CPUThresholdPercent || memoryMB >= float64(level.MemoryThresholdMB) {
			currentLevel = level.Name
		}
	}
	
	// Update degradation state if changed
	previousLevel := rm.degradationState[componentName]
	if currentLevel != previousLevel {
		rm.degradationState[componentName] = currentLevel
		
		// If we moved to a higher degradation level, notify
		if currentLevel != "" {
			var event ThresholdExceededEvent
			if usage.CPUPercent >= componentConfig.MaxCPUPercent {
				event = ThresholdExceededEvent{
					ComponentName: componentName,
					ResourceType:  "DegradationLevel",
					CurrentValue:  usage.CPUPercent,
					ThresholdValue: componentConfig.MaxCPUPercent,
					Timestamp:     time.Now(),
				}
			} else {
				event = ThresholdExceededEvent{
					ComponentName: componentName,
					ResourceType:  "DegradationLevel",
					CurrentValue:  memoryMB,
					ThresholdValue: float64(componentConfig.MaxMemoryMB),
					Timestamp:     time.Now(),
				}
			}
			
			rm.notifyThresholdExceeded(event)
		}
	}
}

// notifyThresholdExceeded notifies handlers of a threshold exceeded event
func (rm *ResourceMonitor) notifyThresholdExceeded(event ThresholdExceededEvent) {
	for _, handler := range rm.handlers {
		go handler(event)
	}
}

// GetTotalResourceUsage returns the total resource usage of the agent
func (rm *ResourceMonitor) GetTotalResourceUsage() ResourceUsage {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return ResourceUsage{
		CPUPercent:      0, // Not available directly
		MemoryBytes:     memStats.Alloc,
		FileDescriptors: 0, // Not available directly in Go
		Goroutines:      runtime.NumGoroutine(),
		LastUpdated:     time.Now(),
	}
}
