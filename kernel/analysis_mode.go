package kernel

import "strings"

func (e *StrategyEngine) buildDecisionAnalysisSystemPrompt() string {
	toggles := e.config.PromptToggles
	if !boolOrDefault(toggles.EnableAnalysisStage, true) {
		return ""
	}

	lang := detectLanguage(e.config.PromptSections.RoleDefinition)
	includeSchema := boolOrDefault(toggles.IncludeSchemaPrompt, true)

	var suffix string
	if lang == LangChinese {
		suffix = `

---

你是一个严格、中立、以数据为基础的交易“分析层”助手。你的工作是把输入的市场数据与持仓信息整理成供后续“决策层”使用的客观事实要点。

规则：
- 仅输出要点列表（最多 10 条），每条尽量 1 句；优先包含明确数值/指标/时间周期。
- 只允许引用输入中明确给出的事实与数值；禁止编造、推断、补全缺失数据。
- 禁止任何方向判断/预测/建议：不要使用看多/看空/上涨/下跌/趋势/大概率等结论性措辞。
- 允许指出不确定性、冲突信号、关键风险与需要满足的条件，但禁止给出任何交易动作/指令（open/close/hold/wait 等）。
- 禁止输出任何 JSON（尤其是决策数组），也不要输出 <decision> / <reasoning> 等标签。
- 纯文本，不要代码块。`
	} else {
		suffix = `

---

You are a strict, neutral, data-grounded analysis-layer assistant for trading. Your job is to summarize the given market/position inputs into objective notes that the later decision layer can use.

Rules:
- Output bullet notes only (max 10 items), ideally 1 sentence each. Prefer explicit numbers/metrics/timeframes when available.
- Only cite facts and numbers explicitly present in the input; never invent, infer, or fill in missing data.
- No directional judgement, predictions, or recommendations. Do NOT use words like bullish/bearish, uptrend/downtrend, likely to rise/fall, etc.
- You may highlight uncertainty, conflicting signals, key risks, and conditions to be met, but you must NOT output any trading actions or instructions (open/close/hold/wait, etc.).
- Do NOT output any JSON (especially no decision arrays), and do NOT use tags like <decision>/<reasoning>.
- Plain text only. Do not use code blocks.`
	}

	if includeSchema {
		return strings.TrimSpace(GetSchemaPromptLite(lang) + suffix)
	}
	return strings.TrimSpace(strings.TrimPrefix(suffix, "\n\n---\n\n"))
}

func appendDecisionAnalysisNotes(userPrompt, notes string) string {
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return userPrompt
	}

	title := "## AI Analysis Notes"
	if detectLanguage(notes) == LangChinese {
		title = "## AI 分析要点"
	}

	var sb strings.Builder
	sb.WriteString(strings.TrimRight(userPrompt, "\n"))
	sb.WriteString("\n\n")
	sb.WriteString(title)
	sb.WriteString("\n")
	sb.WriteString(notes)
	sb.WriteString("\n")
	return sb.String()
}
