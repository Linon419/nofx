package indicator

import (
	"testing"

	"nofx/market"
)

func TestCalculateDivergence_External_PositiveRegular(t *testing.T) {
	// Construct a series with:
	// - A confirmed pivot low at index 5 (prd=2, confirm at index 7)
	// - A lower low at the last bar
	// - An external indicator that is higher at the last bar vs the pivot low
	prices := []float64{
		12, 13, 14, 11, 10, 11, 12, 13, 14, 15, 14, 13, 12, 11, 9,
	}

	n := len(prices)
	klines := make([]market.Kline, 0, n)
	for i, c := range prices {
		klines = append(klines, market.Kline{
			OpenTime: int64(i),
			Open:     c,
			High:     c + 0.5,
			Low:      c - 0.5,
			Close:    c,
			Volume:   1,
		})
	}

	// External series: linear interpolation from 30 at pivotIdx=4 to 40 at lastIdx=14,
	// using an integer slope to avoid floating-point edge cases in line checks.
	external := make([]float64, n)
	for i := 0; i < n; i++ {
		if i < 4 {
			external[i] = 0
			continue
		}
		external[i] = 30 + float64(i-4)
	}

	cfg := DefaultDivergenceConfig()
	cfg.PivotPeriod = 2
	cfg.Source = PivotSourceClose
	cfg.Search = SearchRegular
	cfg.DontConfirm = true
	cfg.MinDivergenceCount = 1
	cfg.MaxPivotPoints = 10
	cfg.MaxBars = 100

	cfg.EnableMACD = false
	cfg.EnableHist = false
	cfg.EnableRSI = false
	cfg.EnableStoch = false
	cfg.EnableCCI = false
	cfg.EnableMOM = false
	cfg.EnableOBV = false
	cfg.EnableVWMACD = false
	cfg.EnableCMF = false
	cfg.EnableMFI = false

	cfg.EnableExternal = true
	cfg.ExternalSeries = external

	got := CalculateDivergence(klines, cfg)
	if got == nil {
		t.Fatalf("expected non-nil result")
	}
	if !got.PassedMinRequired || got.Total == 0 {
		t.Fatalf("expected divergence detected, got passed=%v total=%d", got.PassedMinRequired, got.Total)
	}

	by, ok := got.Indicators["external"]
	if !ok {
		t.Fatalf("expected 'external' in indicators map")
	}
	if by.PositiveRegular == 0 {
		t.Fatalf("expected positive regular divergence for external, got %+v", by)
	}
}
