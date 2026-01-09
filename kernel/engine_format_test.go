package kernel

import (
	"strings"
	"testing"

	"nofx/market"
	"nofx/store"
)

func TestFormatMarketData_TechnicalAnalysisUsesSelectedTimeframesOrder(t *testing.T) {
	cfg := store.StrategyConfig{
		Indicators: store.IndicatorConfig{
			EnableEMA: true,
			Klines: store.KlineConfig{
				PrimaryTimeframe:     "5m",
				EnableMultiTimeframe: true,
				SelectedTimeframes:   []string{"15m", "4h"},
			},
		},
	}
	engine := NewStrategyEngine(&cfg)

	data := &market.Data{
		Symbol: "TESTUSDT",
		TimeframeData: map[string]*market.TimeframeSeriesData{
			"15m": {
				Timeframe: "15m",
				Klines: []market.KlineBar{
					{Time: 1000, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 10},
				},
				EMA21Values:  []float64{1.4},
				EMA55Values:  []float64{1.3},
				EMA100Values: []float64{1.2},
				EMA200Values: []float64{1.1},
			},
			"4h": {
				Timeframe: "4h",
				Klines: []market.KlineBar{
					{Time: 2000, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 10},
				},
				EMA21Values:  []float64{1.4},
				EMA55Values:  []float64{1.3},
				EMA100Values: []float64{1.2},
				EMA200Values: []float64{1.1},
			},
		},
	}

	out := engine.formatMarketData(data)

	idxTf15 := strings.Index(out, "=== 15M Timeframe")
	idxTf4h := strings.Index(out, "=== 4H Timeframe")
	if idxTf15 < 0 || idxTf4h < 0 {
		t.Fatalf("expected timeframe series blocks for selected timeframes, got output:\n%s", out)
	}
	if idxTf15 > idxTf4h {
		t.Fatalf("expected timeframe series blocks ordered by selected_timeframes (15m before 4h), got output:\n%s", out)
	}

	idx15 := strings.Index(out, "=== Technical Analysis (15M) ===")
	if idx15 < 0 {
		t.Fatalf("expected 15m technical analysis block, got output:\n%s", out)
	}
	idx4h := strings.Index(out, "=== Technical Analysis (4H) ===")
	if idx4h < 0 {
		t.Fatalf("expected 4h technical analysis block, got output:\n%s", out)
	}
	if idx15 > idx4h {
		t.Fatalf("expected selected timeframe order (15m before 4h), got output:\n%s", out)
	}

	if !strings.Contains(out, `"ema_snapshot"`) {
		t.Fatalf("expected ema_snapshot injected into technical analysis JSON, got output:\n%s", out)
	}
	if !strings.Contains(out, `"tf":"15m"`) || !strings.Contains(out, `"tf":"4h"`) {
		t.Fatalf("expected ema_snapshot.tf for selected timeframes, got output:\n%s", out)
	}
	if !strings.Contains(out, `"ema21":`) {
		t.Fatalf("expected EMA values in ema_snapshot when available, got output:\n%s", out)
	}
}

func TestFormatMarketData_TechnicalAnalysisOmitsEMASnapshotWhenEMADisabled(t *testing.T) {
	cfg := store.StrategyConfig{
		Indicators: store.IndicatorConfig{
			EnableEMA: false,
			Klines: store.KlineConfig{
				EnableMultiTimeframe: true,
				SelectedTimeframes:   []string{"15m"},
			},
		},
	}
	engine := NewStrategyEngine(&cfg)

	data := &market.Data{
		Symbol: "TESTUSDT",
		TimeframeData: map[string]*market.TimeframeSeriesData{
			"15m": {
				Timeframe: "15m",
				Klines: []market.KlineBar{
					{Time: 1000, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 10},
				},
				EMA21Values:  []float64{1.4},
				EMA55Values:  []float64{1.3},
				EMA100Values: []float64{1.2},
				EMA200Values: []float64{1.1},
			},
		},
	}

	out := engine.formatMarketData(data)

	if strings.Contains(out, `"ema_snapshot"`) {
		t.Fatalf("expected ema_snapshot omitted when EMA is disabled, got output:\n%s", out)
	}
	if strings.Contains(out, `"ema21":`) {
		t.Fatalf("expected no EMA values in technical analysis JSON when EMA is disabled, got output:\n%s", out)
	}
}

func TestFormatMarketData_TechnicalAnalysisEmitsBlockForMissingTimeframe(t *testing.T) {
	cfg := store.StrategyConfig{
		Indicators: store.IndicatorConfig{
			Klines: store.KlineConfig{
				EnableMultiTimeframe: true,
				SelectedTimeframes:   []string{"15m", "1h"},
			},
		},
	}
	engine := NewStrategyEngine(&cfg)

	data := &market.Data{
		Symbol: "TESTUSDT",
		TimeframeData: map[string]*market.TimeframeSeriesData{
			"15m": {
				Timeframe: "15m",
				Klines: []market.KlineBar{
					{Time: 1000, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 10},
				},
			},
		},
	}

	out := engine.formatMarketData(data)

	if !strings.Contains(out, "=== Technical Analysis (1H) ===") {
		t.Fatalf("expected 1h technical analysis block even if missing timeframe data, got output:\n%s", out)
	}
	if !strings.Contains(out, `{"note":"timeframe_data_missing_or_empty"}`) {
		t.Fatalf("expected missing timeframe note, got output:\n%s", out)
	}
}
