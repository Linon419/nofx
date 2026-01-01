package trader

import (
	"encoding/json"
	"testing"
	"time"

	"nofx/decision"
	"nofx/store"
)

type mockExitPlanTrader struct {
	marketPrice         float64
	closeLongCalls      int
	closeShortCalls     int
	lastCloseQty        float64
	cancelStopLossCalls int
	stopLossCalls       int
	lastStopLossQty     float64
	lastStopLossPrice   float64
}

func (m *mockExitPlanTrader) GetBalance() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *mockExitPlanTrader) GetPositions() ([]map[string]interface{}, error) {
	return nil, nil
}

func (m *mockExitPlanTrader) OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return map[string]interface{}{"orderId": "open-long"}, nil
}

func (m *mockExitPlanTrader) OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return map[string]interface{}{"orderId": "open-short"}, nil
}

func (m *mockExitPlanTrader) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
	m.closeLongCalls++
	m.lastCloseQty = quantity
	return map[string]interface{}{"orderId": "close-long"}, nil
}

func (m *mockExitPlanTrader) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
	m.closeShortCalls++
	m.lastCloseQty = quantity
	return map[string]interface{}{"orderId": "close-short"}, nil
}

func (m *mockExitPlanTrader) SetLeverage(symbol string, leverage int) error { return nil }
func (m *mockExitPlanTrader) SetMarginMode(symbol string, isCrossMargin bool) error {
	return nil
}
func (m *mockExitPlanTrader) GetMarketPrice(symbol string) (float64, error) {
	return m.marketPrice, nil
}
func (m *mockExitPlanTrader) SetStopLoss(symbol string, positionSide string, quantity, stopPrice float64) error {
	m.stopLossCalls++
	m.lastStopLossQty = quantity
	m.lastStopLossPrice = stopPrice
	return nil
}
func (m *mockExitPlanTrader) SetTakeProfit(symbol string, positionSide string, quantity, takeProfitPrice float64) error {
	return nil
}
func (m *mockExitPlanTrader) CancelStopLossOrders(symbol string) error {
	m.cancelStopLossCalls++
	return nil
}
func (m *mockExitPlanTrader) CancelTakeProfitOrders(symbol string) error { return nil }
func (m *mockExitPlanTrader) CancelAllOrders(symbol string) error        { return nil }
func (m *mockExitPlanTrader) CancelStopOrders(symbol string) error       { return nil }
func (m *mockExitPlanTrader) FormatQuantity(symbol string, quantity float64) (string, error) {
	return "", nil
}
func (m *mockExitPlanTrader) GetOrderStatus(symbol string, orderID string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
func (m *mockExitPlanTrader) GetClosedPnL(startTime time.Time, limit int) ([]ClosedPnLRecord, error) {
	return nil, nil
}

func TestProcessExitPlanPosition_TierAndStopLossReplace(t *testing.T) {
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})

	plan := decision.ExitPlan{
		PlanID: "plan_tp_tiers_sl_single",
		Children: []decision.ExitPlanChild{
			{
				Component: "tp_tiers",
				Handler:   "tier_take_profit",
				Params: mustMarshalJSON(t, map[string]interface{}{
					"tiers": []map[string]interface{}{
						{"target_price": 105.0, "ratio": 0.5},
						{"target_price": 110.0, "ratio": 0.5},
					},
				}),
			},
			{
				Component: "sl_single",
				Handler:   "tier_stop_loss",
				Params: mustMarshalJSON(t, map[string]interface{}{
					"tiers": []map[string]interface{}{
						{"target_price": 90.0, "ratio": 1.0},
					},
				}),
			},
		},
	}
	snapshot, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal exit plan: %v", err)
	}

	pos := &store.TraderPosition{
		TraderID:         "trader-1",
		ExchangeID:       "exchange-1",
		ExchangeType:     "binance",
		Symbol:           "BTCUSDT",
		Side:             "LONG",
		Quantity:         10,
		EntryPrice:       100,
		EntryTime:        time.Now(),
		ExitPlanSnapshot: string(snapshot),
		Status:           "OPEN",
	}
	if err := st.Position().Create(pos); err != nil {
		t.Fatalf("create position: %v", err)
	}

	mockTrader := &mockExitPlanTrader{marketPrice: 106}
	at := &AutoTrader{
		store:    st,
		trader:   mockTrader,
		exchange: "binance",
	}

	openPos, err := st.Position().GetOpenPositionBySymbol("trader-1", "BTCUSDT", "LONG")
	if err != nil || openPos == nil {
		t.Fatalf("open position missing: %v", err)
	}

	if err := at.processExitPlanPosition(openPos); err != nil {
		t.Fatalf("processExitPlanPosition: %v", err)
	}

	if mockTrader.closeLongCalls != 1 {
		t.Fatalf("expected close long calls=1, got %d", mockTrader.closeLongCalls)
	}
	if mockTrader.lastCloseQty != 5 {
		t.Fatalf("expected close qty=5, got %.2f", mockTrader.lastCloseQty)
	}
	if mockTrader.cancelStopLossCalls != 1 {
		t.Fatalf("expected cancel stop loss calls=1, got %d", mockTrader.cancelStopLossCalls)
	}
	if mockTrader.stopLossCalls != 1 {
		t.Fatalf("expected stop loss calls=1, got %d", mockTrader.stopLossCalls)
	}
	if mockTrader.lastStopLossQty != 5 {
		t.Fatalf("expected stop loss qty=5, got %.2f", mockTrader.lastStopLossQty)
	}

	updated, err := st.Position().GetOpenPositionBySymbol("trader-1", "BTCUSDT", "LONG")
	if err != nil || updated == nil {
		t.Fatalf("updated position missing: %v", err)
	}

	var state exitPlanState
	if err := json.Unmarshal([]byte(updated.ExitPlanState), &state); err != nil {
		t.Fatalf("unmarshal exit plan state: %v", err)
	}
	if state.RemainingQuantity != 5 {
		t.Fatalf("expected remaining quantity=5, got %.2f", state.RemainingQuantity)
	}
	if len(state.Tiers) < 2 || state.Tiers[0].Status != "filled" || state.Tiers[1].Status != "pending" {
		t.Fatalf("unexpected tier state: %+v", state.Tiers)
	}
}

func mustMarshalJSON(t *testing.T, payload interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}
