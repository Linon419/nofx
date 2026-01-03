package trader

import (
	"fmt"
	"nofx/analysis"
	"nofx/decision"
	"nofx/market"
	"strings"
)

// allowAIOpen blocks AI-initiated opens when the decision reasoning references
// data that is not present in the current prompt cycle (anti-hallucination).
//
// This is deliberately conservative: if the model claims evidence that isn't provided,
// we prefer skipping the trade rather than executing a potentially fabricated thesis.
func (at *AutoTrader) allowAIOpen(ctx *decision.Context, d *decision.Decision, claimText string) (bool, string) {
	if at == nil || d == nil {
		return true, "guard_bypass_nil"
	}
	if ctx == nil {
		return true, "guard_bypass_missing_context"
	}

	action := strings.ToLower(strings.TrimSpace(d.Action))
	if action != "open_long" && action != "open_short" {
		return true, "not_open_action"
	}

	symbol := market.Normalize(d.Symbol)
	if strings.TrimSpace(claimText) == "" {
		claimText = d.Reasoning
	}
	r := strings.ToLower(strings.TrimSpace(claimText))
	if symbol == "" || r == "" {
		return true, "no_reasoning_or_symbol"
	}

	if mentionsFundFlowOrOIByTimeframe(r) && !hasQuantDataForSymbol(ctx, symbol) {
		return false, "claims_quant_data_but_missing"
	}

	if ok, why := validatePatternClaims(ctx, symbol, r); !ok {
		return false, why
	}
	if ok, why := validateSqueezeClaims(ctx, symbol, r); !ok {
		return false, why
	}
	if ok, why := validateDivergenceClaims(ctx, symbol, r); !ok {
		return false, why
	}

	return true, "ok"
}

func hasQuantDataForSymbol(ctx *decision.Context, symbol string) bool {
	if ctx == nil || ctx.QuantDataMap == nil {
		return false
	}
	_, ok := ctx.QuantDataMap[symbol]
	if ok {
		return true
	}
	_, ok = ctx.QuantDataMap[strings.ToUpper(symbol)]
	return ok
}

func mentionsFundFlowOrOIByTimeframe(r string) bool {
	if r == "" {
		return false
	}
	hasOI := strings.Contains(r, "open interest") ||
		strings.Contains(r, " oi ") || strings.Contains(r, "oi:") || strings.Contains(r, "oi=") || strings.Contains(r, "oi/") ||
		strings.Contains(r, "持仓量")
	hasFlow := strings.Contains(r, "netflow") || strings.Contains(r, "fund flow") || strings.Contains(r, "资金流") || strings.Contains(r, "净流") || strings.Contains(r, "流入") || strings.Contains(r, "流出") || strings.Contains(r, "机构")
	if !hasOI && !hasFlow {
		return false
	}
	for _, tf := range []string{"5m", "15m", "1h", "4h", "12h", "24h"} {
		if strings.Contains(r, tf) {
			return true
		}
	}
	return false
}

func validatePatternClaims(ctx *decision.Context, symbol string, r string) (bool, string) {
	if !(strings.Contains(r, "double_top") || strings.Contains(r, "double_bottom") || strings.Contains(r, "双顶") || strings.Contains(r, "双底")) {
		return true, "no_pattern_claim"
	}

	tf := extractReasoningTimeframe(r)
	if tf == "" {
		tf = "15m"
	}

	ar, ok := computeAnalysisForSymbolTF(ctx, symbol, tf)
	if !ok || ar == nil || ar.Pattern == nil {
		return false, "claims_pattern_but_analysis_missing"
	}

	claimed := "unknown"
	switch {
	case strings.Contains(r, "double_top") || strings.Contains(r, "双顶"):
		claimed = "double_top"
	case strings.Contains(r, "double_bottom") || strings.Contains(r, "双底"):
		claimed = "double_bottom"
	}
	if claimed == "unknown" {
		return true, "pattern_claim_unchecked"
	}

	detected := strings.ToLower(strings.TrimSpace(string(ar.Pattern.Detected)))
	if detected == "" {
		detected = "none"
	}
	if detected != claimed {
		return false, fmt.Sprintf("claims_%s_but_detected_%s_%s", claimed, detected, tf)
	}
	return true, "pattern_ok"
}

func validateSqueezeClaims(ctx *decision.Context, symbol string, r string) (bool, string) {
	if !(strings.Contains(r, "squeeze") || strings.Contains(r, "挤压") || strings.Contains(r, "warning_l1") || strings.Contains(r, "warning_l2")) {
		return true, "no_squeeze_claim"
	}

	tf := extractReasoningTimeframe(r)
	if tf == "" {
		tf = "15m"
	}

	ar, ok := computeAnalysisForSymbolTF(ctx, symbol, tf)
	if !ok || ar == nil || ar.VolatilityWarning == nil {
		return false, "claims_squeeze_but_analysis_missing"
	}
	if !(ar.VolatilityWarning.IsSqueeze || ar.VolatilityWarning.WarningL1 || ar.VolatilityWarning.WarningL2) {
		return false, fmt.Sprintf("claims_squeeze_but_no_warning_%s", tf)
	}
	return true, "squeeze_ok"
}

func validateDivergenceClaims(ctx *decision.Context, symbol string, r string) (bool, string) {
	if !(strings.Contains(r, "divergence") || strings.Contains(r, "背离")) {
		return true, "no_divergence_claim"
	}

	tf := extractReasoningTimeframe(r)
	if tf == "" {
		tf = "15m"
	}

	ar, ok := computeAnalysisForSymbolTF(ctx, symbol, tf)
	if !ok || ar == nil || ar.Divergence == nil {
		return false, "claims_divergence_but_analysis_missing"
	}
	if len(ar.Divergence.Events) == 0 {
		return false, fmt.Sprintf("claims_divergence_but_none_%s", tf)
	}
	return true, "divergence_ok"
}

func extractReasoningTimeframe(r string) string {
	for _, tf := range []string{"1w", "3d", "1d", "12h", "8h", "6h", "4h", "2h", "1h", "30m", "15m", "5m", "3m", "1m"} {
		if strings.Contains(r, tf) {
			return tf
		}
	}
	if strings.Contains(r, "15min") || strings.Contains(r, "15分钟") {
		return "15m"
	}
	if strings.Contains(r, "1hour") || strings.Contains(r, "1小时") {
		return "1h"
	}
	if strings.Contains(r, "4hour") || strings.Contains(r, "4小时") {
		return "4h"
	}
	return ""
}

func computeAnalysisForSymbolTF(ctx *decision.Context, symbol, tf string) (*analysis.AnalysisResult, bool) {
	if ctx == nil || ctx.MarketDataMap == nil {
		return nil, false
	}
	md, ok := ctx.MarketDataMap[symbol]
	if !ok {
		md, ok = ctx.MarketDataMap[strings.ToUpper(symbol)]
	}
	if !ok || md == nil || md.TimeframeData == nil {
		return nil, false
	}
	tfData := md.TimeframeData[tf]
	if tfData == nil || len(tfData.Klines) == 0 {
		return nil, false
	}
	klines := guardConvertKlineBars(tfData.Klines)
	return analysis.AnalyzeWithDefaults(klines), true
}
