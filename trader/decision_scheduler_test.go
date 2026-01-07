package trader

import (
	"context"
	"testing"
	"time"

	decision "nofx/kernel"
	"nofx/store"
)

func TestNewDecisionScheduler_UsesStrategyAlignedSchedule(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("en")
	cfg.Indicators.Klines.PrimaryTimeframe = "1h"
	cfg.Indicators.Klines.SelectedTimeframes = []string{"1h", "5m"}
	cfg.Indicators.Klines.DecisionIntervalMultiple = 2
	off := 7
	cfg.Indicators.Klines.DecisionOffsetSeconds = &off
	cfg.Indicators.Klines.DecisionRunImmediately = true

	at := &AutoTrader{
		config:         AutoTraderConfig{ScanInterval: 3 * time.Minute},
		strategyEngine: decision.NewStrategyEngine(&cfg),
	}

	sched, effective := at.newDecisionScheduler(context.Background())

	if effective.source != "strategy" {
		t.Fatalf("expected strategy schedule, got %q", effective.source)
	}
	if effective.alignTimeframe != "5m" {
		t.Fatalf("expected alignTimeframe=5m, got %q", effective.alignTimeframe)
	}
	if effective.alignInterval != 5*time.Minute {
		t.Fatalf("expected alignInterval=5m, got %s", effective.alignInterval)
	}
	if effective.interval != 10*time.Minute {
		t.Fatalf("expected interval=10m, got %s", effective.interval)
	}
	if effective.offset != 7*time.Second {
		t.Fatalf("expected offset=7s, got %s", effective.offset)
	}
	if !effective.runImmediately {
		t.Fatalf("expected runImmediately=true")
	}
	if effective.name != "tf-5m-x2" {
		t.Fatalf("expected name=tf-5m-x2, got %q", effective.name)
	}

	if sched.AlignInterval != 5*time.Minute {
		t.Fatalf("expected scheduler.AlignInterval=5m, got %s", sched.AlignInterval)
	}
	if sched.Interval != 10*time.Minute {
		t.Fatalf("expected scheduler.Interval=10m, got %s", sched.Interval)
	}
	if sched.Offset != 7*time.Second {
		t.Fatalf("expected scheduler.Offset=7s, got %s", sched.Offset)
	}
	if !sched.RunImmediately {
		t.Fatalf("expected scheduler.RunImmediately=true")
	}
	if sched.Name != "tf-5m-x2" {
		t.Fatalf("expected scheduler.Name=tf-5m-x2, got %q", sched.Name)
	}
}

func TestNewDecisionScheduler_FallsBackToScanInterval(t *testing.T) {
	at := &AutoTrader{
		config: AutoTraderConfig{ScanInterval: 3 * time.Minute},
	}

	sched, effective := at.newDecisionScheduler(context.Background())

	if effective.source != "scan_interval" {
		t.Fatalf("expected scan_interval schedule, got %q", effective.source)
	}
	if effective.alignInterval != 3*time.Minute || effective.interval != 3*time.Minute {
		t.Fatalf("expected 3m/3m, got align=%s interval=%s", effective.alignInterval, effective.interval)
	}
	if effective.offset != 0 || effective.runImmediately {
		t.Fatalf("expected offset=0 and runImmediately=false, got offset=%s runImmediately=%v", effective.offset, effective.runImmediately)
	}
	if sched.AlignInterval != 3*time.Minute || sched.Interval != 3*time.Minute {
		t.Fatalf("expected scheduler 3m/3m, got align=%s interval=%s", sched.AlignInterval, sched.Interval)
	}
	if sched.Offset != 0 || sched.RunImmediately {
		t.Fatalf("expected scheduler offset=0 and runImmediately=false, got offset=%s runImmediately=%v", sched.Offset, sched.RunImmediately)
	}
}
