package queue

import (
	"context"
	"errors"
	"log"
	"sync"
)

var ErrQueueFull = errors.New("job queue is full")

// Job defines asynchronous work executed by the worker pool.
type Job func(context.Context) error

// WorkerPool provides concurrent background job processing.
type WorkerPool struct {
	workers int
	jobs    chan Job
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func NewWorkerPool(workerCount, queueSize int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 2
	}
	if queueSize <= 0 {
		queueSize = 64
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workers: workerCount,
		jobs:    make(chan Job, queueSize),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (pool *WorkerPool) Start() {
	for workerID := 1; workerID <= pool.workers; workerID++ {
		pool.wg.Add(1)
		go pool.runWorker(workerID)
	}
}

func (pool *WorkerPool) Submit(job Job) error {
	if job == nil {
		return nil
	}

	select {
	case <-pool.ctx.Done():
		return context.Canceled
	case pool.jobs <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

func (pool *WorkerPool) Stop() {
	pool.cancel()
	pool.wg.Wait()
}

func (pool *WorkerPool) runWorker(workerID int) {
	defer pool.wg.Done()

	for {
		select {
		case <-pool.ctx.Done():
			return
		case job := <-pool.jobs:
			if job == nil {
				continue
			}
			if err := job(pool.ctx); err != nil {
				log.Printf("queue worker %d job failed: %v", workerID, err)
			}
		}
	}
}
