package kernel

import (
	"strings"
	"testing"

	"nofx/store"
)

func TestStrategyEngineBuildSystemPrompt_RRInsufficientWaitNoAdjust(t *testing.T) {
	t.Run("English", func(t *testing.T) {
		cfg := store.GetDefaultStrategyConfig("en")
		cfg.RiskControl.MinRiskRewardRatio = 3.0
		engine := NewStrategyEngine(&cfg)
		prompt := engine.BuildSystemPrompt(1000, "")
		if !strings.Contains(prompt, "If RR is insufficient: output wait") {
			t.Fatalf("missing RR wait instruction in prompt")
		}
		if !strings.Contains(prompt, "do NOT move SL/TP") {
			t.Fatalf("missing SL/TP no-adjust instruction in prompt")
		}
	})

	t.Run("Chinese", func(t *testing.T) {
		cfg := store.GetDefaultStrategyConfig("zh")
		cfg.RiskControl.MinRiskRewardRatio = 3.0
		engine := NewStrategyEngine(&cfg)
		prompt := engine.BuildSystemPrompt(1000, "")
		if !strings.Contains(prompt, "RR 不足时：直接输出 wait") {
			t.Fatalf("missing RR wait instruction in prompt")
		}
		if !strings.Contains(prompt, "不要为了满足 RR 去移动止损/止盈") {
			t.Fatalf("missing SL/TP no-adjust instruction in prompt")
		}
	})
}
