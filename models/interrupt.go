package models

import (
	"sync"
)

type Interrupt struct {
	mu sync.Mutex
	ch chan struct{}
	wg sync.WaitGroup
}

func NewInterrupt() *Interrupt {
	return &Interrupt{ch: make(chan struct{})}
}

func (i *Interrupt) Signal() {
	i.mu.Lock()
	select {
	case <-i.ch: // already closed, do nothing
	default:
		close(i.ch)
	}
	i.mu.Unlock()
}

func (i *Interrupt) Reset() {
	i.mu.Lock()
	select {
	case <-i.ch: // already closed, do nothing
	default:
		close(i.ch)
	}
	i.ch = make(chan struct{}) // new channel
	i.mu.Unlock()
}

func (i *Interrupt) Wait() {
	i.wg.Wait()
}

func (i *Interrupt) Add(n int) {
	i.wg.Add(n)
}

func (i *Interrupt) Remove() {
	i.wg.Done()
}

func (i *Interrupt) Done() <-chan struct{} {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.ch
}
