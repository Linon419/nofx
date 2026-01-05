# 决策拆分：分析层 + 决策层（对齐 Brale）

## 背景与目标

当前 NOFX 的 AI 决策为单次调用：同一个模型既要“分析”又要输出最终 JSON 决策，容易出现：

- 输出格式不稳定（缺 JSON、混杂解释）
- 过度推理导致幻觉、风控解释不一致
- 想用“便宜模型做分析 + 贵模型做决策”但缺少配置入口

目标：

- 将决策流程拆为 **分析层 → 决策层** 两次调用
- 分析层输出 **纯文本要点**（不输出任何 JSON 决策）
- 决策层继续输出 **最终 JSON 决策数组**（保持现有解析/工具调用机制）
- 每个 Trader 可配置独立的分析模型（可与决策模型相同或不同）

## 设计概览

### 1) 数据模型

在 `traders` 表新增字段：

- `analysis_ai_model_id`（默认空字符串）
  - 空：分析层复用决策模型（`ai_model_id`）
  - 非空：分析层使用该 `ai_models` 配置对应的 provider/key/url/model

### 2) Trader 运行时（两模型）

- 决策模型：维持现有逻辑（含工具调用、JSON 修复重试等）
- 分析模型：
  - 使用独立的 `mcp.AIClient`（若未配置则回退为决策 client）
  - 用独立 system prompt 约束输出为“文本要点”，禁止任何 JSON/指令

### 3) Kernel 调用链（Brale 风格）

在 `kernel.GetFullDecisionWithStrategyWithAnalysis(...)` 内：

1. 构建 system prompt（决策层）与 user prompt（市场数据 + 可选识图 notes）
2. 调用分析层：
   - 输入：分析层 system prompt + user prompt（同一份市场数据）
   - 输出：文本要点
3. 将分析要点注入决策层 user prompt（新增 `## AI Analysis Notes` 区块）
4. 决策层按既有流程输出最终决策：
   - 优先工具调用（`submit_decisions`）
   - 失败则回退文本解析 + JSON 修复重试

### 4) API / Web UI

- `POST /traders`、`PUT /traders/:id` 支持 `analysis_ai_model_id`
- `GET /traders/:id/config` 返回 `analysis_ai_model_id`
- Web Trader 配置弹窗新增“分析模型（可选）”下拉框

## 风险与取舍

- 该方案每个周期会新增一次 LLM 调用（分析层），成本会增加
- 通过更强的“输出分层约束”，通常能换取更稳定的最终 JSON 与更低幻觉率

