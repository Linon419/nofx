package kernel

import (
	"strings"
	"testing"
)

// TestDataDictionary 测试数据字典定义
func TestDataDictionary(t *testing.T) {
	// 测试账户指标字典
	t.Run("AccountMetrics", func(t *testing.T) {
		equity := DataDictionary["AccountMetrics"]["Equity"]

		if equity.NameZH != "总权益" {
			t.Errorf("Expected NameZH='总权益', got '%s'", equity.NameZH)
		}

		if equity.NameEN != "Total Equity" {
			t.Errorf("Expected NameEN='Total Equity', got '%s'", equity.NameEN)
		}

		if equity.Unit != "USDT" {
			t.Errorf("Expected Unit='USDT', got '%s'", equity.Unit)
		}

		if equity.GetName(LangChinese) != "总权益" {
			t.Errorf("GetName(Chinese) failed")
		}

		if equity.GetName(LangEnglish) != "Total Equity" {
			t.Errorf("GetName(English) failed")
		}
	})

	// 测试持仓指标字典
	t.Run("PositionMetrics", func(t *testing.T) {
		peakPnL := DataDictionary["PositionMetrics"]["PeakPnL%"]

		if peakPnL.NameZH == "" {
			t.Error("PeakPnL% NameZH is empty")
		}

		if peakPnL.NameEN == "" {
			t.Error("PeakPnL% NameEN is empty")
		}

		if !strings.Contains(peakPnL.DescZH, "峰值") {
			t.Error("PeakPnL% DescZH should contain '峰值'")
		}
	})
}

// TestOIInterpretation 测试OI解读
func TestOIInterpretation(t *testing.T) {
	t.Run("OI_Up_Price_Up", func(t *testing.T) {
		if OIInterpretation.OIUp_PriceUp.ZH == "" {
			t.Error("OI Up + Price Up ZH is empty")
		}

		if OIInterpretation.OIUp_PriceUp.EN == "" {
			t.Error("OI Up + Price Up EN is empty")
		}

		if !strings.Contains(OIInterpretation.OIUp_PriceUp.ZH, "多头") {
			t.Error("OI Up + Price Up should indicate bullish trend")
		}
	})
}

// TestCommonMistakes 测试常见错误定义
func TestCommonMistakes(t *testing.T) {
	if len(CommonMistakes) == 0 {
		t.Error("CommonMistakes should not be empty")
	}

	for i, mistake := range CommonMistakes {
		if mistake.ErrorZH == "" {
			t.Errorf("Mistake #%d ErrorZH is empty", i+1)
		}

		if mistake.ErrorEN == "" {
			t.Errorf("Mistake #%d ErrorEN is empty", i+1)
		}

		if mistake.CorrectZH == "" {
			t.Errorf("Mistake #%d CorrectZH is empty", i+1)
		}

		if mistake.CorrectEN == "" {
			t.Errorf("Mistake #%d CorrectEN is empty", i+1)
		}
	}
}

// TestGetSchemaPrompt 测试Schema提示词生成
func TestGetSchemaPrompt(t *testing.T) {
	t.Run("Chinese", func(t *testing.T) {
		prompt := GetSchemaPrompt(LangChinese)

		if prompt == "" {
			t.Fatal("Chinese schema prompt is empty")
		}

		// 验证包含关键内容
		mustContain := []string{
			"数据字典",
			"账户指标",
			"交易指标",
			"持仓指标",
			"市场数据",
			"持仓量(OI)变化解读",
		}

		for _, keyword := range mustContain {
			if !strings.Contains(prompt, keyword) {
				t.Errorf("Chinese prompt should contain '%s'", keyword)
			}
		}
	})

	t.Run("English", func(t *testing.T) {
		prompt := GetSchemaPrompt(LangEnglish)

		if prompt == "" {
			t.Fatal("English schema prompt is empty")
		}

		// 验证包含关键内容
		mustContain := []string{
			"Data Dictionary",
			"Account Metrics",
			"Trade Metrics",
			"Position Metrics",
			"Market Data",
			"Open Interest",
		}

		for _, keyword := range mustContain {
			if !strings.Contains(prompt, keyword) {
				t.Errorf("English prompt should contain '%s'", keyword)
			}
		}
	})

	t.Run("Consistency", func(t *testing.T) {
		promptZH := GetSchemaPrompt(LangChinese)
		promptEN := GetSchemaPrompt(LangEnglish)

		// 两个版本都应该包含相同数量的字段定义
		// 虽然内容不同，但结构应该相似

		zhLines := strings.Split(promptZH, "\n")
		enLines := strings.Split(promptEN, "\n")

		// 行数应该大致相当（允许10%的差异）
		ratio := float64(len(zhLines)) / float64(len(enLines))
		if ratio < 0.9 || ratio > 1.1 {
			t.Logf("Warning: Line count difference is significant (ZH: %d, EN: %d)",
				len(zhLines), len(enLines))
		}
	})
}

// BenchmarkGetSchemaPrompt 性能测试
func BenchmarkGetSchemaPrompt(b *testing.B) {
	b.Run("Chinese", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = GetSchemaPrompt(LangChinese)
		}
	})

	b.Run("English", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = GetSchemaPrompt(LangEnglish)
		}
	})
}

// TestFieldDefinitionMethods 测试字段定义方法
func TestFieldDefinitionMethods(t *testing.T) {
	field := BilingualFieldDef{
		NameZH:    "测试字段",
		NameEN:    "Test Field",
		Unit:      "USDT",
		FormulaZH: "中文公式",
		FormulaEN: "English formula",
		DescZH:    "中文描述",
		DescEN:    "English description",
	}

	// 测试GetName
	if field.GetName(LangChinese) != "测试字段" {
		t.Error("GetName(Chinese) failed")
	}
	if field.GetName(LangEnglish) != "Test Field" {
		t.Error("GetName(English) failed")
	}

	// 测试GetFormula
	if field.GetFormula(LangChinese) != "中文公式" {
		t.Error("GetFormula(Chinese) failed")
	}
	if field.GetFormula(LangEnglish) != "English formula" {
		t.Error("GetFormula(English) failed")
	}

	// 测试GetDesc
	if field.GetDesc(LangChinese) != "中文描述" {
		t.Error("GetDesc(Chinese) failed")
	}
	if field.GetDesc(LangEnglish) != "English description" {
		t.Error("GetDesc(English) failed")
	}
}

// (TradingRules removed from schema; no rule-definition tests.)
