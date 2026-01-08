package kernel

import (
	"bytes"
	"nofx/logger"
	"strings"
	"text/template"
)

// PromptTemplateVars are variables available to editable prompt modules via Go text/template syntax.
// Example: "Equity={{.AccountEquity}}; MaxPositions={{.MaxPositions}}".
type PromptTemplateVars struct {
	Language string
	Variant  string

	AccountEquity float64

	MaxPositions                 int
	BTCETHMaxLeverage            int
	AltcoinMaxLeverage           int
	BTCETHMaxPositionValueRatio  float64
	AltcoinMaxPositionValueRatio float64
	MaxMarginUsagePct            float64
	MinPositionSize              float64
	BTCETHMinPositionSize        float64
	MinRiskRewardRatio           float64
	MinConfidence                int

	ExitPlanID      string
	ExitPlanExample string
}

func renderPromptTemplate(raw string, vars PromptTemplateVars, logKey string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "{{") {
		return raw
	}

	tpl, err := template.New("prompt").Option("missingkey=default").Parse(raw)
	if err != nil {
		logger.Infof("⚠️ prompt template parse failed (%s): %v", logKey, err)
		return raw
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, vars); err != nil {
		logger.Infof("⚠️ prompt template render failed (%s): %v", logKey, err)
		return raw
	}
	return strings.TrimSpace(buf.String())
}
