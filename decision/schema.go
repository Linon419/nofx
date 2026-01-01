package decision

import "fmt"

// ============================================================================
// Trading Data Schema - 交易数据字典
// ============================================================================
// 双语数据字典，支持中文和英文
// 确保AI能够100%理解数据格式，无论使用哪种语言
// ============================================================================

const (
	SchemaVersion = "1.0.0"
)

// Language 语言类型
type Language string

const (
	LangChinese Language = "zh-CN"
	LangEnglish Language = "en-US"
)

// ========== 双语字段定义 ==========

// BilingualFieldDef 双语字段定义
type BilingualFieldDef struct {
	NameZH    string // 中文名称
	NameEN    string // English name
	Unit      string // 单位
	FormulaZH string // 中文公式
	FormulaEN string // English formula
	DescZH    string // 中文描述
	DescEN    string // English description
}

// GetName 获取字段名称（根据语言）
func (d BilingualFieldDef) GetName(lang Language) string {
	if lang == LangChinese {
		return d.NameZH
	}
	return d.NameEN
}

// GetFormula 获取公式（根据语言）
func (d BilingualFieldDef) GetFormula(lang Language) string {
	if lang == LangChinese {
		return d.FormulaZH
	}
	return d.FormulaEN
}

// GetDesc 获取描述（根据语言）
func (d BilingualFieldDef) GetDesc(lang Language) string {
	if lang == LangChinese {
		return d.DescZH
	}
	return d.DescEN
}

// ========== 数据字典 ==========

// DataDictionary 数据字典：定义所有字段的含义
var DataDictionary = map[string]map[string]BilingualFieldDef{
	"AccountMetrics": {
		"Equity": {
			NameZH:    "总权益",
			NameEN:    "Total Equity",
			Unit:      "USDT",
			FormulaZH: "可用余额 + 未实现盈亏",
			FormulaEN: "Available Balance + Unrealized PnL",
			DescZH:    "账户的实际净值，包含所有持仓的浮动盈亏",
			DescEN:    "Actual account value including all unrealized P&L from positions",
		},
		"Balance": {
			NameZH:    "可用余额",
			NameEN:    "Available Balance",
			Unit:      "USDT",
			FormulaZH: "初始资金 + 已实现盈亏",
			FormulaEN: "Initial Capital + Realized PnL",
			DescZH:    "可用于开新仓位的资金，不包括已用保证金",
			DescEN:    "Available funds for opening new positions, excluding used margin",
		},
		"PnL": {
			NameZH:    "总盈亏百分比",
			NameEN:    "Total PnL Percentage",
			Unit:      "%",
			FormulaZH: "(总权益 - 初始资金) / 初始资金 × 100",
			FormulaEN: "(Total Equity - Initial Capital) / Initial Capital × 100",
			DescZH:    "自系统启动以来的总收益率，+15.87%表示盈利15.87%",
			DescEN:    "Total return since inception, +15.87% means 15.87% profit",
		},
		"Margin": {
			NameZH:    "保证金使用率",
			NameEN:    "Margin Usage Rate",
			Unit:      "%",
			FormulaZH: "已用保证金合计 / 总权益 × 100",
			FormulaEN: "Total Used Margin / Total Equity × 100",
			DescZH:    "该值越高，账户风险越大。安全值<30%，危险值>70%",
			DescEN:    "Higher value = higher risk. Safe <30%, Dangerous >70%",
		},
	},

	"TradeMetrics": {
		"Entry": {
			NameZH: "进场价",
			NameEN: "Entry Price",
			Unit:   "USDT",
			DescZH: "开仓时的平均价格",
			DescEN: "Average price when opening position",
		},
		"Exit": {
			NameZH: "出场价",
			NameEN: "Exit Price",
			Unit:   "USDT",
			DescZH: "平仓时的平均价格",
			DescEN: "Average price when closing position",
		},
		"Profit": {
			NameZH:    "已实现盈亏",
			NameEN:    "Realized PnL",
			Unit:      "USDT",
			FormulaZH: "(出场价 - 进场价) / 进场价 × 杠杆 × 仓位价值",
			FormulaEN: "(Exit Price - Entry Price) / Entry Price × Leverage × Position Value",
			DescZH:    "已平仓交易的实际盈亏，包含手续费。正值=盈利，负值=亏损",
			DescEN:    "Actual profit/loss of closed trades including fees. Positive=profit, Negative=loss",
		},
		"PnL%": {
			NameZH:    "盈亏百分比",
			NameEN:    "PnL Percentage",
			Unit:      "%",
			FormulaZH: "(出场价 - 进场价) / 进场价 × 杠杆 × 100",
			FormulaEN: "(Exit - Entry) / Entry × Leverage × 100",
			DescZH:    "已平仓交易的收益率，+6.71%表示盈利6.71%",
			DescEN:    "Return on closed trade, +6.71% means 6.71% profit",
		},
		"HoldDuration": {
			NameZH: "持仓时长",
			NameEN: "Holding Duration",
			Unit:   "minutes",
			DescZH: "从开仓到平仓的时间。<15分钟=超短线，15分钟-4小时=日内，>4小时=波段",
			DescEN: "Time from open to close. <15min=scalping, 15min-4h=intraday, >4h=swing",
		},
	},

	"PositionMetrics": {
		"UnrealizedPnL%": {
			NameZH:    "未实现盈亏百分比",
			NameEN:    "Unrealized PnL Percentage",
			Unit:      "%",
			FormulaZH: "(当前价 - 进场价) / 进场价 × 杠杆 × 100",
			FormulaEN: "(Current Price - Entry Price) / Entry Price × Leverage × 100",
			DescZH:    "当前持仓的浮动盈亏，未平仓前是浮动的",
			DescEN:    "Floating P&L of current position, not realized until closed",
		},
		"PeakPnL%": {
			NameZH: "峰值盈亏百分比",
			NameEN: "Peak PnL Percentage",
			Unit:   "%",
			DescZH: "该持仓曾经达到的峰值未实现盈亏。用于判断是否需要止盈",
			DescEN: "Historical max unrealized PnL for this position. Used for take-profit decisions",
		},
		"Drawdown": {
			NameZH:    "从峰值回撤",
			NameEN:    "Drawdown from Peak",
			Unit:      "%",
			FormulaZH: "当前盈亏% - 峰值盈亏%",
			FormulaEN: "Current PnL% - Peak PnL%",
			DescZH:    "负值表示正在回撤。例如：峰值+5%，当前+3%，回撤=-2%",
			DescEN:    "Negative = pulling back. E.g., Peak +5%, Current +3%, Drawdown = -2%",
		},
		"Leverage": {
			NameZH: "杠杆倍数",
			NameEN: "Leverage",
			Unit:   "x",
			DescZH: "3x表示价格变动1%，持仓盈亏变动3%。杠杆越高，风险越大",
			DescEN: "3x means 1% price move = 3% position PnL. Higher leverage = higher risk",
		},
		"Margin": {
			NameZH:    "占用保证金",
			NameEN:    "Margin Used",
			Unit:      "USDT",
			FormulaZH: "仓位价值 / 杠杆",
			FormulaEN: "Position Value / Leverage",
			DescZH:    "该仓位锁定的保证金金额",
			DescEN:    "Collateral locked for this position",
		},
		"LiqPrice": {
			NameZH: "强平价格",
			NameEN: "Liquidation Price",
			Unit:   "USDT",
			DescZH: "价格触及此值时会被强制平仓。0.0000表示无爆仓风险",
			DescEN: "Price at which position will be force-closed. 0.0000 = no liquidation risk",
		},
	},

	"MarketData": {
		"Volume": {
			NameZH: "成交量",
			NameEN: "Volume",
			Unit:   "base asset",
			DescZH: "该时间段的交易量",
			DescEN: "Trading volume in this period",
		},
		"OI": {
			NameZH: "持仓量",
			NameEN: "Open Interest",
			Unit:   "USDT",
			DescZH: "未平仓合约的总价值。持仓量增加=资金流入，减少=资金流出",
			DescEN: "Total value of open contracts. Increasing OI = capital inflow, decreasing = outflow",
		},
		"OIChange": {
			NameZH: "持仓量变化",
			NameEN: "OI Change",
			Unit:   "USDT & %",
			DescZH: "1小时内持仓量的变化。用于判断市场真实资金流向",
			DescEN: "OI change in 1 hour. Used to determine real capital flow direction",
		},
	},

	"TechnicalAnalysis": {
		"ema_snapshot.tf": {
			NameZH: "EMA 快照周期",
			NameEN: "EMA Snapshot Timeframe",
			Unit:   "",
			DescZH: "该技术分析块对应的周期标识（例如 15m/1h/4h）",
			DescEN: "Timeframe identifier for this technical analysis block (e.g., 15m/1h/4h)",
		},
		"ema_snapshot.price": {
			NameZH: "EMA 快照价格",
			NameEN: "EMA Snapshot Price",
			Unit:   "USDT",
			DescZH: "该周期最新收盘价（用于与 EMA 序列比较）",
			DescEN: "Latest close price for the timeframe (for comparing against EMA chain)",
		},
		"ema_snapshot.ema21": {
			NameZH: "EMA21 快照",
			NameEN: "EMA21 Snapshot",
			Unit:   "USDT",
			DescZH: "该周期最新 EMA21 数值（可能缺失）",
			DescEN: "Latest EMA21 value for the timeframe (may be missing)",
		},
		"ema_snapshot.ema55": {
			NameZH: "EMA55 快照",
			NameEN: "EMA55 Snapshot",
			Unit:   "USDT",
			DescZH: "该周期最新 EMA55 数值（可能缺失）",
			DescEN: "Latest EMA55 value for the timeframe (may be missing)",
		},
		"ema_snapshot.ema100": {
			NameZH: "EMA100 快照",
			NameEN: "EMA100 Snapshot",
			Unit:   "USDT",
			DescZH: "该周期最新 EMA100 数值（可能缺失）",
			DescEN: "Latest EMA100 value for the timeframe (may be missing)",
		},
		"ema_snapshot.ema200": {
			NameZH: "EMA200 快照",
			NameEN: "EMA200 Snapshot",
			Unit:   "USDT",
			DescZH: "该周期最新 EMA200 数值（可能缺失）",
			DescEN: "Latest EMA200 value for the timeframe (may be missing)",
		},
		"wavetrend.value": {
			NameZH: "震荡指标值",
			NameEN: "WaveTrend Value",
			Unit:   "",
			DescZH: "震荡指标值，范围约 -100 ~ 100",
			DescEN: "Oscillator indicator value, range approximately -100 to 100",
		},
		"pattern.detected": {
			NameZH: "检测到的形态",
			NameEN: "Detected Pattern",
			Unit:   "",
			DescZH: "检测到的几何形态: double_bottom/double_top/triangle/compression/none",
			DescEN: "Detected geometric pattern: double_bottom/double_top/triangle/compression/none",
		},
		"pattern.stage": {
			NameZH: "形态阶段",
			NameEN: "Pattern Stage",
			Unit:   "",
			DescZH: "形态阶段: forming/confirmed/failed",
			DescEN: "Pattern stage: forming/confirmed/failed",
		},
		"trend.slope": {
			NameZH: "价格斜率",
			NameEN: "Price Slope",
			Unit:   "",
			DescZH: "价格斜率，正值向上，负值向下",
			DescEN: "Price slope, positive values indicate upward, negative values indicate downward",
		},
		"trend.key_levels": {
			NameZH: "关键价位",
			NameEN: "Key Levels",
			Unit:   "USDT",
			DescZH: "关键支撑阻力价位",
			DescEN: "Key support and resistance price levels",
		},
		"volatility_warning.is_squeeze": {
			NameZH: "Volatility squeeze",
			NameEN: "Volatility Squeeze",
			Unit:   "",
			DescZH: "Bollinger Bands inside Keltner Channels",
			DescEN: "Bollinger Bands inside Keltner Channels",
		},
		"volatility_warning.squeeze_count": {
			NameZH: "Squeeze count",
			NameEN: "Squeeze Count",
			Unit:   "bars",
			DescZH: "Consecutive squeeze bars",
			DescEN: "Consecutive squeeze bars",
		},
		"volatility_warning.warning_l1": {
			NameZH: "Volatility warning L1",
			NameEN: "Volatility Warning L1",
			Unit:   "",
			DescZH: "Squeeze lasts for minimum duration",
			DescEN: "Squeeze lasts for minimum duration",
		},
		"volatility_warning.warning_l2": {
			NameZH: "Volatility warning L2",
			NameEN: "Volatility Warning L2",
			Unit:   "",
			DescZH: "L1 and tightness below threshold",
			DescEN: "L1 and tightness below threshold",
		},
	},
}

// ========== OI解读 ==========

// OIInterpretation OI变化的市场解读（双语）
type OIInterpretationType struct {
	OIUp_PriceUp struct {
		ZH string
		EN string
	}
	OIUp_PriceDown struct {
		ZH string
		EN string
	}
	OIDown_PriceUp struct {
		ZH string
		EN string
	}
	OIDown_PriceDown struct {
		ZH string
		EN string
	}
}

var OIInterpretation = OIInterpretationType{
	OIUp_PriceUp: struct {
		ZH string
		EN string
	}{
		ZH: "强多头趋势（新多单开仓，资金流入做多）",
		EN: "Strong bullish trend (new longs opening, capital flowing into long positions)",
	},
	OIUp_PriceDown: struct {
		ZH string
		EN string
	}{
		ZH: "强空头趋势（新空单开仓，资金流入做空）",
		EN: "Strong bearish trend (new shorts opening, capital flowing into short positions)",
	},
	OIDown_PriceUp: struct {
		ZH string
		EN string
	}{
		ZH: "空头平仓（空头止损离场，可能出现反转）",
		EN: "Shorts covering (shorts stopped out, potential reversal)",
	},
	OIDown_PriceDown: struct {
		ZH string
		EN string
	}{
		ZH: "多头平仓（多头止损离场，可能出现反转）",
		EN: "Longs closing (longs stopped out, potential reversal)",
	},
}

// ========== 常见错误 ==========

// CommonMistake 常见错误定义
type CommonMistake struct {
	ErrorZH   string
	ErrorEN   string
	ExampleZH string
	ExampleEN string
	CorrectZH string
	CorrectEN string
}

var CommonMistakes = []CommonMistake{
	{
		ErrorZH:   "混淆已实现盈亏和未实现盈亏",
		ErrorEN:   "Confusing realized and unrealized P&L",
		ExampleZH: "将历史交易的盈亏与当前持仓的盈亏相加",
		ExampleEN: "Adding historical trade P&L with current position P&L",
		CorrectZH: "已实现盈亏已经计入账户余额，不应重复计算",
		CorrectEN: "Realized P&L is already included in account balance, don't double count",
	},
	{
		ErrorZH:   "忽略杠杆对盈亏的影响",
		ErrorEN:   "Ignoring leverage's impact on P&L",
		ExampleZH: "价格涨1%，认为盈利1%",
		ExampleEN: "Price up 1%, thinking profit is 1%",
		CorrectZH: "3x杠杆时，价格涨1%，实际盈利约3%",
		CorrectEN: "With 3x leverage, 1% price move = ~3% P&L",
	},
	{
		ErrorZH:   "不理解Peak PnL的重要性",
		ErrorEN:   "Not understanding Peak PnL's importance",
		ExampleZH: "只关注当前PnL，不关注回撤",
		ExampleEN: "Only watching current PnL, ignoring drawdown",
		CorrectZH: "当前PnL接近Peak PnL时，应考虑止盈以锁定利润",
		CorrectEN: "When current PnL near Peak PnL, consider taking profit to lock in gains",
	},
	{
		ErrorZH:   "忽略持仓量(OI)变化",
		ErrorEN:   "Ignoring Open Interest changes",
		ExampleZH: "只看价格K线，不看资金流向",
		ExampleEN: "Only watching price candles, not capital flows",
		CorrectZH: "结合OI变化判断趋势的真实性和持续性",
		CorrectEN: "Use OI changes to validate trend authenticity and sustainability",
	},
}

// ========== Prompt生成函数 ==========

// GetSchemaPrompt 生成Schema说明文本，用于AI Prompt
func GetSchemaPrompt(lang Language) string {
	if lang == LangChinese {
		return getSchemaPromptZH()
	}
	return getSchemaPromptEN()
}

// getSchemaPromptZH 生成中文Prompt
func getSchemaPromptZH() string {
	prompt := "# 📖 数据字典\n\n"
	prompt += "## 📊 字段含义说明\n\n"

	// 账户指标
	prompt += "### 账户指标\n"
	for key, field := range DataDictionary["AccountMetrics"] {
		prompt += formatFieldDefZH(key, field)
	}

	// 交易指标
	prompt += "\n### 交易指标\n"
	for key, field := range DataDictionary["TradeMetrics"] {
		prompt += formatFieldDefZH(key, field)
	}

	// 持仓指标
	prompt += "\n### 持仓指标\n"
	for key, field := range DataDictionary["PositionMetrics"] {
		prompt += formatFieldDefZH(key, field)
	}

	// 市场数据
	prompt += "\n### 市场数据\n"
	for key, field := range DataDictionary["MarketData"] {
		prompt += formatFieldDefZH(key, field)
	}

	// 技术分析
	prompt += "\n### 技术分析\n"
	for key, field := range DataDictionary["TechnicalAnalysis"] {
		prompt += formatFieldDefZH(key, field)
	}

	// OI解读
	prompt += "\n## 💹 持仓量(OI)变化解读\n\n"
	prompt += "- **OI增加 + 价格上涨**: " + OIInterpretation.OIUp_PriceUp.ZH + "\n"
	prompt += "- **OI增加 + 价格下跌**: " + OIInterpretation.OIUp_PriceDown.ZH + "\n"
	prompt += "- **OI减少 + 价格上涨**: " + OIInterpretation.OIDown_PriceUp.ZH + "\n"
	prompt += "- **OI减少 + 价格下跌**: " + OIInterpretation.OIDown_PriceDown.ZH + "\n"

	// 常见错误
	prompt += "\n## ⚠️ 常见错误（请避免）\n\n"
	for i, mistake := range CommonMistakes {
		prompt += fmt.Sprintf("**错误%d**: %s\n", i+1, mistake.ErrorZH)
		prompt += "- 错误示例：" + mistake.ExampleZH + "\n"
		prompt += "- 正确做法：" + mistake.CorrectZH + "\n\n"
	}

	return prompt
}

// getSchemaPromptEN 生成英文Prompt
func getSchemaPromptEN() string {
	prompt := "# 📖 Data Dictionary\n\n"
	prompt += "## 📊 Field Definitions\n\n"

	// Account Metrics
	prompt += "### Account Metrics\n"
	for key, field := range DataDictionary["AccountMetrics"] {
		prompt += formatFieldDefEN(key, field)
	}

	// Trade Metrics
	prompt += "\n### Trade Metrics\n"
	for key, field := range DataDictionary["TradeMetrics"] {
		prompt += formatFieldDefEN(key, field)
	}

	// Position Metrics
	prompt += "\n### Position Metrics\n"
	for key, field := range DataDictionary["PositionMetrics"] {
		prompt += formatFieldDefEN(key, field)
	}

	// Market Data
	prompt += "\n### Market Data\n"
	for key, field := range DataDictionary["MarketData"] {
		prompt += formatFieldDefEN(key, field)
	}

	// Technical Analysis
	prompt += "\n### Technical Analysis\n"
	for key, field := range DataDictionary["TechnicalAnalysis"] {
		prompt += formatFieldDefEN(key, field)
	}

	// OI Interpretation
	prompt += "\n## 💹 Open Interest (OI) Change Interpretation\n\n"
	prompt += "- **OI Up + Price Up**: " + OIInterpretation.OIUp_PriceUp.EN + "\n"
	prompt += "- **OI Up + Price Down**: " + OIInterpretation.OIUp_PriceDown.EN + "\n"
	prompt += "- **OI Down + Price Up**: " + OIInterpretation.OIDown_PriceUp.EN + "\n"
	prompt += "- **OI Down + Price Down**: " + OIInterpretation.OIDown_PriceDown.EN + "\n"

	// Common Mistakes
	prompt += "\n## ⚠️ Common Mistakes to Avoid\n\n"
	for i, mistake := range CommonMistakes {
		prompt += fmt.Sprintf("**Mistake %d**: %s\n", i+1, mistake.ErrorEN)
		prompt += "- Bad Example: " + mistake.ExampleEN + "\n"
		prompt += "- Correct Approach: " + mistake.CorrectEN + "\n\n"
	}

	return prompt
}

// formatFieldDefZH 格式化中文字段定义
func formatFieldDefZH(key string, field BilingualFieldDef) string {
	result := "- **" + key + "**（" + field.NameZH + "）: " + field.DescZH
	if field.FormulaZH != "" {
		result += " | 公式: `" + field.FormulaZH + "`"
	}
	if field.Unit != "" {
		result += " | 单位: " + field.Unit
	}
	result += "\n"
	return result
}

// formatFieldDefEN 格式化英文字段定义
func formatFieldDefEN(key string, field BilingualFieldDef) string {
	result := "- **" + key + "** (" + field.NameEN + "): " + field.DescEN
	if field.FormulaEN != "" {
		result += " | Formula: `" + field.FormulaEN + "`"
	}
	if field.Unit != "" {
		result += " | Unit: " + field.Unit
	}
	result += "\n"
	return result
}
