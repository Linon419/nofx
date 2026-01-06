package indicator

import (
	"nofx/market"
	"testing"
)

func TestCalculateCVD_UsesTakerBuyQuoteVolume(t *testing.T) {
	klines := []market.Kline{
		{OpenTime: 1, QuoteVolume: 100, TakerBuyQuoteVolume: 60},
		{OpenTime: 2, QuoteVolume: 200, TakerBuyQuoteVolume: 120},
		{OpenTime: 3, QuoteVolume: 150, TakerBuyQuoteVolume: 30},
	}
	// delta = 2*tbq - qv
	// 1: 120-100=+20
	// 2: 240-200=+40
	// 3: 60-150=-90
	// sum = -30
	res := CalculateCVD(klines)
	if res == nil || !res.Available {
		t.Fatalf("expected available cvd, got %+v", res)
	}
	if res.WindowCVD != -30 {
		t.Fatalf("want window_cvd=-30, got %v", res.WindowCVD)
	}
	if res.LastDelta != -90 {
		t.Fatalf("want last_delta=-90, got %v", res.LastDelta)
	}
}
