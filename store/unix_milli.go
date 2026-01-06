package store

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// UnixMilli stores timestamps as Unix milliseconds (UTC) with backwards-compatible DB scanning.
//
// It supports scanning from:
// - integer types (ms or seconds; seconds will be up-converted)
// - float types (treated as seconds when < 1e12, else ms)
// - time.Time
// - strings/bytes: either numeric (seconds/ms) or timestamp strings like
//   "2006-01-02 15:04:05.999999999+00:00" (legacy SQLite).
type UnixMilli int64

func (u UnixMilli) Value() (driver.Value, error) {
	return int64(u), nil
}

func (u *UnixMilli) Scan(value any) error {
	if u == nil {
		return fmt.Errorf("UnixMilli.Scan: nil receiver")
	}

	switch v := value.(type) {
	case nil:
		*u = 0
		return nil
	case int64:
		*u = UnixMilli(normalizeUnixMillis(v))
		return nil
	case int32:
		*u = UnixMilli(normalizeUnixMillis(int64(v)))
		return nil
	case int:
		*u = UnixMilli(normalizeUnixMillis(int64(v)))
		return nil
	case float64:
		*u = UnixMilli(normalizeUnixMillis(int64(v)))
		return nil
	case float32:
		*u = UnixMilli(normalizeUnixMillis(int64(v)))
		return nil
	case time.Time:
		*u = UnixMilli(v.UTC().UnixMilli())
		return nil
	case []byte:
		return u.scanString(string(v))
	case string:
		return u.scanString(v)
	default:
		return fmt.Errorf("UnixMilli.Scan: unsupported type %T", value)
	}
}

func (u *UnixMilli) scanString(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		*u = 0
		return nil
	}

	// Numeric string (seconds or ms)
	if isNumericString(s) {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("UnixMilli.Scan: parse int %q: %w", s, err)
		}
		*u = UnixMilli(normalizeUnixMillis(n))
		return nil
	}

	// Legacy datetime strings (SQLite) / RFC3339 variants
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05Z07:00",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			*u = UnixMilli(t.UTC().UnixMilli())
			return nil
		}
	}

	return fmt.Errorf("UnixMilli.Scan: unrecognized timestamp %q", s)
}

func isNumericString(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// normalizeUnixMillis tries to interpret n as either seconds or milliseconds.
// Heuristic: values < 1e12 are treated as seconds.
func normalizeUnixMillis(n int64) int64 {
	if n == 0 {
		return 0
	}
	if n < 1_000_000_000_000 {
		return n * 1000
	}
	return n
}

