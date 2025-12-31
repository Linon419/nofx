# Exit Plan Execution Design

Date: 2025-12-31
Status: Draft (validated in chat)

## Goal
Enable strategy-configured exit plans to execute tiered take-profit and ATR-based exits, while keeping the rest of the decision and trading flow intact. Stop-loss uses exchange native orders; take-profit tiers execute via system monitoring and market reduce-only orders.

## Scope
- Add exit plan selection to /strategy (Prompt Sections) with a fixed list of plan IDs:
  - plan_tp_tiers_sl_single (default)
  - plan_tp_single_sl_single
  - plan_sl_atr_tp_single
  - plan_tp_atr_sl_single
- Extend decision JSON to include exit_plan with structured children/params.
- Enforce validation rules for exit_plan structure and parameters.
- Execute tiered take-profit via system monitoring (market reduce-only closes).
- Manage stop-loss via exchange native orders, and re-place stop-loss after partial take-profit.
- Persist exit plan runtime state for recovery after restart.

## Non-Goals
- Full OCO orchestration on the exchange.
- Replacing existing stop_loss/take_profit fields; they remain for compatibility.
- Changing leverage, position sizing, or other risk controls beyond exit plan needs.

## Data Model
### Strategy Config
Add to PromptSections:
- exit_strategy_plan: string (enum of allowed plan IDs, default plan_tp_tiers_sl_single).

### Decision JSON
Add optional field:
- exit_plan: object with children[] (component, handler, params) aligned to brale schema.

### Runtime State (DB)
Create/extend a table to store per-position exit plan state:
- trader_id, symbol, side, entry_price, initial_qty, remaining_qty
- plan_id
- children snapshot
- tier status list (pending/triggering/filled/cancelled)
- stop_loss_order_id (current active)
- updated_at, error_state

## Prompt Changes
In BuildSystemPrompt (Decision Process section), append:
- Requirement to output exit_plan based on selected plan.
- Clarify that exit_plan drives execution; stop_loss/take_profit remain for compatibility.
- State that tp_tiers target_price is absolute price and ratios sum to 1.

## Validation Rules
- exit_plan must match selected plan_id and allowed components/handlers.
- tp_tiers tiers length 1-3; ratios > 0; sum ~1.0 (tolerance 0.99-1.01).
- Long: tp targets strictly increasing; short: strictly decreasing.
- sl_single target in opposite direction (long < entry, short > entry).
- ATR params required and within bounds if using tp_atr/sl_atr.

## Execution Flow
1. Open position with existing flow.
2. Place exchange-native stop-loss (reduce-only) at sl_single target.
3. Save exit plan runtime state.
4. Monitor price at fixed interval (30-60s recommended).
5. When tp_tiers target_price is reached, market reduce-only close ratio of remaining_qty.
6. After partial close, cancel and re-place stop-loss for remaining_qty.
7. Continue until remaining_qty == 0 or all tiers filled.

## Error Handling
- Use a per-symbol+side lock to avoid concurrent tier triggers.
- If market close fails, retry N times; if still failing, close all remaining qty.
- If stop-loss re-place fails, close remaining qty as safety fallback.
- Persist errors; pause exit plan after repeated failures.

## Testing
- Unit tests: exit_plan schema validation and price trigger logic.
- Integration tests: mocked exchange for reduce-only close and stop-loss re-place.
- Recovery test: restart with persisted state and resume execution.

## Migration
- Default existing strategies to plan_tp_tiers_sl_single.
- Backfill null exit_strategy_plan to default at load time.
