package vision

import (
	"bytes"
	"testing"

	"nofx/analysis/indicator"
	"nofx/market"
)

func TestRenderTimeframeChartPNG(t *testing.T) {
	klines := make([]market.KlineBar, 0, 60)
	ema := make([]float64, 0, 60)
	price := 100.0
	for i := 0; i < 60; i++ {
		open := price
		close := price + 0.5
		high := close + 0.2
		low := open - 0.2
		klines = append(klines, market.KlineBar{
			Time:  int64(1700000000000 + i*60000),
			Open:  open,
			High:  high,
			Low:   low,
			Close: close,
		})
		ema = append(ema, (open+close)/2.0)
		price = close
	}

	tf := &market.TimeframeSeriesData{
		Timeframe:    "1h",
		Klines:       klines,
		EMA21Values:  ema,
		EMA55Values:  ema,
		EMA100Values: ema,
		EMA200Values: ema,
	}

	out, err := RenderTimeframeChartPNG("TESTUSDT", "1h", tf, RenderConfig{Width: 640, Height: 360, MaxBars: 50})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected png bytes")
	}
	if !bytes.HasPrefix(out, []byte{0x89, 0x50, 0x4e, 0x47}) {
		t.Fatalf("expected PNG header")
	}
}

func TestBuildBraleHeaderSummary_IncludesDivergence(t *testing.T) {
	div := &indicator.DivergenceResult{
		Total: 1,
		Events: []indicator.DivergenceEvent{
			{Indicator: "rsi", Type: indicator.DivergencePositiveRegular, BarsAgo: 7},
		},
	}

	s := buildBraleHeaderSummary(nil, nil, div, true)
	if s == "" {
		t.Fatalf("expected non-empty summary")
	}
	if !bytes.Contains([]byte(s), []byte("DIV:")) {
		t.Fatalf("expected DIV summary, got: %q", s)
	}
	if !bytes.Contains([]byte(s), []byte("+R")) {
		t.Fatalf("expected +R abbrev, got: %q", s)
	}
}

func TestBuildBraleHeaderSummary_DivNoneWhenEnabled(t *testing.T) {
	s := buildBraleHeaderSummary(nil, nil, &indicator.DivergenceResult{Total: 0}, true)
	if !bytes.Contains([]byte(s), []byte("DIV: none")) {
		t.Fatalf("expected DIV: none, got: %q", s)
	}
}
