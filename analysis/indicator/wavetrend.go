// Package indicator provides technical indicator calculations.
// The primary indicator is WaveTrend-MFI Hybrid with ALMA smoothing,
// which combines WaveTrend oscillator with Money Flow Index.
//
// Design principle: Only output raw indicator values.
// The LLM is responsible for interpretation (overbought/oversold/neutral).
package indicator

import (
	"math"

	"github.com/markcheno/go-talib"

	"nofx/market"
)

// WaveTrendConfig holds configuration for WaveTrend calculation.
type WaveTrendConfig struct {
	// ChannelLen is the length for EMA channel calculation
	ChannelLen int `json:"channel_len"`

	// AvgLen is the length for averaging
	AvgLen int `json:"avg_len"`

	// SmoothLen is the length for ALMA smoothing
	SmoothLen int `json:"smooth_len"`

	// MFILen is the length for Money Flow Index
	MFILen int `json:"mfi_len"`

	// WTWeight is the weight for WaveTrend component (0-1)
	WTWeight float64 `json:"wt_weight"`

	// MFIScale is the scaling factor for MFI component
	MFIScale float64 `json:"mfi_scale"`

	// ALMA parameters
	ALMAOffset float64 `json:"alma_offset"`
	ALMASigma  float64 `json:"alma_sigma"`
}

// DefaultWaveTrendConfig returns the default WaveTrend configuration.
func DefaultWaveTrendConfig() WaveTrendConfig {
	return WaveTrendConfig{
		ChannelLen: 10,
		AvgLen:     8,
		SmoothLen:  5,
		MFILen:     10,
		WTWeight:   0.3,
		MFIScale:   1.5,
		ALMAOffset: 0.85,
		ALMASigma:  6.0,
	}
}

// WaveTrendResult holds the result of WaveTrend calculation.
type WaveTrendResult struct {
	// Value is the final WaveTrend-MFI hybrid value
	// Range is approximately -100 to 100
	// Reference thresholds: >= 50 (overbought zone), <= -50 (oversold zone)
	Value float64 `json:"value"`

	// Series holds historical WaveTrend values (optional, for charting)
	Series []float64 `json:"series,omitempty"`

	// WT1 is the raw WaveTrend line 1 value
	WT1 float64 `json:"wt1,omitempty"`

	// WT2 is the smoothed WaveTrend line 2 value
	WT2 float64 `json:"wt2,omitempty"`

	// MFI is the Money Flow Index value (0-100)
	MFI float64 `json:"mfi,omitempty"`
}

// CalculateWaveTrend computes the WaveTrend-MFI Hybrid indicator.
// It combines WaveTrend oscillator with MFI and applies ALMA smoothing.
func CalculateWaveTrend(klines []market.Kline, cfg WaveTrendConfig) *WaveTrendResult {
	if len(klines) == 0 {
		return &WaveTrendResult{Value: 0}
	}

	// Normalize config with defaults if needed
	cfg = normalizeConfig(cfg)

	// Extract OHLCV data
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

	// Calculate WaveTrend-MFI Hybrid series
	series := calculateWTMFIHybridSeries(highs, lows, closes, volumes, cfg)
	if len(series) == 0 {
		return &WaveTrendResult{Value: 0}
	}

	// Apply post-processing (ALMA smoothing, quantization)
	processed := postProcess(series, cfg.SmoothLen)

	// Get the latest valid value
	value := lastValidValue(processed)

	return &WaveTrendResult{
		Value:  value,
		Series: sanitizeSeries(processed),
	}
}

// CalculateWaveTrendSeriesForChart returns a length-aligned series for charting (same length as klines).
// It keeps NaN gaps so renderers can skip missing points while preserving index alignment.
func CalculateWaveTrendSeriesForChart(klines []market.Kline, cfg WaveTrendConfig) []float64 {
	if len(klines) == 0 {
		return nil
	}

	cfg = normalizeConfig(cfg)

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

	series := calculateWTMFIHybridSeries(highs, lows, closes, volumes, cfg)
	if len(series) == 0 {
		return nil
	}
	return postProcess(series, cfg.SmoothLen)
}

// normalizeConfig ensures config has valid values.
func normalizeConfig(cfg WaveTrendConfig) WaveTrendConfig {
	defaults := DefaultWaveTrendConfig()
	if cfg.ChannelLen <= 0 {
		cfg.ChannelLen = defaults.ChannelLen
	}
	if cfg.AvgLen <= 0 {
		cfg.AvgLen = defaults.AvgLen
	}
	if cfg.SmoothLen <= 0 {
		cfg.SmoothLen = defaults.SmoothLen
	}
	if cfg.MFILen <= 0 {
		cfg.MFILen = defaults.MFILen
	}
	if cfg.WTWeight <= 0 {
		cfg.WTWeight = defaults.WTWeight
	}
	if cfg.MFIScale <= 0 {
		cfg.MFIScale = defaults.MFIScale
	}
	if cfg.ALMAOffset <= 0 {
		cfg.ALMAOffset = defaults.ALMAOffset
	}
	if cfg.ALMASigma <= 0 {
		cfg.ALMASigma = defaults.ALMASigma
	}
	return cfg
}

// calculateWTMFIHybridSeries computes the WaveTrend-MFI hybrid series.
func calculateWTMFIHybridSeries(highs, lows, closes, volumes []float64, cfg WaveTrendConfig) []float64 {
	n := len(closes)
	if n == 0 {
		return nil
	}

	// Calculate HLC3 (Typical Price)
	hlc3 := make([]float64, n)
	for i := range closes {
		hlc3[i] = (highs[i] + lows[i] + closes[i]) / 3.0
	}

	// Calculate ESA (EMA of HLC3)
	esa := ema(hlc3, cfg.ChannelLen)

	// Calculate absolute difference
	absDiff := make([]float64, n)
	for i := range hlc3 {
		absDiff[i] = math.Abs(hlc3[i] - esa[i])
	}

	// Calculate D (EMA of absolute difference)
	d := ema(absDiff, cfg.ChannelLen)

	// Calculate CI (Channel Index)
	ci := make([]float64, n)
	for i := range hlc3 {
		denom := 0.015 * d[i]
		if denom == 0 {
			ci[i] = 0
		} else {
			ci[i] = (hlc3[i] - esa[i]) / denom
		}
	}

	// Calculate WT1 (EMA of CI)
	wt1 := ema(ci, cfg.AvgLen)

	// Calculate WT2 (ALMA of WT1)
	wt2 := alma(wt1, cfg.SmoothLen, cfg.ALMAOffset, cfg.ALMASigma)

	// Calculate MFI
	mfiSeries := calculateMFI(highs, lows, closes, volumes, cfg.MFILen)

	// Combine WT and MFI
	hybrid := make([]float64, n)
	for i := range hybrid {
		// Scale MFI from 0-100 to -50 to +50
		mfiScaled := (mfiSeries[i] - 50) * cfg.MFIScale
		hybrid[i] = cfg.WTWeight*wt2[i] + (1-cfg.WTWeight)*mfiScaled
	}

	// Mark initial values as NaN (not enough data)
	required := maxInt(cfg.ChannelLen, cfg.AvgLen, cfg.SmoothLen, cfg.MFILen) + 1
	if required > n {
		required = n
	}
	for i := 0; i < required; i++ {
		hybrid[i] = math.NaN()
	}

	return hybrid
}

// postProcess applies smoothing and quantization to the series.
func postProcess(series []float64, smoothLen int) []float64 {
	if len(series) == 0 {
		return nil
	}

	const (
		postMult   = 1.2
		oscMax     = 60.0
		oscMin     = -60.0
		stepSize   = 6.6
		quantize   = true
	)

	// Apply multiplier
	processed := make([]float64, len(series))
	for i, v := range series {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			processed[i] = math.NaN()
			continue
		}
		processed[i] = v * postMult
	}

	// Apply ALMA smoothing
	smoothed := alma(processed, smoothLen, 0.85, 6.0)

	// Apply clamping and quantization
	out := make([]float64, len(series))
	for i, v := range smoothed {
		if i < smoothLen-1 || math.IsNaN(v) || math.IsInf(v, 0) || math.IsNaN(processed[i]) {
			out[i] = math.NaN()
			continue
		}
		val := clamp(v, oscMin, oscMax)
		if quantize && stepSize > 0 {
			val = quantizeStep(val, stepSize)
			val = clamp(val, oscMin, oscMax)
		}
		out[i] = val
	}

	return out
}

// ema calculates Exponential Moving Average.
func ema(values []float64, period int) []float64 {
	n := len(values)
	if n == 0 || period <= 0 {
		if n == 0 {
			return nil
		}
		return make([]float64, n)
	}

	result := talib.Ema(values, period)
	if len(result) != n {
		padded := make([]float64, n)
		copy(padded, result)
		return padded
	}
	return result
}

// alma calculates Arnaud Legoux Moving Average.
// offset: controls the center of the window (0.85 typical)
// sigma: controls the width of the Gaussian curve (6.0 typical)
func alma(values []float64, length int, offset, sigma float64) []float64 {
	n := len(values)
	out := make([]float64, n)
	if length <= 0 || n == 0 {
		return out
	}

	m := offset * float64(length-1)
	s := float64(length) / sigma
	denom := 2 * s * s

	for i := range values {
		if i+1 < length {
			out[i] = 0
			continue
		}
		sum := 0.0
		wSum := 0.0
		for j := 0; j < length; j++ {
			idx := i - length + 1 + j
			w := math.Exp(-((float64(j) - m) * (float64(j) - m)) / denom)
			sum += w * values[idx]
			wSum += w
		}
		if wSum == 0 {
			out[i] = 0
		} else {
			out[i] = sum / wSum
		}
	}
	return out
}

// calculateMFI computes Money Flow Index.
func calculateMFI(highs, lows, closes, volumes []float64, period int) []float64 {
	n := len(closes)
	if n == 0 || period <= 0 {
		if n == 0 {
			return nil
		}
		return make([]float64, n)
	}
	result := talib.Mfi(highs, lows, closes, volumes, period)
	if len(result) != n {
		padded := make([]float64, n)
		copy(padded, result)
		return padded
	}
	return result
}

// Helper functions

func clamp(val, minVal, maxVal float64) float64 {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

func quantizeStep(val, step float64) float64 {
	if step <= 0 {
		return val
	}
	return math.Round(val/step) * step
}

func maxInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func lastValidValue(series []float64) float64 {
	for i := len(series) - 1; i >= 0; i-- {
		if !math.IsNaN(series[i]) && !math.IsInf(series[i], 0) {
			return series[i]
		}
	}
	return 0
}

func sanitizeSeries(src []float64) []float64 {
	out := make([]float64, 0, len(src))
	for _, v := range src {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		out = append(out, math.Round(v*10000)/10000)
	}
	return out
}
