package sketch

import (
	"math"
	"sync"
)

// Store is an interface for bucket storage to be used with DDSketch
type Store interface {
	// Add increments the count for the bin at the given index
	Add(index int, count uint64)
	
	// Get returns the count for the bin at the given index
	Get(index int) uint64
	
	// Clear resets the store to an empty state
	Clear()
	
	// GetNonEmptyBuckets returns a map of non-empty bucket indices to counts
	GetNonEmptyBuckets() map[int]uint64
	
	// GetTotalCount returns the sum of counts across all buckets
	GetTotalCount() uint64
	
	// GetMinIndex returns the minimum index with a non-zero count
	GetMinIndex() (int, bool)
	
	// GetMaxIndex returns the maximum index with a non-zero count
	GetMaxIndex() (int, bool)
	
	// Merge merges another store into this one
	Merge(other Store)
	
	// Copy creates a deep copy of the store
	Copy() Store
	
	// GetStoreDensity returns the density of the store (filled/capacity)
	GetStoreDensity() float64
	
	// GetMemoryUsageBytes returns an estimate of memory usage in bytes
	GetMemoryUsageBytes() int64
}

// SparseStore is a memory-efficient implementation of Store using a map
// It's best suited for sparse distributions where most buckets are empty
type SparseStore struct {
	bins            map[int]uint64
	count           uint64
	minIndex        int
	maxIndex        int
	hasElements     bool
	collapseThreshold uint64
	mu              sync.RWMutex
}

// NewSparseStore creates a new sparse store
func NewSparseStore(collapseThreshold uint64) *SparseStore {
	return &SparseStore{
		bins:             make(map[int]uint64),
		collapseThreshold: collapseThreshold,
		minIndex:         math.MaxInt32,
		maxIndex:         math.MinInt32,
	}
}

// Add increments the count for the bin at the given index
func (s *SparseStore) Add(index int, count uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.bins[index] += count
	s.count += count
	
	// Update min/max indices
	if index < s.minIndex {
		s.minIndex = index
	}
	if index > s.maxIndex {
		s.maxIndex = index
	}
	
	s.hasElements = true
	
	// Periodically collapse low-count buckets to save memory
	if len(s.bins) > 1000 { // Only check when we have lots of buckets
		s.collapseBuckets()
	}
}

// Get returns the count for the bin at the given index
func (s *SparseStore) Get(index int) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bins[index]
}

// Clear resets the store to an empty state
func (s *SparseStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.bins = make(map[int]uint64)
	s.count = 0
	s.minIndex = math.MaxInt32
	s.maxIndex = math.MinInt32
	s.hasElements = false
}

// GetNonEmptyBuckets returns a map of non-empty bucket indices to counts
func (s *SparseStore) GetNonEmptyBuckets() map[int]uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Create a copy to avoid concurrent map access
	buckets := make(map[int]uint64, len(s.bins))
	for idx, count := range s.bins {
		buckets[idx] = count
	}
	return buckets
}

// GetTotalCount returns the sum of counts across all buckets
func (s *SparseStore) GetTotalCount() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count
}

// GetMinIndex returns the minimum index with a non-zero count
func (s *SparseStore) GetMinIndex() (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if !s.hasElements {
		return 0, false
	}
	return s.minIndex, true
}

// GetMaxIndex returns the maximum index with a non-zero count
func (s *SparseStore) GetMaxIndex() (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if !s.hasElements {
		return 0, false
	}
	return s.maxIndex, true
}

// Merge merges another store into this one
func (s *SparseStore) Merge(other Store) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Get all non-empty buckets from the other store
	otherBuckets := other.GetNonEmptyBuckets()
	
	// Merge each bucket
	for idx, count := range otherBuckets {
		s.bins[idx] += count
		
		// Update min/max indices
		if idx < s.minIndex {
			s.minIndex = idx
		}
		if idx > s.maxIndex {
			s.maxIndex = idx
		}
	}
	
	// Update total count
	s.count += other.GetTotalCount()
	
	if s.count > 0 {
		s.hasElements = true
	}
	
	// Collapse buckets if we merged a lot
	if len(otherBuckets) > 100 {
		s.collapseBuckets()
	}
}

// Copy creates a deep copy of the store
func (s *SparseStore) Copy() Store {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	newStore := NewSparseStore(s.collapseThreshold)
	newStore.count = s.count
	newStore.minIndex = s.minIndex
	newStore.maxIndex = s.maxIndex
	newStore.hasElements = s.hasElements
	
	// Copy all bins
	for idx, count := range s.bins {
		newStore.bins[idx] = count
	}
	
	return newStore
}

// GetStoreDensity returns the density of the store (filled/capacity)
// For sparse store, this is the fraction of possible indices between
// min and max that are actually filled
func (s *SparseStore) GetStoreDensity() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if !s.hasElements || s.maxIndex <= s.minIndex {
		return 0
	}
	
	// Calculate possible range of indices
	range_ := s.maxIndex - s.minIndex + 1
	
	// Calculate density
	return float64(len(s.bins)) / float64(range_)
}

// GetMemoryUsageBytes returns an estimate of memory usage in bytes
func (s *SparseStore) GetMemoryUsageBytes() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Rough estimate: 
	// - map overhead: 48 bytes
	// - each entry: 16 bytes (key) + 8 bytes (value) + 16 bytes (entry overhead)
	// - other fields: 8 bytes each
	
	mapOverhead := int64(48)
	entriesSize := int64(len(s.bins)) * int64(16+8+16)
	otherFields := int64(8 * 5) // count, minIndex, maxIndex, hasElements, collapseThreshold
	
	return mapOverhead + entriesSize + otherFields
}

// collapseBuckets combines adjacent low-count buckets to save memory
func (s *SparseStore) collapseBuckets() {
	// Only process if we have enough buckets to make it worthwhile
	if len(s.bins) < 10 {
		return
	}
	
	toRemove := make([]int, 0)
	
	// Find buckets with count below threshold
	for idx, count := range s.bins {
		if count <= s.collapseThreshold && idx != s.minIndex && idx != s.maxIndex {
			toRemove = append(toRemove, idx)
		}
	}
	
	// Remove low-count buckets and redistribute their counts
	for _, idx := range toRemove {
		count := s.bins[idx]
		delete(s.bins, idx)
		
		// Find nearest non-empty buckets
		lowerIdx, upperIdx := math.MinInt32, math.MaxInt32
		for bucketIdx := range s.bins {
			if bucketIdx < idx && bucketIdx > lowerIdx {
				lowerIdx = bucketIdx
			}
			if bucketIdx > idx && bucketIdx < upperIdx {
				upperIdx = bucketIdx
			}
		}
		
		// Redistribute count to nearest buckets
		if lowerIdx != math.MinInt32 && upperIdx != math.MaxInt32 {
			// We have both lower and upper buckets, split count
			lowerDist := idx - lowerIdx
			upperDist := upperIdx - idx
			totalDist := lowerDist + upperDist
			
			lowerPortion := float64(upperDist) / float64(totalDist)
			upperPortion := float64(lowerDist) / float64(totalDist)
			
			s.bins[lowerIdx] += uint64(float64(count) * lowerPortion)
			s.bins[upperIdx] += uint64(float64(count) * upperPortion)
		} else if lowerIdx != math.MinInt32 {
			// Only have lower bucket
			s.bins[lowerIdx] += count
		} else if upperIdx != math.MaxInt32 {
			// Only have upper bucket
			s.bins[upperIdx] += count
		} else {
			// Shouldn't happen, but just in case
			// Put count back in index
			s.bins[idx] = count
		}
	}
	
	// Recalculate min/max indices
	s.recalculateMinMax()
}

// recalculateMinMax updates the min and max indices
func (s *SparseStore) recalculateMinMax() {
	if len(s.bins) == 0 {
		s.minIndex = math.MaxInt32
		s.maxIndex = math.MinInt32
		s.hasElements = false
		return
	}
	
	s.minIndex = math.MaxInt32
	s.maxIndex = math.MinInt32
	
	for idx := range s.bins {
		if idx < s.minIndex {
			s.minIndex = idx
		}
		if idx > s.maxIndex {
			s.maxIndex = idx
		}
	}
	
	s.hasElements = true
}

// DenseStore is a memory-efficient implementation of Store using arrays
// It's best suited for dense distributions with a narrow range of values
type DenseStore struct {
	bins       []uint64
	count      uint64
	offset     int
	minIndex   int
	maxIndex   int
	hasElements bool
	mu         sync.RWMutex
}

// NewDenseStore creates a new dense store
func NewDenseStore(initialCapacity int) *DenseStore {
	return &DenseStore{
		bins:       make([]uint64, initialCapacity),
		offset:     0,
		minIndex:   math.MaxInt32,
		maxIndex:   math.MinInt32,
	}
}

// Add increments the count for the bin at the given index
func (d *DenseStore) Add(index int, count uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Ensure the index is within the array bounds
	d.ensureCapacity(index)
	
	// Calculate array index
	arrIdx := index - d.offset
	
	// Increment count
	d.bins[arrIdx] += count
	d.count += count
	
	// Update min/max indices
	if index < d.minIndex {
		d.minIndex = index
	}
	if index > d.maxIndex {
		d.maxIndex = index
	}
	
	d.hasElements = true
}

// Get returns the count for the bin at the given index
func (d *DenseStore) Get(index int) uint64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// Check if index is within bounds
	arrIdx := index - d.offset
	if arrIdx < 0 || arrIdx >= len(d.bins) {
		return 0
	}
	
	return d.bins[arrIdx]
}

// Clear resets the store to an empty state
func (d *DenseStore) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Reset all bins to zero
	for i := range d.bins {
		d.bins[i] = 0
	}
	
	d.count = 0
	d.minIndex = math.MaxInt32
	d.maxIndex = math.MinInt32
	d.hasElements = false
}

// GetNonEmptyBuckets returns a map of non-empty bucket indices to counts
func (d *DenseStore) GetNonEmptyBuckets() map[int]uint64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	buckets := make(map[int]uint64)
	for i, count := range d.bins {
		if count > 0 {
			buckets[i+d.offset] = count
		}
	}
	return buckets
}

// GetTotalCount returns the sum of counts across all buckets
func (d *DenseStore) GetTotalCount() uint64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.count
}

// GetMinIndex returns the minimum index with a non-zero count
func (d *DenseStore) GetMinIndex() (int, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	if !d.hasElements {
		return 0, false
	}
	return d.minIndex, true
}

// GetMaxIndex returns the maximum index with a non-zero count
func (d *DenseStore) GetMaxIndex() (int, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	if !d.hasElements {
		return 0, false
	}
	return d.maxIndex, true
}

// Merge merges another store into this one
func (d *DenseStore) Merge(other Store) {
	// Get all non-empty buckets from the other store
	otherBuckets := other.GetNonEmptyBuckets()
	
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Ensure capacity for all indices
	if len(otherBuckets) > 0 {
		minIdx := math.MaxInt32
		maxIdx := math.MinInt32
		
		for idx := range otherBuckets {
			if idx < minIdx {
				minIdx = idx
			}
			if idx > maxIdx {
				maxIdx = idx
			}
		}
		
		// Ensure we have capacity for all indices
		d.ensureCapacity(minIdx)
		d.ensureCapacity(maxIdx)
	}
	
	// Merge each bucket
	for idx, count := range otherBuckets {
		arrIdx := idx - d.offset
		if arrIdx >= 0 && arrIdx < len(d.bins) {
			d.bins[arrIdx] += count
			
			// Update min/max indices
			if idx < d.minIndex {
				d.minIndex = idx
			}
			if idx > d.maxIndex {
				d.maxIndex = idx
			}
		}
	}
	
	// Update total count
	d.count += other.GetTotalCount()
	
	if d.count > 0 {
		d.hasElements = true
	}
}

// Copy creates a deep copy of the store
func (d *DenseStore) Copy() Store {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	newStore := &DenseStore{
		bins:        make([]uint64, len(d.bins)),
		offset:      d.offset,
		count:       d.count,
		minIndex:    d.minIndex,
		maxIndex:    d.maxIndex,
		hasElements: d.hasElements,
	}
	
	// Copy all bins
	copy(newStore.bins, d.bins)
	
	return newStore
}

// GetStoreDensity returns the density of the store (filled/capacity)
// For dense store, this is the fraction of array elements that are non-zero
func (d *DenseStore) GetStoreDensity() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	if len(d.bins) == 0 {
		return 0
	}
	
	nonEmptyCount := 0
	for _, count := range d.bins {
		if count > 0 {
			nonEmptyCount++
		}
	}
	
	return float64(nonEmptyCount) / float64(len(d.bins))
}

// GetMemoryUsageBytes returns an estimate of memory usage in bytes
func (d *DenseStore) GetMemoryUsageBytes() int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// Rough estimate: 
	// - array overhead: 24 bytes
	// - array elements: 8 bytes each
	// - other fields: 8 bytes each
	
	arrayOverhead := int64(24)
	elementsSize := int64(len(d.bins)) * int64(8)
	otherFields := int64(8 * 5) // count, offset, minIndex, maxIndex, hasElements
	
	return arrayOverhead + elementsSize + otherFields
}

// ensureCapacity ensures the store has capacity for the given index
func (d *DenseStore) ensureCapacity(index int) {
	// If array is empty, initialize with index as the offset
	if len(d.bins) == 0 {
		d.bins = make([]uint64, 1)
		d.offset = index
		return
	}
	
	// Calculate array index
	arrIdx := index - d.offset
	
	// If index is out of bounds, resize the array
	if arrIdx < 0 {
		// Need to expand to the left
		newBins := make([]uint64, len(d.bins)-arrIdx)
		copy(newBins[-arrIdx:], d.bins)
		d.bins = newBins
		d.offset = index
	} else if arrIdx >= len(d.bins) {
		// Need to expand to the right
		newSize := arrIdx + 1
		newBins := make([]uint64, newSize)
		copy(newBins, d.bins)
		d.bins = newBins
	}
}
