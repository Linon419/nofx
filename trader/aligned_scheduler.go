package trader

import (
	"context"
	"time"

	"nofx/logger"
)

type alignedOnceScheduler struct {
	Name           string
	AlignInterval  time.Duration
	Interval       time.Duration
	Offset         time.Duration
	RunImmediately bool

	ctx   context.Context
	nowFn func() time.Time
}

func newAlignedOnceScheduler(ctx context.Context, alignInterval, interval, offset time.Duration) *alignedOnceScheduler {
	if ctx == nil {
		ctx = context.Background()
	}
	return &alignedOnceScheduler{
		AlignInterval: alignInterval,
		Interval:      interval,
		Offset:        offset,
		ctx:           ctx,
		nowFn:         time.Now,
	}
}

func (s *alignedOnceScheduler) Start(task func()) {
	if s == nil {
		return
	}
	if task == nil {
		logger.Warnf("AlignedScheduler: task is nil, exit")
		return
	}
	if s.AlignInterval <= 0 {
		logger.Warnf("AlignedScheduler: invalid align_interval=%s, exit", s.AlignInterval)
		return
	}
	if s.Interval <= 0 {
		logger.Warnf("AlignedScheduler: invalid interval=%s, exit", s.Interval)
		return
	}
	if s.Offset < 0 {
		logger.Warnf("AlignedScheduler: negative offset=%s, clamp to 0", s.Offset)
		s.Offset = 0
	}
	if s.ctx == nil {
		s.ctx = context.Background()
	}
	if s.nowFn == nil {
		s.nowFn = time.Now
	}

	startAt := s.nowFn().UTC()
	prefix := "AlignedScheduler"
	if s.Name != "" {
		prefix = prefix + "[" + s.Name + "]"
	}
	logger.Infof("%s: started align_interval=%s interval=%s offset=%s run_immediately=%v at=%s",
		prefix, s.AlignInterval, s.Interval, s.Offset, s.RunImmediately, startAt.Format(time.RFC3339))

	if s.RunImmediately {
		logger.Infof("%s: RunImmediately=true, execute once before first alignment", prefix)
		task()
	}

	now := s.nowFn().UTC()
	nextClose := now.Truncate(s.AlignInterval).Add(s.AlignInterval)
	firstAt := nextClose.Add(s.Offset)
	if !s.waitUntil(firstAt) {
		return
	}
	task()

	anchor := firstAt.UTC()
	nextAt := nextFixedTimeAfter(anchor, s.Interval, s.nowFn().UTC())

	for {
		if !s.waitUntil(nextAt) {
			return
		}
		task()
		nextAt = nextFixedTimeAfter(anchor, s.Interval, s.nowFn().UTC())
	}
}

func (s *alignedOnceScheduler) waitUntil(target time.Time) bool {
	now := s.nowFn().UTC()
	wait := target.Sub(now)
	if wait <= 0 {
		select {
		case <-s.ctx.Done():
			logger.Infof("AlignedScheduler: ctx done, exit")
			return false
		default:
			return true
		}
	}

	timer := time.NewTimer(wait)
	select {
	case <-s.ctx.Done():
		timer.Stop()
		logger.Infof("AlignedScheduler: ctx done, exit")
		return false
	case <-timer.C:
		return true
	}
}

func nextFixedTimeAfter(anchor time.Time, interval time.Duration, now time.Time) time.Time {
	anchor = anchor.UTC()
	now = now.UTC()
	if interval <= 0 {
		return now
	}
	delta := now.Sub(anchor)
	if delta < 0 {
		return anchor
	}
	k := delta / interval
	return anchor.Add((k + 1) * interval)
}
