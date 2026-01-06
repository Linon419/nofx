package indicator

import (
	"math"
	"nofx/market"
)

// CVDResult represents a cumulative volume delta summary in quote currency.
//
// Calculation per bar (quote currency):
// - buy_quote = taker_buy_quote_volume
// - sell_quote = quote_volume - taker_buy_quote_volume
// - delta_quote = buy_quote - sell_quote = 2*buy_quote - quote_volume
// WindowCVD is the cumulative sum of delta_quote across the provided klines, starting from 0 at the first bar.
type CVDResult struct {
	Available bool    `json:"available"`
	Unit      string  `json:"unit,omitempty"`   // "quote" (e.g. USDT)
	Source    string  `json:"source,omitempty"` // "taker_buy_quote_volume"
	Note      string  `json:"note,omitempty"`
	WindowCVD float64 `json:"window_cvd,omitempty"`
	LastDelta float64 `json:"last_delta,omitempty"`
}

func CalculateCVD(klines []market.Kline) *CVDResult {
	if len(klines) < 2 {
		return &CVDResult{Available: false, Unit: "quote", Source: "taker_buy_quote_volume", Note: "not_enough_klines"}
	}

	var sum float64
	var lastDelta float64
	hasData := false

	for _, k := range klines {
		qv := k.QuoteVolume
		tbq := k.TakerBuyQuoteVolume
		if qv <= 0 || tbq <= 0 || tbq > qv || math.IsNaN(qv) || math.IsNaN(tbq) || math.IsInf(qv, 0) || math.IsInf(tbq, 0) {
			lastDelta = 0
			continue
		}
		hasData = true
		delta := (2 * tbq) - qv
		sum += delta
		lastDelta = delta
	}

	if !hasData {
		return &CVDResult{Available: false, Unit: "quote", Source: "taker_buy_quote_volume", Note: "missing_quote_or_taker_buy_quote_volume"}
	}

	return &CVDResult{
		Available: true,
		Unit:      "quote",
		Source:    "taker_buy_quote_volume",
		WindowCVD: sum,
		LastDelta: lastDelta,
	}
}
