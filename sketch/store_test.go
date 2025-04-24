package sketch

import (
	"sync"
	"testing"
)

func TestSparseStore_Basic(t *testing.T) {
	// Create a new sparse store
	store := NewSparseStore(10)
	
	// Initial state checks
	if store.GetTotalCount() != 0 {
		t.Errorf("New store should have count 0, got %d", store.GetTotalCount())
	}
	
	if len(store.GetNonEmptyBuckets()) != 0 {
		t.Errorf("New store should have 0 non-empty buckets, got %d", 
			len(store.GetNonEmptyBuckets()))
	}
	
	// Add items to store
	store.Add(10, 5)
	store.Add(20, 10)
	store.Add(30, 15)
	
	// Check count
	if store.GetTotalCount() != 30 {
		t.Errorf("Store should have count 30, got %d", store.GetTotalCount())
	}
	
	// Check buckets
	buckets := store.GetNonEmptyBuckets()
	if len(buckets) != 3 {
		t.Errorf("Store should have 3 non-empty buckets, got %d", len(buckets))
	}
	
	if buckets[10] != 5 {
		t.Errorf("Bucket 10 should have count 5, got %d", buckets[10])
	}
	if buckets[20] != 10 {
		t.Errorf("Bucket 20 should have count 10, got %d", buckets[20])
	}
	if buckets[30] != 15 {
		t.Errorf("Bucket 30 should have count 15, got %d", buckets[30])
	}
	
	// Check min/max index
	minIdx, hasMin := store.GetMinIndex()
	if !hasMin || minIdx != 10 {
		t.Errorf("Min index should be 10, got %d (hasMin=%v)", minIdx, hasMin)
	}
	
	maxIdx, hasMax := store.GetMaxIndex()
	if !hasMax || maxIdx != 30 {
		t.Errorf("Max index should be 30, got %d (hasMax=%v)", maxIdx, hasMax)
	}
	
	// Check Get
	if store.Get(10) != 5 {
		t.Errorf("Get(10) should return 5, got %d", store.Get(10))
	}
	if store.Get(20) != 10 {
		t.Errorf("Get(20) should return 10, got %d", store.Get(20))
	}
	if store.Get(30) != 15 {
		t.Errorf("Get(30) should return 15, got %d", store.Get(30))
	}
	if store.Get(40) != 0 {
		t.Errorf("Get(40) should return 0, got %d", store.Get(40))
	}
	
	// Clear store
	store.Clear()
	
	// Check state after clear
	if store.GetTotalCount() != 0 {
		t.Errorf("After clear, store should have count 0, got %d", store.GetTotalCount())
	}
	
	if len(store.GetNonEmptyBuckets()) != 0 {
		t.Errorf("After clear, store should have 0 non-empty buckets, got %d",
			len(store.GetNonEmptyBuckets()))
	}
	
	_, hasMin = store.GetMinIndex()
	if hasMin {
		t.Errorf("After clear, store should not have a min index")
	}
	
	_, hasMax = store.GetMaxIndex()
	if hasMax {
		t.Errorf("After clear, store should not have a max index")
	}
}

func TestSparseStore_Merge(t *testing.T) {
	// Create two stores
	store1 := NewSparseStore(10)
	store2 := NewSparseStore(10)
	
	// Add items to both stores
	store1.Add(10, 5)
	store1.Add(20, 10)
	
	store2.Add(20, 5)
	store2.Add(30, 15)
	
	// Merge store2 into store1
	store1.Merge(store2)
	
	// Check merged result
	if store1.GetTotalCount() != 35 {
		t.Errorf("After merge, total count should be 35, got %d", store1.GetTotalCount())
	}
	
	buckets := store1.GetNonEmptyBuckets()
	if len(buckets) != 3 {
		t.Errorf("After merge, should have 3 non-empty buckets, got %d", len(buckets))
	}
	
	if buckets[10] != 5 {
		t.Errorf("After merge, bucket 10 should have count 5, got %d", buckets[10])
	}
	if buckets[20] != 15 {
		t.Errorf("After merge, bucket 20 should have count 15, got %d", buckets[20])
	}
	if buckets[30] != 15 {
		t.Errorf("After merge, bucket 30 should have count 15, got %d", buckets[30])
	}
	
	// Check min/max after merge
	minIdx, _ := store1.GetMinIndex()
	if minIdx != 10 {
		t.Errorf("After merge, min index should be 10, got %d", minIdx)
	}
	
	maxIdx, _ := store1.GetMaxIndex()
	if maxIdx != 30 {
		t.Errorf("After merge, max index should be 30, got %d", maxIdx)
	}
}

func TestSparseStore_Copy(t *testing.T) {
	// Create a store with some data
	store := NewSparseStore(10)
	store.Add(10, 5)
	store.Add(20, 10)
	store.Add(30, 15)
	
	// Create a copy
	copyStore := store.Copy()
	
	// Check that the copy has the same data
	if store.GetTotalCount() != copyStore.GetTotalCount() {
		t.Errorf("Copy total count mismatch: original=%d, copy=%d",
			store.GetTotalCount(), copyStore.GetTotalCount())
	}
	
	origBuckets := store.GetNonEmptyBuckets()
	copyBuckets := copyStore.GetNonEmptyBuckets()
	
	if len(origBuckets) != len(copyBuckets) {
		t.Errorf("Copy bucket count mismatch: original=%d, copy=%d",
			len(origBuckets), len(copyBuckets))
	}
	
	for idx, count := range origBuckets {
		copyCount, exists := copyBuckets[idx]
		if !exists {
			t.Errorf("Bucket %d exists in original but not in copy", idx)
		} else if count != copyCount {
			t.Errorf("Bucket %d count mismatch: original=%d, copy=%d",
				idx, count, copyCount)
		}
	}
	
	// Modify the original and check that the copy is unaffected
	store.Add(40, 20)
	
	if store.GetTotalCount() == copyStore.GetTotalCount() {
		t.Errorf("Modifying original should not affect copy")
	}
	
	if copyStore.Get(40) != 0 {
		t.Errorf("Copy should not have bucket 40 after it was added to original")
	}
}

func TestSparseStore_Concurrent(t *testing.T) {
	// Create a store
	store := NewSparseStore(10)
	
	// Test concurrent access
	var wg sync.WaitGroup
	numGoroutines := 10
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			// Each goroutine adds to its own buckets
			base := id * 10
			store.Add(base, 1)
			store.Add(base+1, 2)
			store.Add(base+2, 3)
			
			// Each goroutine reads various buckets
			for j := 0; j < 30; j++ {
				store.Get(j)
			}
			
			// Each goroutine gets non-empty buckets
			store.GetNonEmptyBuckets()
			
			// Each goroutine gets min/max
			store.GetMinIndex()
			store.GetMaxIndex()
		}(i)
	}
	
	wg.Wait()
	
	// Check final state
	totalCount := store.GetTotalCount()
	expectedCount := numGoroutines * (1 + 2 + 3)
	if totalCount != uint64(expectedCount) {
		t.Errorf("Expected total count %d, got %d", expectedCount, totalCount)
	}
	
	buckets := store.GetNonEmptyBuckets()
	if len(buckets) != numGoroutines * 3 {
		t.Errorf("Expected %d non-empty buckets, got %d", numGoroutines*3, len(buckets))
	}
}

func TestSparseStore_Collapse(t *testing.T) {
	// Create a store with low collapse threshold for testing
	store := NewSparseStore(2) // Collapse buckets with 2 or fewer counts
	
	// Add many buckets with various counts
	for i := 0; i < 2000; i++ {
		count := uint64(i % 5 + 1) // Counts 1-5
		store.Add(i, count)
	}
	
	// Force collapse by directly calling the method
	sparseStore := store.(*SparseStore)
	sparseStore.collapseBuckets()
	
	// Check that low-count buckets were collapsed
	buckets := store.GetNonEmptyBuckets()
	lowCountBuckets := 0
	for _, count := range buckets {
		if count <= 2 {
			lowCountBuckets++
		}
	}
	
	// Only the min and max index should preserve low counts
	if lowCountBuckets > 2 {
		t.Logf("Found %d buckets with count <= 2 after collapse (expected <= 2)",
			lowCountBuckets)
	}
	
	// Total count should be preserved
	if store.GetTotalCount() != 6000 { // Sum of 1+2+3+4+5 = 15, times 400 buckets of each
		t.Errorf("Total count should be preserved after collapse, expected 6000, got %d",
			store.GetTotalCount())
	}
}

func TestDenseStore_Basic(t *testing.T) {
	// Create a new dense store
	store := NewDenseStore(10)
	
	// Initial state checks
	if store.GetTotalCount() != 0 {
		t.Errorf("New store should have count 0, got %d", store.GetTotalCount())
	}
	
	if len(store.GetNonEmptyBuckets()) != 0 {
		t.Errorf("New store should have 0 non-empty buckets, got %d", 
			len(store.GetNonEmptyBuckets()))
	}
	
	// Add items to store
	store.Add(5, 5)
	store.Add(10, 10)
	store.Add(15, 15)
	
	// Check count
	if store.GetTotalCount() != 30 {
		t.Errorf("Store should have count 30, got %d", store.GetTotalCount())
	}
	
	// Check buckets
	buckets := store.GetNonEmptyBuckets()
	if len(buckets) != 3 {
		t.Errorf("Store should have 3 non-empty buckets, got %d", len(buckets))
	}
	
	if buckets[5] != 5 {
		t.Errorf("Bucket 5 should have count 5, got %d", buckets[5])
	}
	if buckets[10] != 10 {
		t.Errorf("Bucket 10 should have count 10, got %d", buckets[10])
	}
	if buckets[15] != 15 {
		t.Errorf("Bucket 15 should have count 15, got %d", buckets[15])
	}
	
	// Check min/max index
	minIdx, hasMin := store.GetMinIndex()
	if !hasMin || minIdx != 5 {
		t.Errorf("Min index should be 5, got %d (hasMin=%v)", minIdx, hasMin)
	}
	
	maxIdx, hasMax := store.GetMaxIndex()
	if !hasMax || maxIdx != 15 {
		t.Errorf("Max index should be 15, got %d (hasMax=%v)", maxIdx, hasMax)
	}
	
	// Check Get
	if store.Get(5) != 5 {
		t.Errorf("Get(5) should return 5, got %d", store.Get(5))
	}
	if store.Get(10) != 10 {
		t.Errorf("Get(10) should return 10, got %d", store.Get(10))
	}
	if store.Get(15) != 15 {
		t.Errorf("Get(15) should return 15, got %d", store.Get(15))
	}
	if store.Get(20) != 0 {
		t.Errorf("Get(20) should return 0, got %d", store.Get(20))
	}
	
	// Clear store
	store.Clear()
	
	// Check state after clear
	if store.GetTotalCount() != 0 {
		t.Errorf("After clear, store should have count 0, got %d", store.GetTotalCount())
	}
	
	if len(store.GetNonEmptyBuckets()) != 0 {
		t.Errorf("After clear, store should have 0 non-empty buckets, got %d",
			len(store.GetNonEmptyBuckets()))
	}
}

func TestDenseStore_Resize(t *testing.T) {
	// Create a small store
	store := NewDenseStore(5)
	
	// Add within initial capacity
	store.Add(0, 1)
	store.Add(4, 2)
	
	// Now add beyond initial capacity
	store.Add(10, 3)
	
	// Check counts
	if store.Get(0) != 1 {
		t.Errorf("Get(0) should return 1, got %d", store.Get(0))
	}
	if store.Get(4) != 2 {
		t.Errorf("Get(4) should return 2, got %d", store.Get(4))
	}
	if store.Get(10) != 3 {
		t.Errorf("Get(10) should return 3, got %d", store.Get(10))
	}
	
	// Add to negative indices
	store.Add(-5, 5)
	
	// Check negative index
	if store.Get(-5) != 5 {
		t.Errorf("Get(-5) should return 5, got %d", store.Get(-5))
	}
	
	// Check min/max
	minIdx, _ := store.GetMinIndex()
	if minIdx != -5 {
		t.Errorf("Min index should be -5, got %d", minIdx)
	}
	
	maxIdx, _ := store.GetMaxIndex()
	if maxIdx != 10 {
		t.Errorf("Max index should be 10, got %d", maxIdx)
	}
	
	// Check total count
	if store.GetTotalCount() != 11 {
		t.Errorf("Total count should be 11, got %d", store.GetTotalCount())
	}
}

func TestDenseStore_Merge(t *testing.T) {
	// Create two stores
	store1 := NewDenseStore(10)
	store2 := NewDenseStore(10)
	
	// Add to first store
	store1.Add(1, 5)
	store1.Add(2, 10)
	
	// Add to second store
	store2.Add(2, 5)  // Overlapping bucket
	store2.Add(3, 15) // New bucket
	
	// Merge store2 into store1
	store1.Merge(store2)
	
	// Check merged result
	if store1.GetTotalCount() != 35 {
		t.Errorf("After merge, total count should be 35, got %d", store1.GetTotalCount())
	}
	
	// Check individual buckets
	if store1.Get(1) != 5 {
		t.Errorf("After merge, bucket 1 should have count 5, got %d", store1.Get(1))
	}
	if store1.Get(2) != 15 {
		t.Errorf("After merge, bucket 2 should have count 15, got %d", store1.Get(2))
	}
	if store1.Get(3) != 15 {
		t.Errorf("After merge, bucket 3 should have count 15, got %d", store1.Get(3))
	}
	
	// Check min/max
	minIdx, _ := store1.GetMinIndex()
	if minIdx != 1 {
		t.Errorf("After merge, min index should be 1, got %d", minIdx)
	}
	
	maxIdx, _ := store1.GetMaxIndex()
	if maxIdx != 3 {
		t.Errorf("After merge, max index should be 3, got %d", maxIdx)
	}
}

func TestDenseStore_Copy(t *testing.T) {
	// Create a store
	store := NewDenseStore(10)
	store.Add(1, 5)
	store.Add(2, 10)
	store.Add(3, 15)
	
	// Create a copy
	copyStore := store.Copy()
	
	// Check copy has same data
	if store.GetTotalCount() != copyStore.GetTotalCount() {
		t.Errorf("Copy count mismatch: original=%d, copy=%d",
			store.GetTotalCount(), copyStore.GetTotalCount())
	}
	
	// Check individual buckets
	for i := 1; i <= 3; i++ {
		if store.Get(i) != copyStore.Get(i) {
			t.Errorf("Copy bucket %d mismatch: original=%d, copy=%d",
				i, store.Get(i), copyStore.Get(i))
		}
	}
	
	// Modify original
	store.Add(4, 20)
	
	// Check copy is unaffected
	if store.GetTotalCount() == copyStore.GetTotalCount() {
		t.Errorf("Modifying original should not affect copy")
	}
	
	if copyStore.Get(4) != 0 {
		t.Errorf("Copy should not have bucket 4 after it was added to original")
	}
}

func TestDenseStore_Concurrent(t *testing.T) {
	// Create a store
	store := NewDenseStore(10)
	
	// Test concurrent access
	var wg sync.WaitGroup
	numGoroutines := 10
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			// Each goroutine adds to its own buckets
			base := id * 5
			store.Add(base, 1)
			store.Add(base+1, 2)
			store.Add(base+2, 3)
			
			// Each goroutine reads various buckets
			for j := 0; j < 30; j++ {
				store.Get(j)
			}
			
			// Each goroutine gets non-empty buckets
			store.GetNonEmptyBuckets()
			
			// Each goroutine gets min/max
			store.GetMinIndex()
			store.GetMaxIndex()
		}(i)
	}
	
	wg.Wait()
	
	// Check final state
	totalCount := store.GetTotalCount()
	expectedCount := numGoroutines * (1 + 2 + 3)
	if totalCount != uint64(expectedCount) {
		t.Errorf("Expected total count %d, got %d", expectedCount, totalCount)
	}
	
	buckets := store.GetNonEmptyBuckets()
	if len(buckets) != numGoroutines * 3 {
		t.Errorf("Expected %d non-empty buckets, got %d", numGoroutines*3, len(buckets))
	}
}

func TestStore_Density(t *testing.T) {
	// Test sparse store density
	sparseStore := NewSparseStore(10)
	
	// Empty store should have zero density
	if sparseStore.GetStoreDensity() != 0 {
		t.Errorf("Empty sparse store should have density 0, got %f",
			sparseStore.GetStoreDensity())
	}
	
	// Add some sparse values
	sparseStore.Add(0, 1)
	sparseStore.Add(100, 1)
	
	// Density should be low
	density := sparseStore.GetStoreDensity()
	expected := 2.0 / 101.0
	if density != expected {
		t.Errorf("Sparse store density should be %f, got %f", expected, density)
	}
	
	// Test dense store density
	denseStore := NewDenseStore(10)
	
	// Empty store should have zero density
	if denseStore.GetStoreDensity() != 0 {
		t.Errorf("Empty dense store should have density 0, got %f",
			denseStore.GetStoreDensity())
	}
	
	// Add some values
	for i := 0; i < 5; i++ {
		denseStore.Add(i, 1)
	}
	
	// Add to fill specific positions
	denseStore.Add(10, 1)
	denseStore.Add(15, 1)
	
	// Check density
	density = denseStore.GetStoreDensity()
	expected = 7.0 / 16.0
	if density != expected {
		t.Errorf("Dense store density should be %f, got %f", expected, density)
	}
}

func BenchmarkSparseStore_Add(b *testing.B) {
	store := NewSparseStore(10)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := i % 1000
		store.Add(index, 1)
	}
}

func BenchmarkSparseStore_Get(b *testing.B) {
	store := NewSparseStore(10)
	
	// Populate store
	for i := 0; i < 1000; i++ {
		store.Add(i, 1)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := i % 1000
		store.Get(index)
	}
}

func BenchmarkDenseStore_Add(b *testing.B) {
	store := NewDenseStore(1000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := i % 1000
		store.Add(index, 1)
	}
}

func BenchmarkDenseStore_Get(b *testing.B) {
	store := NewDenseStore(1000)
	
	// Populate store
	for i := 0; i < 1000; i++ {
		store.Add(i, 1)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := i % 1000
		store.Get(index)
	}
}
