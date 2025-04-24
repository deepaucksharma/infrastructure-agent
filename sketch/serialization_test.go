package sketch

import (
	"bytes"
	"testing"
)

func TestSerialization_Basic(t *testing.T) {
	// Create a sketch with some data
	config := DefaultConfig().DDSketch
	sketch := NewDDSketch(config)
	
	for i := 1; i <= 100; i++ {
		sketch.Add(float64(i))
	}
	
	// Serialize
	data, err := sketch.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}
	
	// Check serialized data length
	if len(data) < 50 {
		t.Errorf("Serialized data too short: %d bytes", len(data))
	}
	
	// Check magic bytes
	if !bytes.HasPrefix(data, []byte("DDSK")) {
		t.Errorf("Missing magic bytes prefix")
	}
	
	// Create a new sketch and deserialize
	newSketch := NewDDSketch(config)
	err = newSketch.FromBytes(data)
	if err != nil {
		t.Fatalf("FromBytes() returned error: %v", err)
	}
	
	// Compare original and deserialized
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
	
	origSum, _ := sketch.GetSum()
	newSum, _ := newSketch.GetSum()
	if origSum != newSum {
		t.Errorf("Deserialized sum mismatch: original=%f, deserialized=%f",
			origSum, newSum)
	}
	
	// Compare quantiles
	quantiles := []float64{0.5, 0.9, 0.95, 0.99}
	for _, q := range quantiles {
		origVal, _ := sketch.GetValueAtQuantile(q)
		newVal, _ := newSketch.GetValueAtQuantile(q)
		if origVal != newVal {
			t.Errorf("Deserialized quantile mismatch at q=%f: original=%f, deserialized=%f",
				q, origVal, newVal)
		}
	}
}

func TestSerialization_EmptySketch(t *testing.T) {
	// Create an empty sketch
	config := DefaultConfig().DDSketch
	sketch := NewDDSketch(config)
	
	// Serialize
	data, err := sketch.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error for empty sketch: %v", err)
	}
	
	// Create a new sketch and deserialize
	newSketch := NewDDSketch(config)
	err = newSketch.FromBytes(data)
	if err != nil {
		t.Fatalf("FromBytes() returned error for empty sketch: %v", err)
	}
	
	// Check count
	if newSketch.GetCount() != 0 {
		t.Errorf("Deserialized empty sketch should have count 0, got %d", 
			newSketch.GetCount())
	}
	
	// GetMin should return error
	_, err = newSketch.GetMin()
	if err != ErrEmptySketch {
		t.Errorf("GetMin() on deserialized empty sketch should return ErrEmptySketch")
	}
}

func TestSerialization_SparseStore(t *testing.T) {
	// Create a sketch with sparse store
	config := DefaultConfig().DDSketch
	config.UseSparseStore = true
	sketch := NewDDSketch(config)
	
	// Add sparse data
	for i := 1; i <= 100; i += 10 {
		sketch.Add(float64(i))
	}
	
	// Serialize
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
	
	// Check if using sparse store
	ddSketch := newSketch.(*DDSketch)
	if !ddSketch.useSparseStore {
		t.Errorf("Deserialized sketch should be using sparse store")
	}
	
	// Check values
	if newSketch.GetCount() != 10 {
		t.Errorf("Deserialized count should be 10, got %d", newSketch.GetCount())
	}
	
	p50, _ := newSketch.GetValueAtQuantile(0.5)
	if p50 < 40 || p50 > 60 {
		t.Errorf("p50 should be around 50, got %f", p50)
	}
}

func TestSerialization_DenseStore(t *testing.T) {
	// Create a sketch with dense store
	config := DefaultConfig().DDSketch
	config.UseSparseStore = false
	sketch := NewDDSketch(config)
	
	// Add data
	for i := 1; i <= 100; i++ {
		sketch.Add(float64(i))
	}
	
	// Serialize
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
	
	// Check if using dense store
	ddSketch := newSketch.(*DDSketch)
	if ddSketch.useSparseStore {
		t.Errorf("Deserialized sketch should be using dense store")
	}
	
	// Check values
	if newSketch.GetCount() != 100 {
		t.Errorf("Deserialized count should be 100, got %d", newSketch.GetCount())
	}
	
	p50, _ := newSketch.GetValueAtQuantile(0.5)
	if p50 < 45 || p50 > 55 {
		t.Errorf("p50 should be around 50, got %f", p50)
	}
}

func TestSerialization_InvalidData(t *testing.T) {
	// Create a sketch
	config := DefaultConfig().DDSketch
	sketch := NewDDSketch(config)
	
	// Try to deserialize invalid data
	err := sketch.FromBytes([]byte("invalid data"))
	if err == nil {
		t.Errorf("FromBytes() should return error for invalid data")
	}
	
	// Try to deserialize data with invalid magic bytes
	invalidMagic := []byte("XXSK123456789012345678901234567890")
	err = sketch.FromBytes(invalidMagic)
	if err == nil {
		t.Errorf("FromBytes() should return error for invalid magic bytes")
	}
	
	// Try to deserialize data with invalid version
	data, _ := sketch.Bytes()
	data[4] = 99 // Change version byte
	err = sketch.FromBytes(data)
	if err == nil {
		t.Errorf("FromBytes() should return error for invalid version")
	}
}

func TestSerialization_SerializeSlice(t *testing.T) {
	// Create multiple sketches
	config := DefaultConfig().DDSketch
	sketch1 := NewDDSketch(config)
	sketch2 := NewDDSketch(config)
	sketch3 := NewDDSketch(config)
	
	// Add different data to each
	for i := 1; i <= 10; i++ {
		sketch1.Add(float64(i))
	}
	
	for i := 11; i <= 20; i++ {
		sketch2.Add(float64(i))
	}
	
	for i := 21; i <= 30; i++ {
		sketch3.Add(float64(i))
	}
	
	// Serialize the slice
	sketches := []Sketch{sketch1, sketch2, sketch3}
	data, err := SerializeSlice(sketches)
	if err != nil {
		t.Fatalf("SerializeSlice() returned error: %v", err)
	}
	
	// Deserialize the slice
	deserializedSketches, err := DeserializeSlice(data)
	if err != nil {
		t.Fatalf("DeserializeSlice() returned error: %v", err)
	}
	
	// Check number of sketches
	if len(deserializedSketches) != 3 {
		t.Errorf("Expected 3 deserialized sketches, got %d", len(deserializedSketches))
	}
	
	// Check each sketch
	for i, sketch := range deserializedSketches {
		if sketch.GetCount() != 10 {
			t.Errorf("Sketch %d: expected count 10, got %d", i, sketch.GetCount())
		}
		
		min, _ := sketch.GetMin()
		max, _ := sketch.GetMax()
		
		expectedMin := float64(i*10 + 1)
		expectedMax := float64(i*10 + 10)
		
		if min != expectedMin {
			t.Errorf("Sketch %d: expected min %f, got %f", i, expectedMin, min)
		}
		
		if max != expectedMax {
			t.Errorf("Sketch %d: expected max %f, got %f", i, expectedMax, max)
		}
	}
}

func TestSerialization_MergeBytes(t *testing.T) {
	// Create two sketches
	config := DefaultConfig().DDSketch
	sketch1 := NewDDSketch(config)
	sketch2 := NewDDSketch(config)
	
	// Add different data
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
	
	// Merge bytes into sketch1
	err = sketch1.MergeBytes(data)
	if err != nil {
		t.Fatalf("MergeBytes() returned error: %v", err)
	}
	
	// Check merged result
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
	
	// Check quantiles
	p50, _ := sketch1.GetValueAtQuantile(0.5)
	if p50 < 45 || p50 > 55 {
		t.Errorf("After merge, p50 should be around 50, got %f", p50)
	}
}

func BenchmarkSerialization_Bytes(b *testing.B) {
	// Create a sketch with some data
	config := DefaultConfig().DDSketch
	sketch := NewDDSketch(config)
	
	for i := 1; i <= 1000; i++ {
		sketch.Add(float64(i))
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sketch.Bytes()
	}
}

func BenchmarkSerialization_FromBytes(b *testing.B) {
	// Create a sketch with some data
	config := DefaultConfig().DDSketch
	sketch := NewDDSketch(config)
	
	for i := 1; i <= 1000; i++ {
		sketch.Add(float64(i))
	}
	
	// Serialize
	data, _ := sketch.Bytes()
	
	// Create a new sketch for deserialization
	newSketch := NewDDSketch(config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = newSketch.FromBytes(data)
	}
}
