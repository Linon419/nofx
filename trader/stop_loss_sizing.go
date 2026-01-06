package trader

import (
	"math"
	"strings"

	decision "nofx/kernel"
)

func (at *AutoTrader) applyStopLossSizing(dec *decision.Decision, entryPrice float64, equity float64) {
	if at == nil || dec == nil {
		return
	}
	if at.config.StrategyConfig == nil {
		return
	}
	rc := at.config.StrategyConfig.RiskControl
	if !rc.StopLossSizingEnabled {
		return
	}
	if rc.StopLossRiskPct <= 0 {
		return
	}

	action := strings.ToLower(strings.TrimSpace(dec.Action))
	if !strings.HasPrefix(action, "open_") {
		return
	}
	if entryPrice <= 0 || equity <= 0 {
		return
	}
	if dec.StopLoss <= 0 {
		return
	}

	lev := dec.Leverage
	if lev <= 0 {
		lev = 1
	}

	stopDistPct := math.Abs(entryPrice-dec.StopLoss) / entryPrice
	if stopDistPct <= 0 {
		return
	}

	maxLossUSD := equity * (rc.StopLossRiskPct / 100.0)
	if maxLossUSD <= 0 {
		return
	}

	positionSizeUSD := maxLossUSD * float64(lev) / stopDistPct
	if positionSizeUSD <= 0 || math.IsNaN(positionSizeUSD) || math.IsInf(positionSizeUSD, 0) {
		return
	}
	dec.PositionSizeUSD = positionSizeUSD
}

