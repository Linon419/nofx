// Package trend provides trend and structure analysis capabilities.
// It identifies key structural elements including fractal points,
// support/resistance levels, and trend direction.
//
// Design principle: Only output raw values and detected levels.
// The LLM is responsible for interpretation and trading decisions.
package trend

import (
	"fmt"
	"math"
	"sort"
	"time"

	"nofx/market"
)

// StructurePointType represents the type of a structural point.
type StructurePointType string

const (
	FractalHigh StructurePointType = "fractal_high"
	FractalLow  StructurePointType = "fractal_low"
	SwingHigh   StructurePointType = "swing_high"
	SwingLow    StructurePointType = "swing_low"
)

// StructurePoint represents a key structural point in price action.
type StructurePoint struct {
	// Type is the type of structure point
	Type StructurePointType `json:"type"`

	// Price is the price level of this point
	Price float64 `json:"price"`

	// Time is the timestamp of this point
	Time time.Time `json:"time"`

	// Index is the kline index where this point was found
	Index int `json:"index,omitempty"`
}

// KeyLevels holds support and resistance levels.
type KeyLevels struct {
	// Resistance levels (sorted from nearest to farthest)
	Resistance []float64 `json:"resistance"`

	// Support levels (sorted from nearest to farthest)
	Support []float64 `json:"support"`
}

// TrendResult holds the result of trend analysis.
type TrendResult struct {
	// Slope is the linear regression slope of the price series
	// Positive = upward trend, Negative = downward trend
	Slope float64 `json:"slope"`

	// SlopeAngle is the slope expressed in degrees
	SlopeAngle float64 `json:"slope_angle,omitempty"`

	// BaselineDeviation is the percentage deviation of current price from baseline
	BaselineDeviation float64 `json:"baseline_deviation,omitempty"`

	// StructurePoints contains detected fractal/swing points
	StructurePoints []StructurePoint `json:"structure_points,omitempty"`

	// KeyLevels contains support and resistance levels
	KeyLevels *KeyLevels `json:"key_levels,omitempty"`

	// Description provides a text summary of the trend
	Description string `json:"description,omitempty"`
}

// Analyze performs trend and structure analysis on the provided klines.
// It calculates trend slope, identifies fractal points, and determines key levels.
func Analyze(klines []market.Kline) *TrendResult {
	if len(klines) == 0 {
		return &TrendResult{
			Description: "No kline data available",
		}
	}

	// Extract price series
	closes := make([]float64, len(klines))
	highs := make([]float64, len(klines))
	lows := make([]float64, len(klines))
	times := make([]int64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
		highs[i] = k.High
		lows[i] = k.Low
		times[i] = k.OpenTime
	}

	result := &TrendResult{}

	// Calculate trend slope using linear regression
	slope, intercept := fitLine(closes)
	result.Slope = slope
	result.SlopeAngle = slopeToAngle(slope)

	// Calculate baseline deviation
	if len(closes) > 0 {
		lastClose := closes[len(closes)-1]
		baseline := intercept + slope*float64(len(closes)-1)
		if baseline != 0 {
			result.BaselineDeviation = (lastClose - baseline) / baseline * 100
		}
	}

	// Detect fractal/structure points
	result.StructurePoints = detectFractals(highs, lows, times)

	// Calculate key support/resistance levels
	result.KeyLevels = calculateKeyLevels(highs, lows, closes)

	// Generate description
	result.Description = describeTrend(slope, result.BaselineDeviation)

	return result
}

// fitLine performs simple linear regression on a series.
// Returns slope and intercept.
func fitLine(series []float64) (slope, intercept float64) {
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

// slopeToAngle converts a slope value to degrees.
// Uses arctangent for accurate conversion.
func slopeToAngle(slope float64) float64 {
	return math.Atan(slope) * 180.0 / math.Pi
}

// detectFractals identifies fractal high and low points.
// A fractal high is a bar whose high is higher than N bars on each side.
// A fractal low is a bar whose low is lower than N bars on each side.
// Uses Williams' fractal pattern with lookback of 2 bars on each side.
func detectFractals(highs, lows []float64, times []int64) []StructurePoint {
	var points []StructurePoint

	// Lookback period: number of bars on each side to compare
	lookback := 2
	minBars := lookback*2 + 1

	if len(highs) < minBars {
		return points
	}

	// Detect fractal highs
	for i := lookback; i < len(highs)-lookback; i++ {
		isFractalHigh := true
		for j := 1; j <= lookback; j++ {
			if highs[i] <= highs[i-j] || highs[i] <= highs[i+j] {
				isFractalHigh = false
				break
			}
		}
		if isFractalHigh {
			points = append(points, StructurePoint{
				Type:  FractalHigh,
				Price: highs[i],
				Time:  time.UnixMilli(times[i]),
				Index: i,
			})
		}
	}

	// Detect fractal lows
	for i := lookback; i < len(lows)-lookback; i++ {
		isFractalLow := true
		for j := 1; j <= lookback; j++ {
			if lows[i] >= lows[i-j] || lows[i] >= lows[i+j] {
				isFractalLow = false
				break
			}
		}
		if isFractalLow {
			points = append(points, StructurePoint{
				Type:  FractalLow,
				Price: lows[i],
				Time:  time.UnixMilli(times[i]),
				Index: i,
			})
		}
	}

	// Sort by time (most recent first)
	sort.Slice(points, func(i, j int) bool {
		return points[i].Time.After(points[j].Time)
	})

	// Limit to the most recent 10 structure points to avoid noise
	if len(points) > 10 {
		points = points[:10]
	}

	return points
}

// calculateKeyLevels determines support and resistance levels.
// It identifies levels from:
// 1. Recent significant highs and lows (historical key levels)
// 2. EMA dynamic support/resistance (21/55/100/200 periods)
// 3. Clustering nearby levels to reduce noise
func calculateKeyLevels(highs, lows, closes []float64) *KeyLevels {
	levels := &KeyLevels{
		Resistance: make([]float64, 0),
		Support:    make([]float64, 0),
	}

	if len(closes) == 0 {
		return levels
	}

	currentPrice := closes[len(closes)-1]

	// 1. Find recent significant highs and lows
	recentHighs := findSignificantExtremes(highs, true)
	recentLows := findSignificantExtremes(lows, false)

	// 2. Calculate EMA levels (dynamic S/R)
	emaPeriods := []int{21, 55, 100, 200}
	emaLevels := make([]float64, 0)
	for _, period := range emaPeriods {
		if len(closes) >= period {
			ema := calculateEMA(closes, period)
			if ema > 0 {
				emaLevels = append(emaLevels, ema)
			}
		}
	}

	// 3. Classify levels as support or resistance based on current price
	allLevels := make([]float64, 0)
	allLevels = append(allLevels, recentHighs...)
	allLevels = append(allLevels, recentLows...)
	allLevels = append(allLevels, emaLevels...)

	// Cluster nearby levels (within 0.5% of each other)
	clustered := clusterLevels(allLevels, 0.005)

	// Separate into support and resistance
	for _, level := range clustered {
		if level > currentPrice {
			levels.Resistance = append(levels.Resistance, level)
		} else {
			levels.Support = append(levels.Support, level)
		}
	}

	// Sort resistance ascending (nearest first)
	sort.Float64s(levels.Resistance)

	// Sort support descending (nearest first)
	sort.Slice(levels.Support, func(i, j int) bool {
		return levels.Support[i] > levels.Support[j]
	})

	// Limit to top 5 levels each
	if len(levels.Resistance) > 5 {
		levels.Resistance = levels.Resistance[:5]
	}
	if len(levels.Support) > 5 {
		levels.Support = levels.Support[:5]
	}

	return levels
}

// findSignificantExtremes finds the most significant high or low points.
// It looks for local extremes in the second half of the data series.
func findSignificantExtremes(data []float64, findHighs bool) []float64 {
	if len(data) < 10 {
		return nil
	}

	result := make([]float64, 0)

	// Focus on recent data (last 50% of the series)
	startIdx := len(data) / 2
	window := data[startIdx:]

	if findHighs {
		// Find local maxima
		for i := 1; i < len(window)-1; i++ {
			if window[i] > window[i-1] && window[i] > window[i+1] {
				result = append(result, window[i])
			}
		}
		// Also include the global maximum of the window
		maxVal := window[0]
		for _, v := range window {
			if v > maxVal {
				maxVal = v
			}
		}
		result = append(result, maxVal)
	} else {
		// Find local minima
		for i := 1; i < len(window)-1; i++ {
			if window[i] < window[i-1] && window[i] < window[i+1] {
				result = append(result, window[i])
			}
		}
		// Also include the global minimum of the window
		minVal := window[0]
		for _, v := range window {
			if v < minVal {
				minVal = v
			}
		}
		result = append(result, minVal)
	}

	return result
}

// calculateEMA calculates the Exponential Moving Average for a given period.
func calculateEMA(closes []float64, period int) float64 {
	if len(closes) < period {
		return 0
	}

	multiplier := 2.0 / float64(period+1)

	// Start with SMA for the first period
	var sum float64
	for i := 0; i < period; i++ {
		sum += closes[i]
	}
	ema := sum / float64(period)

	// Calculate EMA for the rest
	for i := period; i < len(closes); i++ {
		ema = (closes[i]-ema)*multiplier + ema
	}

	return ema
}

// clusterLevels groups nearby price levels and returns their average.
// Levels within 'tolerance' (as a ratio) of each other are merged.
func clusterLevels(levels []float64, tolerance float64) []float64 {
	if len(levels) == 0 {
		return nil
	}

	// Sort levels
	sorted := make([]float64, len(levels))
	copy(sorted, levels)
	sort.Float64s(sorted)

	result := make([]float64, 0)
	cluster := []float64{sorted[0]}

	for i := 1; i < len(sorted); i++ {
		// Check if current level is within tolerance of the cluster average
		clusterAvg := average(cluster)
		if (sorted[i]-clusterAvg)/clusterAvg <= tolerance {
			cluster = append(cluster, sorted[i])
		} else {
			// Save the cluster average and start a new cluster
			result = append(result, average(cluster))
			cluster = []float64{sorted[i]}
		}
	}

	// Don't forget the last cluster
	result = append(result, average(cluster))

	return result
}

// average calculates the arithmetic mean of a slice.
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// describeTrend generates a text description of the trend.
// Keeps the description factual without directional bias.
func describeTrend(slope, deviation float64) string {
	// Determine slope strength
	var slopeDesc string
	absSlope := math.Abs(slope)
	if absSlope < 0.0001 {
		slopeDesc = "flat"
	} else if absSlope < 0.001 {
		slopeDesc = "slightly"
	} else if absSlope < 0.005 {
		slopeDesc = "moderately"
	} else {
		slopeDesc = "strongly"
	}

	// Determine direction
	var direction string
	if slope > 0.0001 {
		direction = "upward"
	} else if slope < -0.0001 {
		direction = "downward"
	} else {
		direction = "sideways"
	}

	// Determine deviation description
	var deviationDesc string
	absDeviation := math.Abs(deviation)
	if absDeviation < 1 {
		deviationDesc = "close to baseline"
	} else if absDeviation < 3 {
		deviationDesc = "slightly"
	} else if absDeviation < 5 {
		deviationDesc = "moderately"
	} else {
		deviationDesc = "significantly"
	}

	var deviationDir string
	if deviation > 1 {
		deviationDir = "above"
	} else if deviation < -1 {
		deviationDir = "below"
	} else {
		deviationDir = "at"
	}

	if direction == "sideways" {
		if deviationDir == "at" {
			return fmt.Sprintf("Trend: %s, price %s", direction, deviationDesc)
		}
		return fmt.Sprintf("Trend: %s, price %s %s baseline", direction, deviationDesc, deviationDir)
	}

	if deviationDir == "at" {
		return fmt.Sprintf("Trend: %s %s, price %s", slopeDesc, direction, deviationDesc)
	}
	return fmt.Sprintf("Trend: %s %s, price %s %s baseline", slopeDesc, direction, deviationDesc, deviationDir)
}
