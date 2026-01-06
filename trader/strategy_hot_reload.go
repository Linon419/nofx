package trader

import (
	"nofx/kernel"
	"nofx/logger"
	"nofx/store"
)

// QueueStrategyConfigUpdate schedules a strategy config hot-reload for the next cycle.
// It is safe to call from other goroutines (e.g. API handlers).
func (at *AutoTrader) QueueStrategyConfigUpdate(cfg *store.StrategyConfig, reason string) {
	if at == nil || cfg == nil {
		return
	}
	at.pendingStrategyMu.Lock()
	at.pendingStrategyConfig = cfg
	at.pendingStrategyReason = reason
	at.pendingStrategyMu.Unlock()
}

func (at *AutoTrader) applyQueuedStrategyConfigUpdateIfAny() {
	if at == nil {
		return
	}

	at.pendingStrategyMu.Lock()
	cfg := at.pendingStrategyConfig
	reason := at.pendingStrategyReason
	at.pendingStrategyConfig = nil
	at.pendingStrategyReason = ""
	at.pendingStrategyMu.Unlock()

	if cfg == nil {
		return
	}

	at.config.StrategyConfig = cfg
	at.strategyEngine = kernel.NewStrategyEngine(cfg)
	if reason == "" {
		reason = "updated"
	}
	logger.Infof("♻️ Strategy config hot-reloaded: %s", reason)
}
