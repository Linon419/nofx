package trader

import (
	"math"
	"testing"

	decision "nofx/kernel"
	"nofx/store"
)

func TestApplyStopLossSizing_OverridesPositionSize(t *testing.T) {
	at := &AutoTrader{
		config: AutoTraderConfig{
			StrategyConfig: &store.StrategyConfig{
				RiskControl: store.RiskControlConfig{
					ATREnabled:             false,
					StopLossSizingEnabled:  true,
					StopLossRiskPct:        5.0,
					BTCETHMaxLeverage:      50,
					AltcoinMaxLeverage:     20,
					MaxPositions:           1,
					MaxMarginUsage:         1.0,
					MinPositionSize:        0,
					MinRiskRewardRatio:     0,
					MinConfidence:          0,
					BTCETHMaxPositionValueRatio:  100,
					AltcoinMaxPositionValueRatio: 100,
				},
			},
		},
	}

	dec := &decision.Decision{
		Symbol:          "BTCUSDT",
		Action:          "open_long",
		Leverage:        10,
		PositionSizeUSD: 123,
		StopLoss:        90,
		TakeProfit:      150,
	}

	at.applyStopLossSizing(dec, 100, 1000)

	// equity=1000, risk=5% => maxLoss=50; stopDist=10% => size=50*10/0.1 = 5000
	if math.Abs(dec.PositionSizeUSD-5000) > 1e-6 {
		t.Fatalf("expected position_size_usd=5000, got %.6f", dec.PositionSizeUSD)
	}
}

func TestApplyStopLossSizing_NoStopLoss_NoChange(t *testing.T) {
	at := &AutoTrader{
		config: AutoTraderConfig{
			StrategyConfig: &store.StrategyConfig{
				RiskControl: store.RiskControlConfig{
					ATREnabled:            false,
					StopLossSizingEnabled: true,
					StopLossRiskPct:       5.0,
				},
			},
		},
	}

	dec := &decision.Decision{
		Symbol:          "BTCUSDT",
		Action:          "open_long",
		Leverage:        10,
		PositionSizeUSD: 123,
		StopLoss:        0,
		TakeProfit:      150,
	}

	at.applyStopLossSizing(dec, 100, 1000)

	if math.Abs(dec.PositionSizeUSD-123) > 1e-6 {
		t.Fatalf("expected no change, got %.6f", dec.PositionSizeUSD)
	}
}

