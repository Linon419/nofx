package mcp

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// FailoverClient tries multiple underlying AI clients in order.
//
// It is designed to keep trading cycles alive when the primary AI endpoint is flaky
// (e.g. 5xx, timeouts, empty responses). Non-transient errors (e.g. 401/403/invalid key)
// are returned immediately without trying fallbacks.
type FailoverClient struct {
	clients []AIClient
	logger  Logger

	// If a client fails with a transient error, it will be skipped for this duration.
	cooldown time.Duration

	mu            sync.Mutex
	disabledUntil []time.Time
}

func NewFailoverClient(primary AIClient, fallbacks ...AIClient) *FailoverClient {
	all := make([]AIClient, 0, 1+len(fallbacks))
	if primary != nil {
		all = append(all, primary)
	}
	for _, c := range fallbacks {
		if c != nil {
			all = append(all, c)
		}
	}
	fc := &FailoverClient{
		clients:       all,
		cooldown:      10 * time.Minute,
		disabledUntil: make([]time.Time, len(all)),
	}
	fc.logger = fc.inferLogger()
	return fc
}

func (c *FailoverClient) SetAPIKey(apiKey string, customURL string, customModel string) {
	for _, cli := range c.clients {
		if cli != nil {
			cli.SetAPIKey(apiKey, customURL, customModel)
		}
	}
}

func (c *FailoverClient) SetTimeout(timeout time.Duration) {
	for _, cli := range c.clients {
		if cli != nil {
			cli.SetTimeout(timeout)
		}
	}
}

func (c *FailoverClient) CallWithMessages(systemPrompt, userPrompt string) (string, error) {
	if len(c.clients) == 0 {
		return "", fmt.Errorf("no underlying clients configured")
	}

	indexes := c.pickCandidateIndexes()
	var lastErr error

	for _, idx := range indexes {
		cli := c.clients[idx]
		if cli == nil {
			continue
		}

		resp, err := cli.CallWithMessages(systemPrompt, userPrompt)
		if err == nil {
			if idx != 0 {
				c.logger.Warnf("⚠️ [MCP] Failover used backup client %s after primary failure", describeAIClient(cli))
			}
			return resp, nil
		}

		lastErr = err
		if !isTransientAIError(err) {
			return "", err
		}

		c.disableClient(idx, err)
		if idx == 0 && len(indexes) > 1 {
			c.logger.Warnf("⚠️ [MCP] Primary client failed (%v); trying fallbacks...", err)
		}
	}

	return "", lastErr
}

func (c *FailoverClient) CallWithRequest(req *Request) (string, error) {
	if len(c.clients) == 0 {
		return "", fmt.Errorf("no underlying clients configured")
	}

	indexes := c.pickCandidateIndexes()
	var lastErr error

	for _, idx := range indexes {
		cli := c.clients[idx]
		if cli == nil {
			continue
		}

		resp, err := cli.CallWithRequest(req)
		if err == nil {
			if idx != 0 {
				c.logger.Warnf("⚠️ [MCP] Failover used backup client %s after primary failure", describeAIClient(cli))
			}
			return resp, nil
		}

		lastErr = err
		if !isTransientAIError(err) {
			return "", err
		}

		c.disableClient(idx, err)
		if idx == 0 && len(indexes) > 1 {
			c.logger.Warnf("⚠️ [MCP] Primary client failed (%v); trying fallbacks...", err)
		}
	}

	return "", lastErr
}

func (c *FailoverClient) pickCandidateIndexes() []int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	indexes := make([]int, 0, len(c.clients))
	for i := range c.clients {
		if c.disabledUntil[i].IsZero() || !now.Before(c.disabledUntil[i]) {
			indexes = append(indexes, i)
		}
	}
	if len(indexes) > 0 {
		return indexes
	}
	// All are in cooldown: allow one probe cycle to avoid deadlock.
	for i := range c.clients {
		indexes = append(indexes, i)
	}
	return indexes
}

func (c *FailoverClient) disableClient(idx int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if idx < 0 || idx >= len(c.disabledUntil) {
		return
	}
	c.disabledUntil[idx] = time.Now().Add(c.cooldown)
	c.logger.Warnf("⚠️ [MCP] Temporarily disabling client %s for %v due to: %v", describeAIClient(c.clients[idx]), c.cooldown, err)
}

func (c *FailoverClient) inferLogger() Logger {
	for _, cli := range c.clients {
		if cli == nil {
			continue
		}
		// Use clientInfo interface if available
		if ci, ok := cli.(clientInfo); ok {
			if lg := ci.getLogger(); lg != nil {
				return lg
			}
		}
		// Handle SplitClient specially (recursive)
		if sc, ok := cli.(*SplitClient); ok {
			if lg := inferLoggerFromAny(sc.DecisionClient()); lg != nil {
				return lg
			}
			if lg := inferLoggerFromAny(sc.VisionClient()); lg != nil {
				return lg
			}
		}
	}
	return NewNoopLogger()
}

func inferLoggerFromAny(cli AIClient) Logger {
	if cli == nil {
		return nil
	}
	if ci, ok := cli.(clientInfo); ok {
		return ci.getLogger()
	}
	return nil
}

func describeAIClient(cli AIClient) string {
	if cli == nil {
		return "<nil>"
	}
	// Use clientInfo interface if available
	if ci, ok := cli.(clientInfo); ok {
		provider, model, baseURL := ci.getClientInfo()
		return fmt.Sprintf("[Provider: %s, Model: %s, BaseURL: %s]", provider, model, baseURL)
	}
	// Handle SplitClient specially
	if sc, ok := cli.(*SplitClient); ok {
		return fmt.Sprintf("[SplitClient decision=%s vision=%s]", describeAIClient(sc.DecisionClient()), describeAIClient(sc.VisionClient()))
	}
	// Fallback to String() or type name
	if s, ok := cli.(interface{ String() string }); ok {
		return s.String()
	}
	return fmt.Sprintf("%T", cli)
}

func isTransientAIError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()

	// Common wrapper from Client.CallWithMessages/CallWithRequest:
	// "still failed after N retries: API returned error (status 500): ..."
	if code, ok := extractHTTPStatusCode(s); ok {
		if code == http.StatusTooManyRequests || code == http.StatusRequestTimeout || (code >= 500 && code <= 599) {
			return true
		}
	}

	// Some proxies return structurally-valid JSON without choices.
	if strings.Contains(s, "API returned empty response") {
		return true
	}

	// Transport errors (already retryable inside a single client, but may still bubble up).
	transportNeedles := []string{
		"EOF",
		"timeout",
		"context deadline exceeded",
		"connection reset",
		"connection refused",
		"temporary failure",
		"no such host",
		"stream error",
		"INTERNAL_ERROR",
	}
	for _, needle := range transportNeedles {
		if strings.Contains(s, needle) {
			return true
		}
	}

	return false
}

var _ AIClient = (*FailoverClient)(nil)

