package export

import (
	"context"
	"sync"
	"time"
)

// Batch represents a batch of telemetry data to be exported.
type Batch struct {
	// Items holds the telemetry items in this batch
	Items []interface{}
	
	// Size is the current size of the batch
	Size int
	
	// MaxSize is the maximum size of the batch
	MaxSize int
	
	// CreatedAt is the time the batch was created
	CreatedAt time.Time
}

// NewBatch creates a new batch with the given maximum size.
func NewBatch(maxSize int) *Batch {
	return &Batch{
		Items:     make([]interface{}, 0, maxSize),
		Size:      0,
		MaxSize:   maxSize,
		CreatedAt: time.Now(),
	}
}

// Add adds an item to the batch and returns true if the batch is full.
func (b *Batch) Add(item interface{}) bool {
	b.Items = append(b.Items, item)
	b.Size++
	return b.Size >= b.MaxSize
}

// IsFull returns true if the batch is full.
func (b *Batch) IsFull() bool {
	return b.Size >= b.MaxSize
}

// IsEmpty returns true if the batch is empty.
func (b *Batch) IsEmpty() bool {
	return b.Size == 0
}

// Clear clears the batch.
func (b *Batch) Clear() {
	b.Items = b.Items[:0]
	b.Size = 0
	b.CreatedAt = time.Now()
}

// BatchProcessor handles batching of telemetry data.
type BatchProcessor struct {
	config       BatchConfig
	processor    func(context.Context, *Batch) error
	currentBatch *Batch
	mu           sync.Mutex
	ticker       *time.Ticker
	done         chan struct{}
	wg           sync.WaitGroup
}

// NewBatchProcessor creates a new batch processor with the given configuration and processor function.
func NewBatchProcessor(config BatchConfig, processor func(context.Context, *Batch) error) *BatchProcessor {
	bp := &BatchProcessor{
		config:       config,
		processor:    processor,
		currentBatch: NewBatch(config.Size),
		done:         make(chan struct{}),
	}
	
	// Start the ticker to process batches based on timeout
	bp.ticker = time.NewTicker(config.Timeout)
	bp.wg.Add(1)
	go bp.processBatchesByTime()
	
	return bp
}

// Add adds an item to the current batch.
// If the batch becomes full, it is processed asynchronously.
func (bp *BatchProcessor) Add(ctx context.Context, item interface{}) error {
	bp.mu.Lock()
	isFull := bp.currentBatch.Add(item)
	if isFull {
		batch := bp.currentBatch
		bp.currentBatch = NewBatch(bp.config.Size)
		bp.mu.Unlock()
		return bp.processor(ctx, batch)
	}
	bp.mu.Unlock()
	return nil
}

// Flush processes any items in the current batch.
func (bp *BatchProcessor) Flush(ctx context.Context) error {
	bp.mu.Lock()
	if bp.currentBatch.IsEmpty() {
		bp.mu.Unlock()
		return nil
	}
	
	batch := bp.currentBatch
	bp.currentBatch = NewBatch(bp.config.Size)
	bp.mu.Unlock()
	
	return bp.processor(ctx, batch)
}

// Shutdown stops the batch processor and flushes any remaining items.
func (bp *BatchProcessor) Shutdown(ctx context.Context) error {
	close(bp.done)
	bp.ticker.Stop()
	
	// Wait for the ticker goroutine to finish
	bp.wg.Wait()
	
	// Flush any remaining items
	return bp.Flush(ctx)
}

// processBatchesByTime processes batches based on the timeout.
func (bp *BatchProcessor) processBatchesByTime() {
	defer bp.wg.Done()
	
	for {
		select {
		case <-bp.ticker.C:
			// Process any items in the current batch
			err := bp.Flush(context.Background())
			if err != nil {
				// Log or handle error
			}
		case <-bp.done:
			return
		}
	}
}
