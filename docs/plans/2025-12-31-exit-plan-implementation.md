# Exit Plan Implementation Plan
Date: 2025-12-31
Status: Draft

## Goal
Implement real tiered/ATR exit plans with exchange-native stop-loss and system-driven take-profit tiers,
based on the selected plan ID in /strategy.

## Scope
- Add exit_strategy_plan to prompt_sections (backend + frontend).
- Extend decision JSON to include exit_plan structure.
- Validate exit_plan against plan ID constraints.
- Execute TP tiers via system monitor with market reduce-only orders.
- Maintain stop-loss order, re-place after partial TP.
- Persist runtime exit plan state for recovery.

## Plan Steps
1. Data model and API
   - Add prompt_sections.exit_strategy_plan to backend StrategyConfig and API payloads.
   - Add field to web types and /strategy editor with plan dropdown (4 plan IDs).
   - Update default strategy config to include plan_tp_tiers_sl_single.

2. Prompt and schema
   - Update decision schema to include exit_plan object and plan-specific guidance.
   - Extend BuildSystemPrompt Decision Process with exit_plan requirement.

3. Parsing and validation
   - Parse exit_plan in decision JSON.
   - Add validation for tiers, ratios, target_price direction, ATR params.
   - Enforce allowed plan ID list; fallback to default.

4. Runtime state storage
   - Add store/model for exit plan runtime state (per position).
   - Save plan snapshot, tier status, remaining qty, current SL order id.
   - Restore on restart.

5. Execution logic
   - Create exchange-native SL order on open.
   - Add monitoring loop to trigger TP tiers at target_price.
   - Execute market reduce-only close for each tier ratio.
   - After each TP, cancel and re-place SL for remaining qty.

6. Recovery and error handling
   - Serialize per-symbol execution with a lock.
   - Retry strategy for failed market closes and SL re-place.
   - Safety fallback: close remaining qty on repeated failure.

7. Tests and docs
   - Unit tests for validation and trigger logic.
   - Integration tests with mocked trader for TP/SL workflow.
   - Update docs to explain exit_plan execution behavior.

## Checkpoints
- After step 3: confirm decision JSON and validation behavior.
- After step 5: verify TP tiers and SL re-place in a dry run.
- After step 7: review tests and docs.
