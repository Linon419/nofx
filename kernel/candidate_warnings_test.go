package kernel

import (
	"strings"
	"testing"

	"nofx/store"
)

func TestGetCandidateCoinsWithWarnings_StaticIgnoresEnabledFlags(t *testing.T) {
	engine := &StrategyEngine{
		config: &store.StrategyConfig{
			CoinSource: store.CoinSourceConfig{
				SourceType:  "static",
				StaticCoins: []string{"BTCUSDT"},
				UseAI500:    true,
				UseOITop:    true,
				UseOTCTop:   false,
			},
		},
	}

	_, warnings, err := engine.GetCandidateCoinsWithWarnings()
	if err != nil {
		t.Fatalf("GetCandidateCoinsWithWarnings returned error: %v", err)
	}

	if len(warnings) == 0 {
		t.Fatalf("expected warnings, got none")
	}

	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "source_type=static") {
		t.Fatalf("expected warning mentioning source_type=static, got: %q", joined)
	}
	if !strings.Contains(joined, "use_ai500") || !strings.Contains(joined, "use_oi_top") {
		t.Fatalf("expected warning to include ignored flags, got: %q", joined)
	}
	if strings.Contains(joined, "use_otc_top") {
		t.Fatalf("did not expect use_otc_top to be mentioned (it was false), got: %q", joined)
	}
}

