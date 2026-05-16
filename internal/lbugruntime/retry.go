package lbugruntime

import (
	"context"
	"fmt"
	"time"
)

type BusyRetryPolicy struct {
	Attempts  int
	BaseDelay time.Duration
	Sleep     func(context.Context, time.Duration) error
}

func (p BusyRetryPolicy) Run(ctx context.Context, operation func() error) error {
	if operation == nil {
		return fmt.Errorf("operation is nil")
	}
	attempts := p.Attempts
	if attempts <= 0 {
		attempts = 1
	}
	baseDelay := p.BaseDelay
	if baseDelay <= 0 {
		baseDelay = 500 * time.Millisecond
	}
	sleep := p.Sleep
	if sleep == nil {
		sleep = sleepContext
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := operation(); err != nil {
			lastErr = err
			if !IsBusyError(err) || attempt == attempts {
				return err
			}
			if err := sleep(ctx, baseDelay*time.Duration(attempt)); err != nil {
				return err
			}
			continue
		}
		return nil
	}
	return lastErr
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
