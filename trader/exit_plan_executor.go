package trader

import (
	"encoding/json"
	"fmt"
	"math"
	decision "nofx/kernel"
	"nofx/logger"
	"nofx/market"
	"nofx/store"
	"strconv"
	"strings"
	"sync"
	"time"
)

type pendingExitPlan struct {
	Snapshot string
	State    string
}

type exitPlanTierState struct {
	Component   string  `json:"component"`
	TargetPrice float64 `json:"target_price"`
	Ratio       float64 `json:"ratio"`
	Status      string  `json:"status"`
	FilledQty   float64 `json:"filled_qty,omitempty"`
	FilledAt    string  `json:"filled_at,omitempty"`
}

type exitPlanState struct {
	PlanID            string              `json:"plan_id"`
	Symbol            string              `json:"symbol"`
	Side              string              `json:"side"`
	EntryPrice        float64             `json:"entry_price"`
	InitialQuantity   float64             `json:"initial_quantity"`
	RemainingQuantity float64             `json:"remaining_quantity"`
	StopLossPrice     float64             `json:"stop_loss_price"`
	Tiers             []exitPlanTierState `json:"tiers"`
	LastPrice         float64             `json:"last_price,omitempty"`
	LastCheckedAt     string              `json:"last_checked_at,omitempty"`
	LastError         string              `json:"last_error,omitempty"`
	ErrorCount        int                 `json:"error_count,omitempty"`
	Status            string              `json:"status"`
}

type exitPlanTierParams struct {
	Tiers []exitPlanTier `json:"tiers"`
}

type exitPlanTier struct {
	TargetPrice float64 `json:"target_price"`
	Ratio       float64 `json:"ratio"`
}

type exitPlanAtrParams struct {
	Mode                  string  `json:"mode,omitempty"`
	ATRValue              float64 `json:"atr_value"`
	TriggerMultiplier     float64 `json:"trigger_multiplier"`
	TrailMultiplier       float64 `json:"trail_multiplier"`
	InitialStopMultiplier float64 `json:"initial_stop_multiplier,omitempty"`
}

const defaultExitPlanID = "plan_tp_tiers_sl_single"

func (at *AutoTrader) applyExitPlanOnOpen(decision *decision.Decision, positionSide string, entryPrice, quantity float64) bool {
	if decision == nil || decision.ExitPlan == nil {
		return false
	}

	// Align quantities with exchange precision to avoid mismatch between position qty and TP tier sizes.
	normalizedQty := quantity
	if at.trader != nil {
		if qtyStr, err := at.trader.FormatQuantity(decision.Symbol, quantity); err == nil {
			if q, err := strconv.ParseFloat(strings.TrimSpace(qtyStr), 64); err == nil && q > 0 {
				normalizedQty = q
			}
		}
	}

	stopLossPrice := resolveStopLossFromPlan(decision.ExitPlan, decision.StopLoss, decision.Symbol, positionSide, entryPrice)
	if stopLossPrice > 0 {
		if err := at.trader.SetStopLoss(decision.Symbol, positionSide, normalizedQty, stopLossPrice); err != nil {
			logger.Infof("  Failed to set stop loss (exit plan): %v", err)
		}
	}

	snapshotJSON, err := json.Marshal(decision.ExitPlan)
	if err != nil {
		logger.Infof("  Failed to serialize exit plan snapshot: %v", err)
		return true
	}

	state, err := buildExitPlanState(decision.Symbol, positionSide, entryPrice, normalizedQty, decision.ExitPlan)
	if err != nil {
		logger.Infof("  Failed to build exit plan state: %v", err)
	}

	// Option B: place exchange-native TP trigger orders for tiers, fallback to internal close-on-trigger if placement fails.
	if state != nil {
		at.applyTakeProfitTiersOnOpen(decision.Symbol, positionSide, state)
	}

	stateJSON := ""
	if state != nil {
		if data, err := json.Marshal(state); err == nil {
			stateJSON = string(data)
		}
	}

	at.persistExitPlanSnapshot(decision.Symbol, positionSide, string(snapshotJSON), stateJSON)
	return true
}

func (at *AutoTrader) applyTakeProfitTiersOnOpen(symbol string, positionSide string, state *exitPlanState) {
	if at == nil || at.trader == nil || state == nil {
		return
	}
	if state.InitialQuantity <= 0 || len(state.Tiers) == 0 {
		return
	}

	// Only handle TP components here. Stop-loss is managed separately (exchange-native).
	var tpTiers []*exitPlanTierState
	for i := range state.Tiers {
		tier := &state.Tiers[i]
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(tier.Component)), "tp_") {
			tpTiers = append(tpTiers, tier)
		}
	}
	if len(tpTiers) == 0 {
		return
	}

	// Avoid duplicated TP orders (e.g. restart / retry).
	if err := at.trader.CancelTakeProfitOrders(symbol); err != nil {
		logger.Infof("  Failed to cancel existing take-profit orders: %v", err)
	}

	placedAll := true
	for _, tier := range tpTiers {
		// Preserve backward-compatible state: only place tiers that are pending.
		if strings.TrimSpace(strings.ToLower(tier.Status)) != "pending" {
			continue
		}
		if tier.TargetPrice <= 0 || tier.Ratio <= 0 {
			placedAll = false
			break
		}

		tierQty := state.InitialQuantity * tier.Ratio
		if tierQty <= 0 {
			placedAll = false
			break
		}

		if err := at.trader.SetTakeProfit(symbol, strings.ToUpper(normalizeSide(positionSide)), tierQty, tier.TargetPrice); err != nil {
			logger.Infof("  Failed to set take profit (tier target=%.8f ratio=%.4f): %v", tier.TargetPrice, tier.Ratio, err)
			placedAll = false
			break
		}
	}

	if !placedAll {
		// Fallback (A): keep exit-plan internal TP execution; cleanup any partially placed orders.
		if err := at.trader.CancelTakeProfitOrders(symbol); err != nil {
			logger.Infof("  Failed to cleanup take-profit orders after partial placement: %v", err)
		}
		return
	}

	for _, tier := range tpTiers {
		if strings.TrimSpace(strings.ToLower(tier.Status)) == "pending" {
			tier.Status = "placed"
		}
	}
}

func (at *AutoTrader) persistExitPlanSnapshot(symbol, positionSide, snapshotJSON, stateJSON string) {
	if at.store == nil {
		return
	}

	normalizedSymbol := market.Normalize(symbol)
	side := strings.ToUpper(normalizeSide(positionSide))
	updated, err := at.store.Position().UpdateExitPlanForOpenPosition(at.id, normalizedSymbol, side, snapshotJSON, stateJSON)
	if err != nil {
		logger.Infof("  Failed to update exit plan for %s %s: %v", normalizedSymbol, side, err)
	}
	if updated {
		return
	}

	key := exitPlanKey(normalizedSymbol, side)
	at.pendingExitPlansMu.Lock()
	at.pendingExitPlans[key] = pendingExitPlan{Snapshot: snapshotJSON, State: stateJSON}
	at.pendingExitPlansMu.Unlock()
}

func (at *AutoTrader) monitorExitPlans() error {
	if at.store == nil {
		return nil
	}

	positions, err := at.store.Position().GetOpenPositions(at.id)
	if err != nil {
		return err
	}
	if len(positions) == 0 {
		return nil
	}

	at.applyPendingExitPlans(positions)

	for _, pos := range positions {
		if pos.ExitPlanSnapshot == "" {
			continue
		}
		key := exitPlanKey(pos.Symbol, pos.Side)
		at.withExitPlanLock(key, func() {
			if err := at.processExitPlanPosition(pos); err != nil {
				logger.Infof("Exit plan process failed (%s %s): %v", pos.Symbol, pos.Side, err)
			}
		})
	}

	return nil
}

func (at *AutoTrader) applyPendingExitPlans(positions []*store.TraderPosition) {
	if len(positions) == 0 {
		return
	}

	at.pendingExitPlansMu.Lock()
	defer at.pendingExitPlansMu.Unlock()

	for _, pos := range positions {
		if pos.ExitPlanSnapshot != "" {
			continue
		}
		key := exitPlanKey(pos.Symbol, pos.Side)
		pending, ok := at.pendingExitPlans[key]
		if !ok {
			continue
		}

		plan, err := parseExitPlanSnapshot(pending.Snapshot)
		if err != nil {
			logger.Infof("Exit plan snapshot parse failed (%s): %v", key, err)
			continue
		}

		stateJSON := strings.TrimSpace(pending.State)
		if stateJSON == "" {
			state, err := buildExitPlanState(pos.Symbol, pos.Side, pos.EntryPrice, pos.Quantity, plan)
			if err != nil {
				logger.Infof("Exit plan state build failed (%s): %v", key, err)
				continue
			}
			if state != nil {
				if data, err := json.Marshal(state); err == nil {
					stateJSON = string(data)
				}
			}
		}

		if err := at.store.Position().UpdateExitPlanSnapshotAndState(pos.ID, pending.Snapshot, stateJSON); err != nil {
			logger.Infof("Exit plan snapshot update failed (%s): %v", key, err)
			continue
		}
		delete(at.pendingExitPlans, key)
	}
}

func (at *AutoTrader) processExitPlanPosition(pos *store.TraderPosition) error {
	plan, err := parseExitPlanSnapshot(pos.ExitPlanSnapshot)
	if err != nil {
		return err
	}

	state, err := parseExitPlanState(pos.ExitPlanState)
	if err != nil || state == nil {
		state, err = buildExitPlanState(pos.Symbol, pos.Side, pos.EntryPrice, pos.Quantity, plan)
		if err != nil {
			return err
		}
	}

	if state.InitialQuantity <= 0 {
		state.InitialQuantity = pos.EntryQuantity
		if state.InitialQuantity <= 0 {
			state.InitialQuantity = pos.Quantity
		}
	}

	if pos.Quantity > 0 {
		if state.RemainingQuantity <= 0 {
			state.RemainingQuantity = pos.Quantity
		} else {
			prevRemaining := state.RemainingQuantity
			if pos.Quantity < prevRemaining-exitPlanQuantityEpsilon(prevRemaining) {
				// External close detected (e.g. exchange-native TP orders).
				if at.reconcileExitPlanTiersFromQuantity(state, pos.Quantity) {
					if err := at.replaceStopLoss(pos, state); err != nil {
						state.ErrorCount++
						state.LastError = err.Error()
						if state.ErrorCount >= 3 {
							state.Status = "paused"
							_ = at.emergencyClosePosition(pos.Symbol, normalizeSide(pos.Side))
						}
						return at.store.Position().UpdateExitPlanState(pos.ID, mustMarshalState(state))
					}
				}
			} else if pos.Quantity > prevRemaining+exitPlanQuantityEpsilon(prevRemaining) {
				// Position increased externally (rare). Keep state aligned to avoid over-closing.
				state.RemainingQuantity = pos.Quantity
				if state.InitialQuantity < pos.Quantity {
					state.InitialQuantity = pos.Quantity
				}
			}
		}
	}

	price, err := at.trader.GetMarketPrice(pos.Symbol)
	if err != nil || price <= 0 {
		data, dataErr := market.Get(pos.Symbol)
		if dataErr != nil {
			return fmt.Errorf("failed to fetch price: %w", err)
		}
		price = data.CurrentPrice
	}
	if price <= 0 {
		return fmt.Errorf("invalid price for %s", pos.Symbol)
	}

	state.LastPrice = price
	state.LastCheckedAt = time.Now().Format(time.RFC3339)
	if state.Status == "completed" || state.RemainingQuantity <= 0 {
		state.Status = "completed"
		return at.store.Position().UpdateExitPlanState(pos.ID, mustMarshalState(state))
	}

	isLong := normalizeSide(pos.Side) == "long"
	for i := range state.Tiers {
		tier := &state.Tiers[i]
		status := strings.TrimSpace(strings.ToLower(tier.Status))
		if status != "pending" {
			continue
		}
		// "pending" is the internal execution mode (fallback). For exchange-native TP orders we set "placed".
		if !tierTriggered(isLong, price, tier.TargetPrice) {
			continue
		}

		closeQty := state.InitialQuantity * tier.Ratio
		if closeQty > state.RemainingQuantity {
			closeQty = state.RemainingQuantity
		}
		if closeQty <= 0 {
			tier.Status = "filled"
			continue
		}

		if err := at.executeExitPlanClose(pos, closeQty, price); err != nil {
			state.ErrorCount++
			state.LastError = err.Error()
			if state.ErrorCount >= 3 {
				state.Status = "paused"
				_ = at.emergencyClosePosition(pos.Symbol, normalizeSide(pos.Side))
			}
			return at.store.Position().UpdateExitPlanState(pos.ID, mustMarshalState(state))
		}

		state.ErrorCount = 0
		state.RemainingQuantity = math.Max(0, state.RemainingQuantity-closeQty)
		tier.Status = "filled"
		tier.FilledQty = closeQty
		tier.FilledAt = time.Now().Format(time.RFC3339)

		if state.RemainingQuantity <= 0 {
			state.Status = "completed"
			break
		}

		if err := at.replaceStopLoss(pos, state); err != nil {
			state.ErrorCount++
			state.LastError = err.Error()
			if state.ErrorCount >= 3 {
				state.Status = "paused"
				_ = at.emergencyClosePosition(pos.Symbol, normalizeSide(pos.Side))
			}
			return at.store.Position().UpdateExitPlanState(pos.ID, mustMarshalState(state))
		}
	}

	return at.store.Position().UpdateExitPlanState(pos.ID, mustMarshalState(state))
}

func (at *AutoTrader) executeExitPlanClose(pos *store.TraderPosition, quantity, price float64) error {
	var (
		order map[string]interface{}
		err   error
	)
	for i := 0; i < 3; i++ {
		if normalizeSide(pos.Side) == "long" {
			order, err = at.trader.CloseLong(pos.Symbol, quantity)
		} else {
			order, err = at.trader.CloseShort(pos.Symbol, quantity)
		}
		if err == nil {
			at.recordAndConfirmOrder(order, pos.Symbol, "close_"+normalizeSide(pos.Side), quantity, price, 0, pos.EntryPrice)
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return err
}

func (at *AutoTrader) replaceStopLoss(pos *store.TraderPosition, state *exitPlanState) error {
	if state.StopLossPrice <= 0 || state.RemainingQuantity <= 0 {
		return nil
	}

	if err := at.trader.CancelStopLossOrders(pos.Symbol); err != nil {
		logger.Infof("  Failed to cancel stop loss orders: %v", err)
	}

	var err error
	for i := 0; i < 3; i++ {
		err = at.trader.SetStopLoss(pos.Symbol, strings.ToUpper(normalizeSide(pos.Side)), state.RemainingQuantity, state.StopLossPrice)
		if err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return err
}

func exitPlanQuantityEpsilon(ref float64) float64 {
	if ref <= 0 {
		return 0.000001
	}
	// 0.1% or a tiny absolute fallback for integer-qty symbols.
	return math.Max(0.000001, ref*0.001)
}

func (at *AutoTrader) reconcileExitPlanTiersFromQuantity(state *exitPlanState, newRemaining float64) bool {
	if state == nil || newRemaining < 0 {
		return false
	}
	prevRemaining := state.RemainingQuantity
	if prevRemaining <= 0 || newRemaining >= prevRemaining-exitPlanQuantityEpsilon(prevRemaining) {
		return false
	}

	closedQty := prevRemaining - newRemaining
	if closedQty <= 0 {
		return false
	}

	updated := false
	for i := range state.Tiers {
		tier := &state.Tiers[i]
		if closedQty <= exitPlanQuantityEpsilon(prevRemaining) {
			break
		}

		status := strings.TrimSpace(strings.ToLower(tier.Status))
		// Only reconcile exchange-native tiers (placed). Internal tiers update their state when we execute closes.
		if status != "placed" {
			continue
		}

		tierQty := state.InitialQuantity * tier.Ratio
		if tierQty <= 0 {
			continue
		}
		remainingForTier := tierQty - tier.FilledQty
		if remainingForTier <= 0 {
			tier.Status = "filled"
			continue
		}

		fill := math.Min(remainingForTier, closedQty)
		if fill <= 0 {
			continue
		}
		tier.FilledQty += fill
		updated = true
		closedQty -= fill

		if tierQty-tier.FilledQty <= exitPlanQuantityEpsilon(tierQty) {
			tier.Status = "filled"
			tier.FilledAt = time.Now().Format(time.RFC3339)
		}
	}

	state.RemainingQuantity = newRemaining
	if state.RemainingQuantity <= 0 {
		state.Status = "completed"
	}
	return updated
}

func (at *AutoTrader) withExitPlanLock(key string, fn func()) {
	lock := at.getExitPlanLock(key)
	lock.Lock()
	defer lock.Unlock()
	fn()
}

func (at *AutoTrader) getExitPlanLock(key string) *sync.Mutex {
	at.exitPlanLocksMu.Lock()
	defer at.exitPlanLocksMu.Unlock()
	if lock, ok := at.exitPlanLocks[key]; ok {
		return lock
	}
	lock := &sync.Mutex{}
	at.exitPlanLocks[key] = lock
	return lock
}

func exitPlanKey(symbol, side string) string {
	symbol = market.Normalize(symbol)
	side = normalizeSide(side)
	return symbol + "_" + side
}

func normalizeSide(side string) string {
	if strings.EqualFold(side, "LONG") {
		return "long"
	}
	if strings.EqualFold(side, "SHORT") {
		return "short"
	}
	return strings.ToLower(side)
}

func parseExitPlanSnapshot(snapshot string) (*decision.ExitPlan, error) {
	var plan decision.ExitPlan
	if err := json.Unmarshal([]byte(snapshot), &plan); err != nil {
		return nil, err
	}
	if strings.TrimSpace(plan.PlanID) == "" {
		plan.PlanID = defaultExitPlanID
	}
	return &plan, nil
}

func parseExitPlanState(stateJSON string) (*exitPlanState, error) {
	if strings.TrimSpace(stateJSON) == "" {
		return nil, nil
	}
	var state exitPlanState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func buildExitPlanState(symbol, positionSide string, entryPrice, quantity float64, plan *decision.ExitPlan) (*exitPlanState, error) {
	if plan == nil {
		return nil, fmt.Errorf("exit plan is nil")
	}

	state := &exitPlanState{
		PlanID:            normalizePlanID(plan.PlanID),
		Symbol:            market.Normalize(symbol),
		Side:              normalizeSide(positionSide),
		EntryPrice:        entryPrice,
		InitialQuantity:   quantity,
		RemainingQuantity: quantity,
		Status:            "active",
	}

	var stopLossPrice float64
	for _, child := range plan.Children {
		component := strings.TrimSpace(child.Component)
		switch component {
		case "tp_tiers", "tp_single":
			params, err := parseTierParams(child.Params)
			if err != nil {
				return nil, err
			}
			for _, tier := range params.Tiers {
				state.Tiers = append(state.Tiers, exitPlanTierState{
					Component:   component,
					TargetPrice: tier.TargetPrice,
					Ratio:       tier.Ratio,
					Status:      "pending",
				})
			}
		case "tp_atr":
			params, err := parseAtrParams(child.Params)
			if err != nil {
				return nil, err
			}
			targetPrice, err := resolveATRTargetPrice(state.Side, entryPrice, symbol, params)
			if err != nil {
				return nil, err
			}
			state.Tiers = append(state.Tiers, exitPlanTierState{
				Component:   component,
				TargetPrice: targetPrice,
				Ratio:       1,
				Status:      "pending",
			})
		case "sl_single":
			params, err := parseTierParams(child.Params)
			if err != nil {
				return nil, err
			}
			if len(params.Tiers) > 0 {
				stopLossPrice = params.Tiers[0].TargetPrice
			}
		case "sl_atr":
			params, err := parseAtrParams(child.Params)
			if err != nil {
				return nil, err
			}
			stopLossPrice, err = resolveATRStopLoss(state.Side, entryPrice, symbol, params)
			if err != nil {
				return nil, err
			}
		}
	}

	state.StopLossPrice = stopLossPrice
	return state, nil
}

func resolveStopLossFromPlan(plan *decision.ExitPlan, fallback float64, symbol, positionSide string, entryPrice float64) float64 {
	if plan == nil {
		return fallback
	}
	for _, child := range plan.Children {
		switch child.Component {
		case "sl_single":
			params, err := parseTierParams(child.Params)
			if err == nil && len(params.Tiers) > 0 {
				return params.Tiers[0].TargetPrice
			}
		case "sl_atr":
			params, err := parseAtrParams(child.Params)
			if err != nil {
				continue
			}
			price, err := resolveATRStopLoss(normalizeSide(positionSide), entryPrice, symbol, params)
			if err == nil && price > 0 {
				return price
			}
		}
	}
	return fallback
}

func parseTierParams(raw json.RawMessage) (*exitPlanTierParams, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("missing tier params")
	}
	var params exitPlanTierParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, err
	}
	return &params, nil
}

func parseAtrParams(raw json.RawMessage) (*exitPlanAtrParams, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("missing atr params")
	}
	var params exitPlanAtrParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, err
	}
	return &params, nil
}

func resolveATRTargetPrice(side string, entryPrice float64, symbol string, params *exitPlanAtrParams) (float64, error) {
	atrValue := params.ATRValue
	if atrValue <= 0 {
		atrValue = resolveATRValue(symbol)
	}
	if atrValue <= 0 {
		return 0, fmt.Errorf("atr_value unavailable")
	}
	if side == "long" {
		return entryPrice + atrValue*params.TriggerMultiplier, nil
	}
	return entryPrice - atrValue*params.TriggerMultiplier, nil
}

func resolveATRStopLoss(side string, entryPrice float64, symbol string, params *exitPlanAtrParams) (float64, error) {
	atrValue := params.ATRValue
	if atrValue <= 0 {
		atrValue = resolveATRValue(symbol)
	}
	if atrValue <= 0 {
		return 0, fmt.Errorf("atr_value unavailable")
	}
	if side == "long" {
		return entryPrice - atrValue*params.TriggerMultiplier, nil
	}
	return entryPrice + atrValue*params.TriggerMultiplier, nil
}

func resolveATRValue(symbol string) float64 {
	data, err := market.Get(symbol)
	if err != nil {
		return 0
	}
	if data.LongerTermContext != nil && data.LongerTermContext.ATR14 > 0 {
		return data.LongerTermContext.ATR14
	}
	if data.IntradaySeries != nil && data.IntradaySeries.ATR14 > 0 {
		return data.IntradaySeries.ATR14
	}
	return 0
}

func tierTriggered(isLong bool, price, target float64) bool {
	if isLong {
		return price >= target
	}
	return price <= target
}

func normalizePlanID(planID string) string {
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return defaultExitPlanID
	}
	switch planID {
	case "plan_tp_tiers_sl_single", "plan_tp_single_sl_single", "plan_sl_atr_tp_single", "plan_tp_atr_sl_single":
		return planID
	default:
		return defaultExitPlanID
	}
}

func mustMarshalState(state *exitPlanState) string {
	data, err := json.Marshal(state)
	if err != nil {
		logger.Infof("Exit plan state marshal failed: %v", err)
		return ""
	}
	return string(data)
}
