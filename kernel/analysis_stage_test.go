package kernel

import (
	"nofx/market"
	"nofx/mcp"
	"nofx/store"
	"strings"
	"testing"
	"time"
)

type fakeAnalysisClient struct {
	systemPrompts []string
	userPrompts   []string
	output        string
}

func (f *fakeAnalysisClient) SetAPIKey(apiKey string, customURL string, customModel string) {}
func (f *fakeAnalysisClient) SetTimeout(timeout time.Duration)                              {}

func (f *fakeAnalysisClient) CallWithMessages(systemPrompt, userPrompt string) (string, error) {
	f.systemPrompts = append(f.systemPrompts, systemPrompt)
	f.userPrompts = append(f.userPrompts, userPrompt)
	return f.output, nil
}

func (f *fakeAnalysisClient) CallWithRequest(req *mcp.Request) (string, error) {
	// Pre-analysis is expected to use CallWithMessages (text).
	return "", nil
}

var _ mcp.AIClient = (*fakeAnalysisClient)(nil)

type captureToolDecisionClient struct {
	reqs     []*mcp.Request
	toolArgs string
}

func (c *captureToolDecisionClient) SetAPIKey(apiKey string, customURL string, customModel string) {}
func (c *captureToolDecisionClient) SetTimeout(timeout time.Duration)                              {}

func (c *captureToolDecisionClient) CallWithMessages(systemPrompt, userPrompt string) (string, error) {
	return "", nil
}

func (c *captureToolDecisionClient) CallWithRequest(req *mcp.Request) (string, error) {
	c.reqs = append(c.reqs, req)
	return c.toolArgs, nil
}

var _ mcp.AIClient = (*captureToolDecisionClient)(nil)

func TestGetFullDecisionWithStrategyWithAnalysis_InjectsAnalysisNotesIntoDecisionPrompt(t *testing.T) {
	defaultConfig := store.GetDefaultStrategyConfig("en")
	defaultConfig.Vision.Enabled = false
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

	analysis := &fakeAnalysisClient{
		output: "- Key signal A\n- Key risk B",
	}

	decision := &captureToolDecisionClient{
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

	fd, err := GetFullDecisionWithStrategyWithAnalysis(ctx, decision, analysis, engine, "", "", 0)
	if err != nil {
		t.Fatalf("GetFullDecisionWithStrategyWithAnalysis() error = %v", err)
	}
	if fd == nil || len(fd.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %+v", fd)
	}
	if len(analysis.systemPrompts) != 1 || len(analysis.userPrompts) != 1 {
		t.Fatalf("expected 1 analysis call, got %d/%d", len(analysis.systemPrompts), len(analysis.userPrompts))
	}
	if !strings.Contains(strings.ToLower(analysis.systemPrompts[0]), "analysis") {
		t.Fatalf("expected analysis system prompt, got: %q", analysis.systemPrompts[0])
	}
	if len(decision.reqs) != 1 {
		t.Fatalf("expected 1 decision tool request, got %d", len(decision.reqs))
	}
	req := decision.reqs[0]
	if len(req.Messages) < 2 || req.Messages[1].Role != "user" {
		t.Fatalf("expected user message in tool request, got %+v", req.Messages)
	}
	if !strings.Contains(req.Messages[1].Content, "## AI Analysis Notes") {
		t.Fatalf("expected analysis notes section injected, got: %q", req.Messages[1].Content)
	}
	if !strings.Contains(req.Messages[1].Content, "Key signal A") {
		t.Fatalf("expected analysis output injected, got: %q", req.Messages[1].Content)
	}
}
