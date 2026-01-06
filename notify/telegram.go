package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type telegramSendMessageRequest struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

// NormalizeTelegramBotToken removes surrounding and internal whitespace.
// Telegram bot tokens never contain whitespace, but users may paste them with line breaks/spaces.
func NormalizeTelegramBotToken(botToken string) string {
	botToken = strings.TrimSpace(botToken)
	if botToken == "" {
		return ""
	}
	// Remove all whitespace (spaces, newlines, tabs, etc).
	botToken = strings.Join(strings.Fields(botToken), "")

	// Users sometimes paste the full API URL:
	//   https://api.telegram.org/bot<token>/sendMessage
	// Extract the <token> part.
	const apiPrefix = "api.telegram.org/bot"
	lower := strings.ToLower(botToken)
	if idx := strings.Index(lower, apiPrefix); idx >= 0 {
		start := idx + len(apiPrefix)
		rest := botToken[start:]
		restLower := lower[start:]
		// Cut at first '/' or '?' if present.
		cut := len(rest)
		if j := strings.IndexByte(rest, '/'); j >= 0 && j < cut {
			cut = j
		}
		if j := strings.IndexByte(restLower, '?'); j >= 0 && j < cut {
			cut = j
		}
		botToken = rest[:cut]
	}

	// Users may also paste tokens with a leading "bot" prefix. Telegram expects the raw token.
	if strings.HasPrefix(strings.ToLower(botToken), "bot") {
		botToken = botToken[3:]
	}

	return botToken
}

func IsValidTelegramBotToken(botToken string) bool {
	botToken = NormalizeTelegramBotToken(botToken)
	if botToken == "" {
		return false
	}
	parts := strings.Split(botToken, ":")
	if len(parts) != 2 {
		return false
	}
	if parts[0] == "" || parts[1] == "" {
		return false
	}
	// Bot ID (left side) is numeric.
	if _, err := strconv.ParseInt(parts[0], 10, 64); err != nil {
		return false
	}
	// Right side is typically base64url-ish: letters/digits/_/-
	for _, r := range parts[1] {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	// Most tokens are ~35 chars after ":", but keep it loose.
	return len(parts[1]) >= 10
}

func IsValidTelegramChatID(chatID string) bool {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return false
	}
	if strings.HasPrefix(chatID, "@") {
		username := strings.TrimPrefix(chatID, "@")
		// Telegram usernames are 5-32 chars, containing letters, digits, and underscores.
		if len(username) < 5 || len(username) > 32 {
			return false
		}
		for _, r := range username {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
				continue
			}
			return false
		}
		return true
	}
	if strings.HasPrefix(chatID, "-") {
		chatID = strings.TrimPrefix(chatID, "-")
	}
	if chatID == "" {
		return false
	}
	for _, r := range chatID {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func SendTelegramMessage(ctx context.Context, botToken string, chatID string, text string) error {
	botToken = NormalizeTelegramBotToken(botToken)
	chatID = strings.TrimSpace(chatID)
	if botToken == "" {
		return fmt.Errorf("telegram bot token is empty")
	}
	if chatID == "" {
		return fmt.Errorf("telegram chat_id is empty")
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("telegram message text is empty")
	}
	if !IsValidTelegramBotToken(botToken) {
		return fmt.Errorf("telegram bot token is invalid")
	}
	if !IsValidTelegramChatID(chatID) {
		return fmt.Errorf("telegram chat_id is invalid")
	}

	reqBody, err := json.Marshal(telegramSendMessageRequest{
		ChatID:                chatID,
		Text:                  text,
		DisableWebPagePreview: true,
	})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	httpClient := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("telegram sendMessage failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	// Drain body to allow connection reuse.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2048))
	return nil
}
