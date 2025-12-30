# OTC Top Coin Source Design

## Summary

Add a new coin source entry named `otc_top` that adapts the API at
`http://168.138.207.11:3080/api/public/top-otc-crypto`. This source is
configurable in Strategy Studio, can be selected as a standalone source
(`source_type: "otc_top"`), and can be combined with existing sources in
`mixed` mode. The API response is parsed by a new provider that extracts
symbols from `items[]` and orders them by `otc_index` descending. The
resulting symbols flow into existing candidate coin and backtest
resolution logic with a new `sources` tag: `"otc_top"`.

## Goals

- Support the OTC Top API as a first-class coin source.
- Allow standalone selection or mixing with AI500, OI Top, and static coins.
- Keep API parsing robust and consistent with existing SSRF protections.

## Non-Goals

- No changes to the AI500 API format or existing AI500 parsing.
- No new limit field for OTC Top (use full API response).

## Configuration Changes

Update `CoinSourceConfig` to include:

- `UseOTCTop bool`
- `OTCTopAPIURL string`
- Extend `SourceType` enum to include `"otc_top"`

UI changes mirror existing fields for AI500 and OI Top:

- New option in coin source selection: "OTC Top"
- New API URL input when `use_otc_top` is enabled
- i18n strings for labels, placeholder, and validation message

## Provider Changes

Add an OTC Top provider in `provider/data_provider.go`:

- `OTCTopConfig` with `APIURL` and `Timeout`
- `SetOTCTopAPI(apiURL string)`
- `GetOTCTopCoins() ([]OTCTopCoin, error)`
- `GetOTCTopSymbols() ([]string, error)`

Parsing rules:

- Accept API response: `success: true` and `items[]`
- Extract `items[].symbol`
- Order symbols by `otc_index` descending
- Normalize symbols with `market.Normalize`
- Return all symbols (no local limit)
- Use `security.SafeGet` for SSRF protection

## Decision Engine Changes

In `decision/engine.go`:

- Add `source_type: "otc_top"` case in `GetCandidateCoins`
- Add `UseOTCTop` support in `mixed` branch
- Tag `CandidateCoin.Sources` with `"otc_top"`
- Set OTC Top API URL from strategy config

## Backtest and Debate Changes

In `api/backtest.go`:

- Add `"otc_top"` handling in `resolveStrategyCoins`
- Add `UseOTCTop` support in `mixed` branch

In `api/debate.go`:

- Allow auto-selection from OTC Top in `source_type: "otc_top"`
- Extend mixed auto-selection to include OTC Top if enabled

## Error Handling

- `source_type: "otc_top"` + `use_otc_top=false` -> fall back to static coins.
- `source_type: "otc_top"` + API failure -> return error.
- `source_type: "mixed"` -> log OTC Top failure and continue with other sources.
- Missing API URL -> consistent error message from provider.

## Testing

- Unit tests for OTC API parsing and sorting (provider).
- Decision engine tests for `otc_top` and `mixed` behaviors.
- Backtest resolution tests for `otc_top`.
- Manual UI test: Strategy Studio can save and load OTC Top config.

## Rollout

- No default strategy changes.
- Backward compatible: existing configs remain valid.
