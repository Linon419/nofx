package market

import "time"

const defaultKlineCloseGrace = 10 * time.Second

// DropUnclosedKlines removes the latest kline if it has not closed yet.
func DropUnclosedKlines(klines []Kline, timeframe string) []Kline {
	return dropUnclosedKlinesAt(klines, timeframe, time.Now().UTC(), defaultKlineCloseGrace)
}

func dropUnclosedKlinesAt(klines []Kline, timeframe string, now time.Time, grace time.Duration) []Kline {
	if len(klines) == 0 {
		return klines
	}
	dur, err := TFDuration(timeframe)
	if err != nil || dur <= 0 {
		return klines
	}
	if grace < 0 {
		grace = 0
	}
	last := klines[len(klines)-1]
	if last.OpenTime <= 0 {
		return klines
	}
	closeTimeMs := last.OpenTime + dur.Milliseconds()
	cutoffMs := closeTimeMs + grace.Milliseconds()
	if now.UnixMilli() < cutoffMs {
		return klines[:len(klines)-1]
	}
	return klines
}
