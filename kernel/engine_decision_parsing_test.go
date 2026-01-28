package kernel

import (
	"encoding/json"
	"testing"
)

func TestExtractDecisions_NestedArraysInExitPlan(t *testing.T) {
	// 真实场景：<decision> 内的决策 JSON 包含 exit_plan.children[].params.tiers（嵌套数组）。
	// 旧的正则提取容易误截断，导致进入 safe wait。
	raw := `<reasoning>
- HYPEUSDT: 1小时级别处于强势多头排列（EMA 21 > 55 > 100 > 200），价格运行在所有均线上方，趋势特征最明显。
- 15分钟级别：价格在触及 EMA55（约 31.59）完成回踩后强力反弹，目前已站回 EMA21（33.08）上方，回踩确认结束。
- 5分钟级别：形成明显的更高低点（Higher Low）结构，价格在 33.40-33.60 区间稳固，具备入场条件。
- 风险管理：止损设在 32.80，位于 5分钟级别近期波动低点及 15分钟 EMA21 下方；目标位 35.50，盈亏比大于 2.0。
- 其他币种：SOMI 涨幅过大，偏离均线过远，追高风险极高；FOGO 处于 15分钟级别回调中，尚未止跌；PHA 1小时级别均线呈空头排列，属于弱势反弹。
</reasoning>

<decision>
[
  {
    "symbol": "HYPEUSDT",
    "action": "open_long",
    "leverage": 5,
    "position_size_usd": 6.4,
    "stop_loss": 32.8,
    "take_profit": 35.5,
    "confidence": 75,
    "risk_usd": 0.17,
    "reasoning": "1H strong bullish trend with 15M pullback to EMA55 completed and 5M structure showing higher lows.",
    "exit_plan": {
      "plan_id": "plan_tp_tiers_sl_single",
      "children": [
        {
          "component": "tp_tiers",
          "handler": "tier_take_profit",
          "params": {
            "tiers": [
              {"target_price": 34.5, "ratio": 0.4},
              {"target_price": 35.5, "ratio": 0.6}
            ]
          }
        },
        {
          "component": "sl_single",
          "handler": "tier_stop_loss",
          "params": {
            "tiers": [
              {"target_price": 32.8, "ratio": 1.0}
            ]
          }
        }
      ]
    }
  }
]
</decision>`

	decisions, err := extractDecisions(raw)
	if err != nil {
		t.Fatalf("extractDecisions failed: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}
	d := decisions[0]
	if d.Symbol != "HYPEUSDT" || d.Action != "open_long" {
		t.Fatalf("unexpected decision: symbol=%q action=%q", d.Symbol, d.Action)
	}
	if d.ExitPlan == nil || len(d.ExitPlan.Children) != 2 {
		t.Fatalf("expected exit_plan with 2 children, got %+v", d.ExitPlan)
	}

	// 确认 params 保持为可解析的 JSON（而不是被截断/错误提取的数组片段）。
	var tpParams map[string]any
	if err := json.Unmarshal(d.ExitPlan.Children[0].Params, &tpParams); err != nil {
		t.Fatalf("failed to unmarshal tp params: %v (raw=%s)", err, string(d.ExitPlan.Children[0].Params))
	}
}

