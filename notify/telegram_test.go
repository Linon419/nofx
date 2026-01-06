package notify

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNormalizeTelegramBotToken(t *testing.T) {
	cases := map[string]string{
		"  123456:AA_bb-CC \n\t":                                   "123456:AA_bb-CC",
		"bot123456:AA_bb-CC":                                       "123456:AA_bb-CC",
		"BOT123456:AA_bb-CC":                                       "123456:AA_bb-CC",
		"https://api.telegram.org/bot123456:AA_bb-CC/sendMessage":  "123456:AA_bb-CC",
		"https://api.telegram.org/bot123456:AA_bb-CC/getMe?x=1":    "123456:AA_bb-CC",
		"api.telegram.org/bot123456:AA_bb-CC/sendMessage":          "123456:AA_bb-CC",
		"https://api.telegram.org/bot123456:AA_bb-CC/sendMessage\n": "123456:AA_bb-CC",
	}
	for in, want := range cases {
		got := NormalizeTelegramBotToken(in)
		if got != want {
			t.Fatalf("unexpected normalized token for %q: got=%q want=%q", in, got, want)
		}
	}
}

func TestIsValidTelegramBotToken(t *testing.T) {
	valid := []string{
		"123456789:AA_bb-CCddEEffGGhhIIjjKKllMMnn",
		"  123456789:AA_bb-CCddEEffGGhhIIjjKKllMMnn  ",
		"123456789:\nAA_bb-CCddEEffGGhhIIjjKKllMMnn",
		"bot123456789:AA_bb-CCddEEffGGhhIIjjKKllMMnn",
		"https://api.telegram.org/bot123456789:AA_bb-CCddEEffGGhhIIjjKKllMMnn/sendMessage",
	}
	for _, tok := range valid {
		if !IsValidTelegramBotToken(tok) {
			t.Fatalf("expected token to be valid: %q", tok)
		}
	}

	invalid := []string{
		"",
		"codex resume 019b8af1-4d38-7123-8ae7-acb761615d51",
		"no-colon-token",
		"abc:AA_bb-CC",
		"123456789:",
		"123456789:has space",
		"botabc:AA_bb-CC",
	}
	for _, tok := range invalid {
		if IsValidTelegramBotToken(tok) {
			t.Fatalf("expected token to be invalid: %q", tok)
		}
	}
}

func TestIsValidTelegramChatID(t *testing.T) {
	valid := []string{"123", "  6533032832  ", "-1001234567890", "@channelusername", "@user_name_123"}
	for _, id := range valid {
		if !IsValidTelegramChatID(id) {
			t.Fatalf("expected chat_id to be valid: %q", id)
		}
	}

	invalid := []string{"", "abc", "12 34", "-", "@usr", "@user-name", "@user name"}
	for _, id := range invalid {
		if IsValidTelegramChatID(id) {
			t.Fatalf("expected chat_id to be invalid: %q", id)
		}
	}
}

func TestSendTelegramMessageRejectsInvalidInputs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := SendTelegramMessage(ctx, "no-colon-token", "123", "hi"); err == nil || !strings.Contains(err.Error(), "bot token") {
		t.Fatalf("expected invalid bot token error, got: %v", err)
	}

	if err := SendTelegramMessage(ctx, "123:AA_bb-CCddEEffGGhhIIjjKKllMMnn", "not-a-number", "hi"); err == nil || !strings.Contains(err.Error(), "chat_id") {
		t.Fatalf("expected invalid chat_id error, got: %v", err)
	}
}
