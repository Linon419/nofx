package kernel

import (
	"math"
	"nofx/market"
)

// LeverageConfig holds configuration for ATR-based leverage calculation.
type LeverageConfig struct {
	ATRPeriod       int     `json:"atr_period,omitempty"`
	ATRTimeframe    string  `json:"atr_timeframe,omitempty"`
	MaxLeverage     int     `json:"max_leverage,omitempty"`
	MinLeverage     int     `json:"min_leverage,omitempty"`
	StopLossRiskPct float64 `json:"stop_loss_risk_pct,omitempty"`
}

// LeverageResult contains the computed leverage and position sizing info.
type LeverageResult struct {
	Leverage        int     `json:"leverage"`
	ATRValue        float64 `json:"atr_value"`
	MaxATR24h       float64 `json:"max_atr_24h"`
	CurrentPrice    float64 `json:"current_price"`
	PositionSizeUSD float64 `json:"position_size_usd,omitempty"`
	StopDistancePct float64 `json:"stop_distance_pct,omitempty"`
}

// DefaultLeverageConfig returns default values for leverage calculation.
func DefaultLeverageConfig() LeverageConfig {
	return LeverageConfig{
		ATRPeriod:       14,
		ATRTimeframe:    "1d",
		MaxLeverage:     50,
		MinLeverage:     1,
		StopLossRiskPct: 5.0,
	}
}

// NormalizeLeverageConfig fills in default values for missing fields.
func NormalizeLeverageConfig(cfg LeverageConfig) LeverageConfig {
	def := DefaultLeverageConfig()
	if cfg.ATRPeriod <= 0 {
		cfg.ATRPeriod = def.ATRPeriod
	}
	if cfg.ATRTimeframe == "" {
		cfg.ATRTimeframe = def.ATRTimeframe
	} else if norm, err := market.NormalizeTimeframe(cfg.ATRTimeframe); err == nil {
		cfg.ATRTimeframe = norm
	} else {
		cfg.ATRTimeframe = def.ATRTimeframe
	}
	if cfg.MaxLeverage <= 0 {
		cfg.MaxLeverage = def.MaxLeverage
	}
	if cfg.MinLeverage <= 0 {
		cfg.MinLeverage = def.MinLeverage
	}
	if cfg.StopLossRiskPct <= 0 {
		cfg.StopLossRiskPct = def.StopLossRiskPct
	}
	return cfg
}

// CalcATRLeverage computes leverage based on ATR:
// leverage = round(close / max_atr_24h), clamped to [min, max].
func CalcATRLeverage(klines []market.Kline, timeframe string, cfg LeverageConfig) LeverageResult {
	cfg = NormalizeLeverageConfig(cfg)
	if timeframe == "" {
		timeframe = cfg.ATRTimeframe
	} else if norm, err := market.NormalizeTimeframe(timeframe); err == nil {
		timeframe = norm
	} else {
		timeframe = cfg.ATRTimeframe
	}

	result := LeverageResult{
		Leverage: cfg.MinLeverage,
	}

	if len(klines) == 0 {
		return result
	}

	currentPrice := klines[len(klines)-1].Close
	if currentPrice <= 0 {
		return result
	}
	result.CurrentPrice = currentPrice

	atrSeries := atrSeries(klines, cfg.ATRPeriod)
	if len(atrSeries) == 0 {
		return result
	}

	currentATR := lastValidFloat(atrSeries)
	if currentATR <= 0 {
		return result
	}
	result.ATRValue = roundFloat(currentATR, 4)

	barsIn24h := barsInPeriod(timeframe, 24*60)
	lookback := barsIn24h
	if lookback > len(atrSeries) {
		lookback = len(atrSeries)
	}
	if lookback < 1 {
		lookback = 1
	}

	maxATR24h := highestValue(atrSeries, lookback)
	if maxATR24h <= 0 {
		maxATR24h = currentATR
	}
	result.MaxATR24h = roundFloat(maxATR24h, 4)

	leverage := currentPrice / maxATR24h
	leverageInt := int(math.Round(leverage))
	if leverageInt < cfg.MinLeverage {
		leverageInt = cfg.MinLeverage
	}
	if leverageInt > cfg.MaxLeverage {
		leverageInt = cfg.MaxLeverage
	}
	result.Leverage = leverageInt

	return result
}

// CalcPositionSizeByStopLoss calculates position size using stop-loss distance.
// Loss = position_value * stop_distance_pct / leverage
// position_value = (capital * risk_pct%) * leverage / (stop_distance_pct/100)
func CalcPositionSizeByStopLoss(capital float64, riskPct float64, stopDistancePct float64, leverage int) float64 {
	if capital <= 0 || riskPct <= 0 || stopDistancePct <= 0 || leverage <= 0 {
		return 0
	}

	maxLoss := capital * (riskPct / 100.0)
	positionSize := maxLoss * float64(leverage) / (stopDistancePct / 100.0)

	return roundFloat(positionSize, 2)
}

// CalcLeverageWithPositionSize computes both leverage and stop-loss-based position size.
func CalcLeverageWithPositionSize(
	klines []market.Kline,
	timeframe string,
	cfg LeverageConfig,
	capital float64,
	stopDistancePct float64,
) LeverageResult {
	result := CalcATRLeverage(klines, timeframe, cfg)
	if capital > 0 && stopDistancePct > 0 {
		cfg = NormalizeLeverageConfig(cfg)
		result.StopDistancePct = stopDistancePct
		result.PositionSizeUSD = CalcPositionSizeByStopLoss(
			capital,
			cfg.StopLossRiskPct,
			stopDistancePct,
			result.Leverage,
		)
	}
	return result
}

func barsInPeriod(interval string, periodMinutes int) int {
	dur, err := market.TFDuration(interval)
	if err != nil || dur.Minutes() <= 0 {
		return 1
	}
	bars := int(math.Ceil(float64(periodMinutes) / dur.Minutes()))
	if bars < 1 {
		bars = 1
	}
	return bars
}

func atrSeries(klines []market.Kline, period int) []float64 {
	if len(klines) <= period || period <= 0 {
		return nil
	}

	trs := make([]float64, len(klines))
	for i := 1; i < len(klines); i++ {
		high := klines[i].High
		low := klines[i].Low
		prevClose := klines[i-1].Close

		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		trs[i] = math.Max(tr1, math.Max(tr2, tr3))
	}

	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trs[i]
	}
	atr := sum / float64(period)

	atrs := make([]float64, len(klines))
	atrs[period] = atr
	for i := period + 1; i < len(klines); i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
		atrs[i] = atr
	}
	return atrs
}

func highestValue(series []float64, n int) float64 {
	if len(series) == 0 || n <= 0 {
		return 0
	}
	start := len(series) - n
	if start < 0 {
		start = 0
	}
	maxVal := 0.0
	for i := start; i < len(series); i++ {
		v := series[i]
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			continue
		}
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func lastValidFloat(series []float64) float64 {
	for i := len(series) - 1; i >= 0; i-- {
		v := series[i]
		if !math.IsNaN(v) && !math.IsInf(v, 0) && v > 0 {
			return v
		}
	}
	return 0
}

func roundFloat(val float64, decimals int) float64 {
	if decimals < 0 {
		return val
	}
	pow := math.Pow(10, float64(decimals))
	return math.Round(val*pow) / pow
}
