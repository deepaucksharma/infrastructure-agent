package watchdog

import (
	"fmt"
	"sync"
)

// DegradationController manages component degradation
type DegradationController struct {
	// maxLevel is the maximum degradation level
	maxLevel int
	
	// levelActions maps degradation levels to actions
	levelActions map[int][]string
	
	// levelDescriptions maps degradation levels to descriptions
	levelDescriptions map[int]string
	
	// componentLevels tracks current degradation levels by component
	componentLevels map[string]int
	
	// mutex protects the controller state
	mutex sync.RWMutex
}

// NewDegradationController creates a new degradation controller
func NewDegradationController(maxLevel int) (*DegradationController, error) {
	if maxLevel <= 0 {
		return nil, fmt.Errorf("max degradation level must be positive")
	}
	
	return &DegradationController{
		maxLevel:          maxLevel,
		levelActions:      make(map[int][]string),
		levelDescriptions: make(map[int]string),
		componentLevels:   make(map[string]int),
	}, nil
}

// SetLevelActions sets the actions for a degradation level
func (dc *DegradationController) SetLevelActions(level int, actions []string, description string) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	
	if level <= 0 || level > dc.maxLevel {
		return fmt.Errorf("invalid degradation level: %d (max: %d)", level, dc.maxLevel)
	}
	
	dc.levelActions[level] = actions
	dc.levelDescriptions[level] = description
	
	return nil
}

// SetComponentLevel sets the degradation level for a component
func (dc *DegradationController) SetComponentLevel(component string, level int) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	
	if level < 0 || level > dc.maxLevel {
		return fmt.Errorf("invalid degradation level: %d (max: %d)", level, dc.maxLevel)
	}
	
	dc.componentLevels[component] = level
	
	return nil
}

// GetComponentLevel gets the current degradation level for a component
func (dc *DegradationController) GetComponentLevel(component string) int {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	
	level, exists := dc.componentLevels[component]
	if !exists {
		return 0
	}
	
	return level
}

// GetLevelActions gets the actions for a degradation level
func (dc *DegradationController) GetLevelActions(level int) ([]string, error) {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	
	if level <= 0 || level > dc.maxLevel {
		return nil, fmt.Errorf("invalid degradation level: %d (max: %d)", level, dc.maxLevel)
	}
	
	actions, exists := dc.levelActions[level]
	if !exists {
		return []string{}, nil
	}
	
	return actions, nil
}

// GetLevelDescription gets the description for a degradation level
func (dc *DegradationController) GetLevelDescription(level int) (string, error) {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	
	if level <= 0 || level > dc.maxLevel {
		return "", fmt.Errorf("invalid degradation level: %d (max: %d)", level, dc.maxLevel)
	}
	
	description, exists := dc.levelDescriptions[level]
	if !exists {
		return "", nil
	}
	
	return description, nil
}

// GetMaxLevel gets the maximum degradation level
func (dc *DegradationController) GetMaxLevel() int {
	return dc.maxLevel
}

// GetComponentsAtLevel gets all components at a specific degradation level
func (dc *DegradationController) GetComponentsAtLevel(level int) ([]string, error) {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	
	if level < 0 || level > dc.maxLevel {
		return nil, fmt.Errorf("invalid degradation level: %d (max: %d)", level, dc.maxLevel)
	}
	
	var components []string
	
	for component, componentLevel := range dc.componentLevels {
		if componentLevel == level {
			components = append(components, component)
		}
	}
	
	return components, nil
}

// ResetComponent resets a component to no degradation
func (dc *DegradationController) ResetComponent(component string) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	
	dc.componentLevels[component] = 0
}

// ResetAllComponents resets all components to no degradation
func (dc *DegradationController) ResetAllComponents() {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	
	for component := range dc.componentLevels {
		dc.componentLevels[component] = 0
	}
}
