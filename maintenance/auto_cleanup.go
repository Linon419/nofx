package maintenance

import (
	"strconv"
	"strings"
	"time"

	"nofx/logger"
	"nofx/store"
)

const (
	autoCleanupTickInterval  = time.Hour
	autoCleanupVacuumMinGap  = 12 * time.Hour
	autoCleanupDefaultDays   = 3
	autoCleanupDefaultEnable = true
)

type AutoCleanupConfig struct {
	Enabled   bool
	Days      int
	Supported bool
}

type AutoCleanupRunStats struct {
	Traders         int
	DeletedDecisions int64
	DeletedEquity    int64
	Vacuumed         bool
}

func ReadAutoCleanupConfig(st *store.Store) (AutoCleanupConfig, error) {
	if st == nil {
		return AutoCleanupConfig{Enabled: false, Days: autoCleanupDefaultDays, Supported: false}, nil
	}

	cfg := AutoCleanupConfig{
		Enabled:   autoCleanupDefaultEnable,
		Days:      autoCleanupDefaultDays,
		Supported: st.DBType() == store.DBTypeSQLite,
	}

	if raw, err := st.GetSystemConfig(store.SystemConfigAutoCleanupEnabled); err != nil {
		return cfg, err
	} else if strings.TrimSpace(raw) != "" {
		cfg.Enabled = parseBool(raw, autoCleanupDefaultEnable)
	}

	if raw, err := st.GetSystemConfig(store.SystemConfigAutoCleanupDays); err != nil {
		return cfg, err
	} else if strings.TrimSpace(raw) != "" {
		cfg.Days = parseInt(raw, autoCleanupDefaultDays)
	}
	if cfg.Days <= 0 {
		cfg.Days = autoCleanupDefaultDays
	}

	return cfg, nil
}

func SetAutoCleanupEnabled(st *store.Store, enabled bool) error {
	if st == nil {
		return nil
	}
	if err := st.SetSystemConfig(store.SystemConfigAutoCleanupEnabled, formatBool(enabled)); err != nil {
		return err
	}
	// Always keep the (currently fixed) retention days written for transparency.
	if err := st.SetSystemConfig(store.SystemConfigAutoCleanupDays, strconv.Itoa(autoCleanupDefaultDays)); err != nil {
		return err
	}
	return nil
}

func RunAutoCleanupOnce(st *store.Store, days int, allowVacuum bool, lastVacuumAt *time.Time) (AutoCleanupRunStats, error) {
	if st == nil || days <= 0 || st.DBType() != store.DBTypeSQLite {
		return AutoCleanupRunStats{}, nil
	}

	traders, err := st.Trader().ListAll()
	if err != nil {
		return AutoCleanupRunStats{}, err
	}

	stats := AutoCleanupRunStats{Traders: len(traders)}
	for _, t := range traders {
		if t == nil || strings.TrimSpace(t.ID) == "" {
			continue
		}
		if n, err := st.Decision().CleanOldRecords(t.ID, days); err != nil {
			logger.Warnf("auto-cleanup: decision cleanup failed for %s: %v", t.ID, err)
		} else {
			stats.DeletedDecisions += n
		}
		if n, err := st.Equity().CleanOldRecords(t.ID, days); err != nil {
			logger.Warnf("auto-cleanup: equity cleanup failed for %s: %v", t.ID, err)
		} else {
			stats.DeletedEquity += n
		}
	}

	if !allowVacuum {
		return stats, nil
	}
	if (stats.DeletedDecisions+stats.DeletedEquity) <= 0 {
		return stats, nil
	}

	now := time.Now()
	if lastVacuumAt != nil && !lastVacuumAt.IsZero() && now.Sub(*lastVacuumAt) < autoCleanupVacuumMinGap {
		return stats, nil
	}

	if err := st.GormDB().Exec("VACUUM").Error; err != nil {
		logger.Warnf("auto-cleanup: VACUUM failed: %v", err)
		return stats, nil
	}
	stats.Vacuumed = true
	if lastVacuumAt != nil {
		*lastVacuumAt = now
	}
	return stats, nil
}

func StartAutoCleanup(stop <-chan struct{}, st *store.Store) {
	if st == nil || st.DBType() != store.DBTypeSQLite {
		return
	}

	go func() {
		ticker := time.NewTicker(autoCleanupTickInterval)
		defer ticker.Stop()

		var lastVacuumAt time.Time

		run := func() {
			cfg, err := ReadAutoCleanupConfig(st)
			if err != nil {
				logger.Warnf("auto-cleanup: failed to read config: %v", err)
				return
			}
			if !cfg.Supported || !cfg.Enabled {
				return
			}
			stats, err := RunAutoCleanupOnce(st, cfg.Days, true, &lastVacuumAt)
			if err != nil {
				logger.Warnf("auto-cleanup: run failed: %v", err)
				return
			}
			if (stats.DeletedDecisions + stats.DeletedEquity) > 0 {
				logger.Infof("🧹 auto-cleanup: traders=%d decisions=%d equity=%d vacuum=%v",
					stats.Traders, stats.DeletedDecisions, stats.DeletedEquity, stats.Vacuumed)
			}
		}

		run()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

func parseBool(raw string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

func formatBool(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func parseInt(raw string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return def
	}
	return n
}

