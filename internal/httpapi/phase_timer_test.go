package httpapi

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"
)

func TestPhaseTimerStartStopRecordsSinglePhase(t *testing.T) {
	timer := newPhaseTimer()
	timer.start("bm25")
	time.Sleep(20 * time.Millisecond)
	timer.stop()

	phases := timer.summary()
	if phases["bm25"] < 15 {
		t.Fatalf("bm25 duration = %.1f, want >= 15", phases["bm25"])
	}
	if len(phases) != 1 {
		t.Fatalf("phases = %#v, want one phase", phases)
	}
}

func TestPhaseTimerStartImplicitlyStopsPreviousPhase(t *testing.T) {
	timer := newPhaseTimer()
	timer.start("a")
	time.Sleep(10 * time.Millisecond)
	timer.start("b")
	time.Sleep(10 * time.Millisecond)
	timer.stop()

	phases := timer.summary()
	if phases["a"] < 5 || phases["b"] < 5 {
		t.Fatalf("phases = %#v, want both phases recorded", phases)
	}
}

func TestPhaseTimerMarkAccumulatesDurations(t *testing.T) {
	timer := newPhaseTimer()
	timer.mark("x", 5)
	timer.mark("x", 3)
	timer.mark("y", 7)

	phases := timer.summary()
	if phases["x"] != 8 || phases["y"] != 7 {
		t.Fatalf("phases = %#v, want additive marks", phases)
	}
}

func TestPhaseTimerTimeRecordsConcurrentWorkIndependently(t *testing.T) {
	timer := newPhaseTimer()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = timer.time(context.Background(), "a", func(context.Context) error {
			time.Sleep(30 * time.Millisecond)
			return nil
		})
	}()
	go func() {
		defer wg.Done()
		_ = timer.time(context.Background(), "b", func(context.Context) error {
			time.Sleep(80 * time.Millisecond)
			return nil
		})
	}()
	wg.Wait()

	phases := timer.summary()
	if phases["a"] < 25 || phases["a"] >= 80 {
		t.Fatalf("phase a = %.1f, want concurrent independent duration", phases["a"])
	}
	if phases["b"] < 75 {
		t.Fatalf("phase b = %.1f, want >= 75", phases["b"])
	}
}

func TestPhaseTimerMarkRejectsInvalidDurations(t *testing.T) {
	timer := newPhaseTimer()
	timer.mark("x", -1)
	timer.mark("x", math.NaN())
	timer.mark("x", math.Inf(1))

	if _, ok := timer.summary()["x"]; ok {
		t.Fatalf("invalid mark created phase x")
	}
}

func TestPhaseTimerTotalStopsActivePhase(t *testing.T) {
	timer := newPhaseTimer()
	timer.mark("a", 10)
	timer.mark("b", 15)
	timer.start("c")
	time.Sleep(20 * time.Millisecond)

	total := timer.totalMs()
	if total < 40 {
		t.Fatalf("total = %.1f, want >= 40", total)
	}
	if phases := timer.summary(); phases["c"] < 15 {
		t.Fatalf("phases = %#v, want active phase stopped", phases)
	}
}
