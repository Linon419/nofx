// Package indicator provides volatility warning calculations based on
// Bollinger Bands and Keltner Channels squeeze conditions.
package indicator

import (
	"math"

	"nofx/market"
)

// VolatilityWarningConfig holds configuration for volatility warnings.
type VolatilityWarningConfig struct {
	SqueezeLen         int     `json:"squeeze_len"`
	BBMult             float64 `json:"bb_mult"`
	KCMult             float64 `json:"kc_mult"`
	ATRLen             int     `json:"atr_len"`
	MinSqueezeDuration int     `json:"min_squeeze_duration"`
	TightnessL2Ratio   float64 `json:"tightness_l2_ratio"`
}

// DefaultVolatilityWarningConfig returns default settings.
func DefaultVolatilityWarningConfig() VolatilityWarningConfig {
	return VolatilityWarningConfig{
		SqueezeLen:         20,
		BBMult:             2.0,
		KCMult:             1.5,
		ATRLen:             20,
		MinSqueezeDuration: 5,
		TightnessL2Ratio:   0.80,
	}
}

// VolatilityWarningResult holds squeeze state and derived metrics.
type VolatilityWarningResult struct {
	IsSqueeze      bool    `json:"is_squeeze"`
	SqueezeCount   int     `json:"squeeze_count"`
	TightnessRatio float64 `json:"tightness_ratio"`
	KCWidth        float64 `json:"kc_width"`
	BBWidth        float64 `json:"bb_width"`
	KCMid          float64 `json:"kc_mid"`
	BBMid          float64 `json:"bb_mid"`
	KCUpper        float64 `json:"kc_upper"`
	KCLower        float64 `json:"kc_lower"`
	BBUpper        float64 `json:"bb_upper"`
	BBLower        float64 `json:"bb_lower"`
	WarningL1      bool    `json:"warning_l1"`
	WarningL2      bool    `json:"warning_l2"`
}

// CalculateVolatilityWarning computes volatility warning states from klines.
func CalculateVolatilityWarning(klines []market.Kline, cfg VolatilityWarningConfig) *VolatilityWarningResult {
	if len(klines) == 0 {
		return nil
	}

	cfg = normalizeVolatilityWarningConfig(cfg)

	n := len(klines)
	required := maxInt(cfg.SqueezeLen, cfg.ATRLen)
	if n < required {
		return nil
	}

	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	for i, k := range klines {
		highs[i] = k.High
		lows[i] = k.Low
		closes[i] = k.Close
	}

	bbMid := smaSeries(closes, cfg.SqueezeLen)
	bbStd := stdDevSeries(closes, cfg.SqueezeLen)
	kcMid := emaSeries(closes, cfg.SqueezeLen)
	atr := atrSeries(highs, lows, closes, cfg.ATRLen)

	isSqueeze := make([]bool, n)
	for i := 0; i < n; i++ {
		if math.IsNaN(bbMid[i]) || math.IsNaN(bbStd[i]) || math.IsNaN(kcMid[i]) || math.IsNaN(atr[i]) {
			continue
		}
		bbTop := bbMid[i] + cfg.BBMult*bbStd[i]
		bbBot := bbMid[i] - cfg.BBMult*bbStd[i]
		kcTop := kcMid[i] + cfg.KCMult*atr[i]
		kcBot := kcMid[i] - cfg.KCMult*atr[i]
		isSqueeze[i] = bbTop < kcTop && bbBot > kcBot
	}

	last := n - 1
	if math.IsNaN(bbMid[last]) || math.IsNaN(bbStd[last]) || math.IsNaN(kcMid[last]) || math.IsNaN(atr[last]) {
		return nil
	}

	bbTop := bbMid[last] + cfg.BBMult*bbStd[last]
	bbBot := bbMid[last] - cfg.BBMult*bbStd[last]
	kcTop := kcMid[last] + cfg.KCMult*atr[last]
	kcBot := kcMid[last] - cfg.KCMult*atr[last]

	kcWidth := kcTop - kcBot
	bbWidth := bbTop - bbBot

	tightnessRatio := 1.0
	if kcWidth > 0 {
		tightnessRatio = bbWidth / kcWidth
	}

	squeezeCount := consecutiveCount(isSqueeze)
	warningL1 := isSqueeze[last] && squeezeCount >= cfg.MinSqueezeDuration
	warningL2 := warningL1 && tightnessRatio <= cfg.TightnessL2Ratio

	return &VolatilityWarningResult{
		IsSqueeze:      isSqueeze[last],
		SqueezeCount:   squeezeCount,
		TightnessRatio: tightnessRatio,
		KCWidth:        kcWidth,
		BBWidth:        bbWidth,
		KCMid:          kcMid[last],
		BBMid:          bbMid[last],
		KCUpper:        kcTop,
		KCLower:        kcBot,
		BBUpper:        bbTop,
		BBLower:        bbBot,
		WarningL1:      warningL1,
		WarningL2:      warningL2,
	}
}

func normalizeVolatilityWarningConfig(cfg VolatilityWarningConfig) VolatilityWarningConfig {
	defaults := DefaultVolatilityWarningConfig()
	if cfg.SqueezeLen <= 0 {
		cfg.SqueezeLen = defaults.SqueezeLen
	}
	if cfg.BBMult <= 0 {
		cfg.BBMult = defaults.BBMult
	}
	if cfg.KCMult <= 0 {
		cfg.KCMult = defaults.KCMult
	}
	if cfg.ATRLen <= 0 {
		cfg.ATRLen = defaults.ATRLen
	}
	if cfg.MinSqueezeDuration <= 0 {
		cfg.MinSqueezeDuration = defaults.MinSqueezeDuration
	}
	if cfg.TightnessL2Ratio <= 0 {
		cfg.TightnessL2Ratio = defaults.TightnessL2Ratio
	}
	return cfg
}

func smaSeries(values []float64, period int) []float64 {
	out := make([]float64, len(values))
	if period <= 0 {
		fillNaN(out)
		return out
	}
	sum := 0.0
	for i, v := range values {
		sum += v
		if i >= period {
			sum -= values[i-period]
		}
		if i >= period-1 {
			out[i] = sum / float64(period)
		} else {
			out[i] = math.NaN()
		}
	}
	return out
}

func stdDevSeries(values []float64, period int) []float64 {
	out := make([]float64, len(values))
	if period <= 0 {
		fillNaN(out)
		return out
	}
	sum := 0.0
	sumSq := 0.0
	for i, v := range values {
		sum += v
		sumSq += v * v
		if i >= period {
			removed := values[i-period]
			sum -= removed
			sumSq -= removed * removed
		}
		if i >= period-1 {
			mean := sum / float64(period)
			variance := (sumSq / float64(period)) - (mean * mean)
			if variance < 0 {
				variance = 0
			}
			out[i] = math.Sqrt(variance)
		} else {
			out[i] = math.NaN()
		}
	}
	return out
}

func emaSeries(values []float64, period int) []float64 {
	out := make([]float64, len(values))
	if period <= 0 || len(values) == 0 {
		fillNaN(out)
		return out
	}
	if len(values) < period {
		fillNaN(out)
		return out
	}
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += values[i]
		out[i] = math.NaN()
	}
	ema := sum / float64(period)
	out[period-1] = ema
	multiplier := 2.0 / float64(period+1)
	for i := period; i < len(values); i++ {
		ema = (values[i]-ema)*multiplier + ema
		out[i] = ema
	}
	return out
}

func atrSeries(highs, lows, closes []float64, period int) []float64 {
	out := make([]float64, len(closes))
	if period <= 0 || len(closes) == 0 {
		fillNaN(out)
		return out
	}
	if len(closes) < period {
		fillNaN(out)
		return out
	}
	tr := make([]float64, len(closes))
	for i := range closes {
		if i == 0 {
			tr[i] = highs[i] - lows[i]
			continue
		}
		hl := highs[i] - lows[i]
		hc := math.Abs(highs[i] - closes[i-1])
		lc := math.Abs(lows[i] - closes[i-1])
		tr[i] = maxFloat(hl, hc, lc)
	}
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += tr[i]
		out[i] = math.NaN()
	}
	atr := sum / float64(period)
	out[period-1] = atr
	for i := period; i < len(tr); i++ {
		atr = (atr*float64(period-1) + tr[i]) / float64(period)
		out[i] = atr
	}
	return out
}

func consecutiveCount(values []bool) int {
	count := 0
	for i := len(values) - 1; i >= 0; i-- {
		if !values[i] {
			break
		}
		count++
	}
	return count
}

func fillNaN(values []float64) {
	for i := range values {
		values[i] = math.NaN()
	}
}

func maxFloat(values ...float64) float64 {
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
