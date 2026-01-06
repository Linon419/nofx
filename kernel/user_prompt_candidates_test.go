package kernel

import (
	"strings"
	"testing"

	"nofx/market"
	"nofx/store"
)

func TestBuildUserPromptWithOptions_IncludesCandidatesEvenWhenMarketDataMissing(t *testing.T) {
	engine := NewStrategyEngine(&store.StrategyConfig{})

	ctx := &Context{
		CurrentTime:    "2026-01-06 00:00:00 UTC",
		RuntimeMinutes: 1,
		CallCount:      1,
		Account: AccountInfo{
			TotalEquity:      100,
			AvailableBalance: 100,
			TotalPnLPct:      0,
			MarginUsedPct:    0,
			PositionCount:    0,
		},
		Positions:      nil,
		CandidateCoins: []CandidateCoin{{Symbol: "RIVERUSDT", Sources: []string{"ai500"}}, {Symbol: "0GUSDT", Sources: []string{"ai500"}}, {Symbol: "BSVUSDT", Sources: []string{"static"}}},
		MarketDataMap: map[string]*market.Data{
			"RIVERUSDT": {Symbol: "RIVERUSDT", CurrentPrice: 15.9},
			"0GUSDT":    {Symbol: "0GUSDT", CurrentPrice: 0.98},
			// BSVUSDT intentionally missing to simulate upstream fetch/filter mismatch.
		},
	}

	out := engine.BuildUserPromptWithOptions(ctx, UserPromptOptions{})
	if !strings.Contains(out, "## Candidate Coins (3 coins)") {
		t.Fatalf("expected candidate count to reflect candidate list, got output: %q", out)
	}
	if !strings.Contains(out, "### 3. BSVUSDT") {
		t.Fatalf("expected missing-data candidate to still be listed, got output: %q", out)
	}
	if !strings.Contains(out, "Market Data: unavailable for this symbol") {
		t.Fatalf("expected missing-data placeholder for candidate, got output: %q", out)
	}
}

