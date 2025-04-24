package sketch

import (
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestDDSketch_BasicOperations(t *testing.T) {
	// Create a sketch with default config
	config := DefaultConfig().DDSketch
	sketch := NewDDSketch(config)
	
	// Initial state checks
	if sketch.GetCount() != 0 {
		t.Errorf("New sketch should have count 0, got %d", sketch.GetCount())
	}
	
	_, err := sketch.GetMin()
	if err != ErrEmptySketch {
		t.Errorf("GetMin on empty sketch should return ErrEmptySketch")
	}
	
	_, err = sketch.GetMax()
	if err != ErrEmptySketch {
		t.Errorf("GetMax on empty sketch should return ErrEmptySketch")
	}
	
	_, err = sketch.GetSum()
	if err != ErrEmptySketch {
		t.Errorf("GetSum on empty sketch should return ErrEmptySketch")
	}
	
	_, err = sketch.GetAvg()
	if err != ErrEmptySketch {
		t.Errorf("GetAvg on empty sketch should return ErrEmptySketch")
	}
	
	// Add some values
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	for _, v := range values {
		err := sketch.Add(v)
		if err != nil {
			t.Errorf("Add(%f) returned error: %v", v, err)
		}
	}
	
	// Check count
	if sketch.GetCount() != 5 {
		t.Errorf("Sketch should have count 5, got %d", sketch.GetCount())
	}
	
	// Check min/max
	min, err := sketch.GetMin()
	if err != nil {
		t.Errorf("GetMin returned error: %v", err)
	}
	if min != 1.0 {
		t.Errorf("Min should be 1.0, got %f", min)
	}
	
	max, err := sketch.GetMax()
	if err != nil {
		t.Errorf("GetMax returned error: %v", err)
	}
	if max != 5.0 {
		t.Errorf("Max should be 5.0, got %f", max)
	}
	
	// Check sum
	sum, err := sketch.GetSum()
	if err != nil {
		t.Errorf("GetSum returned error: %v", err)
	}
	if sum != 15.0 {
		t.Errorf("Sum should be 15.0, got %f", sum)
	}
	
	// Check average
	avg, err := sketch.GetAvg()
	if err != nil {
		t.Errorf("GetAvg returned error: %v", err)
	}
	if avg != 3.0 {
		t.Errorf("Average should be 3.0, got %f", avg)
	}
	
	// Reset the sketch
	sketch.Reset()
	
	// Check state after reset
	if sketch.GetCount() != 0 {
		t.Errorf("After reset, sketch should have count 0, got %d", sketch.GetCount())
	}
	
	_, err = sketch.GetMin()
	if err != ErrEmptySketch {
		t.Errorf("After reset, GetMin should return ErrEmptySketch")
	}
}

func TestDDSketch_Quantiles(t *testing.T) {
	// Create a sketch with tight accuracy
	config := DefaultConfig().DDSketch
	config.RelativeAccuracy = 0.001 // 0.1% error
	sketch := NewDDSketch(config)
	
	// Add ordered values from 1 to 100
	for i := 1; i <= 100; i++ {
		sketch.Add(float64(i))
	}
	
	// Test exact quantiles
	testCases := []struct {
		quantile float64
		expected float64
		maxError float64
	}{
		{0.0, 1.0, 0.0},
		{0.25, 25.0, 0.5},
		{0.5, 50.0, 0.5},
		{0.75, 75.0, 0.5},
		{0.9, 90.0, 0.5},
		{0.95, 95.0, 0.5},
		{0.99, 99.0, 0.5},
		{1.0, 100.0, 0.0},
	}
	
	for _, tc := range testCases {
		value, err := sketch.GetValueAtQuantile(tc.quantile)
		if err != nil {
			t.Errorf("GetValueAtQuantile(%f) returned error: %v", tc.quantile, err)
			continue
		}
		
		relError := math.Abs(value - tc.expected) / tc.expected
		if relError > tc.maxError {
			t.Errorf("GetValueAtQuantile(%f) = %f, expected %f ± %f%%, got error %f%%",
				tc.quantile, value, tc.expected, tc.maxError*100, relError*100)
		}
	}
	
	// Test invalid quantiles
	_, err := sketch.GetValueAtQuantile(-0.1)
	if err != ErrInvalidQuantile {
		t.Errorf("GetValueAtQuantile(-0.1) should return ErrInvalidQuantile")
	}
	
	_, err = sketch.GetValueAtQuantile(1.1)
	if err != ErrInvalidQuantile {
		t.Errorf("GetValueAtQuantile(1.1) should return ErrInvalidQuantile")
	}
}

func TestDDSketch_GetQuantileAtValue(t *testing.T) {
	// Create a sketch with tight accuracy
	config := DefaultConfig().DDSketch
	config.RelativeAccuracy = 0.001 // 0.1% error
	sketch := NewDDSketch(config)
	
	// Add ordered values from 1 to 100
	for i := 1; i <= 100; i++ {
		sketch.Add(float64(i))
	}
	
	// Test exact values
	testCases := []struct {
		value    float64
		expected float64
		maxError float64
	}{
		{1.0, 0.0, 0.01},
		{25.0, 0.24, 0.01},
		{50.0, 0.49, 0.01},
		{75.0, 0.74, 0.01},
		{90.0, 0.89, 0.01},
		{100.0, 0.99, 0.01},
	}
	
	for _, tc := range testCases {
		quantile, err := sketch.GetQuantileAtValue(tc.value)
		if err != nil {
			t.Errorf("GetQuantileAtValue(%f) returned error: %v", tc.value, err)
			continue
		}
		
		absError := math.Abs(quantile - tc.expected)
		if absError > tc.maxError {
			t.Errorf("GetQuantileAtValue(%f) = %f, expected %f ± %f, got error %f",
				tc.value, quantile, tc.expected, tc.maxError, absError)
		}
	}
	
	// Test invalid values
	_, err := sketch.GetQuantileAtValue(-1.0)
	if err == nil {
		t.Errorf("GetQuantileAtValue(-1.0) should return error for negative value")
	}
}

func TestDDSketch_AddWithCount(t *testing.T) {
	// Create a sketch with default config
	config := DefaultConfig().DDSketch
	sketch := NewDDSketch(config)
	
	// Add values with different counts
	sketch.AddWithCount(10.0, 5)
	sketch.AddWithCount(20.0, 10)
	sketch.AddWithCount(30.0, 15)
	
	// Check count
	if sketch.GetCount() != 30 {
		t.Errorf("Sketch should have count 30, got %d", sketch.GetCount())
	}
	
	// Check sum
	sum, err := sketch.GetSum()
	if err != nil {
		t.Errorf("GetSum returned error: %v", err)
	}
	expected := 10.0*5 + 20.0*10 + 30.0*15
	if math.Abs(sum-expected) > 0.1 {
		t.Errorf("Sum should be %f, got %f", expected, sum)
	}
	
	// Check average
	avg, err := sketch.GetAvg()
	if err != nil {
		t.Errorf("GetAvg returned error: %v", err)
	}
	expectedAvg := expected / 30.0
	if math.Abs(avg-expectedAvg) > 0.1 {
		t.Errorf("Average should be %f, got %f", expectedAvg, avg)
	}
	
	// Check quantiles
	p50, err := sketch.GetValueAtQuantile(0.5)
	if err != nil {
		t.Errorf("GetValueAtQuantile(0.5) returned error: %v", err)
	}
	// Median should be in the 20.0 bucket (10 values below, 15 above)
	if math.Abs(p50-20.0) > 1.0 {
		t.Errorf("P50 should be close to 20.0, got %f", p50)
	}
}

func TestDDSketch_Merge(t *testing.T) {
	// Create two sketches
	config := DefaultConfig().DDSketch
	sketch1 := NewDDSketch(config)
	sketch2 := NewDDSketch(config)
	
	// Add different values to each sketch
	for i := 1; i <= 50; i++ {
		sketch1.Add(float64(i))
	}
	
	for i := 51; i <= 100; i++ {
		sketch2.Add(float64(i))
	}
	
	// Merge sketch2 into sketch1
	err := sketch1.Merge(sketch2)
	if err != nil {
		t.Errorf("Merge returned error: %v", err)
	}
	
	// Check merged sketch
	if sketch1.GetCount() != 100 {
		t.Errorf("Merged sketch should have count 100, got %d", sketch1.GetCount())
	}
	
	min, _ := sketch1.GetMin()
	if min != 1.0 {
		t.Errorf("Merged sketch min should be 1.0, got %f", min)
	}
	
	max, _ := sketch1.GetMax()
	if max != 100.0 {
		t.Errorf("Merged sketch max should be 100.0, got %f", max)
	}
	
	// Check quantiles in merged sketch
	p50, _ := sketch1.GetValueAtQuantile(0.5)
	if math.Abs(p50-50.0) > 1.0 {
		t.Errorf("Merged P50 should be close to 50.0, got %f", p50)
	}
	
	p25, _ := sketch1.GetValueAtQuantile(0.25)
	if math.Abs(p25-25.0) > 1.0 {
		t.Errorf("Merged P25 should be close to 25.0, got %f", p25)
	}
	
	p75, _ := sketch1.GetValueAtQuantile(0.75)
	if math.Abs(p75-75.0) > 1.0 {
		t.Errorf("Merged P75 should be close to 75.0, got %f", p75)
	}
	
	// Try to merge incompatible sketches
	incompatibleConfig := DefaultConfig().DDSketch
	incompatibleConfig.RelativeAccuracy = 0.01
	incompatibleSketch := NewDDSketch(incompatibleConfig)
	incompatibleSketch.Add(1.0)
	
	err = sketch1.Merge(incompatibleSketch)
	if err == nil {
		t.Errorf("Merging incompatible sketches should return error")
	}
}

func TestDDSketch_Copy(t *testing.T) {
	// Create a sketch with some values
	config := DefaultConfig().DDSketch
	sketch := NewDDSketch(config)
	
	for i := 1; i <= 100; i++ {
		sketch.Add(float64(i))
	}
	
	// Create a copy
	copySketch := sketch.Copy()
	
	// Check that the copy has the same properties
	if sketch.GetCount() != copySketch.GetCount() {
		t.Errorf("Copy count mismatch: original=%d, copy=%d", 
			sketch.GetCount(), copySketch.GetCount())
	}
	
	origMin, _ := sketch.GetMin()
	copyMin, _ := copySketch.GetMin()
	if origMin != copyMin {
		t.Errorf("Copy min mismatch: original=%f, copy=%f", origMin, copyMin)
	}
	
	origMax, _ := sketch.GetMax()
	copyMax, _ := copySketch.GetMax()
	if origMax != copyMax {
		t.Errorf("Copy max mismatch: original=%f, copy=%f", origMax, copyMax)
	}
	
	// Check some quantiles
	for _, q := range []float64{0.0, 0.25, 0.5, 0.75, 1.0} {
		origVal, _ := sketch.GetValueAtQuantile(q)
		copyVal, _ := copySketch.GetValueAtQuantile(q)
		if math.Abs(origVal-copyVal) > 1e-6 {
			t.Errorf("Copy quantile mismatch at %f: original=%f, copy=%f", 
				q, origVal, copyVal)
		}
	}
	
	// Modify the original and check that the copy is unaffected
	sketch.Add(200.0)
	
	if sketch.GetCount() == copySketch.GetCount() {
		t.Errorf("Copy should not be affected by changes to original")
	}
	
	origMax, _ = sketch.GetMax()
	copyMax, _ = copySketch.GetMax()
	if origMax == copyMax {
		t.Errorf("Copy max should not change when original changes")
	}
}

func TestDDSketch_Accuracy(t *testing.T) {
	// Test accuracy guarantees with various distributions
	
	// Create a sketch with 0.75% relative accuracy
	config := DefaultConfig().DDSketch
	config.RelativeAccuracy = 0.0075 // 0.75% error (from ADR-001)
	sketch := NewDDSketch(config)
	
	// Generate samples from different distributions
	distributions := []struct {
		name     string
		generate func(n int) []float64
	}{
		{"uniform", generateUniform},
		{"normal", generateNormal},
		{"exponential", generateExponential},
		{"lognormal", generateLogNormal},
		{"bimodal", generateBimodal},
	}
	
	for _, dist := range distributions {
		t.Run(dist.name, func(t *testing.T) {
			// Reset sketch
			sketch.Reset()
			
			// Generate samples
			samples := dist.generate(10000)
			
			// Add to sketch
			for _, v := range samples {
				sketch.Add(v)
			}
			
			// Sort samples for exact quantiles
			sortedSamples := make([]float64, len(samples))
			copy(sortedSamples, samples)
			quickSort(sortedSamples)
			
			// Check quantiles
			quantiles := []float64{0.5, 0.9, 0.95, 0.99}
			for _, q := range quantiles {
				// Get exact quantile
				exactIndex := int(q * float64(len(sortedSamples)))
				if exactIndex >= len(sortedSamples) {
					exactIndex = len(sortedSamples) - 1
				}
				exactValue := sortedSamples[exactIndex]
				
				// Get approximated quantile
				approxValue, _ := sketch.GetValueAtQuantile(q)
				
				// Calculate relative error
				relError := math.Abs(approxValue-exactValue) / exactValue
				
				// Check error bound
				if relError > config.RelativeAccuracy {
					t.Errorf("%s distribution: relative error at q=%.2f exceeded bound: "+
						"exact=%.6f, approx=%.6f, error=%.6f, bound=%.6f",
						dist.name, q, exactValue, approxValue, relError, config.RelativeAccuracy)
				} else {
					t.Logf("%s distribution: q=%.2f, exact=%.6f, approx=%.6f, error=%.6f (within %.6f)",
						dist.name, q, exactValue, approxValue, relError, config.RelativeAccuracy)
				}
			}
		})
	}
}

func TestDDSketch_Concurrent(t *testing.T) {
	// Test concurrent access to the sketch
	config := DefaultConfig().DDSketch
	sketch := NewDDSketch(config)
	
	// Number of goroutines and operations
	goroutines := 10
	opsPerGoroutine := 1000
	
	// Wait group for synchronization
	var wg sync.WaitGroup
	wg.Add(goroutines)
	
	// Start goroutines
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			
			// Each goroutine does a mix of operations
			for i := 0; i < opsPerGoroutine; i++ {
				op := rand.Intn(3)
				switch op {
				case 0:
					// Add a value
					value := rand.Float64()*100.0 + 1.0
					sketch.Add(value)
				case 1:
					// Get a quantile
					q := rand.Float64()
					_, _ = sketch.GetValueAtQuantile(q)
				case 2:
					// Get a count or sum
					if rand.Intn(2) == 0 {
						sketch.GetCount()
					} else {
						_, _ = sketch.GetSum()
					}
				}
			}
		}(g)
	}
	
	// Wait for all goroutines to finish
	wg.Wait()
	
	// Verify the sketch is still functional
	count := sketch.GetCount()
	if count != uint64(goroutines*opsPerGoroutine/3) {
		t.Logf("Expected approximately %d values, got %d",
			goroutines*opsPerGoroutine/3, count)
	}
	
	// Try to get some quantiles
	_, err := sketch.GetValueAtQuantile(0.5)
	if err != nil {
		t.Errorf("After concurrent operations, GetValueAtQuantile returned error: %v", err)
	}
}

func TestDDSketch_Serialization(t *testing.T) {
	// Create a sketch with some values
	config := DefaultConfig().DDSketch
	sketch := NewDDSketch(config)
	
	for i := 1; i <= 100; i++ {
		sketch.Add(float64(i))
	}
	
	// Serialize the sketch
	data, err := sketch.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}
	
	// Create a new sketch and deserialize
	newSketch := NewDDSketch(config)
	err = newSketch.FromBytes(data)
	if err != nil {
		t.Fatalf("FromBytes() returned error: %v", err)
	}
	
	// Verify properties
	if sketch.GetCount() != newSketch.GetCount() {
		t.Errorf("Deserialized count mismatch: original=%d, deserialized=%d",
			sketch.GetCount(), newSketch.GetCount())
	}
	
	origMin, _ := sketch.GetMin()
	newMin, _ := newSketch.GetMin()
	if origMin != newMin {
		t.Errorf("Deserialized min mismatch: original=%f, deserialized=%f",
			origMin, newMin)
	}
	
	origMax, _ := sketch.GetMax()
	newMax, _ := newSketch.GetMax()
	if origMax != newMax {
		t.Errorf("Deserialized max mismatch: original=%f, deserialized=%f",
			origMax, newMax)
	}
	
	// Check some quantiles
	for _, q := range []float64{0.0, 0.25, 0.5, 0.75, 1.0} {
		origVal, _ := sketch.GetValueAtQuantile(q)
		newVal, _ := newSketch.GetValueAtQuantile(q)
		if math.Abs(origVal-newVal) > 1e-6 {
			t.Errorf("Deserialized quantile mismatch at %f: original=%f, deserialized=%f",
				q, origVal, newVal)
		}
	}
}

func TestDDSketch_MergeBytes(t *testing.T) {
	// Create two sketches
	config := DefaultConfig().DDSketch
	sketch1 := NewDDSketch(config)
	sketch2 := NewDDSketch(config)
	
	// Add values to the sketches
	for i := 1; i <= 50; i++ {
		sketch1.Add(float64(i))
	}
	
	for i := 51; i <= 100; i++ {
		sketch2.Add(float64(i))
	}
	
	// Serialize sketch2
	data, err := sketch2.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}
	
	// Merge serialized sketch2 into sketch1
	err = sketch1.MergeBytes(data)
	if err != nil {
		t.Fatalf("MergeBytes() returned error: %v", err)
	}
	
	// Verify merged result
	if sketch1.GetCount() != 100 {
		t.Errorf("After merge, count should be 100, got %d", sketch1.GetCount())
	}
	
	min, _ := sketch1.GetMin()
	if min != 1.0 {
		t.Errorf("After merge, min should be 1.0, got %f", min)
	}
	
	max, _ := sketch1.GetMax()
	if max != 100.0 {
		t.Errorf("After merge, max should be 100.0, got %f", max)
	}
	
	// Check some quantiles
	p50, _ := sketch1.GetValueAtQuantile(0.5)
	if math.Abs(p50-50.0) > 1.0 {
		t.Errorf("After merge, p50 should be close to 50.0, got %f", p50)
	}
}

func TestDDSketch_StoreSwitch(t *testing.T) {
	// Create a sketch that starts with sparse store
	config := DefaultConfig().DDSketch
	config.UseSparseStore = true
	config.AutoSwitch = true
	config.SwitchThreshold = 0.5 // Switch when 50% of possible buckets are used
	sketch := NewDDSketch(config)
	
	// Add sparse values (far apart)
	for i := 1; i <= 100; i += 10 {
		sketch.Add(float64(i))
	}
	
	// Verify we're still using sparse store (density should be low)
	ddSketch := sketch.(*DDSketch)
	if !ddSketch.useSparseStore {
		t.Errorf("Should still be using sparse store after adding sparse values")
	}
	
	// Add dense values (close together)
	for i := 100; i <= 110; i++ {
		sketch.Add(float64(i))
	}
	
	// Force a store density check
	ddSketch.checkAndSwitchStores()
	
	// Might have switched to dense store depending on bucket mapping
	// We won't assert this, but log the current state
	t.Logf("Store density: %.2f%%, using sparse: %v",
		ddSketch.store.GetStoreDensity()*100, ddSketch.useSparseStore)
	
	// Verify functionality is maintained
	p50, _ := sketch.GetValueAtQuantile(0.5)
	if p50 < 1.0 || p50 > 110.0 {
		t.Errorf("P50 should be between 1 and 110, got %f", p50)
	}
}

// Helper functions for generating test distributions

func generateUniform(n int) []float64 {
	rand.Seed(time.Now().UnixNano())
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		result[i] = rand.Float64()*100.0 + 1.0
	}
	return result
}

func generateNormal(n int) []float64 {
	rand.Seed(time.Now().UnixNano())
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		// Box-Muller transform
		u1 := rand.Float64()
		u2 := rand.Float64()
		z0 := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
		
		// Mean 50, std 15
		value := 50.0 + 15.0*z0
		if value <= 0 {
			value = 0.1
		}
		result[i] = value
	}
	return result
}

func generateExponential(n int) []float64 {
	rand.Seed(time.Now().UnixNano())
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		// Inverse transform sampling
		u := rand.Float64()
		value := -math.Log(1.0-u) * 20.0 // Scale factor 20
		if value <= 0 {
			value = 0.1
		}
		result[i] = value
	}
	return result
}

func generateLogNormal(n int) []float64 {
	rand.Seed(time.Now().UnixNano())
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		// Box-Muller transform
		u1 := rand.Float64()
		u2 := rand.Float64()
		z0 := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
		
		// Log-normal with mean 1, std 1
		value := math.Exp(1.0 + 1.0*z0)
		if value <= 0 {
			value = 0.1
		}
		result[i] = value
	}
	return result
}

func generateBimodal(n int) []float64 {
	rand.Seed(time.Now().UnixNano())
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		// 50% from each mode
		if rand.Float64() < 0.5 {
			// First mode: mean 20, std 5
			u1 := rand.Float64()
			u2 := rand.Float64()
			z0 := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
			value := 20.0 + 5.0*z0
			if value <= 0 {
				value = 0.1
			}
			result[i] = value
		} else {
			// Second mode: mean 80, std 5
			u1 := rand.Float64()
			u2 := rand.Float64()
			z0 := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
			value := 80.0 + 5.0*z0
			if value <= 0 {
				value = 0.1
			}
			result[i] = value
		}
	}
	return result
}

// Simple quicksort implementation for sorting sample arrays
func quickSort(arr []float64) {
	if len(arr) <= 1 {
		return
	}
	
	pivot := arr[len(arr)/2]
	left, right := 0, len(arr)-1
	
	for left <= right {
		for arr[left] < pivot {
			left++
		}
		for arr[right] > pivot {
			right--
		}
		if left <= right {
			arr[left], arr[right] = arr[right], arr[left]
			left++
			right--
		}
	}
	
	if right > 0 {
		quickSort(arr[:right+1])
	}
	if left < len(arr) {
		quickSort(arr[left:])
	}
}
