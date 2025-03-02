package xasync

import (
	"sync"
	"time"
)

type Closer interface {
	Closed() <-chan struct{}
	IsClosed() bool
	Wait(dur time.Duration) bool
	Close() bool
}

func NewCloser() *closer {
	return &closer{
		closed: make(chan struct{}),
	}
}

type closer struct {
	closed     chan struct{}
	closedOnce sync.Once
}

func (c *closer) Closed() <-chan struct{} {
	return c.closed
}

func (c *closer) IsClosed() bool {
	select {
	case <-c.Closed():
		return true
	default:
		return false
	}
}

func (c *closer) Wait(dur time.Duration) bool {
	timer := time.NewTimer(dur)
	defer timer.Stop()

	select {
	case <-c.Closed():
		return true
	case <-timer.C:
		return false
	}
}

func (c *closer) Close() bool {
	closed := false

	c.closedOnce.Do(func() {
		close(c.closed)
		closed = true
	})

	return closed
}
