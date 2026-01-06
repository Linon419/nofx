// Package analysis provides technical analysis capabilities for market data.
// It combines pattern detection, trend analysis, and indicator calculations
// to provide structured analysis results for AI decision making.
//
// This module is designed to output raw numerical values without directional
// interpretations. The LLM is responsible for interpreting the values and
// making trading decisions.
package analysis

import (
	"nofx/analysis/indicator"
	"nofx/analysis/pattern"
	"nofx/analysis/trend"
	"nofx/market"
)

// AnalysisResult combines all analysis outputs into a single structure
// for injection into AI prompts.
type AnalysisResult struct {
	// Pattern detection results (double bottom/top, triangle, compression)
	Pattern *pattern.PatternResult `json:"pattern,omitempty"`

	// WaveTrend indicator results
	WaveTrend *indicator.WaveTrendResult `json:"wavetrend,omitempty"`

	// Divergence detection results (multi-indicator divergence)
	Divergence *indicator.DivergenceResult `json:"divergence,omitempty"`

	// Volatility warning results (BB/KC squeeze)
	VolatilityWarning *indicator.VolatilityWarningResult `json:"volatility_warning,omitempty"`

	// Trend analysis results (structure points, key levels)
	Trend *trend.TrendResult `json:"trend,omitempty"`

	// CVD (cumulative volume delta) results (quote currency, taker buy/sell proxy)
	CVD *indicator.CVDResult `json:"cvd,omitempty"`
}

// Config holds configuration for the analysis module.
type Config struct {
	// EnablePattern enables pattern detection
	EnablePattern bool `json:"enable_pattern"`

	// EnableWaveTrend enables WaveTrend indicator calculation
	EnableWaveTrend bool `json:"enable_wavetrend"`

	// EnableDivergence enables divergence detection calculation
	EnableDivergence bool `json:"enable_divergence"`

	// EnableVolatilityWarning enables volatility warning calculation
	EnableVolatilityWarning bool `json:"enable_volatility_warning"`

	// EnableTrend enables trend/structure analysis
	EnableTrend bool `json:"enable_trend"`

	// EnableCVD enables CVD calculation (requires quoteVolume + takerBuyQuoteVolume)
	EnableCVD bool `json:"enable_cvd"`

	// WaveTrend specific settings
	WaveTrend indicator.WaveTrendConfig `json:"wavetrend_config,omitempty"`

	// Divergence specific settings
	Divergence indicator.DivergenceConfig `json:"divergence_config,omitempty"`

	// Volatility warning specific settings
	VolatilityWarning indicator.VolatilityWarningConfig `json:"volatility_warning_config,omitempty"`
}

// DefaultConfig returns the default analysis configuration.
func DefaultConfig() Config {
	return Config{
		EnablePattern:           true,
		EnableWaveTrend:         true,
		EnableDivergence:        true,
		EnableVolatilityWarning: true,
		EnableTrend:             true,
		EnableCVD:               true,
		WaveTrend:               indicator.DefaultWaveTrendConfig(),
		Divergence:              indicator.DefaultDivergenceConfig(),
		VolatilityWarning:       indicator.DefaultVolatilityWarningConfig(),
	}
}

// Analyze performs comprehensive technical analysis on the provided klines.
// It returns an AnalysisResult containing pattern, indicator, and trend data.
func Analyze(klines []market.Kline, cfg Config) *AnalysisResult {
	if len(klines) == 0 {
		return &AnalysisResult{}
	}

	result := &AnalysisResult{}

	// Pattern detection
	if cfg.EnablePattern {
		result.Pattern = pattern.Detect(klines)
	}

	// WaveTrend indicator
	if cfg.EnableWaveTrend {
		result.WaveTrend = indicator.CalculateWaveTrend(klines, cfg.WaveTrend)
	}

	// Divergence
	if cfg.EnableDivergence {
		result.Divergence = indicator.CalculateDivergence(klines, cfg.Divergence)
	}

	// Volatility warning (BB/KC squeeze)
	if cfg.EnableVolatilityWarning {
		result.VolatilityWarning = indicator.CalculateVolatilityWarning(klines, cfg.VolatilityWarning)
	}

	// Trend/structure analysis
	if cfg.EnableTrend {
		result.Trend = trend.Analyze(klines)
	}

	// CVD (quote currency cumulative delta)
	if cfg.EnableCVD {
		result.CVD = indicator.CalculateCVD(klines)
	}

	return result
}

// AnalyzeWithDefaults performs analysis using default configuration.
func AnalyzeWithDefaults(klines []market.Kline) *AnalysisResult {
	return Analyze(klines, DefaultConfig())
}
