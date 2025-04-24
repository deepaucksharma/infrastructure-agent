package sampler

import (
	"container/heap"
	"math/rand"
	"testing"
	"time"
)

func TestProcessHeap_Basic(t *testing.T) {
	// Create a new process heap
	ph := NewProcessHeap(100)
	if ph.Len() != 0 {
		t.Errorf("New heap length should be 0, got %d", ph.Len())
	}

	// Create some test processes
	p1 := &ProcessInfo{PID: 1, Score: 10.0}
	p2 := &ProcessInfo{PID: 2, Score: 20.0}
	p3 := &ProcessInfo{PID: 3, Score: 5.0}

	// Push processes onto the heap
	heap.Push(ph, p1)
	heap.Push(ph, p2)
	heap.Push(ph, p3)

	// Check length
	if ph.Len() != 3 {
		t.Errorf("Heap length should be 3, got %d", ph.Len())
	}

	// Check contains
	if !ph.Contains(1) {
		t.Errorf("Heap should contain PID 1")
	}
	if !ph.Contains(2) {
		t.Errorf("Heap should contain PID 2")
	}
	if !ph.Contains(3) {
		t.Errorf("Heap should contain PID 3")
	}
	if ph.Contains(4) {
		t.Errorf("Heap should not contain PID 4")
	}

	// Pop should return lowest score first (min heap)
	p := heap.Pop(ph).(*ProcessInfo)
	if p.PID != 3 || p.Score != 5.0 {
		t.Errorf("Expected PID 3 with score 5.0, got PID %d with score %.1f", p.PID, p.Score)
	}

	// Check length after pop
	if ph.Len() != 2 {
		t.Errorf("Heap length should be 2 after pop, got %d", ph.Len())
	}

	// Check contains after pop
	if ph.Contains(3) {
		t.Errorf("Heap should not contain PID 3 after pop")
	}

	// Pop again
	p = heap.Pop(ph).(*ProcessInfo)
	if p.PID != 1 || p.Score != 10.0 {
		t.Errorf("Expected PID 1 with score 10.0, got PID %d with score %.1f", p.PID, p.Score)
	}

	// Last pop
	p = heap.Pop(ph).(*ProcessInfo)
	if p.PID != 2 || p.Score != 20.0 {
		t.Errorf("Expected PID 2 with score 20.0, got PID %d with score %.1f", p.PID, p.Score)
	}

	// Heap should be empty
	if ph.Len() != 0 {
		t.Errorf("Heap should be empty, got length %d", ph.Len())
	}
}

func TestProcessHeap_Update(t *testing.T) {
	// Create a new process heap with max size 3
	ph := NewProcessHeap(3)

	// Create test processes
	p1 := &ProcessInfo{PID: 1, Score: 10.0, CPU: 5.0}
	p2 := &ProcessInfo{PID: 2, Score: 20.0, CPU: 10.0}
	p3 := &ProcessInfo{PID: 3, Score: 5.0, CPU: 2.5}

	// Add processes to heap
	ph.Update(p1) // Should add
	ph.Update(p2) // Should add
	ph.Update(p3) // Should add

	// Heap should have 3 processes
	if ph.Len() != 3 {
		t.Errorf("Heap should have 3 processes, got %d", ph.Len())
	}

	// Update an existing process
	p1Updated := &ProcessInfo{PID: 1, Score: 30.0, CPU: 15.0}
	if !ph.Update(p1Updated) {
		t.Errorf("Update should return true for existing process")
	}

	// Get top processes - should be in descending order by score
	top := ph.TopN(3)
	if len(top) != 3 {
		t.Errorf("TopN should return 3 processes, got %d", len(top))
	}

	// Check order by score (descending)
	if top[0].PID != 1 || top[0].Score != 30.0 {
		t.Errorf("Expected PID 1 with score 30.0 at index 0, got PID %d with score %.1f", top[0].PID, top[0].Score)
	}
	if top[1].PID != 2 || top[1].Score != 20.0 {
		t.Errorf("Expected PID 2 with score 20.0 at index 1, got PID %d with score %.1f", top[1].PID, top[1].Score)
	}
	if top[2].PID != 3 || top[2].Score != 5.0 {
		t.Errorf("Expected PID 3 with score 5.0 at index 2, got PID %d with score %.1f", top[2].PID, top[2].Score)
	}

	// Try to add a process with low score when heap is full
	p4 := &ProcessInfo{PID: 4, Score: 2.0, CPU: 1.0}
	if ph.Update(p4) {
		t.Errorf("Update should return false for low score process when heap is full")
	}

	// Try to add a process with high score when heap is full
	p5 := &ProcessInfo{PID: 5, Score: 25.0, CPU: 12.5}
	if !ph.Update(p5) {
		t.Errorf("Update should return true for high score process when heap is full")
	}

	// Heap should still have 3 processes
	if ph.Len() != 3 {
		t.Errorf("Heap should have 3 processes, got %d", ph.Len())
	}

	// Verify that p3 (lowest score) was replaced by p5
	if ph.Contains(3) {
		t.Errorf("Heap should not contain PID 3 (lowest score) after being replaced")
	}
	if !ph.Contains(5) {
		t.Errorf("Heap should contain PID 5 after being added")
	}

	// Check new top processes
	top = ph.TopN(3)
	if top[0].PID != 1 || top[0].Score != 30.0 {
		t.Errorf("Expected PID 1 with score 30.0 at index 0, got PID %d with score %.1f", top[0].PID, top[0].Score)
	}
	if top[1].PID != 5 || top[1].Score != 25.0 {
		t.Errorf("Expected PID 5 with score 25.0 at index 1, got PID %d with score %.1f", top[1].PID, top[1].Score)
	}
	if top[2].PID != 2 || top[2].Score != 20.0 {
		t.Errorf("Expected PID 2 with score 20.0 at index 2, got PID %d with score %.1f", top[2].PID, top[2].Score)
	}
}

func TestProcessHeap_Remove(t *testing.T) {
	// Create a new process heap
	ph := NewProcessHeap(100)

	// Create test processes
	p1 := &ProcessInfo{PID: 1, Score: 10.0}
	p2 := &ProcessInfo{PID: 2, Score: 20.0}
	p3 := &ProcessInfo{PID: 3, Score: 5.0}

	// Add processes to heap
	ph.Update(p1)
	ph.Update(p2)
	ph.Update(p3)

	// Remove a process
	if !ph.Remove(2) {
		t.Errorf("Remove should return true for existing process")
	}

	// Heap should have 2 processes
	if ph.Len() != 2 {
		t.Errorf("Heap should have 2 processes after removal, got %d", ph.Len())
	}

	// PID 2 should not be in the heap
	if ph.Contains(2) {
		t.Errorf("Heap should not contain PID 2 after removal")
	}

	// Try to remove non-existent process
	if ph.Remove(4) {
		t.Errorf("Remove should return false for non-existent process")
	}

	// Heap size should be unchanged
	if ph.Len() != 2 {
		t.Errorf("Heap should still have 2 processes, got %d", ph.Len())
	}
}

func TestProcessHeap_Concurrency(t *testing.T) {
	// This test ensures the heap operations are safe under concurrent access
	ph := NewProcessHeap(1000)

	// Add some initial processes
	for i := 0; i < 100; i++ {
		ph.Update(&ProcessInfo{
			PID:   i,
			Score: float64(i),
		})
	}

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(offset int) {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for j := 0; j < 100; j++ {
				pid := offset*100 + j
				op := r.Intn(3)
				switch op {
				case 0: // Add
					ph.Update(&ProcessInfo{
						PID:   pid,
						Score: float64(pid),
					})
				case 1: // Update
					ph.Update(&ProcessInfo{
						PID:   pid % 100, // Update existing
						Score: float64(r.Intn(1000)),
					})
				case 2: // Remove
					ph.Remove(pid % 100)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify the heap is still functional
	top := ph.TopN(10)
	if len(top) == 0 {
		t.Errorf("TopN should return some processes")
	}

	// Verify heap property (descending order)
	for i := 1; i < len(top); i++ {
		if top[i-1].Score < top[i].Score {
			t.Errorf("Heap property violated: top[%d].Score (%.1f) < top[%d].Score (%.1f)",
				i-1, top[i-1].Score, i, top[i].Score)
		}
	}
}

func BenchmarkProcessHeap_Update(b *testing.B) {
	ph := NewProcessHeap(1000)
	processes := make([]*ProcessInfo, b.N)

	// Generate random processes
	for i := 0; i < b.N; i++ {
		processes[i] = &ProcessInfo{
			PID:   i,
			Score: rand.Float64() * 100,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ph.Update(processes[i])
	}
}

func BenchmarkProcessHeap_TopN(b *testing.B) {
	ph := NewProcessHeap(1000)

	// Add many processes
	for i := 0; i < 1000; i++ {
		ph.Update(&ProcessInfo{
			PID:   i,
			Score: rand.Float64() * 100,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ph.TopN(100)
	}
}
