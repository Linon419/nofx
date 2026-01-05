# 识图 MCP 拆分：每个 Trader 可配置独立识图 AIModel

## 背景与目标

当前系统的“视觉读图（Vision）”与“决策（Decision）”复用同一个 `mcp.AIClient`。这会带来：

- 单个 API Key 同时承担识图与决策，额度/成本压力大
- 识图请求（带图片）与决策请求混用同一模型与上下文，可能增加决策阶段幻觉与不稳定性

目标：

- 为每个 trader 提供可选的“识图专用 AIModel”，与决策 AIModel 分离
- 当识图模型未配置/不可用时自动回退到决策模型，不阻断交易决策

## 设计概览

### 1) 数据模型

- 在 `traders` 表新增字段 `vision_ai_model_id`（默认空字符串）
  - 空：识图阶段复用决策模型（`ai_model_id`）
  - 非空：识图阶段使用该 `ai_models` 配置对应的 provider/key/url/model

### 2) 运行时路由（SplitClient）

新增 `mcp.SplitClient`，实现 `mcp.AIClient`，用于把请求按“是否包含图片”分流：

- `CallWithMessages(...)` → 始终走 **决策 client**
- `CallWithRequest(req)`：
  - 若 `req.Messages[].Parts` 中包含 `image` part，则优先走 **识图 client**
  - 若识图 client 不支持识图或未配置，则自动回退走 **决策 client**

这样可以在不大改调用链的前提下，让 `kernel/vision_mode.go` 的识图阶段自然走“识图专用 client”，而决策阶段保持不变。

### 3) 回退策略

当 `vision_ai_model_id`：

- 为空 / 找不到 / 未启用 / API Key 为空 → 回退到决策模型
- 若最终仍不支持识图 → 跳过识图阶段（继续决策，不阻断）

## 接口与前端

### API

- `POST /traders`：新增可选字段 `vision_ai_model_id`
- `PUT /traders/:id`：新增可选字段 `vision_ai_model_id`
  - 允许显式传空字符串，用于“清空并回退到决策模型”
- `GET /traders/:id/config`：返回 `vision_ai_model_id` 供前端回显

### Web UI

在 `TraderConfigModal` 增加下拉项“识图模型（可选）”：

- 默认：同决策模型（空值）
- 可选：从已启用的 AI 模型中选择一个作为识图模型

## 关键实现点

- `store.Trader` 增加 `VisionAIModelID` 字段，并在 Postgres 已存在表时用 `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` 补齐列
- `manager.TraderManager` 加载 trader 时，额外解析可选的 vision AIModel，并注入到 `trader.AutoTraderConfig`
- `trader.NewAutoTrader` 在创建决策 client 后，按配置创建识图 client，并用 `mcp.NewSplitClient(decision, vision)` 包装
- `kernel/vision_mode.go` 的 `clientSupportsVision()` 增加对 `*mcp.SplitClient` 的识别与“任一侧支持即可”

## 错误处理与可观测性

- 若识图模型配置不合法或缺 key：日志提示并回退
- 若识图请求失败：沿用现有逻辑（该 symbol 的 vision note 失败会被跳过，不影响决策）

## 测试

- `mcp/split_client_test.go`：
  - `CallWithMessages` 始终走决策 client
  - `CallWithRequest`（包含 image parts）优先走识图 client
  - 识图 client 不支持识图时，带图片请求回退到决策 client

