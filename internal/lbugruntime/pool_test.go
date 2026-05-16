package lbugruntime

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestReadPoolLimitsConcurrentCheckout(t *testing.T) {
	var created int32
	pool, err := NewReadPool(1, func() (ReadConnection, error) {
		id := atomic.AddInt32(&created, 1)
		return &fakeReadConnection{id: id}, nil
	})
	if err != nil {
		t.Fatalf("NewReadPool() error = %v", err)
	}
	defer pool.Close()

	conn, release, err := pool.Checkout(context.Background())
	if err != nil {
		t.Fatalf("first Checkout() error = %v", err)
	}
	if conn == nil {
		t.Fatalf("first Checkout() returned nil connection")
	}

	acquired := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, secondRelease, err := pool.Checkout(ctx)
		if err != nil {
			t.Errorf("second Checkout() error = %v", err)
			return
		}
		close(acquired)
		secondRelease()
	}()

	select {
	case <-acquired:
		t.Fatalf("second checkout acquired connection before release")
	case <-time.After(25 * time.Millisecond):
	}
	release()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatalf("second checkout did not acquire after release")
	}
	<-done
	if created != 1 {
		t.Fatalf("created connections = %d, want prewarmed single connection", created)
	}
}

func TestReadPoolExecuteBlocksWritesBeforeQuery(t *testing.T) {
	conn := &fakeReadConnection{}
	pool, err := NewReadPool(1, func() (ReadConnection, error) {
		return conn, nil
	})
	if err != nil {
		t.Fatalf("NewReadPool() error = %v", err)
	}
	defer pool.Close()

	if _, err := pool.Execute(context.Background(), "CREATE (n:File {id: 'x'})"); err == nil {
		t.Fatalf("Execute(write) expected read-only error")
	}
	if conn.queryCount != 0 {
		t.Fatalf("write query reached connection, queryCount = %d", conn.queryCount)
	}

	rows, err := pool.Execute(context.Background(), "MATCH (n) RETURN n")
	if err != nil {
		t.Fatalf("Execute(read) error = %v", err)
	}
	if len(rows) != 1 || rows[0]["ok"] != true {
		t.Fatalf("rows = %#v, want ok row", rows)
	}
}

func TestReadPoolParallelQueriesDrainWaiters(t *testing.T) {
	var created int32
	var queries int32
	pool, err := NewReadPool(4, func() (ReadConnection, error) {
		id := atomic.AddInt32(&created, 1)
		return &fakeReadConnection{id: id, delay: 5 * time.Millisecond, onQuery: func() {
			atomic.AddInt32(&queries, 1)
		}}, nil
	})
	if err != nil {
		t.Fatalf("NewReadPool() error = %v", err)
	}
	defer pool.Close()

	errs := make(chan error, 12)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			rows, err := pool.Execute(context.Background(), "MATCH (n:Function) RETURN n.name AS name")
			if err != nil {
				errs <- fmt.Errorf("query %d: %w", index, err)
				return
			}
			if len(rows) != 1 || rows[0]["ok"] != true {
				errs <- fmt.Errorf("query %d rows = %#v", index, rows)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if created != 4 {
		t.Fatalf("created connections = %d, want prewarmed pool size 4", created)
	}
	if queries != 12 {
		t.Fatalf("queries = %d, want 12", queries)
	}
}

func TestReadPoolCheckoutHonorsContextAndCloseIsIdempotent(t *testing.T) {
	pool, err := NewReadPool(1, func() (ReadConnection, error) {
		return &fakeReadConnection{}, nil
	})
	if err != nil {
		t.Fatalf("NewReadPool() error = %v", err)
	}
	conn, release, err := pool.Checkout(context.Background())
	if err != nil {
		t.Fatalf("Checkout() error = %v", err)
	}
	if conn == nil {
		t.Fatal("Checkout() returned nil connection")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, _, err := pool.Checkout(ctx); err == nil {
		t.Fatalf("Checkout(timeout) error = nil, want context timeout")
	}
	release()
	release()

	if err := pool.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := pool.Close(); err != nil {
		t.Fatalf("Close() second error = %v", err)
	}
	if _, _, err := pool.Checkout(context.Background()); err == nil {
		t.Fatalf("Checkout(closed) error = nil, want closed pool error")
	}
}

type fakeReadConnection struct {
	id         int32
	queryCount int
	closed     bool
	delay      time.Duration
	onQuery    func()
}

func (c *fakeReadConnection) Query(query string) ([]Row, error) {
	c.queryCount++
	if c.onQuery != nil {
		c.onQuery()
	}
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return []Row{{"ok": true, "query": query}}, nil
}

func (c *fakeReadConnection) Close() error {
	c.closed = true
	return nil
}
