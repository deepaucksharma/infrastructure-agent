package sampler

import (
	"container/heap"
	"sync"
)

// ProcessHeap implements a min-heap of processes ordered by score.
// It allows efficient tracking of the top N processes while supporting
// insertion, removal, and updates in O(log N) time.
type ProcessHeap struct {
	processes []*ProcessInfo
	pidMap    map[int]int // Maps PID to index in the heap
	mutex     sync.RWMutex
	maxSize   int
}

// NewProcessHeap creates a new process heap with the specified maximum size.
func NewProcessHeap(maxSize int) *ProcessHeap {
	return &ProcessHeap{
		processes: make([]*ProcessInfo, 0, maxSize),
		pidMap:    make(map[int]int),
		maxSize:   maxSize,
	}
}

// Len returns the number of processes in the heap.
func (h *ProcessHeap) Len() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.processes)
}

// Less returns whether the process at index i has a lower score than the process at index j.
func (h *ProcessHeap) Less(i, j int) bool {
	// Min heap based on score (lower score at the root)
	return h.processes[i].Score < h.processes[j].Score
}

// Swap swaps the processes at indices i and j.
func (h *ProcessHeap) Swap(i, j int) {
	h.processes[i], h.processes[j] = h.processes[j], h.processes[i]
	h.pidMap[h.processes[i].PID] = i
	h.pidMap[h.processes[j].PID] = j
}

// Push adds a process to the heap.
func (h *ProcessHeap) Push(x interface{}) {
	process := x.(*ProcessInfo)
	h.processes = append(h.processes, process)
	h.pidMap[process.PID] = len(h.processes) - 1
}

// Pop removes and returns the process with the lowest score.
func (h *ProcessHeap) Pop() interface{} {
	old := h.processes
	n := len(old)
	process := old[n-1]
	h.processes = old[0 : n-1]
	delete(h.pidMap, process.PID)
	return process
}

// Contains checks if a process with the given PID is in the heap.
func (h *ProcessHeap) Contains(pid int) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	_, exists := h.pidMap[pid]
	return exists
}

// Update updates the score of a process in the heap and rebalances the heap.
func (h *ProcessHeap) Update(process *ProcessInfo) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	idx, exists := h.pidMap[process.PID]
	if !exists {
		// Process not in heap yet
		if len(h.processes) < h.maxSize {
			// Heap not full, add the process
			heap.Push(h, process)
			return true
		} else if len(h.processes) > 0 && process.Score > h.processes[0].Score {
			// Heap full but new process has higher score than minimum
			// Remove lowest scoring process and add the new one
			heap.Pop(h)
			heap.Push(h, process)
			return true
		}
		// Process not important enough to track
		return false
	}

	// Process already in heap, update its score and rebalance
	h.processes[idx].Score = process.Score
	h.processes[idx].CPU = process.CPU
	h.processes[idx].RSS = process.RSS
	h.processes[idx].Command = process.Command
	h.processes[idx].Name = process.Name
	heap.Fix(h, idx)
	return true
}

// Remove removes a process from the heap.
func (h *ProcessHeap) Remove(pid int) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	idx, exists := h.pidMap[pid]
	if !exists {
		return false
	}

	heap.Remove(h, idx)
	return true
}

// TopN returns the top N processes with highest scores.
// This operation is O(N log N) due to sorting.
func (h *ProcessHeap) TopN(n int) []*ProcessInfo {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Copy processes to avoid modifying the heap
	processes := make([]*ProcessInfo, len(h.processes))
	copy(processes, h.processes)

	// Sort by score in descending order
	sort := &processScoreSort{processes: processes}
	sort.Sort()

	// Return at most n processes
	if n > len(processes) {
		n = len(processes)
	}
	return processes[:n]
}

// processScoreSort implements sort.Interface for []*ProcessInfo based on score.
type processScoreSort struct {
	processes []*ProcessInfo
}

func (s *processScoreSort) Len() int {
	return len(s.processes)
}

func (s *processScoreSort) Less(i, j int) bool {
	// Higher score comes first (descending order)
	return s.processes[i].Score > s.processes[j].Score
}

func (s *processScoreSort) Swap(i, j int) {
	s.processes[i], s.processes[j] = s.processes[j], s.processes[i]
}

func (s *processScoreSort) Sort() {
	// Implementation of sort.Sort
	n := s.Len()
	for i := n/2 - 1; i >= 0; i-- {
		s.heapify(n, i)
	}
	for i := n - 1; i >= 0; i-- {
		s.Swap(0, i)
		s.heapify(i, 0)
	}
}

func (s *processScoreSort) heapify(n, i int) {
	largest := i
	left := 2*i + 1
	right := 2*i + 2

	if left < n && s.Less(left, largest) {
		largest = left
	}

	if right < n && s.Less(right, largest) {
		largest = right
	}

	if largest != i {
		s.Swap(i, largest)
		s.heapify(n, largest)
	}
}
