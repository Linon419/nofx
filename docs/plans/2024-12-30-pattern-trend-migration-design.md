# Pattern & Trend 分析模块移植设计

## 概述

将 Brale 项目的 Pattern 检测 + WaveTrend 趋势分析移植到 NOFX，作为数据预处理层，输出结构化 JSON 注入 Prompt。

## 设计原则

- **算法只输出原始数值**，不做方向性判断（超买/超卖/趋势倾向）
- **LLM 负责解读**，根据数值自行决策
- **作为数据层集成**，不改变现有单 Agent 架构

## 模块结构

```
nofx/
├── analysis/
│   ├── pattern/
│   │   └── pattern.go      # 形态检测（双底/双顶/三角/压缩）
│   ├── trend/
│   │   └── trend.go        # 结构位识别（支撑/阻力/分形）
│   ├── indicator/
│   │   └── wavetrend.go    # WT-MFI Hybrid + ALMA 平滑
│   └── analysis.go         # 统一入口，组合所有分析结果
```

## Pattern 检测

### 检测的形态

| 形态 | 检测逻辑 |
|------|----------|
| 双底 (double_bottom) | 两个低点价差 ≤0.4%，中间有反弹 |
| 双顶 (double_top) | 两个高点价差 ≤0.4%，中间有回落 |
| 三角收敛 (triangle) | 高点降低 + 低点抬高 |
| 波动压缩 (compression) | 近期波幅缩减至前期 65% 以下 |

### 输出格式

```json
{
  "pattern": {
    "detected": "double_bottom",
    "stage": "forming",
    "key_levels": {
      "neckline": 96500,
      "low1": 94800,
      "low2": 94650
    }
  }
}
```

### 实现要点

- 使用线性回归（fitLine）计算斜率
- 局部极值检测找高低点
- 返回 `"detected": "none"` 如果没有检测到明确形态

## WaveTrend 指标

### 算法组成

- **WaveTrend**：基于 HLC3 的动量震荡指标
- **MFI**：量价结合的资金流指标
- **ALMA 平滑**：Arnaud Legoux Moving Average，减少滞后
- **阶梯量化**：将连续值转为离散信号

### 输出格式

```json
{
  "wavetrend": {
    "value": 45
  }
}
```

### 阈值参考（供 LLM 解读）

- 超买区：≥ 50
- 超卖区：≤ -50
- 中性区：-50 < value < 50

## Trend 结构位识别

### 识别的结构

| 类型 | 来源 |
|------|------|
| 分形拐点 (Fractal) | 局部高低点作为结构坐标 |
| 动态支撑阻力 | EMA 21/55/100/200 |
| 布林带边界 | 上下轨作为波动区间 |
| 历史关键位 | 近期明显的高低点 |

### 输出格式

```json
{
  "trend": {
    "slope": 0.0023,
    "structure_points": [
      {"type": "fractal_high", "price": 98200, "time": "2024-12-30T10:00:00Z"},
      {"type": "fractal_low", "price": 94500, "time": "2024-12-29T14:00:00Z"}
    ],
    "key_levels": {
      "resistance": [98200, 99500],
      "support": [94500, 93000]
    }
  }
}
```

## 集成方式

### 调用时机

在 `decision/engine.go` 构建 Prompt 前执行分析：

```go
analysisResult := analysis.Analyze(klines, config)
context.Analysis = analysisResult
```

### Prompt 注入

在 User Prompt 的数据部分增加 `technical_analysis` 字段：

```markdown
## Technical Analysis
{analaysisResult JSON}
```

### Schema 更新

在 `decision/schema.go` 增加数据字典（只解释字段含义，不解释交易策略）：

```go
"wavetrend.value": "震荡指标值，范围约 -100 ~ 100"
"pattern.detected": "检测到的几何形态: double_bottom/double_top/triangle/compression/none"
"pattern.stage": "形态阶段: forming/confirmed/failed"
"trend.slope": "价格斜率，正值向上，负值向下"
"trend.key_levels": "关键支撑阻力价位"
```

## 移植来源

从 Brale 项目移植：

- `internal/analysis/pattern/pattern.go` → `analysis/pattern/pattern.go`
- `internal/analysis/indicator/indicator.go` (WT-MFI 部分) → `analysis/indicator/wavetrend.go`
- 结构位识别逻辑 → `analysis/trend/trend.go`

## 完整输出示例

```json
{
  "pattern": {
    "detected": "double_bottom",
    "stage": "forming",
    "key_levels": {
      "neckline": 96500,
      "low1": 94800,
      "low2": 94650
    }
  },
  "wavetrend": {
    "value": 45
  },
  "trend": {
    "slope": 0.0023,
    "structure_points": [
      {"type": "fractal_high", "price": 98200, "time": "2024-12-30T10:00:00Z"},
      {"type": "fractal_low", "price": 94500, "time": "2024-12-29T14:00:00Z"}
    ],
    "key_levels": {
      "resistance": [98200, 99500],
      "support": [94500, 93000]
    }
  }
}
```
