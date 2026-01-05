package kernel

import (
	"strings"
	"testing"
)

func TestValidateDecision_MissingPriceConvertsToWait(t *testing.T) {
	t.Run("keeps existing reasoning", func(t *testing.T) {
		decision := Decision{
			Symbol:          "BSVUSDT",
			Action:          "open_long",
			Leverage:        2,
			PositionSizeUSD: 50,
			StopLoss:        90,
			TakeProfit:      110,
			Confidence:      80,
			RiskUSD:         5,
			Reasoning:       "test",
		}

		priceLookup := func(symbol string) (float64, bool) { return 0, false }

		err := validateDecision(
			&decision,
			priceLookup,
			1000, // accountEquity
			50,   // btcEthLeverage
			40,   // altcoinLeverage
			0.2,  // btcEthPosRatio
			0.1,  // altcoinPosRatio
			12,   // minPositionSize
			true, // enforceMinPositionSize
			3.0,  // minRiskRewardRatio
			"",   // exitPlanID
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if decision.Action != "wait" {
			t.Fatalf("expected action=wait, got %q", decision.Action)
		}
		if decision.Leverage != 0 || decision.PositionSizeUSD != 0 || decision.StopLoss != 0 || decision.TakeProfit != 0 {
			t.Fatalf("expected trade fields cleared, got leverage=%d size=%.2f sl=%.2f tp=%.2f", decision.Leverage, decision.PositionSizeUSD, decision.StopLoss, decision.TakeProfit)
		}
		if decision.ExitPlan != nil || decision.Confidence != 0 || decision.RiskUSD != 0 {
			t.Fatalf("expected non-trade fields cleared, got exitPlan=%v confidence=%d risk=%.2f", decision.ExitPlan, decision.Confidence, decision.RiskUSD)
		}

		if !strings.Contains(decision.Reasoning, "test") ||
			!strings.Contains(decision.Reasoning, "auto-wait:") ||
			!strings.Contains(decision.Reasoning, "missing current price") ||
			!strings.Contains(decision.Reasoning, "BSVUSDT") {
			t.Fatalf("unexpected reasoning: %q", decision.Reasoning)
		}
	})

	t.Run("sets reasoning when empty", func(t *testing.T) {
		decision := Decision{
			Symbol:          "BSVUSDT",
			Action:          "open_short",
			Leverage:        2,
			PositionSizeUSD: 50,
			StopLoss:        110,
			TakeProfit:      90,
		}

		priceLookup := func(symbol string) (float64, bool) { return 0, false }

		err := validateDecision(
			&decision,
			priceLookup,
			1000, // accountEquity
			50,   // btcEthLeverage
			40,   // altcoinLeverage
			0.2,  // btcEthPosRatio
			0.1,  // altcoinPosRatio
			12,   // minPositionSize
			true, // enforceMinPositionSize
			3.0,  // minRiskRewardRatio
			"",   // exitPlanID
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if decision.Action != "wait" {
			t.Fatalf("expected action=wait, got %q", decision.Action)
		}
		if strings.TrimSpace(decision.Reasoning) == "" || !strings.HasPrefix(decision.Reasoning, "auto-wait:") {
			t.Fatalf("expected auto-wait reasoning, got %q", decision.Reasoning)
		}
	})
}
