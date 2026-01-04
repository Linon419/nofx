package decision

import (
	"encoding/json"
	"testing"
)

func fixedEntryPriceLookup(entry float64) priceLookupFunc {
	return func(string) (float64, bool) { return entry, true }
}

func rrEntryPrice(stopLoss, takeProfit, minRR float64) float64 {
	return (takeProfit + minRR*stopLoss) / (1.0 + minRR)
}

func makeExitPlanForAction(t *testing.T, action string) *ExitPlan {
	t.Helper()
	tpTiers := []exitPlanTier{
		{TargetPrice: 110, Ratio: 0.5},
		{TargetPrice: 120, Ratio: 0.5},
	}
	slTier := exitPlanTier{TargetPrice: 90, Ratio: 1.0}

	if action == "open_short" {
		tpTiers = []exitPlanTier{
			{TargetPrice: 90, Ratio: 0.5},
			{TargetPrice: 80, Ratio: 0.5},
		}
		slTier = exitPlanTier{TargetPrice: 110, Ratio: 1.0}
	}

	tpParams, err := json.Marshal(exitPlanTierParams{Tiers: tpTiers})
	if err != nil {
		t.Fatalf("marshal tp tiers: %v", err)
	}
	slParams, err := json.Marshal(exitPlanTierParams{Tiers: []exitPlanTier{slTier}})
	if err != nil {
		t.Fatalf("marshal sl tiers: %v", err)
	}

	return &ExitPlan{
		PlanID: "plan_tp_tiers_sl_single",
		Children: []ExitPlanChild{
			{Component: "tp_tiers", Handler: "tier_take_profit", Params: tpParams},
			{Component: "sl_single", Handler: "tier_stop_loss", Params: slParams},
		},
	}
}

// TestLeverageFallback tests automatic correction when leverage exceeds limit
func TestLeverageFallback(t *testing.T) {
	const minRR = 3.0
	tests := []struct {
		name            string
		decision        Decision
		accountEquity   float64
		btcEthLeverage  int
		altcoinLeverage int
		wantLeverage    int // Expected leverage after correction
		wantError       bool
	}{
		{
			name: "Altcoin leverage exceeded - auto-correct to limit",
			decision: Decision{
				Symbol:          "SOLUSDT",
				Action:          "open_long",
				Leverage:        20, // Exceeds limit
				PositionSizeUSD: 100,
				StopLoss:        50,
				TakeProfit:      200,
			},
			accountEquity:   100,
			btcEthLeverage:  10,
			altcoinLeverage: 5, // Limit 5x
			wantLeverage:    5, // Should be corrected to 5
			wantError:       false,
		},
		{
			name: "BTC leverage exceeded - auto-correct to limit",
			decision: Decision{
				Symbol:          "BTCUSDT",
				Action:          "open_long",
				Leverage:        20, // Exceeds limit
				PositionSizeUSD: 1000,
				StopLoss:        90000,
				TakeProfit:      110000,
			},
			accountEquity:   100,
			btcEthLeverage:  10, // Limit 10x
			altcoinLeverage: 5,
			wantLeverage:    10, // Should be corrected to 10
			wantError:       false,
		},
		{
			name: "Leverage within limit - no correction",
			decision: Decision{
				Symbol:          "ETHUSDT",
				Action:          "open_short",
				Leverage:        5, // Not exceeded
				PositionSizeUSD: 500,
				StopLoss:        4000,
				TakeProfit:      3000,
			},
			accountEquity:   100,
			btcEthLeverage:  10,
			altcoinLeverage: 5,
			wantLeverage:    5, // Stays unchanged
			wantError:       false,
		},
		{
			name: "Leverage is 0 - should error",
			decision: Decision{
				Symbol:          "SOLUSDT",
				Action:          "open_long",
				Leverage:        0, // Invalid
				PositionSizeUSD: 100,
				StopLoss:        50,
				TakeProfit:      200,
			},
			accountEquity:   100,
			btcEthLeverage:  10,
			altcoinLeverage: 5,
			wantLeverage:    0,
			wantError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.decision.ExitPlan = makeExitPlanForAction(t, tt.decision.Action)
			priceLookup := fixedEntryPriceLookup(rrEntryPrice(tt.decision.StopLoss, tt.decision.TakeProfit, minRR))
			// Use default position value ratios for testing (10x for BTC/ETH, 1.5x for altcoins)
			err := validateDecision(
				&tt.decision,
				priceLookup,
				tt.accountEquity,
				tt.btcEthLeverage,
				tt.altcoinLeverage,
				10.0,
				1.5,
				12.0,
				true,
				minRR,
				"plan_tp_tiers_sl_single",
			)

			// Check error status
			if (err != nil) != tt.wantError {
				t.Errorf("validateDecision() error = %v, wantError %v", err, tt.wantError)
				return
			}

			// If shouldn't error, check if leverage was correctly corrected
			if !tt.wantError && tt.decision.Leverage != tt.wantLeverage {
				t.Errorf("Leverage not corrected: got %d, want %d", tt.decision.Leverage, tt.wantLeverage)
			}
		})
	}
}

// contains checks if string contains substring (helper function)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestValidateDecisions_MutatesSlice(t *testing.T) {
	const minRR = 3.0
	decisions := []Decision{
		{
			Symbol:          "SOLUSDT",
			Action:          "open_long",
			Leverage:        20, // exceed limit, should be clamped
			PositionSizeUSD: 100,
			StopLoss:        50,
			TakeProfit:      200,
			ExitPlan:        nil, // should be auto-filled
		},
	}

	priceLookup := fixedEntryPriceLookup(rrEntryPrice(decisions[0].StopLoss, decisions[0].TakeProfit, minRR))
	// Use default position value ratios for testing (10x for BTC/ETH, 1.5x for altcoins)
	err := validateDecisions(decisions, priceLookup, 1000, 10, 5, 10.0, 1.5, 12.0, true, minRR, "plan_tp_tiers_sl_single")
	if err != nil {
		t.Fatalf("validateDecisions() error = %v", err)
	}

	if decisions[0].Leverage != 5 {
		t.Fatalf("expected leverage to be clamped to 5, got %d", decisions[0].Leverage)
	}
	if decisions[0].ExitPlan == nil || len(decisions[0].ExitPlan.Children) == 0 {
		t.Fatalf("expected exit_plan to be auto-filled, got %+v", decisions[0].ExitPlan)
	}
}

func TestValidateDecision_MinPositionSizeDisabled_AllowsSmallBTC(t *testing.T) {
	const minRR = 3.0
	d := Decision{
		Symbol:          "BTCUSDT",
		Action:          "open_long",
		Leverage:        5,
		PositionSizeUSD: 30,
		StopLoss:        90000,
		TakeProfit:      96000,
		Reasoning:       "test",
	}
	d.ExitPlan = makeExitPlanForAction(t, d.Action)

	err := validateDecision(&d, fixedEntryPriceLookup(rrEntryPrice(d.StopLoss, d.TakeProfit, minRR)), 1000, 10, 5, 10.0, 1.5, 12.0, false, minRR, "plan_tp_tiers_sl_single")
	if err != nil {
		t.Fatalf("validateDecision() error = %v", err)
	}
	if d.Action != "open_long" {
		t.Fatalf("expected action to stay open_long, got %s", d.Action)
	}
	if d.PositionSizeUSD != 30 {
		t.Fatalf("expected position_size_usd to stay 30, got %.2f", d.PositionSizeUSD)
	}
}

func TestValidateDecision_MinPositionSizeEnforced_ConvertsImpossibleOpenToWait(t *testing.T) {
	const minRR = 3.0
	d := Decision{
		Symbol:          "BTCUSDT",
		Action:          "open_long",
		Leverage:        5,
		PositionSizeUSD: 30,
		StopLoss:        90000,
		TakeProfit:      96000,
		Reasoning:       "test",
	}
	d.ExitPlan = makeExitPlanForAction(t, d.Action)

	// cap = equity * btcEthPosRatio = 10.6 * 5 = 53 < 60
	err := validateDecision(&d, fixedEntryPriceLookup(rrEntryPrice(d.StopLoss, d.TakeProfit, minRR)), 10.6, 10, 5, 5.0, 1.5, 12.0, true, minRR, "plan_tp_tiers_sl_single")
	if err != nil {
		t.Fatalf("validateDecision() error = %v", err)
	}
	if d.Action != "wait" {
		t.Fatalf("expected action to be converted to wait, got %s", d.Action)
	}
}

func TestValidateDecision_MinPositionSizeEnforced_MinAboveCapConvertsToWaitEvenIfSizeMeetsMin(t *testing.T) {
	const minRR = 3.0
	d := Decision{
		Symbol:          "BTCUSDT",
		Action:          "open_long",
		Leverage:        5,
		PositionSizeUSD: 100,
		StopLoss:        90000,
		TakeProfit:      96000,
		Reasoning:       "test",
	}
	d.ExitPlan = makeExitPlanForAction(t, d.Action)

	// cap = equity * btcEthPosRatio = 10.6 * 5 = 53 < min(100)
	err := validateDecision(&d, fixedEntryPriceLookup(rrEntryPrice(d.StopLoss, d.TakeProfit, minRR)), 10.6, 10, 5, 5.0, 1.5, 100.0, true, minRR, "plan_tp_tiers_sl_single")
	if err != nil {
		t.Fatalf("validateDecision() error = %v", err)
	}
	if d.Action != "wait" {
		t.Fatalf("expected action to be converted to wait, got %s", d.Action)
	}
}

func TestValidateDecision_MinPositionSizeEnforced_UsesConfigForBTCETH(t *testing.T) {
	const minRR = 3.0
	d := Decision{
		Symbol:          "BTCUSDT",
		Action:          "open_long",
		Leverage:        5,
		PositionSizeUSD: 30,
		StopLoss:        90000,
		TakeProfit:      96000,
		Reasoning:       "test",
	}
	d.ExitPlan = makeExitPlanForAction(t, d.Action)

	// equity cap = 1000 * 10 = 10000, so min can be satisfied.
	err := validateDecision(&d, fixedEntryPriceLookup(rrEntryPrice(d.StopLoss, d.TakeProfit, minRR)), 1000, 10, 5, 10.0, 1.5, 100.0, true, minRR, "plan_tp_tiers_sl_single")
	if err != nil {
		t.Fatalf("validateDecision() error = %v", err)
	}
	if d.PositionSizeUSD != 100.0 {
		t.Fatalf("expected BTC position_size_usd to be adjusted to 100, got %.2f", d.PositionSizeUSD)
	}
}

func TestValidateDecision_PositionSizeOverCap_AutoCaps(t *testing.T) {
	const minRR = 3.0
	d := Decision{
		Symbol:          "BTCUSDT",
		Action:          "open_long",
		Leverage:        5,
		PositionSizeUSD: 100,
		StopLoss:        90000,
		TakeProfit:      96000,
		Reasoning:       "test",
	}
	d.ExitPlan = makeExitPlanForAction(t, d.Action)

	// cap = equity * btcEthPosRatio = 10.6 * 5 = 53
	err := validateDecision(&d, fixedEntryPriceLookup(rrEntryPrice(d.StopLoss, d.TakeProfit, minRR)), 10.6, 10, 5, 5.0, 1.5, 1.0, false, minRR, "plan_tp_tiers_sl_single")
	if err != nil {
		t.Fatalf("validateDecision() error = %v", err)
	}
	if d.PositionSizeUSD > 53.0+1e-9 {
		t.Fatalf("expected position_size_usd to be capped to <=53, got %.2f", d.PositionSizeUSD)
	}
}

func TestValidateDecision_RRTooLowAtCurrentPrice_ConvertsToWait(t *testing.T) {
	const minRR = 3.0
	d := Decision{
		Symbol:          "LDOUSDT",
		Action:          "open_long",
		Leverage:        10,
		PositionSizeUSD: 100,
		StopLoss:        0.6100,
		TakeProfit:      0.6550,
		Reasoning:       "test",
	}
	d.ExitPlan = makeExitPlanForAction(t, d.Action)

	// Current price is worse than the required entry for RR>=3, should auto-convert to wait.
	priceLookup := fixedEntryPriceLookup(0.6308)
	err := validateDecision(&d, priceLookup, 1000, 10, 10, 10.0, 1.5, 12.0, true, minRR, "plan_tp_tiers_sl_single")
	if err != nil {
		t.Fatalf("validateDecision() error = %v", err)
	}
	if d.Action != "wait" {
		t.Fatalf("expected action to be converted to wait, got %s", d.Action)
	}
	if d.StopLoss != 0 || d.TakeProfit != 0 || d.Leverage != 0 || d.PositionSizeUSD != 0 {
		t.Fatalf("expected open parameters to be cleared on wait, got leverage=%d size=%.2f sl=%.4f tp=%.4f", d.Leverage, d.PositionSizeUSD, d.StopLoss, d.TakeProfit)
	}
	if !contains(d.Reasoning, "auto-wait") {
		t.Fatalf("expected reasoning to mention auto-wait, got %q", d.Reasoning)
	}
}
