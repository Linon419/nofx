# EMA Period Update Design

## Summary

Replace EMA periods from 20/50 to 21/55/100/200 in backend market data output,
defaults, and documentation, while keeping the existing prompt structure.

## Scope

- Replace EMA20/EMA50 fields with EMA21/EMA55/EMA100/EMA200 in `market/types.go`.
- Update EMA calculations in `market/data.go` for current values, timeframe
  series, intraday series, and longer-term context.
- Update prompt formatting in `decision/engine.go` and `market/data.go` to output
  the new EMA fields.
- Update defaults in `store/strategy.go`, `backtest/config.go`, and UI default
  periods in `web/src/components/strategy/IndicatorEditor.tsx`.
- Update docs to reference EMA21/55/100/200 in examples and guidance.

## Non-Goals

- Do not make EMA output dynamic based on `EMAPeriods` (still fixed fields).
- Do not change frontend chart indicator utilities.

## Data Flow

1. Market data retrieval keeps existing timeframes; `Get()` now fetches 200 bars
   for 3m/4h to support EMA200.
2. EMA values are computed with `calculateEMA` for 21/55/100/200.
3. Output includes `current_ema21/55/100/200` plus EMA21/55/100/200 series per
   timeframe where enough bars exist.

## Error Handling

- EMA calculations return 0 when insufficient bars.
- Series values are appended only when the index has enough bars for the period.

## Testing

- `go test ./market/...`
- Optional: `go test ./...`
