package collector

import (
	"fmt"
	"sync"
)

// ConsumerRegistry manages registered process consumers
type ConsumerRegistry struct {
	consumers map[string]ProcessConsumer
	mutex     sync.RWMutex
}

// NewConsumerRegistry creates a new consumer registry
func NewConsumerRegistry() *ConsumerRegistry {
	return &ConsumerRegistry{
		consumers: make(map[string]ProcessConsumer),
	}
}

// Register adds a consumer to the registry
func (r *ConsumerRegistry) Register(name string, consumer ProcessConsumer) error {
	if name == "" {
		return fmt.Errorf("consumer name cannot be empty")
	}
	
	if consumer == nil {
		return fmt.Errorf("consumer cannot be nil")
	}
	
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if _, exists := r.consumers[name]; exists {
		return fmt.Errorf("consumer '%s' already registered", name)
	}
	
	r.consumers[name] = consumer
	return nil
}

// Unregister removes a consumer from the registry
func (r *ConsumerRegistry) Unregister(name string) error {
	if name == "" {
		return fmt.Errorf("consumer name cannot be empty")
	}
	
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if _, exists := r.consumers[name]; !exists {
		return fmt.Errorf("consumer '%s' not found", name)
	}
	
	delete(r.consumers, name)
	return nil
}

// GetConsumer returns a registered consumer by name
func (r *ConsumerRegistry) GetConsumer(name string) (ProcessConsumer, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	consumer, exists := r.consumers[name]
	return consumer, exists
}

// GetConsumerNames returns all registered consumer names
func (r *ConsumerRegistry) GetConsumerNames() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	names := make([]string, 0, len(r.consumers))
	for name := range r.consumers {
		names = append(names, name)
	}
	return names
}

// NotifyAll sends a process event to all registered consumers
func (r *ConsumerRegistry) NotifyAll(event ProcessEvent) []error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var errors []error
	for name, consumer := range r.consumers {
		err := consumer.HandleProcessEvent(event)
		if err != nil {
			errors = append(errors, fmt.Errorf("consumer '%s' error: %w", name, err))
		}
	}
	
	return errors
}

// NotifyAllAsync sends a process event to all registered consumers asynchronously
func (r *ConsumerRegistry) NotifyAllAsync(event ProcessEvent) {
	// Make a copy of the event to ensure safety
	eventCopy := ProcessEvent{
		Type:      event.Type,
		Process:   event.Process.Clone(),
		Timestamp: event.Timestamp,
	}
	
	// Copy the consumer list to avoid holding the lock during notification
	r.mutex.RLock()
	consumers := make(map[string]ProcessConsumer, len(r.consumers))
	for name, consumer := range r.consumers {
		consumers[name] = consumer
	}
	r.mutex.RUnlock()
	
	// Notify each consumer in a separate goroutine
	for name, consumer := range consumers {
		go func(n string, c ProcessConsumer, e ProcessEvent) {
			// We don't return errors here since this is async
			// Errors should be handled by the consumer or logged
			_ = c.HandleProcessEvent(e)
		}(name, consumer, eventCopy)
	}
}

// ConsumerCount returns the number of registered consumers
func (r *ConsumerRegistry) ConsumerCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	return len(r.consumers)
}
