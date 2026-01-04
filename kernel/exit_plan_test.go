package kernel

import (
	"encoding/json"
	"testing"
)

func TestValidateExitPlan(t *testing.T) {
	makeTierParams := func(t *testing.T, tiers []exitPlanTier) json.RawMessage {
		t.Helper()
		data, err := json.Marshal(exitPlanTierParams{Tiers: tiers})
		if err != nil {
			t.Fatalf("marshal tiers: %v", err)
		}
		return data
	}

	makeAtrParams := func(t *testing.T, params exitPlanAtrParams) json.RawMessage {
		t.Helper()
		data, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("marshal atr params: %v", err)
		}
		return data
	}

	tests := []struct {
		name       string
		action     string
		planID     string
		exitPlan   *ExitPlan
		stopLoss   float64
		takeProfit float64
		wantError  bool
	}{
		{
			name:   "valid tp_tiers + sl_single (long)",
			action: "open_long",
			planID: "plan_tp_tiers_sl_single",
			exitPlan: &ExitPlan{
				PlanID: "plan_tp_tiers_sl_single",
				Children: []ExitPlanChild{
					{
						Component: "tp_tiers",
						Handler:   "tier_take_profit",
						Params: makeTierParams(t, []exitPlanTier{
							{TargetPrice: 110, Ratio: 0.4},
							{TargetPrice: 120, Ratio: 0.3},
							{TargetPrice: 130, Ratio: 0.3},
						}),
					},
					{
						Component: "sl_single",
						Handler:   "tier_stop_loss",
						Params: makeTierParams(t, []exitPlanTier{
							{TargetPrice: 90, Ratio: 1.0},
						}),
					},
				},
			},
			stopLoss:   90,
			takeProfit: 130,
			wantError:  false,
		},
		{
			name:   "invalid tier ratio sum",
			action: "open_long",
			planID: "plan_tp_tiers_sl_single",
			exitPlan: &ExitPlan{
				PlanID: "plan_tp_tiers_sl_single",
				Children: []ExitPlanChild{
					{
						Component: "tp_tiers",
						Handler:   "tier_take_profit",
						Params: makeTierParams(t, []exitPlanTier{
							{TargetPrice: 110, Ratio: 0.5},
							{TargetPrice: 120, Ratio: 0.4},
						}),
					},
					{
						Component: "sl_single",
						Handler:   "tier_stop_loss",
						Params: makeTierParams(t, []exitPlanTier{
							{TargetPrice: 90, Ratio: 1.0},
						}),
					},
				},
			},
			stopLoss:   90,
			takeProfit: 120,
			wantError:  true,
		},
		{
			name:   "invalid tp order for short",
			action: "open_short",
			planID: "plan_tp_tiers_sl_single",
			exitPlan: &ExitPlan{
				PlanID: "plan_tp_tiers_sl_single",
				Children: []ExitPlanChild{
					{
						Component: "tp_tiers",
						Handler:   "tier_take_profit",
						Params: makeTierParams(t, []exitPlanTier{
							{TargetPrice: 110, Ratio: 0.4},
							{TargetPrice: 120, Ratio: 0.3},
							{TargetPrice: 130, Ratio: 0.3},
						}),
					},
					{
						Component: "sl_single",
						Handler:   "tier_stop_loss",
						Params: makeTierParams(t, []exitPlanTier{
							{TargetPrice: 150, Ratio: 1.0},
						}),
					},
				},
			},
			stopLoss:   150,
			takeProfit: 130,
			wantError:  true,
		},
		{
			name:   "missing component",
			action: "open_long",
			planID: "plan_tp_tiers_sl_single",
			exitPlan: &ExitPlan{
				PlanID: "plan_tp_tiers_sl_single",
				Children: []ExitPlanChild{
					{
						Component: "tp_tiers",
						Handler:   "tier_take_profit",
						Params: makeTierParams(t, []exitPlanTier{
							{TargetPrice: 110, Ratio: 1.0},
						}),
					},
				},
			},
			stopLoss:   90,
			takeProfit: 110,
			wantError:  true,
		},
		{
			name:   "invalid atr mode",
			action: "open_long",
			planID: "plan_sl_atr_tp_single",
			exitPlan: &ExitPlan{
				PlanID: "plan_sl_atr_tp_single",
				Children: []ExitPlanChild{
					{
						Component: "sl_atr",
						Handler:   "atr_trailing",
						Params: makeAtrParams(t, exitPlanAtrParams{
							Mode:              "take_profit",
							ATRValue:          10,
							TriggerMultiplier: 2,
							TrailMultiplier:   1,
						}),
					},
					{
						Component: "tp_single",
						Handler:   "tier_take_profit",
						Params: makeTierParams(t, []exitPlanTier{
							{TargetPrice: 110, Ratio: 1.0},
						}),
					},
				},
			},
			stopLoss:   90,
			takeProfit: 110,
			wantError:  true,
		},
		{
			name:       "autofill exit_plan when missing (tp_tiers + sl_single)",
			action:     "open_long",
			planID:     "plan_tp_tiers_sl_single",
			exitPlan:   nil,
			stopLoss:   90,
			takeProfit: 110,
			wantError:  false,
		},
		{
			name:       "autofill exit_plan when missing (sl_atr + tp_single)",
			action:     "open_long",
			planID:     "plan_sl_atr_tp_single",
			exitPlan:   nil,
			stopLoss:   90,
			takeProfit: 110,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec := Decision{
				Action:     tt.action,
				ExitPlan:   tt.exitPlan,
				StopLoss:   tt.stopLoss,
				TakeProfit: tt.takeProfit,
			}
			err := validateExitPlan(&dec, tt.planID)
			if (err != nil) != tt.wantError {
				t.Fatalf("validateExitPlan() error = %v, wantError %v", err, tt.wantError)
			}
			if !tt.wantError && tt.exitPlan == nil && (tt.action == "open_long" || tt.action == "open_short") {
				if dec.ExitPlan == nil || dec.ExitPlan.PlanID == "" || len(dec.ExitPlan.Children) == 0 {
					t.Fatalf("expected exit_plan to be auto-filled, got: %+v", dec.ExitPlan)
				}
			}
		})
	}
}
