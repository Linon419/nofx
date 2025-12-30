package market

import (
	"testing"
	"time"
)

func TestDropUnclosedKlinesAt_DropsBeforeClose(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	klines := []Kline{{OpenTime: base.UnixMilli()}}
	dur, err := TFDuration("1m")
	if err != nil {
		t.Fatalf("unexpected duration error: %v", err)
	}

	now := base.Add(dur - time.Second)
	out := dropUnclosedKlinesAt(klines, "1m", now, 0)
	if len(out) != 0 {
		t.Fatalf("expected last kline to be dropped, got %d", len(out))
	}
}

func TestDropUnclosedKlinesAt_KeepsAfterClose(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	klines := []Kline{{OpenTime: base.UnixMilli()}}
	dur, err := TFDuration("1m")
	if err != nil {
		t.Fatalf("unexpected duration error: %v", err)
	}

	now := base.Add(dur + time.Second)
	out := dropUnclosedKlinesAt(klines, "1m", now, 0)
	if len(out) != 1 {
		t.Fatalf("expected last kline to remain, got %d", len(out))
	}
}
