package market

import (
	"testing"
	"time"
)

func TestMarkKlinesClosedAt_DerivesCloseTimeFromTimeframe(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	klines := []Kline{{OpenTime: base.UnixMilli()}}

	now := base.Add(30 * time.Second)
	out := markKlinesClosedAt(klines, "1m", now)
	if len(out) != 1 {
		t.Fatalf("expected 1 kline, got %d", len(out))
	}
	if out[0].IsClosed {
		t.Fatalf("expected kline to be open (isClosed=false)")
	}

	now2 := base.Add(61 * time.Second)
	out2 := markKlinesClosedAt(klines, "1m", now2)
	if !out2[0].IsClosed {
		t.Fatalf("expected kline to be closed (isClosed=true)")
	}
}

func TestMarkKlinesClosedAt_UsesCloseTimeWhenPresent(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	klines := []Kline{{
		OpenTime:  base.UnixMilli(),
		CloseTime: base.Add(time.Minute).UnixMilli(),
	}}

	out := markKlinesClosedAt(klines, "1m", base.Add(59*time.Second))
	if out[0].IsClosed {
		t.Fatalf("expected kline to be open (isClosed=false)")
	}

	out2 := markKlinesClosedAt(klines, "1m", base.Add(60*time.Second))
	if !out2[0].IsClosed {
		t.Fatalf("expected kline to be closed (isClosed=true)")
	}
}
