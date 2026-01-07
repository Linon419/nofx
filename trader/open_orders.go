package trader

import "nofx/kernel"

// PositionOrderLister is an optional interface implemented by exchange traders that can
// list active orders relevant to an existing position (e.g., TP/SL/conditional reduce-only).
//
// AutoTrader will use this when available to enrich the AI prompt; it is not required for all exchanges.
type PositionOrderLister interface {
	ListPositionOrders(symbol string) ([]kernel.OpenOrderInfo, error)
}

