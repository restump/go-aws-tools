package scan

import (
	"sync"
)

type WorkerPool struct {
	Results chan Result
	waitgrp *sync.WaitGroup
	workers chan *Work
}

type WorkInput struct {
	Input interface{}
	Region string
}

type Work struct {
	WorkFn func(string, interface{})
	WorkIn WorkInput
}

func NewWorkerPool(size int) *WorkerPool {
	pool := &WorkerPool{}
	pool.waitgrp = &sync.WaitGroup{}
	pool.workers = make(chan *Work)
	pool.Results = make(chan Result, size)
	return pool
}

func (p *WorkerPool) AddWorker() {
	p.waitgrp.Add(1)
	go p.doWork()
	return
}

func (p *WorkerPool) AddWork(work *Work) {
	p.workers <- work
	return
}

func (p *WorkerPool) doWork() {
	defer p.waitgrp.Done()
	for work := range p.workers {
		work.WorkFn(work.WorkIn.Region, work.WorkIn.Input)
	}
	return
}

func (p *WorkerPool) CloseWorkers() {
	close(p.workers)
	return
}

func (p *WorkerPool) CloseResults() {
	close(p.Results)
	return
}

func (p *WorkerPool) Wait() {
	p.waitgrp.Wait()
	return
}

func (p *WorkerPool) Done() {
	p.waitgrp.Done()
	return
}

func (p *WorkerPool) AddResult(result Result) {
	p.Results <- result
	return
}

func (p *WorkerPool) GetResults() []Result {
	results := make([]Result, 0)
	for result := range p.Results {
		results = append(results, result)
	}
	return results
}
