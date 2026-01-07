package store

import (
	"fmt"
	"math"
	"nofx/logger"
	"strings"
	"time"
)

// PositionBuilder handles position creation and updates with support for:
// - Position averaging (merging multiple opens)
// - Partial closes (reducing quantity)
// - FIFO matching
// - Time-ordered processing
type PositionBuilder struct {
	positionStore *PositionStore
}

// NewPositionBuilder creates a new PositionBuilder
func NewPositionBuilder(positionStore *PositionStore) *PositionBuilder {
	return &PositionBuilder{
		positionStore: positionStore,
	}
}

// ProcessTrade processes a single trade and updates position accordingly
// tradeTimeMs is Unix milliseconds UTC
func (pb *PositionBuilder) ProcessTrade(
	traderID, exchangeID, exchangeType, symbol, side, action string,
	quantity, price, fee, realizedPnL float64,
	tradeTimeMs int64,
	orderID string,
) error {
	if strings.HasPrefix(action, "open_") {
		return pb.handleOpen(traderID, exchangeID, exchangeType, symbol, side, quantity, price, fee, tradeTimeMs, orderID)
	} else if strings.HasPrefix(action, "close_") {
		return pb.handleClose(traderID, exchangeID, exchangeType, symbol, side, quantity, price, fee, realizedPnL, tradeTimeMs, orderID)
	}
	return nil
}

// handleOpen handles opening positions (create new or average into existing)
// tradeTimeMs is Unix milliseconds UTC
func (pb *PositionBuilder) handleOpen(
	traderID, exchangeID, exchangeType, symbol, side string,
	quantity, price, fee float64,
	tradeTimeMs int64,
	orderID string,
) error {
	// Get existing OPEN position for (symbol, side)
	existing, err := pb.positionStore.GetOpenPositionBySymbol(traderID, symbol, side)
	if err != nil {
		return fmt.Errorf("failed to get open position: %w", err)
	}

	nowMs := time.Now().UTC().UnixMilli()
	if existing == nil {
		// Create new position
		position := &TraderPosition{
			TraderID:           traderID,
			ExchangeID:         exchangeID,
			ExchangeType:       exchangeType,
			ExchangePositionID: fmt.Sprintf("sync_%s_%s_%d", symbol, side, tradeTimeMs),
			Symbol:             symbol,
			Side:               side,
			Quantity:           quantity,
			EntryPrice:         price,
			EntryOrderID:       orderID,
			EntryTime:          UnixMilli(tradeTimeMs),
			Leverage:           1,
			Status:             "OPEN",
			Source:             "sync",
			Fee:                fee,
			CreatedAt:          UnixMilli(nowMs),
			UpdatedAt:          UnixMilli(nowMs),
		}
		return pb.positionStore.CreateOpenPosition(position)
	}

	// Merge: Calculate weighted average entry price and update position
	logger.Infof("  📊 Averaging position: %s %s %.6f @ %.2f + %.6f @ %.2f",
		symbol, side, existing.Quantity, existing.EntryPrice, quantity, price)

	// Also update exchange_id and exchange_type if they were empty
	if existing.ExchangeID == "" || existing.ExchangeType == "" {
		if err := pb.positionStore.UpdatePositionExchangeInfo(existing.ID, exchangeID, exchangeType); err != nil {
			logger.Infof("  ⚠️  Failed to update exchange info: %v", err)
		}
	}

	return pb.positionStore.UpdatePositionQuantityAndPrice(existing.ID, quantity, price, fee)
}

// handleClose handles closing positions (partial or full)
// tradeTimeMs is Unix milliseconds UTC
func (pb *PositionBuilder) handleClose(
	traderID, exchangeID, exchangeType, symbol, side string,
	quantity, price, fee, realizedPnL float64,
	tradeTimeMs int64,
	orderID string,
) error {
	// Get OPEN position
	position, err := pb.positionStore.GetOpenPositionBySymbol(traderID, symbol, side)
	if err != nil {
		return fmt.Errorf("failed to get open position: %w", err)
	}

	if position == nil {
		// No open position found.
		// This can happen when history is incomplete (e.g., the opening trade is outside the sync window),
		// or if trades are processed out of order / database was cleared.
		//
		// Best-effort: create a CLOSED position record from the close trade so history UI can still show it.
		if quantity <= 0 || price <= 0 || tradeTimeMs <= 0 {
			logger.Infof("  ⚠️  No matching open position for %s %s (orderID: %s); invalid close trade (qty=%.8f price=%.8f time=%d), skipping",
				symbol, side, orderID, quantity, price, tradeTimeMs)
			return nil
		}

		entryPrice := price
		if strings.ToUpper(side) == "LONG" {
			entryPrice = price - realizedPnL/quantity
		} else {
			entryPrice = price + realizedPnL/quantity
		}
		if entryPrice <= 0 {
			entryPrice = price
		}

		rec := &ClosedPnLRecord{
			Symbol:      symbol,
			Side:        strings.ToUpper(side),
			EntryPrice:  entryPrice,
			ExitPrice:   price,
			Quantity:    quantity,
			RealizedPnL: realizedPnL,
			Fee:         fee,
			Leverage:    1,
			EntryTime:   tradeTimeMs,
			ExitTime:    tradeTimeMs,
			OrderID:     orderID,
			CloseType:   "unknown",
			ExchangeID:  orderID,
		}

		created, createErr := pb.positionStore.CreateFromClosedPnL(traderID, exchangeID, exchangeType, rec)
		if createErr != nil {
			return fmt.Errorf("failed to create closed position from orphan close: %w", createErr)
		}
		if created {
			logger.Infof("  ✅ Orphan close recorded: %s %s qty=%.6f @ %.2f (entry≈%.2f, PnL=%.2f)", symbol, side, quantity, price, entryPrice, realizedPnL)
		} else {
			logger.Infof("  ⚠️  Orphan close ignored (dedup/invalid): %s %s (orderID: %s)", symbol, side, orderID)
		}
		return nil
	}

	const QUANTITY_TOLERANCE = 0.0001

	// Calculate realized PnL if not provided (some exchanges like Lighter don't return it)
	if realizedPnL == 0 && position.EntryPrice > 0 {
		if side == "LONG" {
			realizedPnL = (price - position.EntryPrice) * quantity
		} else {
			realizedPnL = (position.EntryPrice - price) * quantity
		}
		// Round to 2 decimal places
		realizedPnL = math.Round(realizedPnL*100) / 100
	}

	if quantity < position.Quantity-QUANTITY_TOLERANCE {
		// Partial close: reduce quantity and update weighted average exit price
		logger.Infof("  📉 Partial close: %s %s %.6f → %.6f (closed %.6f @ %.2f, PnL: %.2f)",
			symbol, side, position.Quantity, position.Quantity-quantity, quantity, price, realizedPnL)
		return pb.positionStore.ReducePositionQuantity(position.ID, quantity, price, fee, realizedPnL)
	} else {
		// Full close (or close with tolerance): mark as CLOSED
		closeQty := quantity
		if quantity > position.Quantity {
			logger.Infof("  ⚠️  Over-close detected: %s %s trying to close %.6f but only %.6f open, closing full position",
				symbol, side, quantity, position.Quantity)
			closeQty = position.Quantity
		}

		// Calculate final weighted average exit price
		// Include previously accumulated partial close prices + this final close
		closedBefore := position.EntryQuantity - position.Quantity
		totalClosed := closedBefore + closeQty
		var finalExitPrice float64
		if totalClosed > 0 {
			finalExitPrice = (position.ExitPrice*closedBefore + price*closeQty) / totalClosed
			finalExitPrice = math.Round(finalExitPrice*100) / 100
		} else {
			finalExitPrice = price
		}

		// Calculate total PnL (existing + new)
		totalPnL := position.RealizedPnL + realizedPnL

		// Calculate total fee (existing + new)
		totalFee := position.Fee + fee

		logger.Infof("  ✅ Full close: %s %s %.6f @ %.2f (avg exit: %.2f, entry: %.2f, PnL: %.2f)",
			symbol, side, closeQty, price, finalExitPrice, position.EntryPrice, totalPnL)

		return pb.positionStore.ClosePositionFully(
			position.ID,
			finalExitPrice,
			orderID,
			tradeTimeMs,
			totalPnL,
			totalFee,
			"sync",
		)
	}
}

// quantitiesMatch checks if two quantities are close enough (within tolerance)
func quantitiesMatch(a, b float64) bool {
	const QUANTITY_TOLERANCE = 0.0001
	return math.Abs(a-b) < QUANTITY_TOLERANCE
}
