package lbugruntime

import (
	"context"
	"fmt"
	"sync"
)

type ReadConnection interface {
	Query(query string) ([]Row, error)
	Close() error
}

type ConnectionFactory func() (ReadConnection, error)

type ReadPool struct {
	connections chan ReadConnection
	done        chan struct{}
	all         []ReadConnection
	mu          sync.Mutex
	closed      bool
}

func NewReadPool(size int, factory ConnectionFactory) (*ReadPool, error) {
	if size <= 0 {
		return nil, fmt.Errorf("pool size must be positive")
	}
	if factory == nil {
		return nil, fmt.Errorf("connection factory is nil")
	}
	pool := &ReadPool{
		connections: make(chan ReadConnection, size),
		done:        make(chan struct{}),
		all:         make([]ReadConnection, 0, size),
	}
	for i := 0; i < size; i++ {
		conn, err := factory()
		if err != nil {
			pool.Close()
			return nil, err
		}
		pool.connections <- conn
		pool.all = append(pool.all, conn)
	}
	return pool, nil
}

func (p *ReadPool) Execute(ctx context.Context, query string) ([]Row, error) {
	if err := ValidateReadQuery(query); err != nil {
		return nil, err
	}
	conn, release, err := p.Checkout(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	return conn.Query(query)
}

func (p *ReadPool) Checkout(ctx context.Context) (ReadConnection, func(), error) {
	if p == nil {
		return nil, nil, fmt.Errorf("pool is nil")
	}
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()
	if closed {
		return nil, nil, fmt.Errorf("pool is closed")
	}

	select {
	case conn := <-p.connections:
		released := false
		release := func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			if released {
				return
			}
			released = true
			if p.closed {
				_ = conn.Close()
				return
			}
			p.connections <- conn
		}
		return conn, release, nil
	case <-p.done:
		return nil, nil, fmt.Errorf("pool is closed")
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}

func (p *ReadPool) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	close(p.done)
	p.mu.Unlock()

	var firstErr error
	for _, conn := range p.all {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
