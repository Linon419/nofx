# Aligned Decision Scheduler Design

## Overview
This design aligns live decision cycles to candle close times, mirroring brale's behavior. The minimum timeframe from the strategy configuration becomes the alignment anchor. Decision cadence is expressed as a multiple of that anchor. We also drop the latest unclosed kline (with a grace window) before building indicators and prompts to ensure decisions only use closed data.

## Goals
- Align live decisions to candle closes using the shortest selected timeframe.
- Support a decision cadence multiplier and offset seconds.
- Keep existing trader scan interval as a fallback only.
- Ensure market data excludes unclosed bars in live mode.

## Non-goals
- No changes to backtest timing (already close-based).
- No per-symbol parallel decision loops.
- No UI redesign beyond adding a few strategy fields.

## Strategy Config Additions
Add these fields under `StrategyConfig.Indicators.Klines`:
- `decision_interval_multiple` (int, default 1)
- `decision_offset_seconds` (int, default 10)
- `decision_run_immediately` (bool, default false)

The minimum timeframe is derived from `selected_timeframes`; if empty, use `primary_timeframe`.

## Scheduling Behavior
Create an aligned scheduler utility similar to brale:
- `alignInterval` = min timeframe duration
- `interval` = alignInterval * decision_interval_multiple (clamp to >= alignInterval)
- First run at next candle close + offset seconds
- Optional immediate run if `decision_run_immediately` is true

`AutoTrader.Run()` uses this scheduler to trigger `runCycle()`. If strategy scheduling config is missing or invalid, fall back to the existing `scan_interval_minutes` ticker.

## Market Data Sanitization
Before indicator computation and prompt formatting, drop the last kline if it is not closed yet:
- Parse timeframe -> duration
- Compute close time from last kline open time
- If now < close time + grace (10s), drop the last bar

Apply this in `market.GetWithTimeframes` or in `BuildDataFromKlines` to ensure all downstream indicators use closed bars only.

## Edge Cases
- Invalid timeframe strings: log and fall back to scan interval.
- Empty `selected_timeframes`: use `primary_timeframe`.
- Offset seconds < 0: clamp to 0.
- Decision multiple <= 0: clamp to 1.

## Testing
- Unit tests for timeframe parsing and drop-unclosed logic.
- Scheduler test: next aligned time computation and offset.
- Manual smoke test: set `selected_timeframes=[15m,1h]`, `decision_interval_multiple=1`, verify decisions occur after 15m closes.
