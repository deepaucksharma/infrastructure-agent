package tests

import (
	"testing"

	"github.com/newrelic/infrastructure-agent/watchdog"
	"github.com/stretchr/testify/assert"
)

// TestDegradationControllerCreation tests creating a degradation controller
func TestDegradationControllerCreation(t *testing.T) {
	// Test valid creation
	controller, err := watchdog.NewDegradationController(3)
	assert.NoError(t, err)
	assert.NotNil(t, controller)
	assert.Equal(t, 3, controller.GetMaxLevel())
	
	// Test invalid creation
	controller, err = watchdog.NewDegradationController(0)
	assert.Error(t, err)
	assert.Nil(t, controller)
	
	controller, err = watchdog.NewDegradationController(-1)
	assert.Error(t, err)
	assert.Nil(t, controller)
}

// TestSetLevelActions tests setting actions for a degradation level
func TestSetLevelActions(t *testing.T) {
	controller, err := watchdog.NewDegradationController(3)
	assert.NoError(t, err)
	
	// Test setting valid level actions
	err = controller.SetLevelActions(1, []string{"reduce_frequency"}, "Minor degradation")
	assert.NoError(t, err)
	
	err = controller.SetLevelActions(2, []string{"reduce_frequency", "disable_features"}, "Moderate degradation")
	assert.NoError(t, err)
	
	err = controller.SetLevelActions(3, []string{"reduce_frequency", "disable_features", "minimal_mode"}, "Severe degradation")
	assert.NoError(t, err)
	
	// Test setting invalid level actions
	err = controller.SetLevelActions(0, []string{"action"}, "description")
	assert.Error(t, err)
	
	err = controller.SetLevelActions(4, []string{"action"}, "description")
	assert.Error(t, err)
	
	err = controller.SetLevelActions(-1, []string{"action"}, "description")
	assert.Error(t, err)
}

// TestGetLevelActions tests getting actions for a degradation level
func TestGetLevelActions(t *testing.T) {
	controller, err := watchdog.NewDegradationController(3)
	assert.NoError(t, err)
	
	// Set level actions
	controller.SetLevelActions(1, []string{"action1"}, "description1")
	controller.SetLevelActions(2, []string{"action2a", "action2b"}, "description2")
	
	// Test getting valid level actions
	actions, err := controller.GetLevelActions(1)
	assert.NoError(t, err)
	assert.Equal(t, []string{"action1"}, actions)
	
	actions, err = controller.GetLevelActions(2)
	assert.NoError(t, err)
	assert.Equal(t, []string{"action2a", "action2b"}, actions)
	
	// Test getting actions for level with no actions set
	actions, err = controller.GetLevelActions(3)
	assert.NoError(t, err)
	assert.Empty(t, actions)
	
	// Test getting invalid level actions
	actions, err = controller.GetLevelActions(0)
	assert.Error(t, err)
	assert.Nil(t, actions)
	
	actions, err = controller.GetLevelActions(4)
	assert.Error(t, err)
	assert.Nil(t, actions)
}

// TestGetLevelDescription tests getting descriptions for a degradation level
func TestGetLevelDescription(t *testing.T) {
	controller, err := watchdog.NewDegradationController(3)
	assert.NoError(t, err)
	
	// Set level descriptions
	controller.SetLevelActions(1, []string{"action1"}, "description1")
	controller.SetLevelActions(2, []string{"action2a", "action2b"}, "description2")
	
	// Test getting valid level descriptions
	desc, err := controller.GetLevelDescription(1)
	assert.NoError(t, err)
	assert.Equal(t, "description1", desc)
	
	desc, err = controller.GetLevelDescription(2)
	assert.NoError(t, err)
	assert.Equal(t, "description2", desc)
	
	// Test getting description for level with no description set
	desc, err = controller.GetLevelDescription(3)
	assert.NoError(t, err)
	assert.Empty(t, desc)
	
	// Test getting invalid level descriptions
	desc, err = controller.GetLevelDescription(0)
	assert.Error(t, err)
	assert.Empty(t, desc)
	
	desc, err = controller.GetLevelDescription(4)
	assert.Error(t, err)
	assert.Empty(t, desc)
}

// TestSetComponentLevel tests setting the degradation level for a component
func TestSetComponentLevel(t *testing.T) {
	controller, err := watchdog.NewDegradationController(3)
	assert.NoError(t, err)
	
	// Test setting valid component levels
	err = controller.SetComponentLevel("component1", 1)
	assert.NoError(t, err)
	
	err = controller.SetComponentLevel("component2", 2)
	assert.NoError(t, err)
	
	err = controller.SetComponentLevel("component3", 3)
	assert.NoError(t, err)
	
	// Test setting level 0 (no degradation)
	err = controller.SetComponentLevel("component4", 0)
	assert.NoError(t, err)
	
	// Test setting invalid component levels
	err = controller.SetComponentLevel("component5", 4)
	assert.Error(t, err)
	
	err = controller.SetComponentLevel("component6", -1)
	assert.Error(t, err)
}

// TestGetComponentLevel tests getting the degradation level for a component
func TestGetComponentLevel(t *testing.T) {
	controller, err := watchdog.NewDegradationController(3)
	assert.NoError(t, err)
	
	// Set component levels
	controller.SetComponentLevel("component1", 1)
	controller.SetComponentLevel("component2", 2)
	controller.SetComponentLevel("component3", 3)
	controller.SetComponentLevel("component4", 0)
	
	// Test getting component levels
	level := controller.GetComponentLevel("component1")
	assert.Equal(t, 1, level)
	
	level = controller.GetComponentLevel("component2")
	assert.Equal(t, 2, level)
	
	level = controller.GetComponentLevel("component3")
	assert.Equal(t, 3, level)
	
	level = controller.GetComponentLevel("component4")
	assert.Equal(t, 0, level)
	
	// Test getting level for non-existent component
	level = controller.GetComponentLevel("non-existent")
	assert.Equal(t, 0, level)
}

// TestGetComponentsAtLevel tests getting all components at a specific degradation level
func TestGetComponentsAtLevel(t *testing.T) {
	controller, err := watchdog.NewDegradationController(3)
	assert.NoError(t, err)
	
	// Set component levels
	controller.SetComponentLevel("component1", 1)
	controller.SetComponentLevel("component2", 2)
	controller.SetComponentLevel("component3", 1)
	controller.SetComponentLevel("component4", 3)
	controller.SetComponentLevel("component5", 2)
	controller.SetComponentLevel("component6", 0)
	
	// Test getting components at level 0
	components, err := controller.GetComponentsAtLevel(0)
	assert.NoError(t, err)
	assert.Len(t, components, 1)
	assert.Contains(t, components, "component6")
	
	// Test getting components at level 1
	components, err = controller.GetComponentsAtLevel(1)
	assert.NoError(t, err)
	assert.Len(t, components, 2)
	assert.Contains(t, components, "component1")
	assert.Contains(t, components, "component3")
	
	// Test getting components at level 2
	components, err = controller.GetComponentsAtLevel(2)
	assert.NoError(t, err)
	assert.Len(t, components, 2)
	assert.Contains(t, components, "component2")
	assert.Contains(t, components, "component5")
	
	// Test getting components at level 3
	components, err = controller.GetComponentsAtLevel(3)
	assert.NoError(t, err)
	assert.Len(t, components, 1)
	assert.Contains(t, components, "component4")
	
	// Test getting components at invalid levels
	components, err = controller.GetComponentsAtLevel(-1)
	assert.Error(t, err)
	assert.Nil(t, components)
	
	components, err = controller.GetComponentsAtLevel(4)
	assert.Error(t, err)
	assert.Nil(t, components)
}

// TestResetComponent tests resetting a component to no degradation
func TestResetComponent(t *testing.T) {
	controller, err := watchdog.NewDegradationController(3)
	assert.NoError(t, err)
	
	// Set component levels
	controller.SetComponentLevel("component1", 1)
	controller.SetComponentLevel("component2", 2)
	
	// Verify initial levels
	assert.Equal(t, 1, controller.GetComponentLevel("component1"))
	assert.Equal(t, 2, controller.GetComponentLevel("component2"))
	
	// Reset component1
	controller.ResetComponent("component1")
	
	// Verify reset level
	assert.Equal(t, 0, controller.GetComponentLevel("component1"))
	assert.Equal(t, 2, controller.GetComponentLevel("component2"))
	
	// Reset non-existent component (should be safe)
	controller.ResetComponent("non-existent")
}

// TestResetAllComponents tests resetting all components to no degradation
func TestResetAllComponents(t *testing.T) {
	controller, err := watchdog.NewDegradationController(3)
	assert.NoError(t, err)
	
	// Set component levels
	controller.SetComponentLevel("component1", 1)
	controller.SetComponentLevel("component2", 2)
	controller.SetComponentLevel("component3", 3)
	
	// Verify initial levels
	assert.Equal(t, 1, controller.GetComponentLevel("component1"))
	assert.Equal(t, 2, controller.GetComponentLevel("component2"))
	assert.Equal(t, 3, controller.GetComponentLevel("component3"))
	
	// Reset all components
	controller.ResetAllComponents()
	
	// Verify all are reset
	assert.Equal(t, 0, controller.GetComponentLevel("component1"))
	assert.Equal(t, 0, controller.GetComponentLevel("component2"))
	assert.Equal(t, 0, controller.GetComponentLevel("component3"))
}
