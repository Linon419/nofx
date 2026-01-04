// Package indicator provides divergence detection for multiple indicators.
//
// Ported 1:1 (behaviorally) from TradingView Pine Script:
// `analysis/indicator/背离.pine` (Divergence for Many Indicators v4).
//
// Design principle: Only output raw divergence measurements.
// The LLM is responsible for interpreting trade direction/meaning.
package indicator

import (
	"math"

	"github.com/markcheno/go-talib"

	"nofx/market"
)

type PivotSource string

const (
	PivotSourceClose   PivotSource = "close"
	PivotSourceHighLow PivotSource = "high_low"
)

type DivergenceType string

const (
	DivergencePositiveRegular DivergenceType = "positive_regular"
	DivergenceNegativeRegular DivergenceType = "negative_regular"
	DivergencePositiveHidden  DivergenceType = "positive_hidden"
	DivergenceNegativeHidden  DivergenceType = "negative_hidden"
)

type DivergenceSearchType string

const (
	SearchRegular       DivergenceSearchType = "regular"
	SearchHidden        DivergenceSearchType = "hidden"
	SearchRegularHidden DivergenceSearchType = "regular_hidden"
)

type DivergenceConfig struct {
	PivotPeriod int                  `json:"pivot_period"`
	Source      PivotSource          `json:"source"`
	Search      DivergenceSearchType `json:"search"`

	MinDivergenceCount int `json:"min_divergence_count"`
	MaxPivotPoints     int `json:"max_pivot_points"`
	MaxBars            int `json:"max_bars"`

	DontConfirm bool `json:"dont_confirm"`

	EnableMACD   bool `json:"enable_macd"`
	EnableHist   bool `json:"enable_hist"`
	EnableRSI    bool `json:"enable_rsi"`
	EnableStoch  bool `json:"enable_stoch"`
	EnableCCI    bool `json:"enable_cci"`
	EnableMOM    bool `json:"enable_mom"`
	EnableOBV    bool `json:"enable_obv"`
	EnableVWMACD bool `json:"enable_vwmacd"`
	EnableCMF    bool `json:"enable_cmf"`
	EnableMFI    bool `json:"enable_mfi"`

	EnableExternal bool      `json:"enable_external"`
	ExternalSeries []float64 `json:"-"`
}

func DefaultDivergenceConfig() DivergenceConfig {
	return DivergenceConfig{
		PivotPeriod:        5,
		Source:             PivotSourceClose,
		Search:             SearchRegular,
		MinDivergenceCount: 1,
		MaxPivotPoints:     10,
		MaxBars:            100,
		DontConfirm:        false,

		EnableMACD:   true,
		EnableHist:   true,
		EnableRSI:    true,
		EnableStoch:  true,
		EnableCCI:    true,
		EnableMOM:    true,
		EnableOBV:    true,
		EnableVWMACD: true,
		EnableCMF:    true,
		EnableMFI:    true,

		EnableExternal: false,
	}
}

type DivergenceByType struct {
	PositiveRegular int `json:"positive_regular,omitempty"`
	NegativeRegular int `json:"negative_regular,omitempty"`
	PositiveHidden  int `json:"positive_hidden,omitempty"`
	NegativeHidden  int `json:"negative_hidden,omitempty"`
}

type DivergenceEvent struct {
	Indicator string         `json:"indicator"`
	Type      DivergenceType `json:"type"`
	BarsAgo   int            `json:"bars_ago"`
}

// DivergenceChartEvent represents a divergence signal located at a specific bar index.
// It is intended for chart annotation (multiple historical markers).
type DivergenceChartEvent struct {
	Indicator string         `json:"indicator"`
	Type      DivergenceType `json:"type"`
	Index     int            `json:"index"`
}

type DivergenceResult struct {
	Total             int  `json:"total"`
	MinRequired       int  `json:"min_required,omitempty"`
	PassedMinRequired bool `json:"passed_min_required,omitempty"`

	PositiveRegularCount int `json:"positive_regular_count,omitempty"`
	NegativeRegularCount int `json:"negative_regular_count,omitempty"`
	PositiveHiddenCount  int `json:"positive_hidden_count,omitempty"`
	NegativeHiddenCount  int `json:"negative_hidden_count,omitempty"`

	Indicators map[string]DivergenceByType `json:"indicators,omitempty"`
	Events     []DivergenceEvent           `json:"events,omitempty"`
}

func CalculateDivergence(klines []market.Kline, cfg DivergenceConfig) *DivergenceResult {
	if len(klines) == 0 {
		return &DivergenceResult{Total: 0}
	}

	cfg = normalizeDivergenceConfig(cfg)

	n := len(klines)
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	volumes := make([]float64, n)
	for i, k := range klines {
		highs[i] = k.High
		lows[i] = k.Low
		closes[i] = k.Close
		volumes[i] = k.Volume
	}

	seriesByIndicator := buildDivergenceIndicatorSeries(highs, lows, closes, volumes, cfg)
	if len(seriesByIndicator) == 0 {
		return &DivergenceResult{Total: 0, MinRequired: cfg.MinDivergenceCount, PassedMinRequired: true}
	}

	pivotHighSource, pivotLowSource := pivotSources(highs, lows, closes, cfg.Source)
	phPositions, phValues, plPositions, plValues := buildPivotArrays(pivotHighSource, pivotLowSource, cfg.PivotPeriod, 20)

	lastIndex := n - 1

	indicators := make(map[string]DivergenceByType, len(seriesByIndicator))
	events := make([]DivergenceEvent, 0, 8)

	for name, series := range seriesByIndicator {
		divs := calculateDivergencesAt(
			lastIndex,
			series,
			closes,
			pivotHighSource,
			pivotLowSource,
			phPositions, phValues,
			plPositions, plValues,
			cfg,
		)

		if divs.PositiveRegular == 0 && divs.NegativeRegular == 0 && divs.PositiveHidden == 0 && divs.NegativeHidden == 0 {
			continue
		}

		indicators[name] = divs
		if divs.PositiveRegular > 0 {
			events = append(events, DivergenceEvent{Indicator: name, Type: DivergencePositiveRegular, BarsAgo: divs.PositiveRegular})
		}
		if divs.NegativeRegular > 0 {
			events = append(events, DivergenceEvent{Indicator: name, Type: DivergenceNegativeRegular, BarsAgo: divs.NegativeRegular})
		}
		if divs.PositiveHidden > 0 {
			events = append(events, DivergenceEvent{Indicator: name, Type: DivergencePositiveHidden, BarsAgo: divs.PositiveHidden})
		}
		if divs.NegativeHidden > 0 {
			events = append(events, DivergenceEvent{Indicator: name, Type: DivergenceNegativeHidden, BarsAgo: divs.NegativeHidden})
		}
	}

	// Apply the Pine `showlimit` behavior: if total divergences < min required, zero everything out.
	total := 0
	posReg := 0
	negReg := 0
	posHid := 0
	negHid := 0
	for _, v := range indicators {
		if v.PositiveRegular > 0 {
			total++
			posReg++
		}
		if v.NegativeRegular > 0 {
			total++
			negReg++
		}
		if v.PositiveHidden > 0 {
			total++
			posHid++
		}
		if v.NegativeHidden > 0 {
			total++
			negHid++
		}
	}

	passed := total >= cfg.MinDivergenceCount
	if !passed {
		return &DivergenceResult{
			Total:             0,
			MinRequired:       cfg.MinDivergenceCount,
			PassedMinRequired: false,
		}
	}

	return &DivergenceResult{
		Total:                total,
		MinRequired:          cfg.MinDivergenceCount,
		PassedMinRequired:    true,
		PositiveRegularCount: posReg,
		NegativeRegularCount: negReg,
		PositiveHiddenCount:  posHid,
		NegativeHiddenCount:  negHid,
		Indicators:           indicators,
		Events:               events,
	}
}

// CalculateDivergenceChartEventsInRange computes divergence events for multiple bar indices
// and returns chart events located at the bar where divergence is detected (Index=barIndex).
// This is heavier than CalculateDivergence (which only evaluates the latest bar), so callers
// should keep the range small (e.g. plotted window).
func CalculateDivergenceChartEventsInRange(klines []market.Kline, cfg DivergenceConfig, startIdx, endIdx int) []DivergenceChartEvent {
	if len(klines) == 0 {
		return nil
	}

	cfg = normalizeDivergenceConfig(cfg)

	n := len(klines)
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx <= 0 || endIdx > n {
		endIdx = n
	}
	if endIdx-startIdx < 2 {
		return nil
	}

	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	volumes := make([]float64, n)
	for i, k := range klines {
		highs[i] = k.High
		lows[i] = k.Low
		closes[i] = k.Close
		volumes[i] = k.Volume
	}

	seriesByIndicator := buildDivergenceIndicatorSeries(highs, lows, closes, volumes, cfg)
	if len(seriesByIndicator) == 0 {
		return nil
	}

	pivotHighSource, pivotLowSource := pivotSources(highs, lows, closes, cfg.Source)
	// Keep a bit more pivots for window rendering (still tiny arrays).
	phPositions, phValues, plPositions, plValues := buildPivotArrays(pivotHighSource, pivotLowSource, cfg.PivotPeriod, 60)

	filterPivots := func(barIndex int, positions []int, values []float64) ([]int, []float64) {
		if len(positions) == 0 || len(values) == 0 {
			return nil, nil
		}
		outPos := make([]int, 0, len(positions))
		outVal := make([]float64, 0, len(values))
		for i := 0; i < len(positions) && i < len(values); i++ {
			if positions[i] <= barIndex {
				outPos = append(outPos, positions[i])
				outVal = append(outVal, values[i])
			}
		}
		return outPos, outVal
	}

	out := make([]DivergenceChartEvent, 0, 32)
	for barIndex := startIdx; barIndex < endIdx; barIndex++ {
		phP, phV := filterPivots(barIndex, phPositions, phValues)
		plP, plV := filterPivots(barIndex, plPositions, plValues)
		if len(phP) == 0 && len(plP) == 0 {
			continue
		}

		for name, series := range seriesByIndicator {
			divs := calculateDivergencesAt(
				barIndex,
				series,
				closes,
				pivotHighSource,
				pivotLowSource,
				phP, phV,
				plP, plV,
				cfg,
			)
			if divs.PositiveRegular > 0 {
				out = append(out, DivergenceChartEvent{Indicator: name, Type: DivergencePositiveRegular, Index: barIndex})
			}
			if divs.NegativeRegular > 0 {
				out = append(out, DivergenceChartEvent{Indicator: name, Type: DivergenceNegativeRegular, Index: barIndex})
			}
			if divs.PositiveHidden > 0 {
				out = append(out, DivergenceChartEvent{Indicator: name, Type: DivergencePositiveHidden, Index: barIndex})
			}
			if divs.NegativeHidden > 0 {
				out = append(out, DivergenceChartEvent{Indicator: name, Type: DivergenceNegativeHidden, Index: barIndex})
			}
		}
	}
	return out
}

func normalizeDivergenceConfig(cfg DivergenceConfig) DivergenceConfig {
	defaults := DefaultDivergenceConfig()
	if cfg.PivotPeriod <= 0 {
		cfg.PivotPeriod = defaults.PivotPeriod
	}
	if cfg.Source == "" {
		cfg.Source = defaults.Source
	}
	if cfg.Search == "" {
		cfg.Search = defaults.Search
	}
	if cfg.MinDivergenceCount <= 0 {
		cfg.MinDivergenceCount = defaults.MinDivergenceCount
	}
	if cfg.MaxPivotPoints <= 0 {
		cfg.MaxPivotPoints = defaults.MaxPivotPoints
	}
	if cfg.MaxBars <= 0 {
		cfg.MaxBars = defaults.MaxBars
	}
	return cfg
}

func buildDivergenceIndicatorSeries(highs, lows, closes, volumes []float64, cfg DivergenceConfig) map[string][]float64 {
	n := len(closes)
	out := make(map[string][]float64, 11)

	if cfg.EnableRSI {
		series := talib.Rsi(closes, 14)
		out["rsi"] = withWarmupNaN(series, n, 14)
	}

	if cfg.EnableMACD || cfg.EnableHist {
		macd, _, hist := talib.Macd(closes, 12, 26, 9)
		// Warmup: be conservative to avoid early false detections.
		warm := 26 + 9
		if cfg.EnableMACD {
			out["macd"] = withWarmupNaN(macd, n, warm)
		}
		if cfg.EnableHist {
			out["hist"] = withWarmupNaN(hist, n, warm)
		}
	}

	if cfg.EnableMOM {
		series := talib.Mom(closes, 10)
		out["mom"] = withWarmupNaN(series, n, 10)
	}

	if cfg.EnableCCI {
		series := talib.Cci(highs, lows, closes, 10)
		out["cci"] = withWarmupNaN(series, n, 10)
	}

	if cfg.EnableOBV {
		series := talib.Obv(closes, volumes)
		out["obv"] = padToLength(series, n)
	}

	if cfg.EnableStoch {
		k := stochKSeries(highs, lows, closes, 14)
		stk := smaSeriesNaN(k, 3)
		out["stoch"] = stk
	}

	if cfg.EnableVWMACD {
		maFast := vwmaSeries(closes, volumes, 12)
		maSlow := vwmaSeries(closes, volumes, 26)
		vwmacd := make([]float64, n)
		for i := 0; i < n; i++ {
			if math.IsNaN(maFast[i]) || math.IsNaN(maSlow[i]) {
				vwmacd[i] = math.NaN()
				continue
			}
			vwmacd[i] = maFast[i] - maSlow[i]
		}
		out["vwmacd"] = vwmacd
	}

	if cfg.EnableCMF {
		cmfv := make([]float64, n)
		for i := 0; i < n; i++ {
			den := highs[i] - lows[i]
			if den == 0 {
				cmfv[i] = 0
				continue
			}
			cmfm := ((closes[i] - lows[i]) - (highs[i] - closes[i])) / den
			cmfv[i] = cmfm * volumes[i]
		}
		num := smaSeriesNaN(cmfv, 21)
		den := smaSeriesNaN(volumes, 21)
		cmf := make([]float64, n)
		for i := 0; i < n; i++ {
			if math.IsNaN(num[i]) || math.IsNaN(den[i]) || den[i] == 0 {
				cmf[i] = math.NaN()
				continue
			}
			cmf[i] = num[i] / den[i]
		}
		out["cmf"] = cmf
	}

	if cfg.EnableMFI {
		series := talib.Mfi(highs, lows, closes, volumes, 14)
		out["mfi"] = withWarmupNaN(series, n, 14)
	}

	if cfg.EnableExternal && len(cfg.ExternalSeries) == n {
		out["external"] = padToLength(cfg.ExternalSeries, n)
	}

	return out
}

func pivotSources(highs, lows, closes []float64, source PivotSource) (pivotHighSource, pivotLowSource []float64) {
	switch source {
	case PivotSourceHighLow:
		return highs, lows
	case PivotSourceClose:
		fallthrough
	default:
		return closes, closes
	}
}

func buildPivotArrays(pivotHighSource, pivotLowSource []float64, prd, maxArraySize int) (phPositions []int, phValues []float64, plPositions []int, plValues []float64) {
	n := len(pivotHighSource)
	for i := 0; i < n; i++ {
		if ph, ok := pivotHigh(pivotHighSource, prd, prd, i); ok {
			phPositions = prependInt(phPositions, i, maxArraySize)
			phValues = prependFloat(phValues, ph, maxArraySize)
		}
		if pl, ok := pivotLow(pivotLowSource, prd, prd, i); ok {
			plPositions = prependInt(plPositions, i, maxArraySize)
			plValues = prependFloat(plValues, pl, maxArraySize)
		}
	}
	return phPositions, phValues, plPositions, plValues
}

func calculateDivergencesAt(
	barIndex int,
	indicator, closes, pivotHighSource, pivotLowSource []float64,
	phPositions []int, phValues []float64,
	plPositions []int, plValues []float64,
	cfg DivergenceConfig,
) DivergenceByType {
	divs := DivergenceByType{}

	switch cfg.Search {
	case SearchRegular:
		divs.PositiveRegular = positiveRegularOrHiddenAt(barIndex, indicator, closes, pivotLowSource, plPositions, plValues, cfg, 1)
		divs.NegativeRegular = negativeRegularOrHiddenAt(barIndex, indicator, closes, pivotHighSource, phPositions, phValues, cfg, 1)
	case SearchHidden:
		divs.PositiveHidden = positiveRegularOrHiddenAt(barIndex, indicator, closes, pivotLowSource, plPositions, plValues, cfg, 2)
		divs.NegativeHidden = negativeRegularOrHiddenAt(barIndex, indicator, closes, pivotHighSource, phPositions, phValues, cfg, 2)
	case SearchRegularHidden:
		divs.PositiveRegular = positiveRegularOrHiddenAt(barIndex, indicator, closes, pivotLowSource, plPositions, plValues, cfg, 1)
		divs.NegativeRegular = negativeRegularOrHiddenAt(barIndex, indicator, closes, pivotHighSource, phPositions, phValues, cfg, 1)
		divs.PositiveHidden = positiveRegularOrHiddenAt(barIndex, indicator, closes, pivotLowSource, plPositions, plValues, cfg, 2)
		divs.NegativeHidden = negativeRegularOrHiddenAt(barIndex, indicator, closes, pivotHighSource, phPositions, phValues, cfg, 2)
	default:
		// default matches Pine behavior (Regular)
		divs.PositiveRegular = positiveRegularOrHiddenAt(barIndex, indicator, closes, pivotLowSource, plPositions, plValues, cfg, 1)
		divs.NegativeRegular = negativeRegularOrHiddenAt(barIndex, indicator, closes, pivotHighSource, phPositions, phValues, cfg, 1)
	}

	return divs
}

// cond==1 => positive regular divergence, cond==2 => positive hidden divergence.
func positiveRegularOrHiddenAt(barIndex int, indicator, closes, pivotLowSource []float64, plPositions []int, plValues []float64, cfg DivergenceConfig, cond int) int {
	if barIndex <= 0 || barIndex >= len(indicator) || barIndex >= len(closes) {
		return 0
	}
	if len(plPositions) == 0 || len(plValues) == 0 {
		return 0
	}

	// Pine gate: (dontconfirm or src > src[1] or close > close[1])
	if !cfg.DontConfirm {
		if barIndex < 1 || math.IsNaN(indicator[barIndex]) || math.IsNaN(indicator[barIndex-1]) {
			return 0
		}
		if !(indicator[barIndex] > indicator[barIndex-1] || closes[barIndex] > closes[barIndex-1]) {
			return 0
		}
	}

	startPoint := 0
	if !cfg.DontConfirm {
		startPoint = 1
	}
	if barIndex-startPoint < 0 {
		return 0
	}
	startIdx := barIndex - startPoint

	// Search last pivot points (newest first).
	for x := 0; x < cfg.MaxPivotPoints && x < len(plPositions) && x < len(plValues); x++ {
		confirmIndex := plPositions[x]
		// Equivalent to Pine: len = bar_index - pl_positions[x] + prd
		length := barIndex - confirmIndex + cfg.PivotPeriod
		if length > cfg.MaxBars {
			break
		}
		if length <= 5 {
			continue
		}
		pivotIdx := barIndex - length
		if pivotIdx < 0 || pivotIdx >= len(indicator) || pivotIdx >= len(closes) || pivotIdx >= len(pivotLowSource) {
			continue
		}

		srcStart := indicator[startIdx]
		srcPivot := indicator[pivotIdx]
		if math.IsNaN(srcStart) || math.IsNaN(srcPivot) {
			continue
		}
		priceStart := pivotLowSource[startIdx]
		pricePivot := plValues[x]

		ok := false
		if cond == 1 {
			// Positive regular: indicator higher low + price lower low.
			ok = srcStart > srcPivot && priceStart < pricePivot
		} else if cond == 2 {
			// Positive hidden: indicator lower low + price higher low.
			ok = srcStart < srcPivot && priceStart > pricePivot
		}
		if !ok {
			continue
		}

		slope1 := (srcStart - srcPivot) / float64(length-startPoint)
		virtualLine1 := srcStart - slope1
		slope2 := (closes[startIdx] - closes[pivotIdx]) / float64(length-startPoint)
		virtualLine2 := closes[startIdx] - slope2

		arrived := true
		for y := 1 + startPoint; y <= length-1; y++ {
			idx := barIndex - y
			if idx < 0 {
				arrived = false
				break
			}
			if math.IsNaN(indicator[idx]) {
				arrived = false
				break
			}
			// Pine: if src[y] < virtual_line1 or close[y] < virtual_line2 => invalid
			if indicator[idx] < virtualLine1 || closes[idx] < virtualLine2 {
				arrived = false
				break
			}
			virtualLine1 -= slope1
			virtualLine2 -= slope2
		}

		if arrived {
			return length
		}
	}

	return 0
}

// cond==1 => negative regular divergence, cond==2 => negative hidden divergence.
func negativeRegularOrHiddenAt(barIndex int, indicator, closes, pivotHighSource []float64, phPositions []int, phValues []float64, cfg DivergenceConfig, cond int) int {
	if barIndex <= 0 || barIndex >= len(indicator) || barIndex >= len(closes) {
		return 0
	}
	if len(phPositions) == 0 || len(phValues) == 0 {
		return 0
	}

	// Pine gate: (dontconfirm or src < src[1] or close < close[1])
	if !cfg.DontConfirm {
		if barIndex < 1 || math.IsNaN(indicator[barIndex]) || math.IsNaN(indicator[barIndex-1]) {
			return 0
		}
		if !(indicator[barIndex] < indicator[barIndex-1] || closes[barIndex] < closes[barIndex-1]) {
			return 0
		}
	}

	startPoint := 0
	if !cfg.DontConfirm {
		startPoint = 1
	}
	if barIndex-startPoint < 0 {
		return 0
	}
	startIdx := barIndex - startPoint

	for x := 0; x < cfg.MaxPivotPoints && x < len(phPositions) && x < len(phValues); x++ {
		confirmIndex := phPositions[x]
		length := barIndex - confirmIndex + cfg.PivotPeriod
		if length > cfg.MaxBars {
			break
		}
		if length <= 5 {
			continue
		}
		pivotIdx := barIndex - length
		if pivotIdx < 0 || pivotIdx >= len(indicator) || pivotIdx >= len(closes) || pivotIdx >= len(pivotHighSource) {
			continue
		}

		srcStart := indicator[startIdx]
		srcPivot := indicator[pivotIdx]
		if math.IsNaN(srcStart) || math.IsNaN(srcPivot) {
			continue
		}
		priceStart := pivotHighSource[startIdx]
		pricePivot := phValues[x]

		ok := false
		if cond == 1 {
			// Negative regular: indicator lower high + price higher high.
			ok = srcStart < srcPivot && priceStart > pricePivot
		} else if cond == 2 {
			// Negative hidden: indicator higher high + price lower high.
			ok = srcStart > srcPivot && priceStart < pricePivot
		}
		if !ok {
			continue
		}

		slope1 := (srcStart - srcPivot) / float64(length-startPoint)
		virtualLine1 := srcStart - slope1
		slope2 := (closes[startIdx] - closes[pivotIdx]) / float64(length-startPoint)
		virtualLine2 := closes[startIdx] - slope2

		arrived := true
		for y := 1 + startPoint; y <= length-1; y++ {
			idx := barIndex - y
			if idx < 0 {
				arrived = false
				break
			}
			if math.IsNaN(indicator[idx]) {
				arrived = false
				break
			}
			// Pine: if src[y] > virtual_line1 or close[y] > virtual_line2 => invalid
			if indicator[idx] > virtualLine1 || closes[idx] > virtualLine2 {
				arrived = false
				break
			}
			virtualLine1 -= slope1
			virtualLine2 -= slope2
		}

		if arrived {
			return length
		}
	}

	return 0
}

func pivotHigh(series []float64, left, right, i int) (float64, bool) {
	pivotIdx := i - right
	if pivotIdx < left {
		return math.NaN(), false
	}
	start := pivotIdx - left
	end := pivotIdx + right
	if start < 0 || end >= len(series) {
		return math.NaN(), false
	}
	pivot := series[pivotIdx]
	if math.IsNaN(pivot) {
		return math.NaN(), false
	}
	for j := start; j <= end; j++ {
		if j == pivotIdx {
			continue
		}
		v := series[j]
		if math.IsNaN(v) {
			return math.NaN(), false
		}
		// Strict (TradingView pivot logic: equal highs do not form a pivot high).
		if v >= pivot {
			return math.NaN(), false
		}
	}
	return pivot, true
}

func pivotLow(series []float64, left, right, i int) (float64, bool) {
	pivotIdx := i - right
	if pivotIdx < left {
		return math.NaN(), false
	}
	start := pivotIdx - left
	end := pivotIdx + right
	if start < 0 || end >= len(series) {
		return math.NaN(), false
	}
	pivot := series[pivotIdx]
	if math.IsNaN(pivot) {
		return math.NaN(), false
	}
	for j := start; j <= end; j++ {
		if j == pivotIdx {
			continue
		}
		v := series[j]
		if math.IsNaN(v) {
			return math.NaN(), false
		}
		if v <= pivot {
			return math.NaN(), false
		}
	}
	return pivot, true
}

func prependInt(dst []int, v int, maxLen int) []int {
	dst = append([]int{v}, dst...)
	if maxLen > 0 && len(dst) > maxLen {
		dst = dst[:maxLen]
	}
	return dst
}

func prependFloat(dst []float64, v float64, maxLen int) []float64 {
	dst = append([]float64{v}, dst...)
	if maxLen > 0 && len(dst) > maxLen {
		dst = dst[:maxLen]
	}
	return dst
}

func padToLength(src []float64, n int) []float64 {
	if len(src) == n {
		out := make([]float64, n)
		copy(out, src)
		return out
	}
	out := make([]float64, n)
	copy(out, src)
	for i := len(src); i < n; i++ {
		out[i] = math.NaN()
	}
	return out
}

func withWarmupNaN(src []float64, n, warmup int) []float64 {
	out := padToLength(src, n)
	if warmup <= 0 {
		return out
	}
	if warmup > n {
		warmup = n
	}
	for i := 0; i < warmup; i++ {
		out[i] = math.NaN()
	}
	return out
}

func smaSeriesNaN(values []float64, period int) []float64 {
	out := make([]float64, len(values))
	if period <= 0 {
		for i := range out {
			out[i] = math.NaN()
		}
		return out
	}
	sum := 0.0
	valid := 0
	window := make([]float64, 0, period)

	for i, v := range values {
		window = append(window, v)
		if !math.IsNaN(v) {
			sum += v
			valid++
		}
		if len(window) > period {
			removed := window[0]
			window = window[1:]
			if !math.IsNaN(removed) {
				sum -= removed
				valid--
			}
		}
		if len(window) == period && valid == period {
			out[i] = sum / float64(period)
		} else {
			out[i] = math.NaN()
		}
	}

	return out
}

func vwmaSeries(closes, volumes []float64, period int) []float64 {
	n := len(closes)
	out := make([]float64, n)
	if period <= 0 {
		for i := range out {
			out[i] = math.NaN()
		}
		return out
	}
	var sumPV, sumV float64
	for i := 0; i < n; i++ {
		sumPV += closes[i] * volumes[i]
		sumV += volumes[i]
		if i >= period {
			sumPV -= closes[i-period] * volumes[i-period]
			sumV -= volumes[i-period]
		}
		if i >= period-1 {
			if sumV == 0 {
				out[i] = math.NaN()
			} else {
				out[i] = sumPV / sumV
			}
		} else {
			out[i] = math.NaN()
		}
	}
	return out
}

func stochKSeries(highs, lows, closes []float64, period int) []float64 {
	n := len(closes)
	out := make([]float64, n)
	if period <= 0 {
		for i := range out {
			out[i] = math.NaN()
		}
		return out
	}
	for i := 0; i < n; i++ {
		if i < period-1 {
			out[i] = math.NaN()
			continue
		}
		lowMin := lows[i-period+1]
		highMax := highs[i-period+1]
		for j := i - period + 1; j <= i; j++ {
			if lows[j] < lowMin {
				lowMin = lows[j]
			}
			if highs[j] > highMax {
				highMax = highs[j]
			}
		}
		den := highMax - lowMin
		if den == 0 {
			out[i] = 0
			continue
		}
		out[i] = 100 * (closes[i] - lowMin) / den
	}
	return out
}
