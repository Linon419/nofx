package trader

import (
	"fmt"
	"math"
	"nofx/logger"
	"nofx/market"
	"nofx/store"
	"strings"
	"time"
)

type stopLossFlipConfig struct {
	enabled     bool
	runnerRatio float64
	trailPct    float64
	pollEvery   time.Duration
}

type stopLossFlipPosInfo struct {
	side      string // "long"/"short"
	qtyAbs    float64
	markPrice float64
}

func (at *AutoTrader) getStopLossFlipConfig() stopLossFlipConfig {
	cfg := stopLossFlipConfig{
		enabled:     false,
		runnerRatio: 0.3,
		trailPct:    0.003,
		pollEvery:   10 * time.Second,
	}
	if at == nil || at.config.StrategyConfig == nil {
		return cfg
	}

	rc := at.config.StrategyConfig.RiskControl
	cfg.enabled = rc.StopLossFlipEnabled

	if rc.StopLossFlipRunnerRatio > 0 && rc.StopLossFlipRunnerRatio < 1 {
		cfg.runnerRatio = rc.StopLossFlipRunnerRatio
	}
	if rc.StopLossFlipTrailPct > 0 && rc.StopLossFlipTrailPct < 0.2 {
		cfg.trailPct = rc.StopLossFlipTrailPct
	}
	if rc.StopLossFlipPollSecs > 0 && rc.StopLossFlipPollSecs <= 300 {
		cfg.pollEvery = time.Duration(rc.StopLossFlipPollSecs) * time.Second
	}

	return cfg
}

func normalizeClosedSide(side string) string {
	if strings.EqualFold(side, "LONG") || strings.EqualFold(side, "BUY") {
		return "LONG"
	}
	if strings.EqualFold(side, "SHORT") || strings.EqualFold(side, "SELL") {
		return "SHORT"
	}
	s := strings.ToLower(strings.TrimSpace(side))
	if s == "long" {
		return "LONG"
	}
	if s == "short" {
		return "SHORT"
	}
	return strings.ToUpper(s)
}

func reverseSide(side string) string {
	if strings.EqualFold(side, "LONG") {
		return "SHORT"
	}
	return "LONG"
}

func calcRecoveryTarget(entryPrice, stopLossPrice float64) float64 {
	// Recovery target price after a stop-out, based on full reverse size:
	// T = 2*S - E
	return 2*stopLossPrice - entryPrice
}

func isNearStopLoss(exitPrice, stopLossPrice float64, tolerancePct float64) bool {
	if stopLossPrice <= 0 || exitPrice <= 0 {
		return false
	}
	diff := math.Abs(exitPrice-stopLossPrice) / stopLossPrice
	return diff <= tolerancePct
}

func (at *AutoTrader) armStopLossFlip(symbol, side string, entryPrice, stopLossPrice, quantity float64, leverage int) {
	cfg := at.getStopLossFlipConfig()
	if !cfg.enabled || at.store == nil {
		return
	}
	if entryPrice <= 0 || stopLossPrice <= 0 || quantity <= 0 {
		return
	}

	symbol = market.Normalize(symbol)
	side = strings.ToUpper(strings.TrimSpace(side))
	if side != "LONG" && side != "SHORT" {
		return
	}

	// Set reverse conditional order on exchange (triggered at stop-loss price)
	if err := at.trader.SetReverseOrder(symbol, side, quantity, stopLossPrice, leverage); err != nil {
		logger.Infof("[StopLossFlip] failed to set reverse order: %v", err)
		// Continue to create task record anyway for tracking
	}

	task := &store.StopLossFlipTask{
		TraderID:      at.id,
		ExchangeID:    at.exchangeID,
		ExchangeType:  at.exchange,
		Symbol:        symbol,
		Side:          side,
		EntryPrice:    entryPrice,
		StopLossPrice: stopLossPrice,
		Quantity:      quantity,
		Leverage:      leverage,
		RunnerRatio:   cfg.runnerRatio,
		TrailPct:      cfg.trailPct,
		Status:        store.StopLossFlipArmed,
	}

	if err := at.store.StopLossFlip().Create(task); err != nil {
		logger.Infof("[StopLossFlip] failed to arm: %v", err)
		return
	}
	logger.Infof("[StopLossFlip] armed: %s %s qty=%.6f entry=%.6f sl=%.6f lev=%dx",
		symbol, side, quantity, entryPrice, stopLossPrice, leverage)
}

func (at *AutoTrader) startStopLossFlipMonitor() {
	if at == nil || at.store == nil {
		return
	}

	cfg := at.getStopLossFlipConfig()
	if !cfg.enabled {
		return
	}

	at.monitorWg.Add(1)
	go func() {
		defer at.monitorWg.Done()

		lastSync := time.Now().UTC().Add(-1 * time.Minute)
		ticker := time.NewTicker(cfg.pollEvery)
		defer ticker.Stop()

		logger.Infof("[StopLossFlip] monitor started (poll=%v)", cfg.pollEvery)

		for {
			select {
			case <-ticker.C:
				at.stopLossFlipTick(&lastSync)
			case <-at.stopMonitorCh:
				logger.Infof("[StopLossFlip] monitor stopped")
				return
			}
		}
	}()
}

func (at *AutoTrader) stopLossFlipTick(lastSync *time.Time) {
	cfg := at.getStopLossFlipConfig()
	if !cfg.enabled || at.store == nil {
		return
	}

	start := time.Now().UTC().Add(-30 * time.Second)
	if lastSync != nil && !lastSync.IsZero() {
		start = *lastSync
	}

	records, err := at.trader.GetClosedPnL(start, 50)
	if err == nil && len(records) > 0 {
		maxExit := start
		for _, rec := range records {
			if rec.ExitTime.After(maxExit) {
				maxExit = rec.ExitTime
			}
			at.handleClosedPnLRecord(&rec)
		}
		if lastSync != nil {
			*lastSync = maxExit.Add(1 * time.Millisecond)
		}
	}

	at.updateActiveStopLossFlips(cfg)
}

func (at *AutoTrader) handleClosedPnLRecord(rec *ClosedPnLRecord) {
	if rec == nil || at.store == nil {
		return
	}

	symbol := market.Normalize(rec.Symbol)
	side := normalizeClosedSide(rec.Side)
	if symbol == "" || (side != "LONG" && side != "SHORT") {
		return
	}

	task, err := at.store.StopLossFlip().GetLatestArmed(at.id, at.exchangeID, symbol, side)
	if err != nil || task == nil {
		return
	}

	// Detect stop-loss by price comparison (tolerance 0.5%)
	closeType := rec.CloseType
	if closeType == "unknown" || closeType == "" {
		if isNearStopLoss(rec.ExitPrice, task.StopLossPrice, 0.005) {
			closeType = "stop_loss"
			logger.Infof("[StopLossFlip] detected stop-loss by price: exit=%.4f sl=%.4f", rec.ExitPrice, task.StopLossPrice)
		} else {
			return
		}
	} else if strings.ToLower(strings.TrimSpace(closeType)) != "stop_loss" {
		return
	}

	updates := map[string]interface{}{
		"status":           store.StopLossFlipTriggered,
		"close_type":       closeType,
		"close_order_id":   rec.OrderID,
		"close_time_ms":    rec.ExitTime.UTC().UnixMilli(),
		"close_exit_price": rec.ExitPrice,
	}
	_ = at.store.StopLossFlip().Update(task.ID, updates)

	if err := at.tryExecuteStopLossFlip(task); err != nil {
		_ = at.store.StopLossFlip().Update(task.ID, map[string]interface{}{
			"status":        store.StopLossFlipFailed,
			"error_message": err.Error(),
		})
		logger.Infof("[StopLossFlip] failed: %s %s: %v", symbol, side, err)
	}
}

func (at *AutoTrader) isSymbolFlat(symbol string) (bool, error) {
	positions, err := at.trader.GetPositions()
	if err != nil {
		return false, err
	}
	symN := market.Normalize(symbol)
	for _, p := range positions {
		ps, _ := p["symbol"].(string)
		if market.Normalize(ps) != symN {
			continue
		}
		amt, _ := p["positionAmt"].(float64)
		if math.Abs(amt) > 0 {
			return false, nil
		}
	}
	return true, nil
}

func (at *AutoTrader) tryExecuteStopLossFlip(task *store.StopLossFlipTask) error {
	if task == nil {
		return nil
	}
	if task.Status != store.StopLossFlipTriggered && task.Status != store.StopLossFlipArmed {
		return nil
	}

	if err := at.trader.CancelStopOrders(task.Symbol); err != nil {
		logger.Infof("[StopLossFlip] cancel stop orders failed (%s): %v", task.Symbol, err)
	}

	flat, err := at.isSymbolFlat(task.Symbol)
	if err != nil {
		return fmt.Errorf("failed to check positions: %w", err)
	}
	if !flat {
		return fmt.Errorf("position not flat; skip flip")
	}

	revSide := reverseSide(task.Side) // LONG->SHORT, SHORT->LONG
	if err := at.trader.SetMarginMode(task.Symbol, at.config.IsCrossMargin); err != nil {
		logger.Infof("[StopLossFlip] set margin mode failed (%s): %v", task.Symbol, err)
	}

	// Open reverse position at market using the closed quantity.
	var order map[string]interface{}
	if revSide == "LONG" {
		order, err = at.trader.OpenLong(task.Symbol, task.Quantity, task.Leverage)
	} else {
		order, err = at.trader.OpenShort(task.Symbol, task.Quantity, task.Leverage)
	}
	if err != nil {
		return fmt.Errorf("failed to open reverse position: %w", err)
	}

	reverseOrderID := fmt.Sprintf("%v", order["orderId"])
	reverseEntryApprox := task.CloseExitPrice
	if reverseEntryApprox <= 0 {
		if p, perr := at.trader.GetMarketPrice(task.Symbol); perr == nil && p > 0 {
			reverseEntryApprox = p
		}
	}

	recoveryTarget := calcRecoveryTarget(task.EntryPrice, task.StopLossPrice)
	if recoveryTarget <= 0 {
		return fmt.Errorf("invalid recovery target (entry=%.6f sl=%.6f)", task.EntryPrice, task.StopLossPrice)
	}

	// Protective stop for the reverse: original entry price.
	if err := at.trader.SetStopLoss(task.Symbol, revSide, task.Quantity, task.EntryPrice); err != nil {
		logger.Infof("[StopLossFlip] set reverse stop-loss failed (%s): %v", task.Symbol, err)
	}

	// Partial recovery TP: close (1-runnerRatio) at recovery target.
	closeRatio := 1.0 - task.RunnerRatio
	if closeRatio <= 0 || closeRatio >= 1 {
		closeRatio = 0.7
	}
	tpQty := task.Quantity * closeRatio
	if tpQty > 0 {
		if err := at.trader.SetTakeProfit(task.Symbol, revSide, tpQty, recoveryTarget); err != nil {
			logger.Infof("[StopLossFlip] set recovery take-profit failed (%s): %v", task.Symbol, err)
		}
	}

	_ = at.store.StopLossFlip().Update(task.ID, map[string]interface{}{
		"status":              store.StopLossFlipReversed,
		"reverse_side":        revSide,
		"reverse_order_id":    reverseOrderID,
		"reverse_entry_price": reverseEntryApprox,
		"recovery_target":     recoveryTarget,
		"current_stop_price":  task.EntryPrice,
		"last_extreme_price":  reverseEntryApprox,
	})

	logger.Infof("[StopLossFlip] reversed: %s %s qty=%.6f -> %s (tp70%%@%.6f, runner30%% trail=%.3f%%)",
		task.Symbol, task.Side, task.Quantity, revSide, recoveryTarget, task.TrailPct*100)
	return nil
}

func (at *AutoTrader) updateActiveStopLossFlips(cfg stopLossFlipConfig) {
	if at.store == nil {
		return
	}
	tasks, err := at.store.StopLossFlip().ListActive(at.id, at.exchangeID)
	if err != nil || len(tasks) == 0 {
		return
	}

	positions, err := at.trader.GetPositions()
	if err != nil {
		return
	}

	posMap := make(map[string]stopLossFlipPosInfo)
	for _, p := range positions {
		s, _ := p["symbol"].(string)
		side, _ := p["side"].(string)
		amt, _ := p["positionAmt"].(float64)
		mark, _ := p["markPrice"].(float64)

		sN := market.Normalize(s)
		if sN == "" {
			continue
		}
		posMap[sN] = stopLossFlipPosInfo{
			side:      strings.ToLower(side),
			qtyAbs:    math.Abs(amt),
			markPrice: mark,
		}
	}

	for _, t := range tasks {
		sym := market.Normalize(t.Symbol)
		p, ok := posMap[sym]
		if !ok || p.qtyAbs <= 0 {
			_ = at.store.StopLossFlip().Update(t.ID, map[string]interface{}{"status": store.StopLossFlipDone})
			continue
		}

		revSide := strings.ToLower(strings.TrimSpace(t.ReverseSide))
		if revSide == "" {
			continue
		}
		if p.side != strings.ToLower(revSide) {
			// In one-way mode, this likely means manual interference; stop managing.
			continue
		}

		switch t.Status {
		case store.StopLossFlipReversed:
			// Detect partial recovery close by qty shrink.
			targetRunnerQty := t.Quantity * t.RunnerRatio
			if targetRunnerQty <= 0 {
				targetRunnerQty = t.Quantity * 0.3
			}
			if p.qtyAbs <= targetRunnerQty*1.05 {
				breakEven := t.CloseExitPrice
				if breakEven <= 0 {
					breakEven = t.ReverseEntryPrice
				}
				if breakEven <= 0 {
					breakEven = p.markPrice
				}
				_ = at.trader.CancelStopLossOrders(t.Symbol)
				_ = at.trader.SetStopLoss(t.Symbol, strings.ToUpper(t.ReverseSide), p.qtyAbs, breakEven)
				_ = at.store.StopLossFlip().Update(t.ID, map[string]interface{}{
					"status":             store.StopLossFlipRunner,
					"current_stop_price": breakEven,
					"last_extreme_price": p.markPrice,
				})
			}

		case store.StopLossFlipRunner:
			at.updateRunnerTrailingStop(&t, p, cfg)
		}
	}
}

func (at *AutoTrader) updateRunnerTrailingStop(task *store.StopLossFlipTask, p stopLossFlipPosInfo, cfg stopLossFlipConfig) {
	if task == nil || p.qtyAbs <= 0 {
		return
	}
	if cfg.trailPct <= 0 {
		return
	}

	currentPrice := p.markPrice
	if mp, err := at.trader.GetMarketPrice(task.Symbol); err == nil && mp > 0 {
		currentPrice = mp
	}
	if currentPrice <= 0 {
		return
	}

	revSide := strings.ToUpper(strings.TrimSpace(task.ReverseSide))
	if revSide != "LONG" && revSide != "SHORT" {
		return
	}

	extreme := task.LastExtremePrice
	if extreme <= 0 {
		extreme = currentPrice
	}

	newExtreme := extreme
	if revSide == "LONG" {
		newExtreme = math.Max(extreme, currentPrice)
	} else {
		newExtreme = math.Min(extreme, currentPrice)
	}

	desiredStop := task.CurrentStopPrice
	if revSide == "LONG" {
		candidate := newExtreme * (1 - cfg.trailPct)
		// Keep stop below current to avoid immediate trigger.
		if candidate >= currentPrice {
			candidate = currentPrice * 0.999
		}
		if candidate > desiredStop {
			desiredStop = candidate
		}
	} else { // SHORT
		candidate := newExtreme * (1 + cfg.trailPct)
		// Keep stop above current to avoid immediate trigger.
		if candidate <= currentPrice {
			candidate = currentPrice * 1.001
		}
		if desiredStop == 0 || candidate < desiredStop {
			desiredStop = candidate
		}
	}

	// Reduce churn: update only if stop moved meaningfully.
	if task.CurrentStopPrice > 0 {
		movePct := math.Abs(desiredStop-task.CurrentStopPrice) / task.CurrentStopPrice
		if movePct < 0.001 { // <0.1%
			_ = at.store.StopLossFlip().Update(task.ID, map[string]interface{}{"last_extreme_price": newExtreme})
			return
		}
	}

	_ = at.trader.CancelStopLossOrders(task.Symbol)
	if err := at.trader.SetStopLoss(task.Symbol, revSide, p.qtyAbs, desiredStop); err != nil {
		logger.Infof("[StopLossFlip] update trailing stop failed (%s): %v", task.Symbol, err)
		return
	}

	_ = at.store.StopLossFlip().Update(task.ID, map[string]interface{}{
		"last_extreme_price": newExtreme,
		"current_stop_price": desiredStop,
	})
}
