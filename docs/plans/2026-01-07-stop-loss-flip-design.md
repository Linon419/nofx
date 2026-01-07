# Stop-Loss Flip (One-Way) Design

Date: 2026-01-07  
Status: Draft

## Goal

When a position is stopped out (stop-loss triggers), automatically open an equal-sized reverse position to reduce the impact of a bad decision, recover losses, and keep a small “runner” to capture continued trend.

This design targets **one-way (net) position mode** behavior (no true long+short hedge on the same symbol). If the exchange/account is in hedge mode, a later iteration can add true “near-SL hedge orders”.

## Core Behavior (One-Way / Net)

### 1) Arm on open

When the system executes an AI open decision (`open_long`/`open_short`) and successfully places the exchange-native stop-loss, it stores an “armed flip” record with:

- symbol, side, leverage
- entry_price (approx = open time market price)
- stop_loss_price (from AI decision)
- quantity (derived from position_size_usd / price)
- timestamps
- status = `ARMED`

### 2) Trigger on stop-loss close

A background monitor polls `Trader.GetClosedPnL()` and looks for records with:

- `CloseType == "stop_loss"`
- matching `symbol` + `side`
- a matching “armed flip” record still in `ARMED`

When detected, the monitor:

1. Cancels remaining stop/TP orders for the symbol (best-effort)
2. Verifies the position is flat (no open position for that symbol)
3. Opens the **reverse position** at market using the same leverage and the closed quantity

### 3) Recovery target + runner

Let:

- `E` = original entry price
- `S` = original stop-loss price

After stop-loss triggers, the bot opens the reverse position and sets:

- **Reverse stop-loss**: `E` (caps additional loss if price reverts)
- **Recovery TP (partial)**: target price `T = 2*S - E`
- **Partial close**: close `70%` at `T`, leave `30%` as runner

This “recovery” target aims to roughly offset the original 1R loss when the reverse move reaches `T`.

### 4) Runner management

After the partial recovery TP is executed (detected by current position qty shrinking to ~30%), the bot:

- moves stop-loss up to “break-even” (approx = stop-loss exit price)
- then trails the stop using a percent trail (default configurable)

## Safety / Guards

- Feature is **disabled by default**
- Only triggers on `CloseType == "stop_loss"`
- Only triggers when a matching `ARMED` record exists
- Only triggers once per armed record (status transition enforced)
- Best-effort order cancel to avoid stale conditional orders

## Configuration (Risk Control)

Add fields under `risk_control`:

- `stop_loss_flip_enabled` (bool, default false)
- `stop_loss_flip_runner_ratio` (float, default 0.3)
- `stop_loss_flip_trail_pct` (float, default 0.003 = 0.3%)
- `stop_loss_flip_poll_secs` (int, default 10)

## Testing

- Unit test recovery target calculation (`T = 2*S - E`) for long/short cases
- Smoke test: `go test ./...`

