package kernel

import "testing"

func TestSelectVisionSymbols_IncludesCandidatesWithoutMarketData(t *testing.T) {
	ctx := &Context{
		CandidateCoins: []CandidateCoin{
			{Symbol: "PEPEUSDT"},
			{Symbol: "UNIUSDT"},
			{Symbol: "LDOUSDT"},
			{Symbol: "BTCUSDT"},
			{Symbol: "ZECUSDT"},
		},
	}

	got := selectVisionSymbols(ctx, 5, []string{"15m", "4h"})
	want := []string{"PEPEUSDT", "UNIUSDT", "LDOUSDT", "BTCUSDT", "ZECUSDT"}

	if len(got) != len(want) {
		t.Fatalf("expected %d symbols, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected got[%d]=%q, got %q (%v)", i, want[i], got[i], got)
		}
	}
}
