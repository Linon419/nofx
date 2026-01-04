package kernel

import (
	"fmt"
	"nofx/market"
	"nofx/mcp"
	"nofx/store"
	"testing"
	"time"
)

type fakeAIClient struct {
	responses []string
	calls     int
}

func (f *fakeAIClient) SetAPIKey(apiKey string, customURL string, customModel string) {}

func (f *fakeAIClient) SetTimeout(timeout time.Duration) {}

func (f *fakeAIClient) CallWithMessages(systemPrompt, userPrompt string) (string, error) {
	if f.calls >= len(f.responses) {
		return f.responses[len(f.responses)-1], nil
	}
	resp := f.responses[f.calls]
	f.calls++
	return resp, nil
}

func (f *fakeAIClient) CallWithRequest(req *mcp.Request) (string, error) {
	return "", fmt.Errorf("CallWithRequest not supported in this test client")
}

var _ mcp.AIClient = (*fakeAIClient)(nil)

func TestGetFullDecisionWithStrategy_RetryWhenMissingJSONDecisionArray(t *testing.T) {
	prevDelay := decisionFormatRetryBaseDelay
	decisionFormatRetryBaseDelay = 0
	t.Cleanup(func() { decisionFormatRetryBaseDelay = prevDelay })

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

	client := &fakeAIClient{
		responses: []string{
			"<reasoning>analysis</reasoning>\n<decision>\nOPEN_LONG SOLUSDT\n</decision>",
			"<reasoning>ok</reasoning>\n<decision>\n```json\n[\n  {\n    \"symbol\": \"SOLUSDT\",\n    \"action\": \"open_long\",\n    \"leverage\": 2,\n    \"position_size_usd\": 20,\n    \"stop_loss\": 90,\n    \"take_profit\": 150,\n    \"confidence\": 80,\n    \"risk_usd\": 1,\n    \"reasoning\": \"test\"\n  }\n]\n```\n</decision>",
		},
	}

	fd, err := GetFullDecisionWithStrategy(ctx, client, engine, "", "", 0)
	if err != nil {
		t.Fatalf("GetFullDecisionWithStrategy() error = %v", err)
	}
	if client.calls != 2 {
		t.Fatalf("expected 2 AI calls (1 retry), got %d", client.calls)
	}
	if fd == nil || len(fd.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %+v", fd)
	}
	if fd.Decisions[0].Action != "open_long" || fd.Decisions[0].Symbol != "SOLUSDT" {
		t.Fatalf("unexpected decision: %+v", fd.Decisions[0])
	}
}

func TestGetFullDecisionWithStrategy_ReturnsSafeWaitAfterRetriesExhausted(t *testing.T) {
	prevDelay := decisionFormatRetryBaseDelay
	decisionFormatRetryBaseDelay = 0
	t.Cleanup(func() { decisionFormatRetryBaseDelay = prevDelay })

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

	client := &fakeAIClient{
		responses: []string{
			"<reasoning>analysis</reasoning>\n<decision>\n(no json)\n</decision>",
			"<reasoning>analysis2</reasoning>\n<decision>\n(still no json)\n</decision>",
			"<reasoning>analysis3</reasoning>\n<decision>\n(still no json)\n</decision>",
		},
	}

	fd, err := GetFullDecisionWithStrategy(ctx, client, engine, "", "", 0)
	if err != nil {
		t.Fatalf("GetFullDecisionWithStrategy() error = %v", err)
	}
	if client.calls != 3 {
		t.Fatalf("expected 3 AI calls (2 retries), got %d", client.calls)
	}
	if fd == nil || len(fd.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %+v", fd)
	}
	if fd.Decisions[0].Action != "wait" || fd.Decisions[0].Symbol != "ALL" {
		t.Fatalf("unexpected decision: %+v", fd.Decisions[0])
	}
}
