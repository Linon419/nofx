package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"nofx/config"
	"nofx/crypto"
	"nofx/decision"
	"nofx/logger"
	"nofx/mcp"
)

type ModelProbeRequest struct {
	Provider        string `json:"provider"`
	APIKey          string `json:"apiKey"`
	CustomAPIURL    string `json:"customApiUrl"`
	CustomModelName string `json:"customModelName"`
}

type ModelTestResponse struct {
	Provider    string              `json:"provider"`
	Model       string              `json:"model"`
	BaseURL     string              `json:"baseUrl"`
	LatencyMs   int64               `json:"latency_ms"`
	Reasoning   string              `json:"reasoning,omitempty"`
	Decisions   []decision.Decision `json:"decisions"`
	RawToolArgs string              `json:"raw_tool_args"`
}

func (s *Server) handleListRemoteModels(c *gin.Context) {
	var req ModelProbeRequest
	if err := s.bindEncryptedOrPlain(c, &req); err != nil {
		return
	}

	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	apiKey := strings.TrimSpace(req.APIKey)
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is required"})
		return
	}
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "apiKey is required"})
		return
	}

	aiClient := newAIClientForProvider(provider)
	aiClient.SetAPIKey(apiKey, strings.TrimSpace(req.CustomAPIURL), strings.TrimSpace(req.CustomModelName))

	lister, ok := aiClient.(interface {
		ListModels(ctx context.Context) ([]mcp.ModelListItem, error)
	})
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI client does not support model listing"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	models, err := lister.ListModels(ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to fetch models: %v", err)})
		return
	}

	c.JSON(http.StatusOK, models)
}

func (s *Server) handleTestModelConfigToolCall(c *gin.Context) {
	var req ModelProbeRequest
	if err := s.bindEncryptedOrPlain(c, &req); err != nil {
		return
	}

	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	apiKey := strings.TrimSpace(req.APIKey)
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is required"})
		return
	}
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "apiKey is required"})
		return
	}

	aiClient := newAIClientForProvider(provider)
	aiClient.SetAPIKey(apiKey, strings.TrimSpace(req.CustomAPIURL), strings.TrimSpace(req.CustomModelName))

	toolParams := decision.BuildDecisionToolParameters("")
	systemPrompt := "You are a trading decision engine. You must call the function tool and return no free-form text."
	userPrompt := "Create exactly 1 decision: {symbol: BTCUSDT, action: wait, reasoning: \"test\"}."

	mcpReq := mcp.NewRequestBuilder().
		WithSystemPrompt(systemPrompt).
		WithUserPrompt(userPrompt).
		AddFunction("submit_decisions", "Submit trading decisions as structured JSON (no free-form text).", toolParams).
		WithToolChoice("required").
		WithMaxTokens(800).
		MustBuild()

	start := time.Now()
	rawArgs, err := aiClient.CallWithRequest(mcpReq)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("AI API call failed: %v", err)})
		return
	}

	reasoning, decisionsOut, err := decision.ParseDecisionToolArguments(rawArgs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to parse tool args: %v", err), "raw_tool_args": rawArgs})
		return
	}

	baseURL := ""
	modelName := ""
	// Best-effort: read BaseURL/Model from the embedded client when possible
	switch cc := aiClient.(type) {
	case *mcp.OpenAIClient:
		baseURL = cc.BaseURL
		modelName = cc.Model
	case *mcp.ClaudeClient:
		baseURL = cc.BaseURL
		modelName = cc.Model
	case *mcp.DeepSeekClient:
		baseURL = cc.BaseURL
		modelName = cc.Model
	case *mcp.QwenClient:
		baseURL = cc.BaseURL
		modelName = cc.Model
	case *mcp.GeminiClient:
		baseURL = cc.BaseURL
		modelName = cc.Model
	case *mcp.GrokClient:
		baseURL = cc.BaseURL
		modelName = cc.Model
	case *mcp.KimiClient:
		baseURL = cc.BaseURL
		modelName = cc.Model
	case *mcp.Client:
		baseURL = cc.BaseURL
		modelName = cc.Model
	}

	c.JSON(http.StatusOK, ModelTestResponse{
		Provider:    provider,
		Model:       modelName,
		BaseURL:     baseURL,
		LatencyMs:   latency,
		Reasoning:   reasoning,
		Decisions:   decisionsOut,
		RawToolArgs: rawArgs,
	})
}

func newAIClientForProvider(provider string) mcp.AIClient {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "qwen":
		return mcp.NewQwenClientWithOptions()
	case "deepseek":
		return mcp.NewDeepSeekClientWithOptions()
	case "claude":
		return mcp.NewClaudeClientWithOptions()
	case "kimi":
		return mcp.NewKimiClientWithOptions()
	case "gemini":
		return mcp.NewGeminiClientWithOptions()
	case "grok":
		return mcp.NewGrokClientWithOptions()
	case "openai":
		return mcp.NewOpenAIClientWithOptions()
	default:
		return mcp.NewClient()
	}
}

func (s *Server) bindEncryptedOrPlain(c *gin.Context, out any) error {
	cfg := config.Get()

	bodyBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return err
	}

	if !cfg.TransportEncryption {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return err
		}
		return nil
	}

	var encryptedPayload crypto.EncryptedPayload
	if err := json.Unmarshal(bodyBytes, &encryptedPayload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format, encrypted transmission required"})
		return err
	}
	if encryptedPayload.WrappedKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "This endpoint only supports encrypted transmission, please use encrypted client",
			"code":    "ENCRYPTION_REQUIRED",
			"message": "Encrypted transmission is required for security reasons",
		})
		return fmt.Errorf("encryption required")
	}

	decrypted, err := s.cryptoHandler.cryptoService.DecryptSensitiveData(&encryptedPayload)
	if err != nil {
		logger.Infof("❌ Failed to decrypt request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decrypt data"})
		return err
	}

	if err := json.Unmarshal([]byte(decrypted), out); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse decrypted data"})
		return err
	}

	return nil
}
