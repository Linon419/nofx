package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestPositionBuilder_OrphanClose_CreatesClosedPosition(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	defer st.db.Close()

	pb := NewPositionBuilder(st.Position())

	traderID := "t_test"
	exchangeID := "ex_test"
	exchangeType := "bitget"
	symbol := "BTCUSDT"
	side := "LONG"

	// Close trade arrives without a recorded open position (e.g., open is outside sync window).
	quantity := 0.1
	exitPrice := 10000.0
	realizedPnL := 50.0  // profit
	fee := 0.2
	tradeTimeMs := time.Now().UTC().UnixMilli()

	if err := pb.ProcessTrade(
		traderID, exchangeID, exchangeType, symbol, side, "close_long",
		quantity, exitPrice, fee, realizedPnL,
		tradeTimeMs, "trade-1",
	); err != nil {
		t.Fatalf("ProcessTrade: %v", err)
	}

	closed, err := st.Position().GetClosedPositions(traderID, 10)
	if err != nil {
		t.Fatalf("GetClosedPositions: %v", err)
	}
	if len(closed) != 1 {
		t.Fatalf("closed positions = %d, want 1", len(closed))
	}
	got := closed[0]
	if got.Symbol != symbol {
		t.Fatalf("symbol = %q, want %q", got.Symbol, symbol)
	}
	if got.Status != "CLOSED" {
		t.Fatalf("status = %q, want %q", got.Status, "CLOSED")
	}
	if got.ExitPrice != exitPrice {
		t.Fatalf("exit_price = %v, want %v", got.ExitPrice, exitPrice)
	}
	if got.Quantity != quantity {
		t.Fatalf("quantity = %v, want %v", got.Quantity, quantity)
	}
	if got.RealizedPnL != realizedPnL {
		t.Fatalf("realized_pnl = %v, want %v", got.RealizedPnL, realizedPnL)
	}

	// Entry price is inferred from PnL for LONG: entry = exit - pnl/qty.
	wantEntry := exitPrice - realizedPnL/quantity
	if diff := got.EntryPrice - wantEntry; diff < -1e-6 || diff > 1e-6 {
		t.Fatalf("entry_price = %v, want %v", got.EntryPrice, wantEntry)
	}
}

