package processor

import (
	"context"
	"sync"
)

// FFmpegWorkerPool manages a pool of workers for FFmpeg operations
type FFmpegWorkerPool struct {
	maxWorkers int
	sem        chan struct{}
	jobs       map[string]context.CancelFunc
	mu         sync.RWMutex
}

// NewFFmpegWorkerPool creates a new worker pool with the specified maximum number of concurrent workers
func NewFFmpegWorkerPool(maxWorkers int) *FFmpegWorkerPool {
	return &FFmpegWorkerPool{
		maxWorkers: maxWorkers,
		sem:        make(chan struct{}, maxWorkers),
		jobs:       make(map[string]context.CancelFunc),
	}
}

// AcquireWorker attempts to acquire a worker from the pool
// Returns a context and cancel function, or an error if the context is done before a worker is available
func (p *FFmpegWorkerPool) AcquireWorker(ctx context.Context, jobID string) (context.Context, error) {
	select {
	case p.sem <- struct{}{}:
		jobCtx, cancel := context.WithCancel(ctx)
		p.mu.Lock()
		p.jobs[jobID] = cancel
		p.mu.Unlock()
		return jobCtx, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ReleaseWorker releases a worker back to the pool
func (p *FFmpegWorkerPool) ReleaseWorker(jobID string) {
	p.mu.Lock()
	if cancel, exists := p.jobs[jobID]; exists {
		cancel()
		delete(p.jobs, jobID)
	}
	p.mu.Unlock()
	<-p.sem
}

// CancelJob cancels a specific job if it's running
func (p *FFmpegWorkerPool) CancelJob(jobID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if cancel, exists := p.jobs[jobID]; exists {
		cancel()
		delete(p.jobs, jobID)
		return true
	}
	return false
}

// ActiveWorkers returns the current number of active workers
func (p *FFmpegWorkerPool) ActiveWorkers() int {
	return len(p.sem)
}
