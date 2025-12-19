package dsl

import (
	"fmt"
	"sync"
)

// StepsOutputManager manages step outputs for expression evaluation
type StepsOutputManager struct {
	mu      sync.RWMutex                      // Story 1.5: 支持运行时更新
	outputs map[string]map[string]interface{} // stepID → outputs
}

// NewStepsOutputManager creates a new steps output manager
func NewStepsOutputManager() *StepsOutputManager {
	return &StepsOutputManager{
		outputs: make(map[string]map[string]interface{}),
	}
}

// Set stores outputs for a step
func (m *StepsOutputManager) Set(stepID string, outputs map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputs[stepID] = outputs
}

// Update incrementally updates outputs for a step (Story 1.5: 运行时更新)
func (m *StepsOutputManager) Update(stepID string, outputs map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.outputs[stepID] == nil {
		m.outputs[stepID] = make(map[string]interface{})
	}

	for k, v := range outputs {
		m.outputs[stepID][k] = v
	}
}

// Get retrieves a specific output value from a step
func (m *StepsOutputManager) Get(stepID, key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stepOutputs, exists := m.outputs[stepID]
	if !exists {
		return nil, fmt.Errorf("step '%s' not found or not executed", stepID)
	}

	value, exists := stepOutputs[key]
	if !exists {
		// Build available outputs list for error message
		available := make([]string, 0, len(stepOutputs))
		for k := range stepOutputs {
			available = append(available, k)
		}
		return nil, fmt.Errorf("output '%s' not found in step '%s'. Available: %v", key, stepID, available)
	}

	return value, nil
}

// ToContext converts outputs to context format for expression evaluation
func (m *StepsOutputManager) ToContext() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]interface{})
	for stepID, outputs := range m.outputs {
		result[stepID] = map[string]interface{}{
			"outputs": outputs,
		}
	}
	return result
}
