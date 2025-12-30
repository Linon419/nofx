// Package pattern provides chart pattern detection capabilities.
// It detects common chart patterns such as double bottom, double top,
// triangle convergence, and volatility compression.
//
// Design principle: Only output pattern detection results without
// directional bias. The LLM is responsible for interpretation.
package pattern

import (
	"fmt"
	"math"

	"nofx/market"
)

// PatternType represents the type of detected pattern.
type PatternType string

const (
	PatternNone         PatternType = "none"
	PatternDoubleBottom PatternType = "double_bottom"
	PatternDoubleTop    PatternType = "double_top"
	PatternTriangle     PatternType = "triangle"
	PatternCompression  PatternType = "compression"
)

// PatternStage represents the stage of pattern formation.
type PatternStage string

const (
	StageForming   PatternStage = "forming"
	StageConfirmed PatternStage = "confirmed"
	StageFailed    PatternStage = "failed"
)

// KeyLevels holds key price levels for a detected pattern.
type KeyLevels struct {
	// Neckline is the breakout level for patterns like double bottom/top
	Neckline float64 `json:"neckline,omitempty"`

	// Low1 is the first low point (for double bottom)
	Low1 float64 `json:"low1,omitempty"`

	// Low2 is the second low point (for double bottom)
	Low2 float64 `json:"low2,omitempty"`

	// High1 is the first high point (for double top)
	High1 float64 `json:"high1,omitempty"`

	// High2 is the second high point (for double top)
	High2 float64 `json:"high2,omitempty"`

	// UpperBound is the upper boundary (for triangle/compression)
	UpperBound float64 `json:"upper_bound,omitempty"`

	// LowerBound is the lower boundary (for triangle/compression)
	LowerBound float64 `json:"lower_bound,omitempty"`
}

// PatternResult holds the result of pattern detection.
type PatternResult struct {
	// Detected is the type of pattern detected
	Detected PatternType `json:"detected"`

	// Stage is the current stage of pattern formation
	Stage PatternStage `json:"stage,omitempty"`

	// KeyLevels contains important price levels for the pattern
	KeyLevels *KeyLevels `json:"key_levels,omitempty"`

	// Description provides a text description of the pattern
	Description string `json:"description,omitempty"`

	// Confidence is a value between 0 and 1 indicating detection confidence
	Confidence float64 `json:"confidence,omitempty"`
}

// Detect analyzes klines for chart patterns and returns detection results.
// It checks for double bottom, double top, triangle, and compression patterns.
func Detect(klines []market.Kline) *PatternResult {
	if len(klines) == 0 {
		return &PatternResult{
			Detected:    PatternNone,
			Description: "No kline data available",
		}
	}

	// Extract price series
	closes := make([]float64, len(klines))
	highs := make([]float64, len(klines))
	lows := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
		highs[i] = k.High
		lows[i] = k.Low
	}

	// Try to detect patterns in order of priority
	if result := detectDoubleBottom(closes, lows); result != nil {
		return result
	}

	if result := detectDoubleTop(highs); result != nil {
		return result
	}

	if result := detectTriangle(highs, lows); result != nil {
		return result
	}

	if result := detectCompression(highs, lows); result != nil {
		return result
	}

	return &PatternResult{
		Detected:    PatternNone,
		Description: "No significant pattern detected",
	}
}

// detectDoubleBottom checks for double bottom pattern.
// Double bottom: two low points within 0.4% price difference, with a bounce in between.
func detectDoubleBottom(closes, lows []float64) *PatternResult {
	if len(lows) < 20 {
		return nil
	}

	// Use the second half of the data to find lows
	window := lows[len(lows)/2:]

	// Find first minimum
	min1, idx1 := minWithIndex(window)

	// Create a copy and mask out the area around the first minimum
	remove := make([]float64, len(window))
	copy(remove, window)
	for i := idx1 - 2; i <= idx1+2 && i >= 0 && i < len(remove); i++ {
		remove[i] = math.MaxFloat64
	}

	// Find second minimum
	min2, idx2 := minWithIndex(remove)

	// Check if the two minimums are within 0.4% of each other
	diff := math.Abs(min1-min2) / math.Max(min1, 1)
	if diff <= 0.004 && idx2 >= 3 {
		supportLevel := (min1 + min2) / 2

		// Find the neckline (high point between the two lows)
		startIdx := minInt(idx1, idx2)
		endIdx := maxInt(idx1, idx2)
		if endIdx > startIdx {
			neckline := maxSlice(window[startIdx:endIdx])
			return &PatternResult{
				Detected:    PatternDoubleBottom,
				Stage:       StageForming,
				Description: fmt.Sprintf("Double bottom forming at support level ~%.2f", supportLevel),
				Confidence:  1.0 - diff/0.004, // Higher confidence for smaller difference
				KeyLevels: &KeyLevels{
					Low1:     min1,
					Low2:     min2,
					Neckline: neckline,
				},
			}
		}

		return &PatternResult{
			Detected:    PatternDoubleBottom,
			Stage:       StageForming,
			Description: fmt.Sprintf("Double bottom forming at support level ~%.2f", supportLevel),
			Confidence:  1.0 - diff/0.004,
			KeyLevels: &KeyLevels{
				Low1: min1,
				Low2: min2,
			},
		}
	}

	return nil
}

// detectDoubleTop checks for double top pattern.
// Double top: two high points within 0.4% price difference, with a pullback in between.
func detectDoubleTop(highs []float64) *PatternResult {
	if len(highs) < 20 {
		return nil
	}

	// Use the second half of the data to find highs
	window := highs[len(highs)/2:]

	// Find first maximum
	max1, idx1 := maxWithIndex(window)

	// Create a copy and mask out the area around the first maximum
	remove := make([]float64, len(window))
	copy(remove, window)
	for i := idx1 - 2; i <= idx1+2 && i >= 0 && i < len(remove); i++ {
		remove[i] = -math.MaxFloat64
	}

	// Find second maximum
	max2, idx2 := maxWithIndex(remove)

	// Check if the two maximums are within 0.4% of each other
	diff := math.Abs(max1-max2) / math.Max(max1, 1)
	if diff <= 0.004 && idx2 >= 3 {
		resistanceLevel := (max1 + max2) / 2

		// Find the neckline (low point between the two highs)
		startIdx := minInt(idx1, idx2)
		endIdx := maxInt(idx1, idx2)
		if endIdx > startIdx {
			neckline := minSlice(window[startIdx:endIdx])
			return &PatternResult{
				Detected:    PatternDoubleTop,
				Stage:       StageForming,
				Description: fmt.Sprintf("Double top forming at resistance level ~%.2f", resistanceLevel),
				Confidence:  1.0 - diff/0.004,
				KeyLevels: &KeyLevels{
					High1:    max1,
					High2:    max2,
					Neckline: neckline,
				},
			}
		}

		return &PatternResult{
			Detected:    PatternDoubleTop,
			Stage:       StageForming,
			Description: fmt.Sprintf("Double top forming at resistance level ~%.2f", resistanceLevel),
			Confidence:  1.0 - diff/0.004,
			KeyLevels: &KeyLevels{
				High1: max1,
				High2: max2,
			},
		}
	}

	return nil
}

// detectTriangle checks for triangle convergence pattern.
// Triangle: highs declining + lows rising, forming a converging pattern.
func detectTriangle(highs, lows []float64) *PatternResult {
	if len(highs) < 30 {
		return nil
	}

	// Compare first half vs second half
	firstHigh := maxSlice(highs[:len(highs)/2])
	lastHigh := maxSlice(highs[len(highs)/2:])
	firstLow := minSlice(lows[:len(lows)/2])
	lastLow := minSlice(lows[len(lows)/2:])

	// Triangle condition: highs declining AND lows rising
	if lastHigh < firstHigh && lastLow > firstLow {
		widthDelta := (firstHigh - firstLow) - (lastHigh - lastLow)
		if widthDelta/firstHigh > 0.05 {
			return &PatternResult{
				Detected:    PatternTriangle,
				Stage:       StageForming,
				Description: "Price range converging, possible symmetric triangle",
				Confidence:  math.Min(widthDelta/firstHigh*10, 1.0), // Scale confidence
				KeyLevels: &KeyLevels{
					UpperBound: lastHigh,
					LowerBound: lastLow,
				},
			}
		}
	}

	return nil
}

// detectCompression checks for volatility compression pattern.
// Compression: recent volatility range is less than 65% of prior range.
func detectCompression(highs, lows []float64) *PatternResult {
	if len(highs) < 40 {
		return nil
	}

	// Calculate range for first half (as percentage of high)
	firstHigh := maxSlice(highs[:len(highs)/2])
	firstLow := minSlice(lows[:len(lows)/2])
	first := (firstHigh - firstLow) / firstHigh

	// Calculate range for second half (as percentage of high)
	secondHigh := maxSlice(highs[len(highs)/2:])
	secondLow := minSlice(lows[len(lows)/2:])
	second := (secondHigh - secondLow) / secondHigh

	// Compression condition: second half range < 65% of first half range
	if second < first*0.65 {
		compressionRatio := second / first
		return &PatternResult{
			Detected:    PatternCompression,
			Stage:       StageForming,
			Description: "Volatility rapidly compressing, watch for breakout direction",
			Confidence:  1.0 - compressionRatio, // Higher confidence for more compression
			KeyLevels: &KeyLevels{
				UpperBound: secondHigh,
				LowerBound: secondLow,
			},
		}
	}

	return nil
}

// Helper functions for finding min/max values

func minSlice(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := math.MaxFloat64
	for _, v := range values {
		if v < m {
			m = v
		}
	}
	return m
}

func maxSlice(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := -math.MaxFloat64
	for _, v := range values {
		if v > m {
			m = v
		}
	}
	return m
}

func minWithIndex(values []float64) (float64, int) {
	m := math.MaxFloat64
	idx := -1
	for i, v := range values {
		if v < m {
			m = v
			idx = i
		}
	}
	return m, idx
}

func maxWithIndex(values []float64) (float64, int) {
	m := -math.MaxFloat64
	idx := -1
	for i, v := range values {
		if v > m {
			m = v
			idx = i
		}
	}
	return m, idx
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// FitLine performs linear regression on a series and returns slope and intercept.
// This can be used for trend analysis in conjunction with pattern detection.
func FitLine(series []float64) (slope, intercept float64) {
	if len(series) == 0 {
		return 0, 0
	}
	var sumX, sumY, sumXY, sumXX float64
	n := float64(len(series))
	for i, y := range series {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}
	denom := n*sumXX - sumX*sumX
	if denom == 0 {
		return 0, series[len(series)-1]
	}
	slope = (n*sumXY - sumX*sumY) / denom
	intercept = (sumY - slope*sumX) / n
	return
}
