package coinank

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("COINANK_ENABLE_TESTS") != "1" {
		os.Exit(0)
	}
	if os.Getenv("COINANK_API_KEY") == "" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}
