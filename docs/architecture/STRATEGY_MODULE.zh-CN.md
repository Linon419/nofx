# NOFX 绛栫暐妯″潡鎶€鏈枃妗?

**璇█:** [English](STRATEGY_MODULE.md) | [涓枃](STRATEGY_MODULE.zh-CN.md)

## 姒傝堪

鏈枃妗ｈ缁嗘弿杩?NOFX 绛栫暐妯″潡鐨勫畬鏁存暟鎹祦绋嬶紝鍖呮嫭甯佺閫夋嫨銆佹暟鎹粍瑁呫€佹彁绀鸿瘝鏋勫缓銆丄I 璇锋眰銆佸搷搴旇В鏋愬拰鍐崇瓥鎵ц銆?

---

## 瀹屾暣鏁版嵁娴佺▼鍥?

```
鈹屸攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹?
鈹?                   浜ゆ槗鍛ㄦ湡 (姣?N 鍒嗛挓)                          鈹?
鈹斺攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹?

1. 甯佺閫夋嫨 (GetCandidateCoins)
   鈹溾攢 Static (闈欐€佸垪琛?
   鈹溾攢 AI500 Pool (AI璇勫垎姹?

   - OTC Top (OTC index ranking)
   鈹斺攢 Mixed (娣峰悎妯″紡)
        鈫?
2. 鏁版嵁缁勮 (buildTradingContext)
   鈹溾攢 璐︽埛浣欓 鈫?equity, available, unrealizedPnL
   鈹溾攢 褰撳墠鎸佷粨 鈫?symbol, side, entry, mark, qty, leverage
   鈹溾攢 K绾挎暟鎹?鈫?OHLCV (5m, 15m, 1h, 4h)
   鈹溾攢 鎶€鏈寚鏍?鈫?EMA, MACD, RSI, ATR, Volume
   鈹溾攢 閾句笂鏁版嵁 鈫?OI, Funding Rate
   鈹溾攢 閲忓寲鏁版嵁 鈫?璧勯噾娴佸悜, OI鍙樺寲 (鍙€?
   鈹斺攢 鏈€杩戜氦鏄?鈫?鏈€杩?0绗斿凡骞充粨
        鈫?
3. 绯荤粺鎻愮ず璇?(BuildSystemPrompt)
   鈹溾攢 瑙掕壊瀹氫箟
   鈹溾攢 浜ゆ槗妯″紡 (aggressive/conservative/scalping)
   鈹溾攢 纭€х害鏉?(浠ｇ爜寮哄埗鎵ц)
   鈹溾攢 AI寮曞 (寤鸿鍊?
   鈹溾攢 浜ゆ槗棰戠巼
   鈹溾攢 鍏ュ満鏍囧噯
   鈹溾攢 鍐崇瓥娴佺▼
   鈹斺攢 杈撳嚭鏍煎紡 (XML + JSON)
        鈫?
4. 鐢ㄦ埛鎻愮ず璇?(BuildUserPrompt)
   鈹溾攢 绯荤粺鐘舵€?(鏃堕棿, 鍛ㄦ湡鍙?
   鈹溾攢 BTC甯傚満姒傝
   鈹溾攢 璐︽埛淇℃伅
   鈹溾攢 褰撳墠鎸佷粨 (鍚妧鏈寚鏍?
   鈹溾攢 鍊欓€夊竵绉?(瀹屾暣甯傚満鏁版嵁)
   鈹斺攢 "璇峰垎鏋愬苟杈撳嚭鍐崇瓥..."
        鈫?
5. AI璇锋眰 (CallWithMessages)
   鈹溾攢 閫夋嫨AI妯″瀷
   鈹溾攢 POST: system_prompt + user_prompt
   鈹溾攢 瓒呮椂: 120绉? 閲嶈瘯: 3娆?
   鈹斺攢 杩斿洖鍘熷鍝嶅簲
        鈫?
6. AI瑙ｆ瀽 (parseFullDecisionResponse)
   鈹溾攢 鎻愬彇鎬濈淮閾?<reasoning>
   鈹溾攢 鎻愬彇JSON鍐崇瓥 <decision>
   鈹溾攢 淇瀛楃缂栫爜
   鈹溾攢 楠岃瘉JSON鏍煎紡
   鈹溾攢 瑙ｆ瀽鍐崇瓥鏁扮粍
   鈹斺攢 楠岃瘉椋庢帶鍙傛暟
        鈫?
7. 鍐崇瓥鎵ц
   鈹溾攢 鎺掑簭: 骞充粨浼樺厛 鈫?寮€浠?鈫?hold/wait
   鈹溾攢 椋庢帶寮哄埗鎵ц
   鈹溾攢 鎻愪氦璁㈠崟
   鈹溾攢 纭鎴愪氦
   鈹斺攢 璁板綍鍒版暟鎹簱
```

---

## 1. 甯佺閫夋嫨 (Coin Selection)

**鏍稿績鏂囦欢:** `decision/engine.go:380-454`

**鍏ュ彛鏂规硶:** `StrategyEngine.GetCandidateCoins()`

### 1.1 闈欐€佸竵绉嶅垪琛?(Static)

```go
// decision/engine.go:395-403
if config.CoinSource.SourceType == "static" {
    for _, symbol := range config.CoinSource.StaticCoins {
        coins = append(coins, CandidateCoin{
            Symbol:  market.Normalize(symbol),
            Sources: []string{"static"},
        })
    }
}
```

- **閰嶇疆:** `StrategyConfig.CoinSource.StaticCoins`
- **鐢ㄩ€?** 鎵嬪姩鎸囧畾浜ゆ槗甯佺
- **鏍囩:** `["static"]`

### 1.2 AI500 甯佺姹?(CoinPool)

```go
// decision/engine.go:405-406, 456-474
func (e *StrategyEngine) getCoinPoolCoins(limit int) []CandidateCoin {
    coins, err := e.provider.GetTopRatedCoins(limit)
    // ...
    for _, coin := range coins {
        result = append(result, CandidateCoin{
            Symbol:  coin.Symbol,
            Sources: []string{"ai500"},
        })
    }
}
```

- **API:** `config.CoinSource.CoinPoolAPIURL` (榛樿: `http://nofxaios.com:30006/api/ai500/list`)
- **鐢ㄩ€?** 鑾峰彇 AI 璇勫垎鏈€楂樼殑 N 涓竵绉?
- **鏍囩:** `["ai500"]`


   - OTC Top (OTC index ranking)

```go
// decision/engine.go:408-409, 476-498
func (e *StrategyEngine) getOITopCoins() []CandidateCoin {
    positions, err := e.provider.GetOITopPositions()
    // ...
    for _, pos := range positions {
        result = append(result, CandidateCoin{
            Symbol:  pos.Symbol,
            Sources: []string{"oi_top"},
        })
    }
}
```

- **API:** `config.CoinSource.OITopAPIURL`
- **鐢ㄩ€?** 鑾峰彇鎸佷粨閲忓闀挎渶蹇殑甯佺
- **鏍囩:** `["oi_top"]`

### 1.4 OTC Top (OTC index ranking)

```go
// decision/engine.go:501-520
func (e *StrategyEngine) getOTCTopCoins() ([]CandidateCoin, error) {
    symbols, err := e.provider.GetOTCTopSymbols()
    // ...
    for _, symbol := range symbols {
        result = append(result, CandidateCoin{
            Symbol:  symbol,
            Sources: []string{"otc_top"},
        })
    }
}
```

- **API:** `config.CoinSource.OTCTopAPIURL`
- **Usage:** Get coins with highest OTC index
- **Tag:** `["otc_top"]`

### 1.5 Mixed

```go
// decision/engine.go:411-449
if config.CoinSource.SourceType == "mixed" {
    if config.CoinSource.UseCoinPool {
        // Add AI500 coins
    }
    if config.CoinSource.UseOITop {
        // Add OI Top coins
    }
    if config.CoinSource.UseOTCTop {
        // Add OTC Top coins
    }
    if len(config.CoinSource.StaticCoins) > 0 {
        // Add static coins
    }
    // Deduplicate and merge, keep multi-source tags
}
```

- **Features:** Use multiple data sources
- **Tag example:** `["ai500", "oi_top", "otc_top"]` (multi-source coins)

---
## 2. 鏁版嵁缁勮 (Data Assembly)

**鏍稿績鏂囦欢:** `trader/auto_trader.go:562-791`, `decision/engine.go:299-374`

**鍏ュ彛鏂规硶:** `AutoTrader.buildTradingContext()`

### 2.1 璐︽埛鏁版嵁

```go
// trader/auto_trader.go:565-583
balance, err := at.trader.GetBalance()
equity := balance["total_equity"].(float64)
available := balance["available_balance"].(float64)
unrealizedPnL := balance["total_pnl"].(float64)
```

**鎻愬彇瀛楁:**
- `total_equity` - 璐︽埛鎬绘潈鐩?
- `available_balance` - 鍙敤浣欓
- `total_pnl` - 鏈疄鐜扮泩浜?

### 2.2 鎸佷粨鏁版嵁

```go
// trader/auto_trader.go:588-682
positions, err := at.trader.GetPositions()
for _, pos := range positions {
    position := decision.Position{
        Symbol:           pos.Symbol,
        Side:             pos.Side,          // "long" / "short"
        EntryPrice:       pos.EntryPrice,
        MarkPrice:        pos.MarkPrice,
        Quantity:         pos.Quantity,
        Leverage:         pos.Leverage,
        UnrealizedPnL:    pos.UnrealizedPnL,
        LiquidationPrice: pos.LiquidationPrice,
    }
}
```

### 2.3 甯傚満鏁版嵁鑾峰彇

```go
// decision/engine.go:299-374
func (e *StrategyEngine) fetchMarketDataWithStrategy(symbols []string) map[string]*market.Data {
    timeframes := config.Indicators.Klines.SelectedTimeframes  // ["5m", "15m", "1h", "4h"]
    primaryTF := config.Indicators.Klines.PrimaryTimeframe     // "5m"
    count := config.Indicators.Klines.PrimaryCount             // 30

    for _, symbol := range symbols {
        data := market.GetWithTimeframes(symbol, timeframes, primaryTF, count)
        result[symbol] = data
    }
}
```

### 2.4 鎶€鏈寚鏍囪绠?

**鏂囦欢:** `market/data.go:59-98`

| 鎸囨爣 | 閰嶇疆 | 璁＄畻鏂规硶 |
|------|------|----------|
| **EMA** | `EnableEMA`, `EMAPeriods` | `calculateEMA(klines, period)` |
| **MACD** | `EnableMACD` | `calculateMACD(klines)` - 12/26/9 |
| **RSI** | `EnableRSI`, `RSIPeriods` | `calculateRSI(klines, period)` |
| **ATR** | `EnableATR`, `ATRPeriods` | `calculateATR(klines, period)` |
| **Volume** | `EnableVolume` | 鍘熷鎴愪氦閲忔暟鎹?|
| **OI** | `EnableOI` | 鎸佷粨閲忔暟鎹?|
| **Funding Rate** | `EnableFundingRate` | 璧勯噾璐圭巼 |

### 2.5 閲忓寲鏁版嵁 (鍙€?

```go
// trader/auto_trader.go:759-778
if config.Indicators.EnableQuantData {
    quantData := provider.GetQuantData(symbol)
    // 鍖呭惈: 璧勯噾娴佸悜銆丱I鍙樺寲銆佷环鏍煎彉鍖?
}
```

**鏁版嵁缁撴瀯:**
```go
QuantData {
    Netflow {
        Institution: {Future, Spot},  // 鏈烘瀯璧勯噾娴?
        Personal: {Future, Spot}      // 鏁ｆ埛璧勯噾娴?
    },
    OI {
        CurrentOI: float64,
        Delta: {1h, 4h, 24h}          // OI鍙樺寲
    },
    PriceChange {
        "1h", "4h", "24h": float64    // 浠锋牸鍙樺寲鐧惧垎姣?
    }
}
```

---

## 3. 绯荤粺鎻愮ず璇?(System Prompt)

**鏍稿績鏂囦欢:** `decision/engine.go:700-818`

**鍏ュ彛鏂规硶:** `StrategyEngine.BuildSystemPrompt(accountEquity, variant)`

### 3.1 鎻愮ず璇嶇粨鏋?(8涓儴鍒?

```
1. 瑙掕壊瀹氫箟          [鍙紪杈慮
2. 浜ゆ槗妯″紡鍙樹綋      [杩愯鏃剁‘瀹歖
3. 纭€х害鏉?         [浠ｇ爜寮哄埗 + AI寮曞]
4. 浜ゆ槗棰戠巼          [鍙紪杈慮
5. 鍏ュ満鏍囧噯          [鍙紪杈慮
6. 鍐崇瓥娴佺▼          [鍙紪杈慮
7. 杈撳嚭鏍煎紡          [鍥哄畾XML + JSON缁撴瀯]
8. 鑷畾涔夋彁绀鸿瘝      [鍙€塢
```

### 3.2 瑙掕壊瀹氫箟

```go
// decision/engine.go:706-713
roleDefinition := config.PromptSections.RoleDefinition
if roleDefinition == "" {
    roleDefinition = "You are a professional cryptocurrency trading AI..."
}
```

### 3.3 浜ゆ槗妯″紡鍙樹綋

| 妯″紡 | 鐗圭偣 |
|------|------|
| `aggressive` | 瓒嬪娍绐佺牬锛岃緝楂樹粨浣嶅蹇嶅害 |
| `conservative` | 澶氫俊鍙风‘璁わ紝淇濆畧璧勯噾绠＄悊 |
| `scalping` | 鐭嚎鍔ㄩ噺锛岀揣姝㈢泩 |

### 3.4 纭€х害鏉?

**浠ｇ爜寮哄埗鎵ц (CODE ENFORCED):**

```go
// decision/engine.go:725-749
maxPositions := config.RiskControl.MaxPositions           // 榛樿: 3
altcoinMaxRatio := config.RiskControl.AltcoinMaxPositionValueRatio  // 榛樿: 1.0
btcethMaxRatio := config.RiskControl.BTCETHMaxPositionValueRatio    // 榛樿: 5.0
maxMarginUsage := config.RiskControl.MaxMarginUsage       // 榛樿: 90%
minPositionSize := config.RiskControl.MinPositionSize     // 榛樿: 12 USDT
```

**AI寮曞 (寤鸿鍊?:**

```go
altcoinMaxLeverage := config.RiskControl.AltcoinMaxLeverage  // 榛樿: 5x
btcethMaxLeverage := config.RiskControl.BTCETHMaxLeverage    // 榛樿: 5x
minRiskRewardRatio := config.RiskControl.MinRiskRewardRatio  // 榛樿: 1:3
minConfidence := config.RiskControl.MinConfidence            // 榛樿: 75
```

### 3.5 杈撳嚭鏍煎紡瑕佹眰

```xml
<reasoning>
[鎬濈淮閾惧垎鏋愯繃绋媇
</reasoning>

<decision>
```json
[
  {
    "symbol": "BTCUSDT",
    "action": "open_long",
    "leverage": 5,
    "position_size_usd": 100.00,
    "stop_loss": 65000.00,
    "take_profit": 72000.00,
    "confidence": 85,
    "risk_usd": 20.00,
    "reasoning": "..."
  }
]
```
</decision>
```

---

## 4. 鐢ㄦ埛鎻愮ず璇?(User Prompt)

**鏍稿績鏂囦欢:** `decision/engine.go:884-1007`

**鍏ュ彛鏂规硶:** `StrategyEngine.BuildUserPrompt(ctx)`

### 4.1 鎻愮ず璇嶅唴瀹圭粨鏋?

```
1. 绯荤粺鐘舵€?         [鏃堕棿, 鍛ㄦ湡鍙? 杩愯鏃堕暱]
2. BTC甯傚満姒傝      [浠锋牸, 娑ㄨ穼骞? MACD, RSI]
3. 璐︽埛淇℃伅          [鏉冪泭, 浣欓%, 鐩堜簭%, 淇濊瘉閲?, 鎸佷粨鏁癩
4. 鏈€杩戞垚浜?         [鏈€杩?3绗斿凡骞充粨浜ゆ槗]
5. 褰撳墠鎸佷粨          [璇︾粏鎸佷粨鏁版嵁 + 鎶€鏈寚鏍嘳
6. 鍊欓€夊竵绉?         [瀹屾暣甯傚満鏁版嵁]
7. 閲忓寲鏁版嵁          [璧勯噾娴佸悜, OI鏁版嵁] (鍙€?
8. OI鎺掕鏁版嵁        [甯傚満OI鍙樺寲鎺掕] (鍙€?
```

### 4.2 璐︽埛淇℃伅鏍煎紡

```
Account: Equity 1000.00 | Balance 800.00 (80.0%) | PnL +5.5% | Margin 20.0% | Positions 2
```

### 4.3 鎸佷粨淇℃伅鏍煎紡

```
1. BTCUSDT LONG | Entry 68000.0000 Current 69500.0000
   Qty 0.0100 | Position Value $695.00
   PnL +2.21% | Amount +$15.00
   Peak PnL +3.50% | Leverage 5x
   Margin $139.00 | Liquidation Price 55000.0000
   Holding Duration 2 hours 30 minutes

   Market: price=69500, ema21=68800, ema55=68000, ema100=67000, ema200=66000, macd=150.5, rsi7=62.3
   OI: Latest=15000000, Avg=14500000
   Funding Rate: 0.0100%
```

### 4.4 鍊欓€夊竵绉嶆牸寮?

```
### 1. ETHUSDT (AI500+OI_Top dual signal)

current_price = 3500.00, current_ema21 = 3450.00, current_ema55 = 3400.00, current_ema100 = 3300.00, current_ema200 = 3200.00, current_macd = 25.5, current_rsi7 = 58.0

Open Interest: Latest: 8500000.00 Average: 8200000.00
Funding Rate: 0.0050

=== 5M TIMEFRAME (oldest 鈫?latest) ===
Prices: [3480, 3485, 3490, 3495, 3500]
Volumes: [1000, 1200, 1100, 1300, 1150]
EMA21: [3470, 3475, 3478, 3482, 3485]
EMA55: [3460, 3465, 3468, 3472, 3475]
EMA100: [3430, 3435, 3438, 3440, 3442]
EMA200: [3380, 3385, 3390, 3395, 3400]
MACD: [20.1, 21.5, 22.8, 24.0, 25.5]
RSI7: [55.0, 56.2, 57.1, 57.8, 58.0]

=== 15M TIMEFRAME ===
...
```

---

## 5. AI璇锋眰 (AI Request)

**鏍稿績鏂囦欢:** `decision/engine.go:222-293`, `mcp/client.go:136-150`

### 5.1 璇锋眰娴佺▼

```go
// decision/engine.go:263-268
aiCallStart := time.Now()
aiResponse, err := mcpClient.CallWithMessages(systemPrompt, userPrompt)
aiCallDuration := time.Since(aiCallStart)
```

### 5.2 鏀寔鐨凙I妯″瀷

| 妯″瀷 | 瀹㈡埛绔枃浠?| 榛樿妯″瀷 |
|------|-----------|----------|
| **DeepSeek** | `mcp/deepseek_client.go` | deepseek-chat |
| **Qwen** | `mcp/qwen_client.go` | qwen-max |
| **Claude** | `mcp/claude_client.go` | claude-3-5-sonnet |
| **Gemini** | `mcp/gemini_client.go` | gemini-pro |
| **Grok** | `mcp/grok_client.go` | grok-beta |
| **OpenAI** | `mcp/openai_client.go` | gpt-5.2 |
| **Kimi** | `mcp/kimi_client.go` | moonshot-v1-8k |

### 5.3 璇锋眰鍙傛暟

```go
// mcp/client.go
Timeout: 120 seconds
MaxRetries: 3
RetryDelay: 2 seconds (exponential backoff)
```

---

## 6. AI鍝嶅簲瑙ｆ瀽 (Response Parsing)

**鏍稿績鏂囦欢:** `decision/engine.go:1303-1604`

**鍏ュ彛鏂规硶:** `parseFullDecisionResponse(response, accountEquity, leverage, ratio)`

### 6.1 瑙ｆ瀽娴佺▼

```
鍘熷AI鍝嶅簲 (鏂囨湰)
    鈫?
1. 鎻愬彇鎬濈淮閾? [extractCoTTrace()]
    鈫?
2. 鎻愬彇JSON鍐崇瓥  [extractDecisions()]
    鈫?
3. 楠岃瘉JSON鏍煎紡  [validateJSONFormat()]
    鈫?
4. 瑙ｆ瀽JSON  [json.Unmarshal()]
    鈫?
5. 楠岃瘉鍐崇瓥  [validateDecisions()]
    鈫?
6. 鏋勫缓FullDecision  [杩斿洖缁撴瀯鍖栫粨鏋淽
```

### 6.2 鎬濈淮閾炬彁鍙?

```go
// decision/engine.go:1327-1345
func extractCoTTrace(response string) string {
    // 浼樺厛绾?: <reasoning> XML鏍囩
    if match := reReasoningTag.FindStringSubmatch(response); len(match) > 1 {
        return strings.TrimSpace(match[1])
    }
    // 浼樺厛绾?: <decision>鏍囩涔嬪墠鐨勬枃鏈?
    // 浼樺厛绾?: JSON [ 涔嬪墠鐨勬枃鏈?
    // 浼樺厛绾?: 瀹屾暣鍝嶅簲
}
```

### 6.3 JSON鍐崇瓥鎻愬彇

```go
// decision/engine.go:1347-1408
func extractDecisions(response string) (string, error) {
    // 1. 绉婚櫎涓嶅彲瑙佸瓧绗?
    response = removeInvisibleRunes(response)

    // 2. 淇瀛楃缂栫爜
    response = fixMissingQuotes(response)

    // 3. 鎻愬彇JSON (浼樺厛绾?
    //    - <decision> XML鏍囩 + ```json
    //    - 鐙珛 ```json 浠ｇ爜鍧?
    //    - 瑁窲SON鏁扮粍
}
```

### 6.4 瀛楃缂栫爜淇

```go
// decision/engine.go:1410-1432
func fixMissingQuotes(s string) string {
    // 涓枃寮曞彿 鈫?ASCII
    s = strings.ReplaceAll(s, """, "\"")
    s = strings.ReplaceAll(s, """, "\"")

    // 涓枃鎷彿 鈫?ASCII
    s = strings.ReplaceAll(s, "锛?, "[")
    s = strings.ReplaceAll(s, "锛?, "]")
    s = strings.ReplaceAll(s, "锝?, "{")
    s = strings.ReplaceAll(s, "锝?, "}")

    // 涓枃鏍囩偣 鈫?ASCII
    s = strings.ReplaceAll(s, "锛?, ":")
    s = strings.ReplaceAll(s, "锛?, ",")
}
```

### 6.5 鍐崇瓥楠岃瘉

```go
// decision/engine.go:1480-1602
func validateDecisions(decisions []Decision, equity, leverage, ratio float64) error {
    for _, d := range decisions {
        // 1. 楠岃瘉action绫诲瀷
        validActions := []string{"open_long", "open_short", "close_long", "close_short", "hold", "wait"}

        // 2. 寮€浠撻獙璇?
        if isOpenAction(d.Action) {
            // 鏉犳潌鑼冨洿妫€鏌?
            // 浠撲綅澶у皬妫€鏌?
            // 姝㈡崯姝㈢泩妫€鏌?
            // 椋庨櫓鍥炴姤姣旀鏌?
            // 缃俊搴︽鏌?
        }

        // 3. 骞充粨楠岃瘉
        if isCloseAction(d.Action) {
            // Symbol蹇呴』瀛樺湪
        }
    }
}
```

### 6.6 Decision缁撴瀯浣?

```go
// decision/engine.go:128-143
type Decision struct {
    Symbol          string   // 浜ゆ槗瀵? "BTCUSDT"
    Action          string   // "open_long", "open_short", "close_long", "close_short", "hold", "wait"
    Leverage        int      // 鏉犳潌鍊嶆暟
    PositionSizeUSD float64  // 浠撲綅浠峰€?(USDT)
    StopLoss        float64  // 姝㈡崯浠锋牸
    TakeProfit      float64  // 姝㈢泩浠锋牸
    Confidence      int      // 缃俊搴?0-100
    RiskUSD         float64  // 鏈€澶ч闄?(USDT)
    Reasoning       string   // 鍐崇瓥鐞嗙敱
}
```

---

## 7. 鍐崇瓥鎵ц (Execution)

**鏍稿績鏂囦欢:** `trader/auto_trader.go:392-560`

### 7.1 鍐崇瓥鎺掑簭

```go
// trader/auto_trader.go:519-526
sort.SliceStable(decisions, func(i, j int) bool {
    priority := map[string]int{
        "close_long": 1, "close_short": 1,  // 鏈€楂樹紭鍏堢骇
        "open_long": 2, "open_short": 2,    // 娆′紭鍏堢骇
        "hold": 3, "wait": 3,               // 鏈€浣庝紭鍏堢骇
    }
    return priority[decisions[i].Action] < priority[decisions[j].Action]
})
```

### 7.2 椋庢帶寮哄埗鎵ц

**鏂囦欢:** `trader/auto_trader.go:1769-1851`

| 妫€鏌ラ」 | 鏂规硶 | 鍔ㄤ綔 |
|--------|------|------|
| 鏈€澶ф寔浠撴暟 | `enforceMaxPositions()` | 鎷掔粷鏂板紑浠?|
| 浠撲綅浠峰€间笂闄?| `enforcePositionValueRatio()` | 鑷姩缂╁噺浠撲綅 |
| 鏈€灏忎粨浣?| `enforceMinPositionSize()` | 鎷掔粷杩囧皬璁㈠崟 |
| 淇濊瘉閲戣皟鏁?| 鑷姩璁＄畻 | 鏍规嵁鍙敤浣欓璋冩暣 |

### 7.3 璁㈠崟鎵ц

```go
// trader/auto_trader.go:1631-1767
func (at *AutoTrader) recordAndConfirmOrder(orderID, symbol, side, action string) {
    // 1. 杞璁㈠崟鐘舵€?(5娆￠噸璇? 500ms闂撮殧)
    for i := 0; i < 5; i++ {
        status := at.trader.GetOrderStatus(orderID)
        if status.Status == "FILLED" {
            break
        }
        time.Sleep(500 * time.Millisecond)
    }

    // 2. 鎻愬彇鎴愪氦淇℃伅
    filledPrice := status.AvgPrice
    filledQty := status.FilledQty
    fee := status.Fee

    // 3. 璁板綍鍒版暟鎹簱
    at.store.Position().SaveOrder(...)
}
```

### 7.4 鍐崇瓥鏃ュ織淇濆瓨

```go
// trader/auto_trader.go:1235-1256
record := &store.DecisionRecord{
    CycleNumber:    cycleNumber,
    TraderID:       traderID,
    Timestamp:      time.Now(),
    SystemPrompt:   systemPrompt,     // 瀹屾暣绯荤粺鎻愮ず璇?
    InputPrompt:    userPrompt,       // 瀹屾暣鐢ㄦ埛鎻愮ず璇?
    CoTTrace:       cotTrace,         // AI鎬濈淮閾?
    DecisionJSON:   decisionsJSON,    // 瑙ｆ瀽鍚庣殑鍐崇瓥
    RawResponse:    rawResponse,      // 鍘熷AI鍝嶅簲
    ExecutionLog:   executionResults, // 鎵ц缁撴灉
    CandidateCoins: candidateCoins,   // 鍊欓€夊竵绉?
    Success:        success,          // 鎵ц鐘舵€?
}
at.store.Decision().LogDecision(record)
```

---

## 鏍稿績鏂囦欢绱㈠紩

| 妯″潡 | 鏂囦欢 | 鍏抽敭鏂规硶 |
|------|------|----------|
| **涓诲惊鐜?* | `trader/auto_trader.go` | `Run()`, `runCycle()`, `buildTradingContext()` |
| **甯佺閫夋嫨** | `decision/engine.go:380-454` | `GetCandidateCoins()` |
| **鏁版嵁鑾峰彇** | `market/data.go` | `Get()`, `GetWithTimeframes()` |
| **鎸囨爣璁＄畻** | `market/data.go:59-98` | `calculateEMA()`, `calculateMACD()`, `calculateRSI()` |
| **绯荤粺鎻愮ず璇?* | `decision/engine.go:700-818` | `BuildSystemPrompt()` |
| **鐢ㄦ埛鎻愮ず璇?* | `decision/engine.go:884-1007` | `BuildUserPrompt()` |
| **甯傚満鏁版嵁鏍煎紡鍖?* | `decision/engine.go:1029-1099` | `formatMarketData()` |
| **AI璇锋眰** | `decision/engine.go:222-293` | `GetFullDecisionWithStrategy()` |
| **MCP瀹㈡埛绔?* | `mcp/client.go:136-150` | `CallWithMessages()` |
| **鍝嶅簲瑙ｆ瀽** | `decision/engine.go:1303-1604` | `parseFullDecisionResponse()` |
| **鎬濈淮閾炬彁鍙?* | `decision/engine.go:1327-1345` | `extractCoTTrace()` |
| **JSON鎻愬彇** | `decision/engine.go:1347-1408` | `extractDecisions()` |
| **鍐崇瓥楠岃瘉** | `decision/engine.go:1480-1602` | `validateDecisions()` |
| **椋庢帶鎵ц** | `trader/auto_trader.go:1769-1851` | `enforceMaxPositions()`, `enforcePositionValueRatio()` |
| **绛栫暐閰嶇疆** | `store/strategy.go` | `StrategyConfig`, `RiskControlConfig` |
| **数据提供者** | `provider/data_provider.go` | `GetAI500Data()`, `GetOITopPositions()`, `GetOTCTopSymbols()` |

---

## 閰嶇疆鍙傝€?

### 绛栫暐閰嶇疆缁撴瀯

```go
// store/strategy.go
type StrategyConfig struct {
    // 甯佺鏉ユ簮
    CoinSource struct {
        SourceType     string   // "static", "coinpool", "oi_top", "otc_top", "mixed"
        StaticCoins    []string // Static coin list
        UseCoinPool    bool     // Use AI500
        UseOITop       bool     // Use OI ranking
        UseOTCTop      bool     // Use OTC Top
        CoinPoolLimit  int      // AI500 fetch limit
        CoinPoolAPIURL string   // AI500 API URL
        OITopAPIURL    string   // OI ranking API URL
        OTCTopAPIURL   string   // OTC Top API URL
    }
    // 鎶€鏈寚鏍?
    Indicators struct {
        EnableEMA         bool
        EMAPeriods        []int   // [21, 55, 100, 200]
        EnableMACD        bool
        EnableRSI         bool
        RSIPeriods        []int   // [7, 14]
        EnableATR         bool
        ATRPeriods        []int   // [14]
        EnableVolume      bool
        EnableOI          bool
        EnableFundingRate bool
        EnableQuantData   bool
        EnableOIRanking   bool

        Klines struct {
            PrimaryTimeframe   string   // "5m"
            SelectedTimeframes []string // ["5m", "15m", "1h", "4h"]
            PrimaryCount       int      // 30
        }
    }

    // 椋庢帶閰嶇疆
    RiskControl struct {
        MaxPositions               int     // 鏈€澶ф寔浠撴暟
        BTCETHMaxLeverage          int     // BTC/ETH鏈€澶ф潬鏉?
        AltcoinMaxLeverage         int     // 灞卞甯佹渶澶ф潬鏉?
        BTCETHMaxPositionValueRatio float64 // BTC/ETH浠撲綅姣斾緥涓婇檺
        AltcoinMaxPositionValueRatio float64 // 灞卞甯佷粨浣嶆瘮渚嬩笂闄?
        MaxMarginUsage             float64 // 鏈€澶т繚璇侀噾浣跨敤鐜?
        MinPositionSize            float64 // 鏈€灏忎粨浣?
        MinRiskRewardRatio         float64 // 鏈€灏忛闄╁洖鎶ユ瘮
        MinConfidence              int     // 鏈€灏忕疆淇″害
    }

    // 鎻愮ず璇嶉儴鍒?
    PromptSections struct {
        RoleDefinition   string
        TradingFrequency string
        EntryStandards   string
        DecisionProcess  string
        RecentTradesLimit int // Recent closed trades in User Prompt
    }

    // 鑷畾涔夋彁绀鸿瘝
    CustomPrompt string
}
```

---

**鏂囨。鐗堟湰:** 1.0.0
**鏈€鍚庢洿鏂?** 2025-01-15







