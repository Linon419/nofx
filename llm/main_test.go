package llm

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("QWEN_APP_ID") == "" || os.Getenv("QWEN_API_KEY") == "" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}
