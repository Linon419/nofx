package trader

import "strings"

func calcRiskRewardRatio(action string, entryPrice, stopLoss, takeProfit float64) float64 {
	action = strings.ToLower(strings.TrimSpace(action))
	if !strings.HasPrefix(action, "open_") {
		return 0
	}
	if entryPrice <= 0 || stopLoss <= 0 || takeProfit <= 0 {
		return 0
	}

	var risk, reward float64
	if strings.Contains(action, "long") {
		risk = entryPrice - stopLoss
		reward = takeProfit - entryPrice
	} else if strings.Contains(action, "short") {
		risk = stopLoss - entryPrice
		reward = entryPrice - takeProfit
	} else {
		return 0
	}

	if risk <= 0 || reward <= 0 {
		return 0
	}
	return reward / risk
}

