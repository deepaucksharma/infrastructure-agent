package sketch

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Register the DDSketch at package initialization
func init() {
	RegisterSketch("ddsketch", func() Sketch {
		return NewDDSketch(DefaultConfig().DDSketch)
	})
}

// DDSketch implements a sketch that provides accurate quantile approximation
// with relative-error guarantees.
// Based on the paper "DDSketch: A fast and fully-mergeable quantile sketch with
// relative-error guarantees" by Masson, Rim, Lee
type DDSketch struct {
	gamma        float64    // Relative accuracy parameter
	multiplier   float64    // Mapping multiplier (1/ln(1+gamma))
	offset       float64    // Mapping offset
	minValue     float64    // Minimum allowed value
	maxValue     float64    // Maximum allowed value
	
	store        Store      // Bucket store (sparse or dense)
	useSparseStore bool     // Whether to use sparse store
	autoSwitch   bool       // Whether to automatically switch between stores
	switchThreshold float64 // Density threshold for switching to dense store
	
	min          float64    // Minimum value seen
	max          float64    // Maximum value seen
	sum          float64    // Sum of all values
	count        uint64     // Count of all values
	
	sparseStore  Store      // Sparse store reference
	denseStore   Store      // Dense store reference
	
	startTime    time.Time  // Time when the sketch was created
	lastSwitch   time.Time  // Time of last store switch
	
	mutex        sync.RWMutex
}

// NewDDSketch creates a new DDSketch with the given configuration
func NewDDSketch(config DDSketchConfig) *DDSketch {
	gamma, multiplier, offset := config.LogarithmicMapping()
	
	var store Store
	if config.UseSparseStore {
		store = NewSparseStore(config.CollapseThreshold)
	} else {
		store = NewDenseStore(config.InitialCapacity)
	}
	
	// Create both store types for potential switching
	sparseStore := NewSparseStore(config.CollapseThreshold)
	denseStore := NewDenseStore(config.InitialCapacity)
	
	return &DDSketch{
		gamma:        gamma,
		multiplier:   multiplier,
		offset:       offset,
		minValue:     config.MinValue,
		maxValue:     config.MaxValue,
		store:        store,
		useSparseStore: config.UseSparseStore,
		autoSwitch:   config.AutoSwitch,
		switchThreshold: config.SwitchThreshold,
		min:          math.Inf(1),
		max:          math.Inf(-1),
		sum:          0,
		count:        0,
		sparseStore:  sparseStore,
		denseStore:   denseStore,
		startTime:    time.Now(),
		lastSwitch:   time.Now(),
	}
}

// Add adds a value to the sketch
func (d *DDSketch) Add(value float64) error {
	return d.AddWithCount(value, 1)
}

// AddWithCount adds a value to the sketch with a specific count
func (d *DDSketch) AddWithCount(value float64, count uint64) error {
	// Validate input
	if value <= 0 {
		return fmt.Errorf("value must be positive: %f", value)
	}
	if count == 0 {
		return nil
	}
	
	// Bound value to min/max range
	if value < d.minValue {
		value = d.minValue
	} else if value > d.maxValue {
		value = d.maxValue
	}
	
	// Calculate bucket index using logarithmic mapping
	index := d.valueToIndex(value)
	
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	// Add to store
	d.store.Add(index, count)
	
	// Update statistics
	d.count += count
	d.sum += value * float64(count)
	
	// Update min/max values
	if value < d.min {
		d.min = value
	}
	if value > d.max {
		d.max = value
	}
	
	// Check if we should switch store type
	if d.autoSwitch && time.Since(d.lastSwitch) > time.Second {
		d.checkAndSwitchStores()
	}
	
	return nil
}

// GetValueAtQuantile returns the value at the specified quantile
func (d *DDSketch) GetValueAtQuantile(q float64) (float64, error) {
	// Validate input
	if q < 0 || q > 1 {
		return 0, ErrInvalidQuantile
	}
	
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	// Empty sketch check
	if d.count == 0 {
		return 0, ErrEmptySketch
	}
	
	// Handle edge cases
	if q == 0 {
		return d.min, nil
	}
	if q == 1 {
		return d.max, nil
	}
	
	// Calculate rank
	rank := uint64(math.Ceil(q * float64(d.count)))
	
	// Find the bucket that contains the rank
	minIndex, hasMin := d.store.GetMinIndex()
	maxIndex, hasMax := d.store.GetMaxIndex()
	
	if !hasMin || !hasMax {
		return 0, ErrEmptySketch
	}
	
	// Walk through buckets to find the one containing the rank
	var sum uint64
	for i := minIndex; i <= maxIndex; i++ {
		sum += d.store.Get(i)
		if sum >= rank {
			// Found the bucket, convert index to value
			return d.indexToValue(i), nil
		}
	}
	
	// Fallback in case of unexpected error
	return d.max, nil
}

// GetQuantileAtValue returns the quantile at which value falls
func (d *DDSketch) GetQuantileAtValue(value float64) (float64, error) {
	// Validate input
	if value <= 0 {
		return 0, fmt.Errorf("value must be positive: %f", value)
	}
	
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	// Empty sketch check
	if d.count == 0 {
		return 0, ErrEmptySketch
	}
	
	// Bound value to min/max range
	if value < d.minValue {
		value = d.minValue
	} else if value > d.maxValue {
		value = d.maxValue
	}
	
	// Handle edge cases
	if value <= d.min {
		return 0, nil
	}
	if value >= d.max {
		return 1, nil
	}
	
	// Calculate bucket index using logarithmic mapping
	index := d.valueToIndex(value)
	
	// Find the number of elements below this value
	minIndex, hasMin := d.store.GetMinIndex()
	
	if !hasMin {
		return 0, ErrEmptySketch
	}
	
	// Sum counts up to the index
	var sum uint64
	for i := minIndex; i < index; i++ {
		sum += d.store.Get(i)
	}
	
	// Calculate quantile
	return float64(sum) / float64(d.count), nil
}

// GetCount returns the total count of values in the sketch
func (d *DDSketch) GetCount() uint64 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.count
}

// GetMin returns the minimum value added to the sketch
func (d *DDSketch) GetMin() (float64, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	if d.count == 0 {
		return 0, ErrEmptySketch
	}
	
	return d.min, nil
}

// GetMax returns the maximum value added to the sketch
func (d *DDSketch) GetMax() (float64, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	if d.count == 0 {
		return 0, ErrEmptySketch
	}
	
	return d.max, nil
}

// GetSum returns the sum of all values added to the sketch
func (d *DDSketch) GetSum() (float64, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	if d.count == 0 {
		return 0, ErrEmptySketch
	}
	
	return d.sum, nil
}

// GetAvg returns the average of all values added to the sketch
func (d *DDSketch) GetAvg() (float64, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	if d.count == 0 {
		return 0, ErrEmptySketch
	}
	
	return d.sum / float64(d.count), nil
}

// Merge merges another sketch into this one
func (d *DDSketch) Merge(other Sketch) error {
	otherDD, ok := other.(*DDSketch)
	if !ok {
		return ErrIncompatibleSketches
	}
	
	// Check compatibility
	if d.gamma != otherDD.gamma {
		return fmt.Errorf("cannot merge sketches with different gamma values: %f != %f", 
			d.gamma, otherDD.gamma)
	}
	
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	otherDD.mutex.RLock()
	defer otherDD.mutex.RUnlock()
	
	// Merge store data
	d.store.Merge(otherDD.store)
	
	// Update statistics
	d.count += otherDD.count
	d.sum += otherDD.sum
	
	// Update min/max values
	if otherDD.min < d.min {
		d.min = otherDD.min
	}
	if otherDD.max > d.max {
		d.max = otherDD.max
	}
	
	// Check if we should switch store type after merge
	if d.autoSwitch {
		d.checkAndSwitchStores()
	}
	
	return nil
}

// Copy creates a deep copy of the sketch
func (d *DDSketch) Copy() Sketch {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	newDD := &DDSketch{
		gamma:        d.gamma,
		multiplier:   d.multiplier,
		offset:       d.offset,
		minValue:     d.minValue,
		maxValue:     d.maxValue,
		useSparseStore: d.useSparseStore,
		autoSwitch:   d.autoSwitch,
		switchThreshold: d.switchThreshold,
		min:          d.min,
		max:          d.max,
		sum:          d.sum,
		count:        d.count,
		startTime:    d.startTime,
		lastSwitch:   d.lastSwitch,
	}
	
	// Create fresh stores
	newDD.sparseStore = NewSparseStore(10)
	newDD.denseStore = NewDenseStore(128)
	
	// Copy the active store
	if d.useSparseStore {
		newDD.store = d.store.Copy()
		newDD.sparseStore = newDD.store
		// Initialize dense store as empty
		newDD.denseStore = NewDenseStore(128)
	} else {
		newDD.store = d.store.Copy()
		newDD.denseStore = newDD.store
		// Initialize sparse store as empty
		newDD.sparseStore = NewSparseStore(10)
	}
	
	return newDD
}

// Reset resets the sketch to an empty state
func (d *DDSketch) Reset() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	d.store.Clear()
	d.min = math.Inf(1)
	d.max = math.Inf(-1)
	d.sum = 0
	d.count = 0
	
	// Reset both store types
	d.sparseStore.Clear()
	d.denseStore.Clear()
}

// Bytes returns a serialized representation of the sketch
// Actual implementation will be in serialization.go
func (d *DDSketch) Bytes() ([]byte, error) {
	return nil, fmt.Errorf("not implemented - see serialization.go")
}

// FromBytes populates the sketch from a serialized representation
// Actual implementation will be in serialization.go
func (d *DDSketch) FromBytes(data []byte) error {
	return fmt.Errorf("not implemented - see serialization.go")
}

// Resources returns resource usage of the sketch itself
func (d *DDSketch) Resources() map[string]float64 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return map[string]float64{
		"sketch_count":          float64(d.count),
		"sketch_buckets":        float64(len(d.store.GetNonEmptyBuckets())),
		"sketch_memory_bytes":   float64(d.store.GetMemoryUsageBytes()),
		"sketch_store_density":  d.store.GetStoreDensity() * 100, // as percentage
		"sketch_uptime_seconds": time.Since(d.startTime).Seconds(),
	}
}

// valueToIndex maps a value to a bucket index
func (d *DDSketch) valueToIndex(value float64) int {
	if value <= 0 {
		// Should never happen due to validation, but just in case
		return math.MinInt32
	}
	
	// Apply logarithmic mapping
	index := int(math.Ceil(d.multiplier * math.Log(value) - d.offset))
	return index
}

// indexToValue maps a bucket index to a representative value
func (d *DDSketch) indexToValue(index int) float64 {
	value := math.Exp((float64(index) + d.offset) / d.multiplier)
	return value
}

// checkAndSwitchStores checks if we should switch between sparse and dense stores
func (d *DDSketch) checkAndSwitchStores() {
	// Only check periodically to avoid overhead
	now := time.Now()
	if now.Sub(d.lastSwitch) < time.Second {
		return
	}
	d.lastSwitch = now
	
	// Get current store density
	density := d.store.GetStoreDensity()
	
	if d.useSparseStore && density > d.switchThreshold {
		// Switch from sparse to dense
		d.denseStore.Clear()
		d.denseStore.Merge(d.store)
		d.store = d.denseStore
		d.useSparseStore = false
		fmt.Printf("AgentDiagEvent: DDSketch switched from sparse to dense store (density: %.2f%%)\n", 
			density * 100)
	} else if !d.useSparseStore && density < d.switchThreshold/2 {
		// Switch from dense to sparse
		d.sparseStore.Clear()
		d.sparseStore.Merge(d.store)
		d.store = d.sparseStore
		d.useSparseStore = true
		fmt.Printf("AgentDiagEvent: DDSketch switched from dense to sparse store (density: %.2f%%)\n", 
			density * 100)
	}
}
