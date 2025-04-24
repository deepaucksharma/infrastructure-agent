package sketch

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// Magic bytes to identify serialized DDSketch
var ddSketchMagic = [4]byte{'D', 'D', 'S', 'K'}

// Current serialization version
const serializationVersion = uint8(1)

// Serialization flags
const (
	flagSparseStore = 1 << 0
	flagHasMin      = 1 << 1
	flagHasMax      = 1 << 2
	flagHasSum      = 1 << 3
)

// SerializedSketch represents a serialized sketch
type SerializedSketch struct {
	// Header
	Magic     [4]byte // Magic bytes: "DDSK"
	Version   uint8   // Serialization version
	Flags     uint8   // Feature flags
	
	// Parameters
	Gamma     float64 // Relative accuracy
	MinValue  float64 // Minimum allowed value
	MaxValue  float64 // Maximum allowed value
	
	// Statistics
	Count     uint64  // Total count
	Min       float64 // Minimum value (if flag set)
	Max       float64 // Maximum value (if flag set)
	Sum       float64 // Sum of values (if flag set)
	
	// Buckets
	NumBuckets uint32             // Number of buckets
	Buckets    map[int32]uint64   // Bucket index -> count
}

// Bytes returns a serialized representation of the DDSketch
func (d *DDSketch) Bytes() ([]byte, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	// Prepare the serialized sketch
	var flags uint8
	if d.useSparseStore {
		flags |= flagSparseStore
	}
	if !math.IsInf(d.min, 1) {
		flags |= flagHasMin
	}
	if !math.IsInf(d.max, -1) {
		flags |= flagHasMax
	}
	if d.sum > 0 {
		flags |= flagHasSum
	}
	
	// Get non-empty buckets
	buckets := d.store.GetNonEmptyBuckets()
	numBuckets := len(buckets)
	
	// Create buffer for serialization
	buf := bytes.NewBuffer(make([]byte, 0, 128+numBuckets*12))
	
	// Write header
	buf.Write(ddSketchMagic[:])
	buf.WriteByte(serializationVersion)
	buf.WriteByte(flags)
	
	// Write parameters
	binary.Write(buf, binary.LittleEndian, d.gamma)
	binary.Write(buf, binary.LittleEndian, d.minValue)
	binary.Write(buf, binary.LittleEndian, d.maxValue)
	
	// Write statistics
	binary.Write(buf, binary.LittleEndian, d.count)
	
	if flags&flagHasMin != 0 {
		binary.Write(buf, binary.LittleEndian, d.min)
	}
	if flags&flagHasMax != 0 {
		binary.Write(buf, binary.LittleEndian, d.max)
	}
	if flags&flagHasSum != 0 {
		binary.Write(buf, binary.LittleEndian, d.sum)
	}
	
	// Write buckets
	binary.Write(buf, binary.LittleEndian, uint32(numBuckets))
	
	for idx, count := range buckets {
		binary.Write(buf, binary.LittleEndian, int32(idx))
		binary.Write(buf, binary.LittleEndian, count)
	}
	
	return buf.Bytes(), nil
}

// FromBytes populates the DDSketch from a serialized representation
func (d *DDSketch) FromBytes(data []byte) error {
	if len(data) < 4+1+1+8+8+8+8 { // Magic + Version + Flags + Gamma + MinValue + MaxValue + Count
		return fmt.Errorf("data too short for DDSketch header")
	}
	
	// Lock for modification
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	// Reset sketch
	d.Reset()
	
	// Read from buffer
	buf := bytes.NewBuffer(data)
	
	// Check magic bytes
	magic := [4]byte{}
	buf.Read(magic[:])
	if magic != ddSketchMagic {
		return fmt.Errorf("invalid magic bytes: expected DDSK")
	}
	
	// Read version
	version, _ := buf.ReadByte()
	if version != serializationVersion {
		return fmt.Errorf("unsupported version: %d", version)
	}
	
	// Read flags
	flags, _ := buf.ReadByte()
	useSparseStore := flags&flagSparseStore != 0
	hasMin := flags&flagHasMin != 0
	hasMax := flags&flagHasMax != 0
	hasSum := flags&flagHasSum != 0
	
	// Read parameters
	binary.Read(buf, binary.LittleEndian, &d.gamma)
	binary.Read(buf, binary.LittleEndian, &d.minValue)
	binary.Read(buf, binary.LittleEndian, &d.maxValue)
	
	// Recalculate multiplier and offset
	d.multiplier = 1.0 / math.Log1p(d.gamma)
	d.offset = 0
	
	// Read statistics
	binary.Read(buf, binary.LittleEndian, &d.count)
	
	if hasMin {
		binary.Read(buf, binary.LittleEndian, &d.min)
	} else {
		d.min = math.Inf(1)
	}
	
	if hasMax {
		binary.Read(buf, binary.LittleEndian, &d.max)
	} else {
		d.max = math.Inf(-1)
	}
	
	if hasSum {
		binary.Read(buf, binary.LittleEndian, &d.sum)
	} else {
		d.sum = 0
	}
	
	// Read buckets
	var numBuckets uint32
	binary.Read(buf, binary.LittleEndian, &numBuckets)
	
	// Choose store type
	if useSparseStore {
		d.store = d.sparseStore
		d.useSparseStore = true
	} else {
		d.store = d.denseStore
		d.useSparseStore = false
	}
	
	// Clear store
	d.store.Clear()
	
	// Read buckets
	for i := uint32(0); i < numBuckets; i++ {
		var idx int32
		var count uint64
		binary.Read(buf, binary.LittleEndian, &idx)
		binary.Read(buf, binary.LittleEndian, &count)
		d.store.Add(int(idx), count)
	}
	
	return nil
}

// MergeBytes merges a serialized sketch into this sketch
func (d *DDSketch) MergeBytes(data []byte) error {
	// Create temporary sketch
	temp := NewDDSketch(DefaultConfig().DDSketch)
	
	// Deserialize into temporary sketch
	if err := temp.FromBytes(data); err != nil {
		return err
	}
	
	// Merge temporary sketch into this one
	return d.Merge(temp)
}

// SerializeSlice serializes multiple sketches into a single byte slice
func SerializeSlice(sketches []Sketch) ([]byte, error) {
	// Create buffer
	buf := bytes.NewBuffer(nil)
	
	// Write number of sketches
	binary.Write(buf, binary.LittleEndian, uint32(len(sketches)))
	
	// Serialize each sketch
	for _, sketch := range sketches {
		ddsketch, ok := sketch.(*DDSketch)
		if !ok {
			return nil, fmt.Errorf("only DDSketch is supported")
		}
		
		// Serialize sketch
		data, err := ddsketch.Bytes()
		if err != nil {
			return nil, err
		}
		
		// Write size and data
		binary.Write(buf, binary.LittleEndian, uint32(len(data)))
		buf.Write(data)
	}
	
	return buf.Bytes(), nil
}

// DeserializeSlice deserializes multiple sketches from a single byte slice
func DeserializeSlice(data []byte) ([]Sketch, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("data too short for slice header")
	}
	
	// Read from buffer
	buf := bytes.NewBuffer(data)
	
	// Read number of sketches
	var numSketches uint32
	binary.Read(buf, binary.LittleEndian, &numSketches)
	
	// Deserialize each sketch
	sketches := make([]Sketch, numSketches)
	for i := uint32(0); i < numSketches; i++ {
		// Read size
		var size uint32
		binary.Read(buf, binary.LittleEndian, &size)
		
		// Read data
		data := make([]byte, size)
		buf.Read(data)
		
		// Deserialize sketch
		sketch := NewDDSketch(DefaultConfig().DDSketch)
		if err := sketch.FromBytes(data); err != nil {
			return nil, err
		}
		
		sketches[i] = sketch
	}
	
	return sketches, nil
}
