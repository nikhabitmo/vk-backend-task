package pool

import (
	"fmt"
	"sync"
	"time"
)

type Pool interface {
	AddWorker()
	RemoveWorker()
	Close()
}

type Worker struct {
	id   int
	jobs <-chan string
	stop chan bool
	wg   *sync.WaitGroup
}

func (w *Worker) Start() {
	defer w.wg.Done()
	for {
		select {
		case job, ok := <-w.jobs:
			if !ok {
				return
			}

			fmt.Printf("Worker %d: processing job: %s\n", w.id, job)
			time.Sleep(1 * time.Second)

		case <-w.stop:
			fmt.Printf("Worker %d stopped.\n", w.id)
			return
		}
	}
}

type WorkerPool struct {
	workers []*Worker
	jobs    chan string
	wg      sync.WaitGroup
	mu      sync.Mutex
}

func New(numWorkers int, ch chan string) *WorkerPool {
	pool := &WorkerPool{jobs: ch, workers: make([]*Worker, 0)}

	for range numWorkers {
		pool.AddWorker()
	}

	return pool
}

func (p *WorkerPool) AddWorker() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.workers == nil {
		fmt.Println("Pool is closed. Cannot add worker.")
		return
	}

	id := len(p.workers) + 1
	worker := &Worker{
		id:   id,
		jobs: p.jobs,
		stop: make(chan bool),
		wg:   &p.wg,
	}

	p.workers = append(p.workers, worker)
	p.wg.Add(1)
	fmt.Printf("Worker %d added.\n", worker.id)
	go worker.Start()
}

func (p *WorkerPool) RemoveWorker() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.workers == nil {
		fmt.Println("Pool is closed. Cannot remove worker.")
		return
	}

	if len(p.workers) == 0 {
		fmt.Println("No workers to remove.")
		return
	}

	worker := p.workers[len(p.workers)-1]
	p.workers = p.workers[:len(p.workers)-1]

	worker.stop <- true
}

func (p *WorkerPool) Close() {
	for _, worker := range p.workers {
		close(worker.stop)
	}
	p.workers = nil
	p.wg.Wait()
}
