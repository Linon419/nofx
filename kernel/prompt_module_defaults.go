package kernel

import (
	"fmt"
	"nofx/store"
	"strings"
)

type PromptModuleDefaults struct {
	SchemaPrompt              string `json:"schema_prompt"`
	SchemaPromptLite          string `json:"schema_prompt_lite"`
	ModeVariantAggressive     string `json:"mode_variant_aggressive"`
	ModeVariantConservative   string `json:"mode_variant_conservative"`
	ModeVariantScalping       string `json:"mode_variant_scalping"`
	HardConstraints           string `json:"hard_constraints"`
	OutputFormat              string `json:"output_format"`
	AnalysisCorePrompt        string `json:"analysis_core_prompt"`
	VisionSystemPrompt        string `json:"vision_system_prompt"`
	VisionSystemPromptVerbose string `json:"vision_system_prompt_verbose"`
	VisionSystemPromptConcise string `json:"vision_system_prompt_concise"`
}

func defaultModeVariantAggressive() string {
	return strings.TrimSpace(`## Mode: Aggressive
- Prioritize capturing trend breakouts, can build positions in batches when confidence >=70
- Allow higher positions, but must strictly set stop-loss and explain risk-reward ratio`)
}

func defaultModeVariantConservative() string {
	return strings.TrimSpace(`## Mode: Conservative
- Only open positions when multiple signals resonate
- Prioritize cash preservation, must pause for multiple periods after consecutive losses`)
}

func defaultModeVariantScalping() string {
	return strings.TrimSpace(`## Mode: Scalping
- Focus on short-term momentum, smaller profit targets but require quick action
- If price doesn't move as expected within two bars, immediately reduce position or stop-loss`)
}

func defaultAnalysisCorePrompt(lang Language) string {
	if lang == LangChinese {
		return strings.TrimSpace(`
你是一个严格、中立、以数据为基础的交易“分析层”助手。你的工作是把输入的市场数据与持仓信息整理成供后续“决策层”使用的客观事实要点。

规则：
- 仅输出要点列表（最多 10 条），每条尽量 1 句；优先包含明确数值/指标/时间周期。
- 只允许引用输入中明确给出的事实与数值；禁止编造、推断、补全缺失数据。
- 禁止任何方向判断/预测/建议：不要使用看多/看空/上涨/下跌/趋势/大概率等结论性措辞。
- 允许指出不确定性、冲突信号、关键风险与需要满足的条件，但禁止给出任何交易动作/指令（open/close/hold/wait 等）。
- 禁止输出任何 JSON（尤其是决策数组），也不要输出 <decision> / <reasoning> 等标签。
- 纯文本，不要代码块。`)
	}

	return strings.TrimSpace(`
You are a strict, neutral, data-grounded analysis-layer assistant for trading. Your job is to summarize the given market/position inputs into objective notes that the later decision layer can use.

Rules:
- Output bullet notes only (max 10 items), ideally 1 sentence each. Prefer explicit numbers/metrics/timeframes when available.
- Only cite facts and numbers explicitly present in the input; never invent, infer, or fill in missing data.
- No directional judgement, predictions, or recommendations. Do NOT use words like bullish/bearish, uptrend/downtrend, likely to rise/fall, etc.
- You may highlight uncertainty, conflicting signals, key risks, and conditions to be met, but you must NOT output any trading actions or instructions (open/close/hold/wait, etc.).
- Do NOT output any JSON (especially no decision arrays), and do NOT use tags like <decision>/<reasoning>.
- Plain text only. Do not use code blocks.`)
}

func defaultVisionSystemPromptConcise() string {
	return strings.TrimSpace(`
You are a strict chart-reading assistant.

Rules:
- Only describe what is directly visible in the images.
- No predictions or recommendations; do NOT output any trading actions.
- If unclear/not visible, say so.
- Plain text only (no code blocks). Keep it concise.

中文（同样只做读图要点，不做交易结论）：
- 只描述图上可见信息，不做预测/建议/开平仓指令。
- 不清楚就说不清楚。
- 纯文本，尽量简短。`)
}

func defaultHardConstraintsBlock(
	lang Language,
	accountEquity float64,
	riskControl store.RiskControlConfig,
	btcEthPosValueRatio float64,
	altcoinPosValueRatio float64,
	minPositionSize float64,
	btcEthMinPositionSize float64,
	enforceMinPositionSize bool,
) string {
	var sb strings.Builder

	sb.WriteString("# Hard Constraints (Risk Control)\n\n")
	sb.WriteString("## CODE ENFORCED (Backend validation, cannot be bypassed):\n")
	sb.WriteString(fmt.Sprintf("- Max Positions: %d coins simultaneously\n", riskControl.MaxPositions))
	sb.WriteString(fmt.Sprintf("- Position Value Limit (Altcoins): max %.0f USDT (= equity %.0f * %.1fx)\n",
		accountEquity*altcoinPosValueRatio, accountEquity, altcoinPosValueRatio))
	sb.WriteString(fmt.Sprintf("- Position Value Limit (BTC/ETH): max %.0f USDT (= equity %.0f * %.1fx)\n",
		accountEquity*btcEthPosValueRatio, accountEquity, btcEthPosValueRatio))
	sb.WriteString(fmt.Sprintf("- Max Margin Usage: <=%.0f%%\n", riskControl.MaxMarginUsage*100))
	if enforceMinPositionSize {
		sb.WriteString(fmt.Sprintf("- Min Position Size (Altcoins): >=%.0f USDT\n", minPositionSize))
		sb.WriteString(fmt.Sprintf("- Min Position Size (BTC/ETH): >=%.0f USDT\n\n", btcEthMinPositionSize))
	} else {
		sb.WriteString("- Min Position Size: disabled (exchange may reject small orders)\n\n")
	}

	sb.WriteString("## AI GUIDED (Recommended, you should follow):\n")
	sb.WriteString(fmt.Sprintf("- Trading Leverage: Altcoins max %dx | BTC/ETH max %dx\n",
		riskControl.AltcoinMaxLeverage, riskControl.BTCETHMaxLeverage))
	sb.WriteString(fmt.Sprintf("- Risk-Reward Ratio: >=1:%.1f (take_profit / stop_loss)\n", riskControl.MinRiskRewardRatio))
	if lang == LangChinese {
		sb.WriteString("- RR 不足时：直接输出 wait；不要为了满足 RR 去移动止损/止盈（SL/TP）。优先保持结构失效位的止损与合理目标位。\n")
	} else {
		sb.WriteString("- If RR is insufficient: output wait; do NOT move SL/TP just to satisfy the RR constraint. Keep SL at the structural invalidation level and TP at a realistic target.\n")
	}
	sb.WriteString(fmt.Sprintf("- Min Confidence: >=%d to open position\n\n", riskControl.MinConfidence))

	return strings.TrimSpace(sb.String())
}

func defaultOutputFormatBlock(
	accountEquity float64,
	riskControl store.RiskControlConfig,
	btcEthPosValueRatio float64,
	exitPlanExample string,
) string {
	var sb strings.Builder

	sb.WriteString("# Output Format (Strictly Follow)\n\n")
	sb.WriteString("**Must use XML tags <reasoning> and <decision> to separate brief public rationale and decision JSON, avoiding parsing errors**\n\n")
	sb.WriteString("## Format Requirements\n\n")
	sb.WriteString("<reasoning>\n")
	sb.WriteString("Requirements:\n")
	sb.WriteString("- Write brief public rationale notes grouped by symbol (recommended: 2-6 bullets per symbol, total <= 40 lines).\n")
	sb.WriteString("- Do NOT output chain-of-thought.\n")
	sb.WriteString("</reasoning>\n\n")
	sb.WriteString("<decision>\n")
	sb.WriteString("Requirements:\n")
	sb.WriteString("- Only output a pure JSON array inside <decision> ... </decision> (no extra text).\n")
	sb.WriteString("- Do NOT use code fences (no ```).\n\n")
	sb.WriteString("[\n")
	examplePositionSize := accountEquity * btcEthPosValueRatio
	sb.WriteString("  {\n")
	sb.WriteString("    \"symbol\": \"BTCUSDT\",\n")
	sb.WriteString("    \"action\": \"open_short\",\n")
	sb.WriteString(fmt.Sprintf("    \"leverage\": %d,\n", riskControl.BTCETHMaxLeverage))
	sb.WriteString(fmt.Sprintf("    \"position_size_usd\": %.0f,\n", examplePositionSize))
	sb.WriteString("    \"stop_loss\": 97000,\n")
	sb.WriteString("    \"take_profit\": 91000,\n")
	sb.WriteString("    \"confidence\": 85,\n")
	sb.WriteString("    \"risk_usd\": 300,\n")
	sb.WriteString("    \"reasoning\": \"One sentence summary of why this action is taken\"")
	if exitPlanExample != "" {
		sb.WriteString(",\n")
		sb.WriteString(exitPlanExample)
	} else {
		sb.WriteString("\n")
	}
	sb.WriteString("  },\n")
	sb.WriteString("  {\"symbol\": \"ETHUSDT\", \"action\": \"close_long\"}\n")
	sb.WriteString("]\n")
	sb.WriteString("</decision>\n\n")
	sb.WriteString("## Field Description\n\n")
	sb.WriteString("- `action`: open_long | open_short | close_long | close_short | hold | wait\n")
	sb.WriteString(fmt.Sprintf("- `confidence`: 0-100 (opening recommended >=%d)\n", riskControl.MinConfidence))
	sb.WriteString("- Required when opening: leverage, position_size_usd, stop_loss, take_profit, confidence, risk_usd\n")
	sb.WriteString("- `exit_plan`: required when opening if exit plan is configured; must match plan_id and component rules\n")
	sb.WriteString("- **IMPORTANT**: All numeric values must be calculated numbers, NOT formulas/expressions (e.g., use `27.76` not `3000 * 0.01`)\n\n")
	sb.WriteString("## JSON Strictness (Must Follow)\n\n")
	sb.WriteString("- No range/approx symbols in JSON: do not use `~` or `～` anywhere (e.g., use `88336`, not `~88336` or `88000~89000`)\n")
	sb.WriteString("- No thousand separators in JSON numbers (e.g., `98000`, not `98,000`)\n")
	sb.WriteString("- No comments in JSON (no `//` or `/* */`)\n\n")
	sb.WriteString("- No trailing commas in JSON\n\n")

	return strings.TrimSpace(sb.String())
}

func (e *StrategyEngine) BuildPromptModuleDefaults(accountEquity float64, variant string) PromptModuleDefaults {
	lang := detectLanguage(e.config.PromptSections.RoleDefinition)
	rc := e.config.RiskControl
	toggles := e.config.PromptToggles
	exitPlanID := normalizeExitPlanID(e.config.PromptSections.ExitStrategyPlan)
	exitPlanExample := buildExitPlanExample(exitPlanID)

	btcEthPosValueRatio := rc.BTCETHMaxPositionValueRatio
	if btcEthPosValueRatio <= 0 {
		btcEthPosValueRatio = 5.0
	}
	altcoinPosValueRatio := rc.AltcoinMaxPositionValueRatio
	if altcoinPosValueRatio <= 0 {
		altcoinPosValueRatio = 1.0
	}

	minPositionSize := rc.MinPositionSize
	if minPositionSize <= 0 {
		minPositionSize = 12
	}
	btcEthMinPositionSize := minPositionSize
	if btcEthMinPositionSize < 60 {
		btcEthMinPositionSize = 60
	}
	enforceMinPositionSize := true
	if rc.EnforceMinPositionSize != nil {
		enforceMinPositionSize = *rc.EnforceMinPositionSize
	}

	visionVerbose := e.buildVisionSystemPromptBilingual()
	visionConcise := defaultVisionSystemPromptConcise()
	visionEffective := visionVerbose
	if !boolOrDefault(toggles.UseVerboseVisionSystemPrompt, true) {
		visionEffective = visionConcise
	}

	return PromptModuleDefaults{
		SchemaPrompt:              strings.TrimSpace(GetSchemaPrompt(lang)),
		SchemaPromptLite:          strings.TrimSpace(GetSchemaPromptLite(lang)),
		ModeVariantAggressive:     defaultModeVariantAggressive(),
		ModeVariantConservative:   defaultModeVariantConservative(),
		ModeVariantScalping:       defaultModeVariantScalping(),
		HardConstraints:           defaultHardConstraintsBlock(lang, accountEquity, rc, btcEthPosValueRatio, altcoinPosValueRatio, minPositionSize, btcEthMinPositionSize, enforceMinPositionSize),
		OutputFormat:              defaultOutputFormatBlock(accountEquity, rc, btcEthPosValueRatio, exitPlanExample),
		AnalysisCorePrompt:        defaultAnalysisCorePrompt(lang),
		VisionSystemPrompt:        strings.TrimSpace(visionEffective),
		VisionSystemPromptVerbose: strings.TrimSpace(visionVerbose),
		VisionSystemPromptConcise: strings.TrimSpace(visionConcise),
	}
}
