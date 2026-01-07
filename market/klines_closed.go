package market

import "time"

// MarkKlinesClosed sets kline.IsClosed based on now and each kline's close-time.
//
// Behavior:
// - If CloseTime is present (> OpenTime), use it.
// - Otherwise, best-effort derive close time from OpenTime + timeframe duration.
// - If close time cannot be determined, default to closed (IsClosed=true).
//
// This avoids silently dropping unclosed klines: consumers can decide how to use the flag.
func MarkKlinesClosed(klines []Kline, timeframe string) []Kline {
	return markKlinesClosedAt(klines, timeframe, time.Now().UTC())
}

func markKlinesClosedAt(klines []Kline, timeframe string, now time.Time) []Kline {
	if len(klines) == 0 {
		return klines
	}

	dur, _ := TFDuration(timeframe) // best-effort; may be unknown (e.g. "1M")
	durMs := dur.Milliseconds()
	nowMs := now.UnixMilli()

	for i := range klines {
		openMs := klines[i].OpenTime
		closeMs := klines[i].CloseTime

		// Some sources may not set a meaningful CloseTime; derive it when possible.
		if closeMs <= openMs && openMs > 0 && durMs > 0 {
			closeMs = openMs + durMs
		}

		if closeMs <= 0 {
			klines[i].IsClosed = true
			continue
		}

		klines[i].IsClosed = nowMs >= closeMs
	}

	return klines
}
