package trader

import (
	"context"
	"fmt"
	"time"

	"nofx/market"
)

type decisionSchedule struct {
	source         string
	alignTimeframe string
	alignInterval  time.Duration
	interval       time.Duration
	offset         time.Duration
	runImmediately bool
	name           string
}

func (at *AutoTrader) newDecisionScheduler(ctx context.Context) (*alignedOnceScheduler, decisionSchedule) {
	// Default: fall back to scan interval.
	alignInterval := at.config.ScanInterval
	if alignInterval <= 0 {
		alignInterval = 3 * time.Minute
	}
	effective := decisionSchedule{
		source:         "scan_interval",
		alignInterval:  alignInterval,
		interval:       alignInterval,
		offset:         0,
		runImmediately: false,
	}

	if at != nil && at.strategyEngine != nil {
		cfg := at.strategyEngine.GetConfig()
		if cfg != nil {
			tf, tfDur, ok := pickAlignTimeframe(cfg.Indicators.Klines.SelectedTimeframes, cfg.Indicators.Klines.PrimaryTimeframe)
			if ok && tfDur > 0 {
				multiple := cfg.Indicators.Klines.DecisionIntervalMultiple
				if multiple <= 0 {
					multiple = 1
				}
				offsetSeconds := 10
				if cfg.Indicators.Klines.DecisionOffsetSeconds != nil {
					offsetSeconds = *cfg.Indicators.Klines.DecisionOffsetSeconds
				}
				if offsetSeconds < 0 {
					offsetSeconds = 0
				}

				effective = decisionSchedule{
					source:         "strategy",
					alignTimeframe: tf,
					alignInterval:  tfDur,
					interval:       tfDur * time.Duration(multiple),
					offset:         time.Duration(offsetSeconds) * time.Second,
					runImmediately: cfg.Indicators.Klines.DecisionRunImmediately,
					name:           fmt.Sprintf("tf-%s-x%d", tf, multiple),
				}
			}
		}
	}

	sched := newAlignedOnceScheduler(ctx, effective.alignInterval, effective.interval, effective.offset)
	sched.Name = effective.name
	sched.RunImmediately = effective.runImmediately
	return sched, effective
}

func pickAlignTimeframe(selected []string, primary string) (string, time.Duration, bool) {
	minTF := ""
	minDur := time.Duration(0)

	for _, tf := range selected {
		norm, err := market.NormalizeTimeframe(tf)
		if err != nil {
			continue
		}
		dur, err := market.TFDuration(norm)
		if err != nil || dur <= 0 {
			continue
		}
		if minTF == "" || dur < minDur {
			minTF = norm
			minDur = dur
		}
	}

	if minTF != "" {
		return minTF, minDur, true
	}

	norm, err := market.NormalizeTimeframe(primary)
	if err != nil {
		return "", 0, false
	}
	dur, err := market.TFDuration(norm)
	if err != nil || dur <= 0 {
		return "", 0, false
	}
	return norm, dur, true
}
