package store

import (
	"testing"
	"time"
)

func TestUnixMilliScan_LegacySQLiteTimestampString(t *testing.T) {
	var u UnixMilli
	if err := u.Scan("2026-01-05 22:37:46.041+00:00"); err != nil {
		t.Fatalf("scan: %v", err)
	}

	want := time.Date(2026, 1, 5, 22, 37, 46, 41*1_000_000, time.UTC).UnixMilli()
	if int64(u) != want {
		t.Fatalf("want %d, got %d", want, int64(u))
	}
}

func TestUnixMilliScan_NumericSecondsAndMillis(t *testing.T) {
	var u UnixMilli
	if err := u.Scan("1700000000"); err != nil {
		t.Fatalf("scan seconds: %v", err)
	}
	if int64(u) != 1700000000*1000 {
		t.Fatalf("seconds normalize: got %d", int64(u))
	}

	if err := u.Scan("1700000000000"); err != nil {
		t.Fatalf("scan millis: %v", err)
	}
	if int64(u) != 1700000000000 {
		t.Fatalf("millis normalize: got %d", int64(u))
	}
}

