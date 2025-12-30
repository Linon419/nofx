# Recent Trades Limit Config (Strategy Prompt Context)

## Goal
Allow each strategy to control how many recent closed trades are injected into the User Prompt. Default to 3.

## Approach
Add `prompt_sections.recent_trades_limit` to the strategy configuration. The backend uses it when calling `GetRecentTrades`, with a fallback to 3 when unset or <= 0. The frontend exposes a numeric control in Strategy Studio (Prompt Sections editor).

## Data Flow
- Strategy config stores `prompt_sections.recent_trades_limit`.
- AutoTrader reads the strategy config and uses the limit to fetch recent trades.
- Prompt building remains unchanged and only renders the section when trades exist.

## UI
Prompt Sections editor shows a number input (min 1) with a reset-to-default button. It displays 3 as the default even if the field is absent.

## Defaults & Backward Compatibility
`GetDefaultStrategyConfig` sets `RecentTradesLimit` to 3. Older strategies without the field fall back to 3 in AutoTrader.

## Docs
Update STRATEGY_MODULE docs to describe ¡°Last N closed trades (default 3)¡± and add `RecentTradesLimit` to the PromptSections struct listing.

## Testing
Run `go test ./...` and review failures unrelated to this change.
