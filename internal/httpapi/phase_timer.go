package httpapi

import (
	"context"
	"math"
	"sync"
	"time"
)

type phaseTimer struct {
	mu      sync.Mutex
	phases  map[string]float64
	current string
	started time.Time
}

func newPhaseTimer() *phaseTimer {
	return &phaseTimer{phases: map[string]float64{}}
}

func (timer *phaseTimer) start(phase string) {
	timer.mu.Lock()
	defer timer.mu.Unlock()
	timer.stopLocked()
	timer.current = phase
	timer.started = time.Now()
}

func (timer *phaseTimer) stop() {
	timer.mu.Lock()
	defer timer.mu.Unlock()
	timer.stopLocked()
}

func (timer *phaseTimer) mark(phase string, durationMs float64) {
	if math.IsNaN(durationMs) || math.IsInf(durationMs, 0) || durationMs < 0 {
		return
	}
	timer.mu.Lock()
	defer timer.mu.Unlock()
	timer.phases[phase] += durationMs
}

func (timer *phaseTimer) time(ctx context.Context, phase string, fn func(context.Context) error) error {
	started := time.Now()
	err := fn(ctx)
	timer.mark(phase, elapsedMillis(started))
	return err
}

func (timer *phaseTimer) summary() map[string]float64 {
	timer.stop()
	timer.mu.Lock()
	defer timer.mu.Unlock()
	out := make(map[string]float64, len(timer.phases))
	for phase, duration := range timer.phases {
		out[phase] = roundMillis(duration)
	}
	return out
}

func (timer *phaseTimer) totalMs() float64 {
	timer.stop()
	timer.mu.Lock()
	defer timer.mu.Unlock()
	var total float64
	for _, duration := range timer.phases {
		total += duration
	}
	return roundMillis(total)
}

func (timer *phaseTimer) stopLocked() {
	if timer.current == "" {
		return
	}
	timer.phases[timer.current] += elapsedMillis(timer.started)
	timer.current = ""
	timer.started = time.Time{}
}

func elapsedMillis(started time.Time) float64 {
	return float64(time.Since(started).Nanoseconds()) / float64(time.Millisecond)
}

func roundMillis(value float64) float64 {
	return math.Round(value*10) / 10
}
