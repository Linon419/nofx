package kernel

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

const defaultExitPlanID = "plan_tp_tiers_sl_single"

type exitPlanRequirement struct {
	PlanID     string
	Components []string
}

var exitPlanRequirements = map[string]exitPlanRequirement{
	"plan_tp_tiers_sl_single": {
		PlanID:     "plan_tp_tiers_sl_single",
		Components: []string{"tp_tiers", "sl_single"},
	},
	"plan_tp_single_sl_single": {
		PlanID:     "plan_tp_single_sl_single",
		Components: []string{"tp_single", "sl_single"},
	},
	"plan_sl_atr_tp_single": {
		PlanID:     "plan_sl_atr_tp_single",
		Components: []string{"sl_atr", "tp_single"},
	},
	"plan_tp_atr_sl_single": {
		PlanID:     "plan_tp_atr_sl_single",
		Components: []string{"tp_atr", "sl_single"},
	},
}

var exitPlanComponentHandlers = map[string]string{
	"tp_tiers":  "tier_take_profit",
	"tp_single": "tier_take_profit",
	"sl_single": "tier_stop_loss",
	"sl_tiers":  "tier_stop_loss",
	"tp_atr":    "atr_trailing",
	"sl_atr":    "atr_trailing",
}

var exitPlanComponentModes = map[string]string{
	"tp_atr": "take_profit",
	"sl_atr": "stop_loss",
}

// ExitPlan represents the structured exit plan output by the model.
type ExitPlan struct {
	PlanID               string          `json:"plan_id,omitempty"`
	FinalTakeProfitPrice float64         `json:"final_take_profit_price,omitempty"`
	FinalStopLossPrice   float64         `json:"final_stop_loss_price,omitempty"`
	Children             []ExitPlanChild `json:"children,omitempty"`
}

// ExitPlanChild represents a single exit plan component.
type ExitPlanChild struct {
	Component string          `json:"component"`
	Handler   string          `json:"handler"`
	Params    json.RawMessage `json:"params,omitempty"`
}

type exitPlanTierParams struct {
	Tiers []exitPlanTier `json:"tiers"`
}

type exitPlanTier struct {
	TargetPrice float64 `json:"target_price"`
	Ratio       float64 `json:"ratio"`
}

type exitPlanAtrParams struct {
	Mode                  string  `json:"mode,omitempty"`
	ATRValue              float64 `json:"atr_value"`
	TriggerMultiplier     float64 `json:"trigger_multiplier"`
	TrailMultiplier       float64 `json:"trail_multiplier"`
	InitialStopMultiplier float64 `json:"initial_stop_multiplier,omitempty"`
}

func fillExitPlanFromLegacyFields(decision *Decision, expectedPlanID string) error {
	if decision == nil {
		return fmt.Errorf("decision is nil")
	}

	if decision.Action != "open_long" && decision.Action != "open_short" {
		return nil
	}

	planID := normalizeExitPlanID(expectedPlanID)
	if planID == "" {
		return nil
	}

	if decision.StopLoss <= 0 || decision.TakeProfit <= 0 {
		return fmt.Errorf("exit_plan is required when action is %s (stop_loss and take_profit must be provided to auto-generate exit_plan)", decision.Action)
	}

	makeTierParams := func(tiers []exitPlanTier) (json.RawMessage, error) {
		data, err := json.Marshal(exitPlanTierParams{Tiers: tiers})
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	makeAtrParams := func(params exitPlanAtrParams) (json.RawMessage, error) {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	children := make([]ExitPlanChild, 0, 2)
	switch planID {
	case "plan_tp_single_sl_single":
		tpParams, err := makeTierParams([]exitPlanTier{{TargetPrice: decision.TakeProfit, Ratio: 1.0}})
		if err != nil {
			return err
		}
		slParams, err := makeTierParams([]exitPlanTier{{TargetPrice: decision.StopLoss, Ratio: 1.0}})
		if err != nil {
			return err
		}
		children = append(children,
			ExitPlanChild{Component: "tp_single", Handler: exitPlanComponentHandlers["tp_single"], Params: tpParams},
			ExitPlanChild{Component: "sl_single", Handler: exitPlanComponentHandlers["sl_single"], Params: slParams},
		)
	case "plan_sl_atr_tp_single":
		atrParams, err := makeAtrParams(exitPlanAtrParams{
			Mode:              exitPlanComponentModes["sl_atr"],
			ATRValue:          0,
			TriggerMultiplier: 2.0,
			TrailMultiplier:   1.0,
		})
		if err != nil {
			return err
		}
		tpParams, err := makeTierParams([]exitPlanTier{{TargetPrice: decision.TakeProfit, Ratio: 1.0}})
		if err != nil {
			return err
		}
		children = append(children,
			ExitPlanChild{Component: "sl_atr", Handler: exitPlanComponentHandlers["sl_atr"], Params: atrParams},
			ExitPlanChild{Component: "tp_single", Handler: exitPlanComponentHandlers["tp_single"], Params: tpParams},
		)
	case "plan_tp_atr_sl_single":
		atrParams, err := makeAtrParams(exitPlanAtrParams{
			Mode:              exitPlanComponentModes["tp_atr"],
			ATRValue:          0,
			TriggerMultiplier: 2.0,
			TrailMultiplier:   1.0,
		})
		if err != nil {
			return err
		}
		slParams, err := makeTierParams([]exitPlanTier{{TargetPrice: decision.StopLoss, Ratio: 1.0}})
		if err != nil {
			return err
		}
		children = append(children,
			ExitPlanChild{Component: "tp_atr", Handler: exitPlanComponentHandlers["tp_atr"], Params: atrParams},
			ExitPlanChild{Component: "sl_single", Handler: exitPlanComponentHandlers["sl_single"], Params: slParams},
		)
	default:
		tpParams, err := makeTierParams([]exitPlanTier{{TargetPrice: decision.TakeProfit, Ratio: 1.0}})
		if err != nil {
			return err
		}
		slParams, err := makeTierParams([]exitPlanTier{{TargetPrice: decision.StopLoss, Ratio: 1.0}})
		if err != nil {
			return err
		}
		children = append(children,
			ExitPlanChild{Component: "tp_tiers", Handler: exitPlanComponentHandlers["tp_tiers"], Params: tpParams},
			ExitPlanChild{Component: "sl_single", Handler: exitPlanComponentHandlers["sl_single"], Params: slParams},
		)
	}

	decision.ExitPlan = &ExitPlan{
		PlanID:               planID,
		FinalTakeProfitPrice: decision.TakeProfit,
		FinalStopLossPrice:   decision.StopLoss,
		Children:             children,
	}
	return nil
}

func normalizeExitPlanID(planID string) string {
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return defaultExitPlanID
	}
	if _, ok := exitPlanRequirements[planID]; ok {
		return planID
	}
	return defaultExitPlanID
}

func buildExitPlanPrompt(planID string) string {
	planID = normalizeExitPlanID(planID)
	requirement, ok := exitPlanRequirements[planID]
	if !ok {
		return ""
	}

	componentHints := make([]string, 0, len(requirement.Components))
	hasATR := false
	for _, component := range requirement.Components {
		handler := exitPlanComponentHandlers[component]
		componentHints = append(componentHints, fmt.Sprintf("%s (%s)", component, handler))
		if strings.HasSuffix(component, "_atr") {
			hasATR = true
		}
	}

	var sb strings.Builder
	sb.WriteString("## Exit Plan (Optional Output)\n")
	sb.WriteString(fmt.Sprintf("- plan_id: %s\n", planID))
	sb.WriteString(fmt.Sprintf("- children must include: %s\n", strings.Join(componentHints, ", ")))
	sb.WriteString("- tiers: 1-3, ratio sum = 1, target_price absolute\n")
	sb.WriteString("- long: tp target_price strictly increases; sl strictly decreases. short: reversed\n")
	sb.WriteString("- tp_single/sl_single must use a single tier with ratio = 1\n")
	if hasATR {
		sb.WriteString("- tp_atr/sl_atr must include atr_value, trigger_multiplier, trail_multiplier; mode must match component (atr_value can be 0, system overrides)\n")
	}
	sb.WriteString("- You MAY omit exit_plan; the backend will auto-generate a valid exit_plan from stop_loss/take_profit.\n")
	sb.WriteString("- If you include exit_plan, keep it minimal to avoid truncation (1 tier is acceptable).\n")
	sb.WriteString("- Still output stop_loss and take_profit for compatibility\n\n")
	return sb.String()
}

func buildExitPlanExample(planID string) string {
	planID = normalizeExitPlanID(planID)
	switch planID {
	case "plan_tp_single_sl_single":
		return fmt.Sprintf("    \"exit_plan\": {\n      \"plan_id\": \"%s\",\n      \"children\": [\n        {\"component\": \"tp_single\", \"handler\": \"tier_take_profit\", \"params\": {\"tiers\": [{\"target_price\": 91000, \"ratio\": 1.0}]}},\n        {\"component\": \"sl_single\", \"handler\": \"tier_stop_loss\", \"params\": {\"tiers\": [{\"target_price\": 97000, \"ratio\": 1.0}]}}\n      ]\n    }\n", planID)
	case "plan_sl_atr_tp_single":
		return fmt.Sprintf("    \"exit_plan\": {\n      \"plan_id\": \"%s\",\n      \"children\": [\n        {\"component\": \"sl_atr\", \"handler\": \"atr_trailing\", \"params\": {\"mode\": \"stop_loss\", \"atr_value\": 0, \"trigger_multiplier\": 2.0, \"trail_multiplier\": 1.0}},\n        {\"component\": \"tp_single\", \"handler\": \"tier_take_profit\", \"params\": {\"tiers\": [{\"target_price\": 91000, \"ratio\": 1.0}]}}\n      ]\n    }\n", planID)
	case "plan_tp_atr_sl_single":
		return fmt.Sprintf("    \"exit_plan\": {\n      \"plan_id\": \"%s\",\n      \"children\": [\n        {\"component\": \"tp_atr\", \"handler\": \"atr_trailing\", \"params\": {\"mode\": \"take_profit\", \"atr_value\": 0, \"trigger_multiplier\": 2.0, \"trail_multiplier\": 1.0}},\n        {\"component\": \"sl_single\", \"handler\": \"tier_stop_loss\", \"params\": {\"tiers\": [{\"target_price\": 97000, \"ratio\": 1.0}]}}\n      ]\n    }\n", planID)
	default:
		return fmt.Sprintf("    \"exit_plan\": {\n      \"plan_id\": \"%s\",\n      \"children\": [\n        {\"component\": \"tp_tiers\", \"handler\": \"tier_take_profit\", \"params\": {\"tiers\": [{\"target_price\": 95000, \"ratio\": 0.4}, {\"target_price\": 93000, \"ratio\": 0.3}, {\"target_price\": 91000, \"ratio\": 0.3}]}},\n        {\"component\": \"sl_single\", \"handler\": \"tier_stop_loss\", \"params\": {\"tiers\": [{\"target_price\": 97000, \"ratio\": 1.0}]}}\n      ]\n    }\n", planID)
	}
}

func validateExitPlan(decision *Decision, expectedPlanID string) error {
	if decision.Action != "open_long" && decision.Action != "open_short" {
		return nil
	}

	planID := normalizeExitPlanID(expectedPlanID)
	if planID == "" {
		return nil
	}

	if decision.ExitPlan == nil || len(decision.ExitPlan.Children) == 0 {
		if err := fillExitPlanFromLegacyFields(decision, planID); err != nil {
			return err
		}
	}

	if strings.TrimSpace(decision.ExitPlan.PlanID) == "" {
		decision.ExitPlan.PlanID = planID
	}

	if decision.ExitPlan.PlanID != planID {
		return fmt.Errorf("exit_plan.plan_id must be %s", planID)
	}

	requirement, ok := exitPlanRequirements[planID]
	if !ok {
		return fmt.Errorf("unsupported exit_plan plan_id: %s", planID)
	}

	return validateExitPlanChildren(decision, requirement)
}

func validateExitPlanChildren(decision *Decision, requirement exitPlanRequirement) error {
	children := decision.ExitPlan.Children
	if len(children) == 0 {
		return fmt.Errorf("exit_plan.children is required")
	}
	if len(children) != len(requirement.Components) {
		return fmt.Errorf("exit_plan.children must contain %d components", len(requirement.Components))
	}

	required := make(map[string]bool, len(requirement.Components))
	for _, component := range requirement.Components {
		required[component] = true
	}

	seen := make(map[string]bool, len(children))
	for _, child := range children {
		component := strings.TrimSpace(child.Component)
		handler := strings.TrimSpace(child.Handler)
		if component == "" || handler == "" {
			return fmt.Errorf("exit_plan child component/handler required")
		}
		if seen[component] {
			return fmt.Errorf("exit_plan child component duplicated: %s", component)
		}

		expectedHandler, ok := exitPlanComponentHandlers[component]
		if !ok {
			return fmt.Errorf("exit_plan component not supported: %s", component)
		}
		if handler != expectedHandler {
			return fmt.Errorf("exit_plan component %s must use handler %s", component, expectedHandler)
		}
		if !required[component] {
			return fmt.Errorf("exit_plan component %s not allowed for plan %s", component, requirement.PlanID)
		}

		if strings.HasSuffix(component, "_atr") {
			if err := validateAtrParams(child.Params, component); err != nil {
				return err
			}
		} else {
			if err := validateTierParams(child.Params, component, decision.Action); err != nil {
				return err
			}
		}

		seen[component] = true
	}

	for _, component := range requirement.Components {
		if !seen[component] {
			return fmt.Errorf("exit_plan missing component %s", component)
		}
	}

	return nil
}

func validateTierParams(raw json.RawMessage, component, action string) error {
	if len(raw) == 0 {
		return fmt.Errorf("exit_plan %s params required", component)
	}

	var params exitPlanTierParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return fmt.Errorf("exit_plan %s params invalid: %w", component, err)
	}

	if len(params.Tiers) == 0 || len(params.Tiers) > 3 {
		return fmt.Errorf("exit_plan %s tiers must be 1-3 items", component)
	}

	if strings.HasSuffix(component, "_single") && len(params.Tiers) != 1 {
		return fmt.Errorf("exit_plan %s must use a single tier", component)
	}

	isTakeProfit := strings.HasPrefix(component, "tp_")
	ascending := expectsAscending(action, isTakeProfit)
	var totalRatio float64
	var prev float64
	for i, tier := range params.Tiers {
		if tier.TargetPrice <= 0 {
			return fmt.Errorf("exit_plan %s tier target_price must be > 0", component)
		}
		if tier.Ratio <= 0 || tier.Ratio > 1 {
			return fmt.Errorf("exit_plan %s tier ratio must be within (0, 1]", component)
		}
		if i > 0 {
			if ascending && tier.TargetPrice <= prev {
				return fmt.Errorf("exit_plan %s target_price must be strictly increasing", component)
			}
			if !ascending && tier.TargetPrice >= prev {
				return fmt.Errorf("exit_plan %s target_price must be strictly decreasing", component)
			}
		}
		totalRatio += tier.Ratio
		prev = tier.TargetPrice
	}

	if math.Abs(totalRatio-1) > 0.001 {
		return fmt.Errorf("exit_plan %s tier ratios must sum to 1 (got %.4f)", component, totalRatio)
	}

	if strings.HasSuffix(component, "_single") && math.Abs(params.Tiers[0].Ratio-1) > 0.001 {
		return fmt.Errorf("exit_plan %s tier ratio must be 1", component)
	}

	return nil
}

func validateAtrParams(raw json.RawMessage, component string) error {
	if len(raw) == 0 {
		return fmt.Errorf("exit_plan %s params required", component)
	}

	var params exitPlanAtrParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return fmt.Errorf("exit_plan %s params invalid: %w", component, err)
	}

	expectedMode := exitPlanComponentModes[component]
	if params.Mode == "" || params.Mode != expectedMode {
		return fmt.Errorf("exit_plan %s mode must be %s", component, expectedMode)
	}

	if params.ATRValue < 0 {
		return fmt.Errorf("exit_plan %s atr_value must be >= 0", component)
	}
	if params.TriggerMultiplier < 1.0 || params.TriggerMultiplier > 5.0 {
		return fmt.Errorf("exit_plan %s trigger_multiplier must be between 1.0 and 5.0", component)
	}
	if params.TrailMultiplier < 0.5 {
		return fmt.Errorf("exit_plan %s trail_multiplier must be >= 0.5", component)
	}
	if params.TriggerMultiplier > 0 && params.TrailMultiplier >= params.TriggerMultiplier {
		return fmt.Errorf("exit_plan %s trail_multiplier must be < trigger_multiplier", component)
	}
	if params.InitialStopMultiplier != 0 && params.InitialStopMultiplier < 1.0 {
		return fmt.Errorf("exit_plan %s initial_stop_multiplier must be >= 1.0", component)
	}

	return nil
}

func expectsAscending(action string, isTakeProfit bool) bool {
	if action == "open_long" {
		return isTakeProfit
	}
	if action == "open_short" {
		return !isTakeProfit
	}
	return true
}
