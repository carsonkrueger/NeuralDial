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

// signals an interrupt making all interrupt listeners exit
func (i *Interrupt) Signal() {
	i.mu.Lock()
	select {
	case <-i.ch: // already closed, do nothing
	default:
		close(i.ch)
	}
	i.mu.Unlock()
}

// resets the interrupt making all interrupt listeners exit and creates a new channel
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

// waits for the waitgroup to finish
func (i *Interrupt) Wait() {
	i.wg.Wait()
}

// adds a waitgroup entry to be waited on
func (i *Interrupt) Add(n int) {
	i.wg.Add(n)
}

// removes a waitgroup entry
func (i *Interrupt) Remove() {
	i.wg.Done()
}

// the channel that signals the interrupt
func (i *Interrupt) Done() <-chan struct{} {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.ch
}
