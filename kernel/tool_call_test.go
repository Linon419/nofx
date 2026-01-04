package kernel

import (
	"fmt"
	"nofx/market"
	"nofx/mcp"
	"nofx/store"
	"testing"
	"time"
)

type fakeToolOnlyAIClient struct {
	toolArgs string
	calls    int
}

func (f *fakeToolOnlyAIClient) SetAPIKey(apiKey string, customURL string, customModel string) {}
func (f *fakeToolOnlyAIClient) SetTimeout(timeout time.Duration)                              {}

func (f *fakeToolOnlyAIClient) CallWithMessages(systemPrompt, userPrompt string) (string, error) {
	return "", fmt.Errorf("CallWithMessages should not be used when tool calling succeeds")
}

func (f *fakeToolOnlyAIClient) CallWithRequest(req *mcp.Request) (string, error) {
	f.calls++
	return f.toolArgs, nil
}

var _ mcp.AIClient = (*fakeToolOnlyAIClient)(nil)

func TestGetFullDecisionWithStrategy_ToolCallingPreferred(t *testing.T) {
	defaultConfig := store.GetDefaultStrategyConfig("en")
	engine := NewStrategyEngine(&defaultConfig)

	ctx := &Context{
		CurrentTime:    time.Now().Format(time.RFC3339),
		RuntimeMinutes: 1,
		CallCount:      1,
		Account: AccountInfo{
			TotalEquity:      100,
			AvailableBalance: 100,
			MarginUsed:       0,
			MarginUsedPct:    0,
			PositionCount:    0,
		},
		Positions:      []PositionInfo{},
		CandidateCoins: []CandidateCoin{{Symbol: "SOLUSDT"}},
		MarketDataMap: map[string]*market.Data{
			"SOLUSDT": {Symbol: "SOLUSDT", CurrentPrice: 100, CurrentEMA21: 95, CurrentEMA55: 90, CurrentEMA100: 80, CurrentEMA200: 70},
		},
		MultiTFMarket: make(map[string]map[string]*market.Data),
		OITopDataMap:  make(map[string]*OITopData),
		QuantDataMap:  make(map[string]*QuantData),
	}

	client := &fakeToolOnlyAIClient{
		toolArgs: `{
  "reasoning": "tool reasoning",
  "decisions": [
    {
      "symbol": "SOLUSDT",
      "action": "open_long",
      "leverage": 2,
      "position_size_usd": 20,
      "stop_loss": 90,
      "take_profit": 150,
      "confidence": 80,
      "risk_usd": 1,
      "reasoning": "open long"
    }
  ]
}`,
	}

	fd, err := GetFullDecisionWithStrategy(ctx, client, engine, "", "", 0)
	if err != nil {
		t.Fatalf("GetFullDecisionWithStrategy() error = %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("expected 1 tool call, got %d", client.calls)
	}
	if fd == nil || len(fd.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %+v", fd)
	}
	if fd.CoTTrace != "tool reasoning" {
		t.Fatalf("expected CoTTrace from tool reasoning, got %q", fd.CoTTrace)
	}
	if fd.Decisions[0].Action != "open_long" || fd.Decisions[0].Symbol != "SOLUSDT" {
		t.Fatalf("unexpected decision: %+v", fd.Decisions[0])
	}
}
