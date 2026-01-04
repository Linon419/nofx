package trader

import (
	"fmt"
	"nofx/analysis"
	decision "nofx/kernel"
	"nofx/market"
	"sort"
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

	if ok, why := validatePatternClaimsV2(ctx, symbol, r); !ok {
		return false, why
	}
	if ok, why := validateSqueezeClaimsV2(ctx, symbol, r); !ok {
		return false, why
	}
	if ok, why := validateDivergenceClaimsV2(ctx, symbol, r); !ok {
		return false, why
	}

	return true, "ok"
}

// extractSymbolSectionFromCoT attempts to extract the section for a single symbol from a full-chain CoT blob.
// This prevents cross-symbol claims (e.g. "ZEC 15m squeeze") from blocking a different symbol's order.
func extractSymbolSectionFromCoT(cotTrace string, symbol string) string {
	cot := strings.TrimSpace(cotTrace)
	if cot == "" {
		return ""
	}
	sym := market.Normalize(symbol)
	if sym == "" {
		return ""
	}
	symUpper := strings.ToUpper(sym)

	lines := strings.Split(cot, "\n")
	start := -1
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if !(strings.HasPrefix(trim, "###") || strings.HasPrefix(trim, "##")) {
			continue
		}
		if strings.Contains(strings.ToUpper(trim), symUpper) {
			start = i
			break
		}
	}
	if start == -1 {
		return ""
	}

	end := len(lines)
	for j := start + 1; j < len(lines); j++ {
		trim := strings.TrimSpace(lines[j])
		if !(strings.HasPrefix(trim, "###") || strings.HasPrefix(trim, "##")) {
			continue
		}
		upper := strings.ToUpper(trim)
		isSymbolHeader := strings.Contains(upper, "USDT") || strings.Contains(strings.ToLower(trim), "xyz:")
		if isSymbolHeader && !strings.Contains(upper, symUpper) {
			end = j
			break
		}
	}

	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
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

func validatePatternClaimsV2(ctx *decision.Context, symbol string, r string) (bool, string) {
	if !hasAssertedKeyword(r, []string{
		"double_top",
		"double_bottom",
		"\u53cc\u9876",
		"\u53cc\u5e95",
	}) {
		return true, "no_pattern_claim"
	}

	claimed := "unknown"
	switch {
	case hasAssertedKeyword(r, []string{"double_top", "\u53cc\u9876"}):
		claimed = "double_top"
	case hasAssertedKeyword(r, []string{"double_bottom", "\u53cc\u5e95"}):
		claimed = "double_bottom"
	}
	if claimed == "unknown" {
		return true, "pattern_claim_unchecked"
	}

	keywords := []string{"double_top", "\u53cc\u9876"}
	if claimed == "double_bottom" {
		keywords = []string{"double_bottom", "\u53cc\u5e95"}
	}
	explicitTF := extractTimeframeNearAssertedKeyword(r, keywords)
	tfs := timeframesForClaimCheckV2(ctx, symbol, explicitTF)
	if len(tfs) == 0 {
		return false, "claims_pattern_but_analysis_missing"
	}

	anyAnalysis := false
	firstTF := ""
	firstDetected := "none"
	for _, tf := range tfs {
		ar, ok := computeAnalysisForSymbolTF(ctx, symbol, tf)
		if !ok || ar == nil || ar.Pattern == nil {
			continue
		}
		anyAnalysis = true
		detected := strings.ToLower(strings.TrimSpace(string(ar.Pattern.Detected)))
		if detected == "" {
			detected = "none"
		}
		if firstTF == "" {
			firstTF = tf
			firstDetected = detected
		}
		if detected == claimed {
			return true, "pattern_ok"
		}
	}
	if !anyAnalysis {
		return false, "claims_pattern_but_analysis_missing"
	}
	return false, fmt.Sprintf("claims_%s_but_detected_%s_%s", claimed, firstDetected, firstTF)
}

func validateSqueezeClaimsV2(ctx *decision.Context, symbol string, r string) (bool, string) {
	if !hasAssertedKeyword(r, []string{
		"squeeze",
		"\u6324\u538b",
		"warning_l1",
		"warning_l2",
	}) {
		return true, "no_squeeze_claim"
	}

	explicitTF := extractTimeframeNearAssertedKeyword(r, []string{
		"squeeze",
		"\u6324\u538b",
		"warning_l1",
		"warning_l2",
	})
	tfs := timeframesForClaimCheckV2(ctx, symbol, explicitTF)
	if len(tfs) == 0 {
		return false, "claims_squeeze_but_analysis_missing"
	}

	anyAnalysis := false
	firstTF := ""
	for _, tf := range tfs {
		ar, ok := computeAnalysisForSymbolTF(ctx, symbol, tf)
		if !ok || ar == nil || ar.VolatilityWarning == nil {
			continue
		}
		anyAnalysis = true
		if firstTF == "" {
			firstTF = tf
		}
		if ar.VolatilityWarning.IsSqueeze || ar.VolatilityWarning.WarningL1 || ar.VolatilityWarning.WarningL2 {
			return true, "squeeze_ok"
		}
	}
	if !anyAnalysis {
		return false, "claims_squeeze_but_analysis_missing"
	}
	if firstTF == "" {
		firstTF = "any"
	}
	return false, fmt.Sprintf("claims_squeeze_but_no_warning_%s", firstTF)
}

func validateDivergenceClaimsV2(ctx *decision.Context, symbol string, r string) (bool, string) {
	if !hasAssertedKeyword(r, []string{"divergence", "\u80cc\u79bb"}) {
		return true, "no_divergence_claim"
	}

	explicitTF := extractTimeframeNearAssertedKeyword(r, []string{"divergence", "\u80cc\u79bb"})
	tfs := timeframesForClaimCheckV2(ctx, symbol, explicitTF)
	if len(tfs) == 0 {
		return false, "claims_divergence_but_analysis_missing"
	}

	anyAnalysis := false
	firstTF := ""
	for _, tf := range tfs {
		ar, ok := computeAnalysisForSymbolTF(ctx, symbol, tf)
		if !ok || ar == nil || ar.Divergence == nil {
			continue
		}
		anyAnalysis = true
		if firstTF == "" {
			firstTF = tf
		}
		if len(ar.Divergence.Events) > 0 {
			return true, "divergence_ok"
		}
	}
	if !anyAnalysis {
		return false, "claims_divergence_but_analysis_missing"
	}
	if firstTF == "" {
		firstTF = "any"
	}
	return false, fmt.Sprintf("claims_divergence_but_none_%s", firstTF)
}

func timeframesForClaimCheckV2(ctx *decision.Context, symbol string, explicitTF string) []string {
	explicitTF = strings.TrimSpace(explicitTF)
	if explicitTF != "" {
		return []string{explicitTF}
	}
	if ctx == nil || ctx.MarketDataMap == nil {
		return nil
	}

	md, ok := ctx.MarketDataMap[symbol]
	if !ok {
		md, ok = ctx.MarketDataMap[strings.ToUpper(symbol)]
	}
	if !ok || md == nil || md.TimeframeData == nil {
		return nil
	}

	seen := make(map[string]bool, len(md.TimeframeData))
	out := make([]string, 0, len(md.TimeframeData))
	add := func(tf string) {
		tf = strings.TrimSpace(tf)
		if tf == "" || seen[tf] {
			return
		}
		tfData := md.TimeframeData[tf]
		if tfData == nil || len(tfData.Klines) == 0 {
			return
		}
		seen[tf] = true
		out = append(out, tf)
	}

	for _, tf := range ctx.Timeframes {
		add(tf)
	}
	for _, tf := range []string{"15m", "1h", "4h", "12h", "1d"} {
		add(tf)
	}

	rest := make([]string, 0, len(md.TimeframeData))
	for tf := range md.TimeframeData {
		tf = strings.TrimSpace(tf)
		if tf == "" || seen[tf] {
			continue
		}
		tfData := md.TimeframeData[tf]
		if tfData == nil || len(tfData.Klines) == 0 {
			continue
		}
		rest = append(rest, tf)
	}
	sort.Strings(rest)
	for _, tf := range rest {
		add(tf)
	}

	return out
}

func extractTimeframeNearKeyword(r string, keywords []string) string {
	r = strings.ToLower(strings.TrimSpace(r))
	if r == "" || len(keywords) == 0 {
		return ""
	}

	idx := -1
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		if i := strings.Index(r, kw); i >= 0 && (idx == -1 || i < idx) {
			idx = i
		}
	}
	if idx < 0 {
		return ""
	}

	start := idx - 64
	if start < 0 {
		start = 0
	}
	end := idx + 64
	if end > len(r) {
		end = len(r)
	}
	window := r[start:end]
	return extractReasoningTimeframe(window)
}

func extractTimeframeNearAssertedKeyword(r string, keywords []string) string {
	r = strings.ToLower(strings.TrimSpace(r))
	if r == "" || len(keywords) == 0 {
		return ""
	}

	best := -1
	bestLen := 0
	for _, kw := range keywords {
		kw = strings.ToLower(strings.TrimSpace(kw))
		if kw == "" {
			continue
		}
		searchFrom := 0
		for {
			i := strings.Index(r[searchFrom:], kw)
			if i < 0 {
				break
			}
			i += searchFrom
			if keywordIsAssertedAt(r, i, kw) && (best == -1 || i < best) {
				best = i
				bestLen = len(kw)
			}
			searchFrom = i + len(kw)
			if searchFrom >= len(r) {
				break
			}
		}
	}
	if best < 0 {
		return ""
	}
	start := best - 64
	if start < 0 {
		start = 0
	}
	end := best + bestLen + 64
	if end > len(r) {
		end = len(r)
	}
	window := r[start:end]
	kwRel := best - start
	return extractNearestTimeframe(window, kwRel)
}

func hasAssertedKeyword(r string, keywords []string) bool {
	r = strings.ToLower(r)
	for _, kw := range keywords {
		kw = strings.ToLower(strings.TrimSpace(kw))
		if kw == "" {
			continue
		}
		searchFrom := 0
		for {
			i := strings.Index(r[searchFrom:], kw)
			if i < 0 {
				break
			}
			i += searchFrom
			if keywordIsAssertedAt(r, i, kw) {
				return true
			}
			searchFrom = i + len(kw)
			if searchFrom >= len(r) {
				break
			}
		}
	}
	return false
}

func keywordIsAssertedAt(r string, idx int, kw string) bool {
	if idx < 0 || idx+len(kw) > len(r) {
		return false
	}
	beforeStart := idx - 28
	if beforeStart < 0 {
		beforeStart = 0
	}
	afterEnd := idx + len(kw) + 28
	if afterEnd > len(r) {
		afterEnd = len(r)
	}
	before := r[beforeStart:idx]
	after := r[idx+len(kw) : afterEnd]

	if containsNegationNearEnd(before) || containsNegationNearStart(after) {
		return false
	}
	if containsHedgeNearEnd(before) || containsHedgeNearStart(after) {
		return false
	}
	return true
}

func extractNearestTimeframe(window string, kwPos int) string {
	window = strings.ToLower(window)
	if window == "" || kwPos < 0 {
		return ""
	}

	type cand struct {
		tf   string
		dist int
		pos  int
	}
	best := cand{dist: 1 << 30, pos: -1}

	addMatch := func(tf string, pos int) {
		if pos < 0 {
			return
		}
		d := pos - kwPos
		if d < 0 {
			d = -d
		}
		if d < best.dist {
			best = cand{tf: tf, dist: d, pos: pos}
		}
	}

	findAll := func(tf string, needles []string) {
		for _, needle := range needles {
			if needle == "" {
				continue
			}
			searchFrom := 0
			for {
				i := strings.Index(window[searchFrom:], needle)
				if i < 0 {
					break
				}
				i += searchFrom
				if isTokenMatch(window, i, len(needle)) {
					addMatch(tf, i)
				}
				searchFrom = i + len(needle)
				if searchFrom >= len(window) {
					break
				}
			}
		}
	}

	for _, tf := range []string{"1w", "3d", "1d", "12h", "8h", "6h", "4h", "2h", "1h", "30m", "15m", "5m", "3m", "1m"} {
		findAll(tf, []string{tf})
	}
	findAll("15m", []string{"15min", "15分钟"})
	findAll("1h", []string{"1hour", "1小时"})
	findAll("4h", []string{"4hour", "4小时"})

	if best.pos < 0 {
		return ""
	}
	return best.tf
}

func isTokenMatch(s string, pos, length int) bool {
	if pos < 0 || length <= 0 || pos+length > len(s) {
		return false
	}
	isAlphaNum := func(b byte) bool {
		return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
	}
	if pos > 0 && isAlphaNum(s[pos-1]) {
		return false
	}
	if pos+length < len(s) && isAlphaNum(s[pos+length]) {
		return false
	}
	return true
}

func containsNegationNearEnd(s string) bool {
	if s == "" {
		return false
	}
	cut := len(s) - 12
	if cut < 0 {
		cut = 0
	}
	return containsAny(s[cut:], []string{
		"no ",
		"not ",
		"without ",
		"lack",
		"lacks",
		"lacking",
		"isn't",
		"isnt",
		"aren't",
		"arent",
		"never",
		"none",
		"不是",
		"并非",
		"没有",
		"无",
		"未",
		"不构成",
		"未形成",
		"未构成",
		"否",
	})
}

func containsNegationNearStart(s string) bool {
	if s == "" {
		return false
	}
	if len(s) > 12 {
		s = s[:12]
	}
	return containsAny(s, []string{
		"not",
		"no",
		"without",
		"lack",
		"不是",
		"并非",
		"没有",
		"无",
		"未",
		"否",
	})
}

func containsHedgeNearEnd(s string) bool {
	if s == "" {
		return false
	}
	cut := len(s) - 18
	if cut < 0 {
		cut = 0
	}
	return containsAny(s[cut:], []string{
		"maybe",
		"possible",
		"possibly",
		"might",
		"could",
		"potential",
		"risk",
		"watch",
		"unclear",
		"not sure",
		"?",
		"可能",
		"或许",
		"疑似",
		"风险",
		"留意",
		"观察",
		"不确定",
	})
}

func containsHedgeNearStart(s string) bool {
	if s == "" {
		return false
	}
	if len(s) > 18 {
		s = s[:18]
	}
	return containsAny(s, []string{
		"?",
		"maybe",
		"possible",
		"possibly",
		"might",
		"could",
		"unclear",
		"not sure",
		"可能",
		"或许",
		"疑似",
		"不确定",
	})
}

func containsAny(s string, terms []string) bool {
	for _, t := range terms {
		if t == "" {
			continue
		}
		if strings.Contains(s, t) {
			return true
		}
	}
	return false
}
