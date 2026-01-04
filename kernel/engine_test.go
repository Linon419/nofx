package kernel

import (
	"strings"
	"testing"

	"nofx/market"
	"nofx/store"
)

func TestExtractDecisions_AllowsTildeInReasoning(t *testing.T) {
	aiResponse := "<reasoning>\n" +
		"BTC bullish structure; waiting for pullback.\n" +
		"</reasoning>\n\n" +
		"<decision>\n" +
		"```json\n" +
		"[\n" +
		"  {\n" +
		"    \"symbol\": \"BTCUSDT\",\n" +
		"    \"action\": \"wait\",\n" +
		"    \"confidence\": 45,\n" +
		"    \"reasoning\": \"Wait for pullback to EMA21 ~88336 area before considering long entry.\"\n" +
		"  }\n" +
		"]\n" +
		"```\n" +
		"</decision>"

	decisions, err := extractDecisions(aiResponse)
	if err != nil {
		t.Fatalf("extractDecisions returned error: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}
	if decisions[0].Symbol != "BTCUSDT" {
		t.Fatalf("expected symbol BTCUSDT, got %q", decisions[0].Symbol)
	}
	if decisions[0].Action != "wait" {
		t.Fatalf("expected action wait, got %q", decisions[0].Action)
	}
	if decisions[0].Confidence != 45 {
		t.Fatalf("expected confidence 45, got %d", decisions[0].Confidence)
	}
	if decisions[0].Reasoning == "" {
		t.Fatalf("expected reasoning to be captured")
	}
}

func TestValidateJSONFormat_DoesNotRejectTilde(t *testing.T) {
	jsonContent := `[
  {"symbol":"BTCUSDT","action":"wait","confidence":45,"reasoning":"~88336 is mentioned here"}
]`

	if err := validateJSONFormat(jsonContent); err != nil {
		t.Fatalf("validateJSONFormat returned error: %v", err)
	}
}

func TestFormatMarketData_PrintsTechnicalAnalysisForSelectedTimeframes(t *testing.T) {
	cfg := &store.StrategyConfig{
		Indicators: store.IndicatorConfig{
			EnableEMA:  true,
			EnableMACD: true,
			EnableRSI:  true,
			Klines: store.KlineConfig{
				PrimaryTimeframe:   "3m",
				SelectedTimeframes: []string{"3m", "15m", "1h"},
			},
		},
	}
	engine := &StrategyEngine{config: cfg}

	nowMs := int64(1735640000000)
	makeBars := func(stepMs int64, n int) []market.KlineBar {
		bars := make([]market.KlineBar, 0, n)
		price := 100.0
		for i := 0; i < n; i++ {
			price += 0.1
			bars = append(bars, market.KlineBar{
				Time:   nowMs + int64(i)*stepMs,
				Open:   price - 0.05,
				High:   price + 0.10,
				Low:    price - 0.10,
				Close:  price,
				Volume: 1000 + float64(i),
			})
		}
		return bars
	}

	data := &market.Data{
		Symbol:       "TESTUSDT",
		CurrentPrice: 100.0,
		TimeframeData: map[string]*market.TimeframeSeriesData{
			"3m":  {Timeframe: "3m", Klines: makeBars(3*60*1000, 60)},
			"15m": {Timeframe: "15m", Klines: makeBars(15*60*1000, 60)},
			"1h":  {Timeframe: "1h", Klines: makeBars(60*60*1000, 60)},
		},
	}

	out := engine.formatMarketData(data)

	for _, marker := range []string{
		"=== Technical Analysis (3M) ===",
		"=== Technical Analysis (15M) ===",
		"=== Technical Analysis (1H) ===",
		"\"wavetrend\"",
	} {
		if !strings.Contains(out, marker) {
			t.Fatalf("expected output to contain %q", marker)
		}
	}

	if strings.Contains(out, "鈫?") {
		t.Fatalf("expected timeframe header to not contain mojibake arrow")
	}
}

func TestFormatQuantData_NoMojibakePrefix(t *testing.T) {
	cfg := &store.StrategyConfig{
		Indicators: store.IndicatorConfig{
			EnableQuantNetflow: true,
		},
	}
	engine := &StrategyEngine{config: cfg}

	out := engine.formatQuantData(&QuantData{
		Symbol: "ZECUSDT",
		PriceChange: map[string]float64{
			"1h": 0.01,
		},
	})

	if !strings.Contains(out, "=== ZECUSDT Quantitative Data ===") {
		t.Fatalf("expected quantitative data header, got: %q", out)
	}
	if strings.Contains(out, "馃") {
		t.Fatalf("expected quantitative data header to not contain mojibake emoji prefix")
	}
}
