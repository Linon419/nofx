package trader

import (
	"fmt"
	"math"
	"nofx/analysis"
	atrend "nofx/analysis/trend"
	"nofx/decision"
	"nofx/market"
	"nofx/store"
	"strings"
)

const (
	aiCloseGuardLiquidationBufferPct = 0.02 // 2% from liquidation price
)

// allowAIClose enforces: AI cannot close an open position unless hard conditions trigger.
//
// Hard conditions (user-defined):
// A) Price touches planned stop loss (exit_plan_state.stop_loss_price)
// B) Price touches final take profit (highest/lowest TP tier target) but position still open
// C) HTF structure reversal (1h/4h) with close confirmation (breaks latest fractal level and slope flips)
// D) System risk hard trigger (near liquidation / margin usage / exit plan paused)
func (at *AutoTrader) allowAIClose(ctx *decision.Context, d *decision.Decision) (bool, string) {
	if at == nil || d == nil {
		return true, "guard_bypass_nil"
	}
	if ctx == nil {
		return true, "guard_bypass_missing_context"
	}
	if at.store == nil {
		return true, "guard_bypass_missing_store"
	}

	action := strings.ToLower(strings.TrimSpace(d.Action))
	if action != "close_long" && action != "close_short" {
		return true, "not_close_action"
	}

	positionSide := "LONG"
	if action == "close_short" {
		positionSide = "SHORT"
	}

	symbol := market.Normalize(d.Symbol)
	if symbol == "" {
		return true, "guard_bypass_empty_symbol"
	}

	openPos, err := at.store.Position().GetOpenPositionBySymbol(at.id, symbol, positionSide)
	if err != nil {
		return true, fmt.Sprintf("guard_bypass_position_lookup_failed:%v", err)
	}
	if openPos == nil {
		return true, "guard_bypass_no_open_position"
	}

	currentPrice := at.resolveCurrentPrice(ctx, symbol)
	if currentPrice <= 0 {
		return true, "guard_bypass_invalid_price"
	}

	// D) System risk hard trigger
	if ok, why := at.guardRiskHardTrigger(ctx, openPos, currentPrice); ok {
		return true, "D_" + why
	}

	state, _ := at.resolveExitPlanState(openPos)
	if state != nil {
		if strings.TrimSpace(state.Status) == "paused" || state.ErrorCount >= 3 {
			return true, "D_exit_plan_paused"
		}

		// A) Stop loss touched
		if state.StopLossPrice > 0 && guardStopLossTouched(positionSide, currentPrice, state.StopLossPrice) {
			return true, "A_stop_loss_touched"
		}

		// B) Final TP touched
		if tp, ok := guardFinalTakeProfitTarget(positionSide, state.Tiers); ok && guardTakeProfitTouched(positionSide, currentPrice, tp) {
			return true, "B_take_profit_touched"
		}
	}

	// C) Higher timeframe structure reversal (confirmed on close)
	if ok, why := at.guardHTFStructureReversal(ctx, symbol, positionSide); ok {
		return true, "C_" + why
	}

	return false, "blocked_exit_plan_active_no_hard_trigger"
}

func (at *AutoTrader) resolveCurrentPrice(ctx *decision.Context, symbol string) float64 {
	if ctx == nil {
		return 0
	}
	if ctx.MarketDataMap != nil {
		if md, ok := ctx.MarketDataMap[symbol]; ok && md != nil && md.CurrentPrice > 0 {
			return md.CurrentPrice
		}
		// Fallback for non-normalized keys
		if md, ok := ctx.MarketDataMap[strings.ToUpper(symbol)]; ok && md != nil && md.CurrentPrice > 0 {
			return md.CurrentPrice
		}
	}
	price, err := at.trader.GetMarketPrice(symbol)
	if err == nil && price > 0 {
		return price
	}
	if md, err := market.Get(symbol); err == nil {
		return md.CurrentPrice
	}
	return 0
}

func (at *AutoTrader) resolveExitPlanState(pos *store.TraderPosition) (*exitPlanState, error) {
	if pos == nil {
		return nil, nil
	}

	if strings.TrimSpace(pos.ExitPlanState) != "" {
		state, err := parseExitPlanState(pos.ExitPlanState)
		if err == nil && state != nil {
			return state, nil
		}
	}

	if strings.TrimSpace(pos.ExitPlanSnapshot) == "" {
		return nil, nil
	}
	plan, err := parseExitPlanSnapshot(pos.ExitPlanSnapshot)
	if err != nil {
		return nil, err
	}
	return buildExitPlanState(pos.Symbol, pos.Side, pos.EntryPrice, pos.Quantity, plan)
}

func (at *AutoTrader) guardRiskHardTrigger(ctx *decision.Context, pos *store.TraderPosition, currentPrice float64) (bool, string) {
	if ctx == nil || pos == nil || currentPrice <= 0 {
		return false, ""
	}

	// Margin usage hard limit
	maxMarginUsage := 0.9
	if at != nil && at.config.StrategyConfig != nil {
		if v := at.config.StrategyConfig.RiskControl.MaxMarginUsage; v > 0 {
			maxMarginUsage = v
		}
	}
	if ctx.Account.MarginUsedPct >= maxMarginUsage && maxMarginUsage > 0 {
		return true, "margin_usage_high"
	}

	// Near liquidation (position-level)
	for _, p := range ctx.Positions {
		if market.Normalize(p.Symbol) != market.Normalize(pos.Symbol) {
			continue
		}
		if strings.ToLower(strings.TrimSpace(p.Side)) != strings.ToLower(strings.TrimSpace(normalizeSide(pos.Side))) {
			continue
		}

		liq := p.LiquidationPrice
		if liq <= 0 {
			return false, ""
		}

		side := normalizeSide(pos.Side)
		if side == "long" {
			dist := (currentPrice - liq) / currentPrice
			if dist <= aiCloseGuardLiquidationBufferPct {
				return true, "near_liquidation"
			}
		} else {
			dist := (liq - currentPrice) / currentPrice
			if dist <= aiCloseGuardLiquidationBufferPct {
				return true, "near_liquidation"
			}
		}
		return false, ""
	}

	return false, ""
}

func guardStopLossTouched(positionSide string, currentPrice, stopLoss float64) bool {
	if currentPrice <= 0 || stopLoss <= 0 {
		return false
	}
	if strings.ToUpper(strings.TrimSpace(positionSide)) == "SHORT" {
		return currentPrice >= stopLoss
	}
	return currentPrice <= stopLoss
}

func guardTakeProfitTouched(positionSide string, currentPrice, takeProfit float64) bool {
	if currentPrice <= 0 || takeProfit <= 0 {
		return false
	}
	if strings.ToUpper(strings.TrimSpace(positionSide)) == "SHORT" {
		return currentPrice <= takeProfit
	}
	return currentPrice >= takeProfit
}

func guardFinalTakeProfitTarget(positionSide string, tiers []exitPlanTierState) (float64, bool) {
	best := 0.0
	found := false
	if strings.ToUpper(strings.TrimSpace(positionSide)) == "SHORT" {
		best = math.Inf(1)
	}
	for _, t := range tiers {
		if t.TargetPrice <= 0 {
			continue
		}
		found = true
		if strings.ToUpper(strings.TrimSpace(positionSide)) == "SHORT" {
			if t.TargetPrice < best {
				best = t.TargetPrice
			}
		} else {
			if t.TargetPrice > best {
				best = t.TargetPrice
			}
		}
	}
	if !found || math.IsInf(best, 1) {
		return 0, false
	}
	return best, true
}

func (at *AutoTrader) guardHTFStructureReversal(ctx *decision.Context, symbol, positionSide string) (bool, string) {
	if ctx == nil || symbol == "" {
		return false, ""
	}
	if ctx.MarketDataMap == nil {
		return false, ""
	}
	md, ok := ctx.MarketDataMap[symbol]
	if !ok || md == nil || len(md.TimeframeData) == 0 {
		return false, ""
	}

	// Prefer 4h + 1h when available; require all available HTF frames to confirm.
	timeframes := []string{"4h", "1h"}
	confirmed := 0
	required := 0

	for _, tf := range timeframes {
		tfData := md.TimeframeData[tf]
		if tfData == nil || len(tfData.Klines) < 10 {
			continue
		}
		required++

		klines := guardConvertKlineBars(tfData.Klines)
		ar := analysis.AnalyzeWithDefaults(klines)
		if ar == nil || ar.Trend == nil || len(tfData.Klines) == 0 {
			continue
		}

		lastClose := tfData.Klines[len(tfData.Klines)-1].Close
		if lastClose <= 0 {
			continue
		}

		if guardTrendStructureReversed(positionSide, lastClose, ar.Trend) {
			confirmed++
		}
	}

	if required == 0 {
		return false, ""
	}
	if confirmed == required {
		return true, fmt.Sprintf("htf_structure_reversal_%dof%d", confirmed, required)
	}
	return false, ""
}

func guardTrendStructureReversed(positionSide string, lastClose float64, tr *atrend.TrendResult) bool {
	if tr == nil || lastClose <= 0 {
		return false
	}

	side := strings.ToUpper(strings.TrimSpace(positionSide))
	if side == "SHORT" {
		if tr.Slope <= 0 {
			return false
		}
		high, ok := guardLatestStructurePoint(tr.StructurePoints, atrend.FractalHigh)
		return ok && lastClose >= high
	}

	// LONG
	if tr.Slope >= 0 {
		return false
	}
	low, ok := guardLatestStructurePoint(tr.StructurePoints, atrend.FractalLow)
	return ok && lastClose <= low
}

func guardLatestStructurePoint(points []atrend.StructurePoint, want atrend.StructurePointType) (float64, bool) {
	for _, p := range points {
		if p.Type == want && p.Price > 0 {
			return p.Price, true
		}
	}
	return 0, false
}

func guardConvertKlineBars(bars []market.KlineBar) []market.Kline {
	klines := make([]market.Kline, 0, len(bars))
	for _, b := range bars {
		if b.Time <= 0 {
			continue
		}
		klines = append(klines, market.Kline{
			OpenTime:  b.Time,
			Open:      b.Open,
			High:      b.High,
			Low:       b.Low,
			Close:     b.Close,
			Volume:    b.Volume,
			CloseTime: b.Time,
		})
	}
	return klines
}
