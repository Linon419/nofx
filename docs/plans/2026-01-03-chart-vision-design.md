# Chart Vision (K线/指标出图 + 读图决策) - Design
Date: 2026-01-03
Branch: feature/chart-vision
Status: Draft

## Goal
Generate candlestick charts with key indicators (EMA/WaveTrend/divergence/structure/pattern) on the backend, and attach those images to the AI decision request so the model can "read the chart" during decision making.

Targets:
- Support OpenAI-compatible vision (OpenAI + Gemini(OpenAI-compatible endpoint))
- Support Anthropic Claude vision (/messages API)
- Backward compatible: text-only models continue to work (skip images)

Non-goals (v1):
- Frontend-driven screenshot pipeline
- Storing images permanently or showing charts in UI (optional later)

## High-level Flow
1) Trader builds context (klines, EMA, TA JSON).
2) Backend renders 1-3 PNG chart(s) per symbol (per selected timeframe), including:
   - Candles
   - EMA21/55/100/200 overlays
   - Optional overlays: structure points (fractal highs/lows), pattern labels, divergence markers
3) StrategyEngine composes the normal system/user prompts + attaches chart images as multimodal content.
4) mcp client sends multimodal request when provider supports it; otherwise sends text-only request.
5) Parse tool-call decision JSON exactly as today.

## Chart Rendering Spec (Backend)
- Renderer: Go PNG generation (recommended: `go-chart` or `gonum/plot`), deterministic output.
- Default image size: 1024x640 (configurable), light background for readability.
- Minimum bars: 200 (or available), last bar aligned to current time.
- Overlays:
  - EMA lines: color-coded, legend in top-left.
  - WaveTrend: optional separate subpanel (v1 may omit to reduce complexity).
  - Divergence: mark latest divergence events (from `analysis.AnalysisResult.Divergence.Events`) as arrows/labels.
  - Pattern: annotate detected pattern + key levels (neckline, highs/lows).
  - Structure: mark fractal highs/lows + key support/resistance bands.

Performance controls:
- Only render for:
  - Current open positions (always), and
  - Top N candidates (default N=3), and
  - Only selected timeframes (default: 15m + 1h; optionally 4h).
- Cache by (symbol, timeframe, last_close_time, indicator config hash) with small in-memory LRU.

## Multimodal Message Model (Internal)
Current `mcp.Message` only supports `Content string`. We introduce a backward-compatible content union:
- Keep `Content string` for text-only.
- Add `Parts []ContentPart` (optional) for multimodal:
  - `type=text, text=...`
  - `type=image, media_type=image/png, data_base64=...`

RequestBuilder will accept either:
- `WithUserPrompt(text)` (text-only), or
- `WithUserParts(parts)` (multimodal), and similar for system.

## Provider Mapping
### OpenAI / OpenAI-Compatible (Gemini via /openai)
Send `messages[].content` as an array:
- `{type:"text", text:"..."}`
- `{type:"image_url", image_url:{url:"data:image/png;base64,..."}}`

### Claude (/v1/messages)
Send `messages[].content` as an array:
- `{type:"text", text:"..."}`
- `{type:"image", source:{type:"base64", media_type:"image/png", data:"..."}}`
System prompt remains top-level `system`.

If provider/model does not support images:
- Do NOT attach images
- Add a short notice: "Charts not attached for this provider/model."

## Prompt Contract (Model Instructions)
Add a short system section when images exist:
- "You must treat the attached charts as the primary source of truth for price/EMA/structure; the textual TA JSON is a supplement."
- "If chart content and text conflict, prefer chart."
- "Do not hallucinate chart features; only describe what is visible."

## Configuration
Add strategy config knobs (backend + UI later):
- `vision.enabled` (default false)
- `vision.max_symbols` (default 3 + open positions)
- `vision.timeframes` (default ["15m","1h"])
- `vision.image_width/height`
- `vision.attach_mode`: `none | candidates | positions | both`

## Security / Safety
- Images are derived only from market data already in prompt; no external fetch.
- Limit total payload size (e.g., <= 6 images, each <= 300KB).

## Testing Plan
- Unit: chart render returns non-empty PNG; deterministic size.
- Unit: request serialization for OpenAI and Claude includes correct image parts.
- Integration (manual): run one cycle with vision enabled and confirm AI sees image (e.g., model describes visible EMA/candle).

