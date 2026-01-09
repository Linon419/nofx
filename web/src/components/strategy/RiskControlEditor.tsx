import { Shield, AlertTriangle } from 'lucide-react'
import type { RiskControlConfig } from '../../types'

interface RiskControlEditorProps {
  config: RiskControlConfig
  onChange: (config: RiskControlConfig) => void
  disabled?: boolean
  language: string
}

export function RiskControlEditor({
  config,
  onChange,
  disabled,
  language,
}: RiskControlEditorProps) {
  const t = (key: string) => {
    const translations: Record<string, Record<string, string>> = {
      positionLimits: { zh: '持仓限制', en: 'Position Limits' },
      maxPositions: { zh: '最大持仓数', en: 'Max Positions' },
      maxPositionsDesc: { zh: '同时持有的最大币种数量', en: 'Maximum coins held simultaneously' },
      // Trading leverage (exchange leverage)
      tradingLeverage: { zh: '交易杠杆（交易所）', en: 'Trading Leverage (Exchange)' },
      btcEthLeverage: { zh: 'BTC/ETH 交易杠杆', en: 'BTC/ETH Trading Leverage' },
      btcEthLeverageDesc: { zh: '开仓使用的交易所杠杆', en: 'Exchange leverage for opening positions' },
      altcoinLeverage: { zh: '山寨币交易杠杆', en: 'Altcoin Trading Leverage' },
      altcoinLeverageDesc: { zh: '开仓使用的交易所杠杆', en: 'Exchange leverage for opening positions' },
      atrLeverage: { zh: 'ATR 杠杆与止损定仓', en: 'ATR Leverage & Stop-Loss Sizing' },
      atrLeverageDesc: { zh: '杠杆=round(close/max_atr_24h)，仓位按止损距离确定', en: 'Leverage=round(close/max_atr_24h), position size by stop-loss distance' },
      atrEnabled: { zh: '启用 ATR 杠杆', en: 'Enable ATR leverage' },
      atrEnabledDesc: { zh: '开启后系统会覆盖杠杆和仓位大小', en: 'When enabled, system overrides leverage and position size' },
      atrPeriod: { zh: 'ATR 周期', en: 'ATR Period' },
      atrTimeframe: { zh: 'ATR 时间周期', en: 'ATR Timeframe' },
      stopLossRiskPct: { zh: '单笔风险(%)', en: 'Risk Per Trade (%)' },
      stopLossRiskPctDesc: { zh: '单笔最大亏损占净值比例', en: 'Max loss as % of equity' },
      stopLossSizing: { zh: '强制止损定仓（代码强制）', en: 'Stop-Loss Sizing (CODE ENFORCED)' },
      stopLossSizingDesc: { zh: '关闭ATR时，系统会根据止损距离 + 杠杆重算 position_size_usd', en: 'When ATR is OFF, system recomputes position_size_usd using stop distance and leverage' },
      stopLossSizingEnabled: { zh: '开启止损定仓', en: 'Enable stop-loss sizing' },
      stopLossSizingEnabledDesc: { zh: '需要提供 stop_loss，系统在下单前覆盖 position_size_usd', en: 'Requires stop_loss; system overrides position_size_usd before placing orders' },
      // Position value ratio (risk control) - CODE ENFORCED
      positionValueRatio: { zh: '仓位价值比例（代码强制）', en: 'Position Value Ratio (CODE ENFORCED)' },
      positionValueRatioDesc: { zh: '仓位名义价值/净值，代码强制', en: 'Position notional value / equity, enforced by code' },
      btcEthPositionValueRatio: { zh: 'BTC/ETH 仓位比例', en: 'BTC/ETH Position Value Ratio' },
      btcEthPositionValueRatioDesc: { zh: '最大仓位 = 净值 × 该比例（代码强制）', en: 'Max position value = equity ? this ratio (CODE ENFORCED)' },
      altcoinPositionValueRatio: { zh: '山寨币仓位比例', en: 'Altcoin Position Value Ratio' },
      altcoinPositionValueRatioDesc: { zh: '最大仓位 = 净值 × 该比例（代码强制）', en: 'Max position value = equity ? this ratio (CODE ENFORCED)' },
      riskParameters: { zh: '风险参数', en: 'Risk Parameters' },
      minRiskReward: { zh: '最小盈亏比', en: 'Min Risk/Reward Ratio' },
      minRiskRewardDesc: { zh: '开仓所需的最低盈亏比', en: 'Minimum profit ratio for opening' },
      minOIValue: { zh: '最小OI流动性门槛（M USD）', en: 'Min OI Liquidity (M USD)' },
      minOIValueDesc: { zh: '候选币需满足最小OI名义价值；设为0关闭该过滤', en: 'Candidate coins must meet a minimum OI notional; set to 0 to disable this filter' },
      maxMarginUsage: { zh: '最大保证金使用率（代码强制）', en: 'Max Margin Usage (CODE ENFORCED)' },
      maxMarginUsageDesc: { zh: '最大保证金使用率，代码强制', en: 'Maximum margin utilization, enforced by code' },
      entryRequirements: { zh: '开仓要求', en: 'Entry Requirements' },
      minPositionSize: { zh: '最小开仓金额', en: 'Min Position Size' },
      minPositionSizeDesc: { zh: 'USDT 最小名义价值', en: 'Minimum notional value in USDT' },
      enforceMinPositionSize: { zh: '启用最小开仓金额校验', en: 'Enforce min position size' },
      enforceMinPositionSizeDesc: { zh: '关闭后允许更小的单（可能被交易所拒单）', en: 'If disabled, system may place tiny orders (exchange may reject)' },
      minConfidence: { zh: '最低信心', en: 'Min Confidence' },
      minConfidenceDesc: { zh: 'AI 开仓的信心阈值', en: 'AI confidence threshold for entry' },
      stopLossFlip: { zh: '止损反手（反向开仓）', en: 'Stop-loss Flip (Reverse Entry)' },
      stopLossFlipDesc: { zh: '止损触发后自动反向开仓（适用于单向/净持仓逻辑）', en: 'Auto reverse entry after stop-loss triggers (one-way/net logic)' },
      stopLossFlipEnabled: { zh: '启用止损反手', en: 'Enable stop-loss flip' },
      stopLossFlipRunnerRatio: { zh: 'Runner 比例', en: 'Runner Ratio' },
      stopLossFlipRunnerRatioDesc: { zh: '反向单部分止盈后保留的持仓比例（0~1）', en: 'Portion kept as runner after partial recovery TP (0~1)' },
      stopLossFlipTrailPct: { zh: '跟踪止损 (%)', en: 'Trail Stop (%)' },
      stopLossFlipTrailPctDesc: { zh: '输入百分比，例如 0.3 表示 0.3%（保存为 0.003）', en: 'Enter percent, e.g. 0.3 means 0.3% (stored as 0.003)' },
      stopLossFlipPollSecs: { zh: '轮询间隔 (秒)', en: 'Poll Interval (sec)' },
      stopLossFlipPollSecsDesc: { zh: '检查止损触发/反手状态的轮询频率', en: 'How often to poll for stop-loss trigger/flip status' },
      drawdownClose: { zh: '盈利回撤保护（自动平仓）', en: 'Profit Drawdown Protection (Auto Close)' },
      drawdownCloseDesc: { zh: '盈利后出现大幅回撤时自动平仓，防止利润回吐', en: 'Auto close when profit pulls back sharply from peak' },
      drawdownCloseEnabled: { zh: '启用回撤保护', en: 'Enable drawdown protection' },
      drawdownCloseMinProfitPct: { zh: '最小盈利(%)', en: 'Min Profit (%)' },
      drawdownCloseMinProfitPctDesc: { zh: '当前盈利达到该阈值后才允许触发回撤平仓', en: 'Only triggers when current profit is above this threshold' },
      drawdownClosePct: { zh: '回撤阈值(%)', en: 'Drawdown Threshold (%)' },
      drawdownClosePctDesc: { zh: '从峰值盈利回撤的百分比，达到后触发平仓', en: 'Percent drawdown from peak profit to trigger close' },
    }
    return translations[key]?.[language] || key
  }

  const updateField = <K extends keyof RiskControlConfig>(
    key: K,
    value: RiskControlConfig[K]
  ) => {
    if (!disabled) {
      onChange({ ...config, [key]: value })
    }
  }

  const atrEnabled = config.atr_enabled ?? false
  const stopLossSizingEnabled = config.stop_loss_sizing_enabled ?? false
  const stopLossSizingEffective = atrEnabled || stopLossSizingEnabled
  const enforceMinPositionSize = config.enforce_min_position_size ?? true
  const stopLossFlipEnabled = config.stop_loss_flip_enabled ?? false
  const stopLossFlipInputDisabled = disabled || !stopLossFlipEnabled
  const stopLossFlipRunnerRatio = config.stop_loss_flip_runner_ratio ?? 0.3
  const stopLossFlipTrailPct = config.stop_loss_flip_trail_pct ?? 0.003
  const drawdownCloseEnabled = config.drawdown_close_enabled ?? true
  const drawdownCloseInputDisabled = disabled || !drawdownCloseEnabled
  const drawdownCloseMinProfitPct = config.drawdown_close_min_profit_pct ?? 5
  const drawdownClosePct = config.drawdown_close_pct ?? 40

  return (
    <div className="space-y-6">
      {/* Position Limits */}
      <div>
        <div className="flex items-center gap-2 mb-4">
          <Shield className="w-5 h-5" style={{ color: '#F0B90B' }} />
          <h3 className="font-medium" style={{ color: '#EAECEF' }}>
            {t('positionLimits')}
          </h3>
        </div>

        <div className="grid grid-cols-1 gap-4 mb-4">
          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('maxPositions')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('maxPositionsDesc')}
            </p>
            <input
              type="number"
              value={config.max_positions ?? 3}
              onChange={(e) =>
                updateField('max_positions', parseInt(e.target.value) || 3)
              }
              disabled={disabled}
              min={1}
              max={10}
              className="w-32 px-3 py-2 rounded"
              style={{
                background: '#1E2329',
                border: '1px solid #2B3139',
                color: '#EAECEF',
              }}
            />
          </div>
        </div>

        {/* Trading Leverage (Exchange) */}
        <div className="mb-2">
          <p className="text-xs font-medium mb-2" style={{ color: '#F0B90B' }}>
            {t('tradingLeverage')}
          </p>
        </div>
        <div className="grid grid-cols-2 gap-4 mb-4">
          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('btcEthLeverage')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('btcEthLeverageDesc')}
            </p>
            <div className="flex items-center gap-2">
              <input
                type="range"
                value={config.btc_eth_max_leverage ?? 5}
                onChange={(e) =>
                  updateField('btc_eth_max_leverage', parseInt(e.target.value))
                }
                disabled={disabled}
                min={1}
                max={40}
                className="flex-1 accent-yellow-500"
              />
              <span
                className="w-12 text-center font-mono"
                style={{ color: '#F0B90B' }}
              >
                {config.btc_eth_max_leverage ?? 5}x
              </span>
            </div>
          </div>

          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('altcoinLeverage')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('altcoinLeverageDesc')}
            </p>
            <div className="flex items-center gap-2">
              <input
                type="range"
                value={config.altcoin_max_leverage ?? 5}
                onChange={(e) =>
                  updateField('altcoin_max_leverage', parseInt(e.target.value))
                }
                disabled={disabled}
                min={1}
                max={40}
                className="flex-1 accent-yellow-500"
              />
              <span
                className="w-12 text-center font-mono"
                style={{ color: '#F0B90B' }}
              >
                {config.altcoin_max_leverage ?? 5}x
              </span>
            </div>
          </div>
        </div>

        {/* ATR Leverage & Stop-Loss Sizing */}
        <div className="mb-2">
          <p className="text-xs font-medium mb-2" style={{ color: '#F0B90B' }}>
            {t('atrLeverage')}
          </p>
          <p className="text-xs" style={{ color: '#848E9C' }}>
            {t('atrLeverageDesc')}
          </p>
        </div>
        <div className="grid grid-cols-2 gap-4 mb-4">
          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('atrEnabled')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('atrEnabledDesc')}
            </p>
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={atrEnabled}
                onChange={(e) => updateField('atr_enabled', e.target.checked)}
                disabled={disabled}
                className="accent-yellow-500"
              />
              <span style={{ color: '#F0B90B' }}>
                {atrEnabled ? 'ON' : 'OFF'}
              </span>
            </label>
          </div>

          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('stopLossRiskPct')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('stopLossRiskPctDesc')}
            </p>
            <div className="flex items-center">
              <input
                type="number"
                value={config.stop_loss_risk_pct ?? 5}
                onChange={(e) =>
                  updateField('stop_loss_risk_pct', parseFloat(e.target.value) || 5)
                }
                disabled={disabled || !stopLossSizingEffective}
                min={0.1}
                max={20}
                step={0.1}
                className="w-24 px-3 py-2 rounded"
                style={{
                  background: '#1E2329',
                  border: '1px solid #2B3139',
                  color: '#EAECEF',
                }}
              />
              <span className="ml-2" style={{ color: '#848E9C' }}>
                %
              </span>
            </div>
          </div>

          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('stopLossSizing')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('stopLossSizingDesc')}
            </p>
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={stopLossSizingEnabled}
                onChange={(e) =>
                  updateField('stop_loss_sizing_enabled', e.target.checked)
                }
                disabled={disabled || atrEnabled}
                className="accent-yellow-500"
              />
              <span style={{ color: atrEnabled ? '#848E9C' : '#F0B90B' }}>
                {atrEnabled ? 'Controlled by ATR' : stopLossSizingEnabled ? 'ON' : 'OFF'}
              </span>
            </label>
            <p className="text-xs mt-2" style={{ color: '#848E9C' }}>
              {t('stopLossSizingEnabledDesc')}
            </p>
          </div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('atrPeriod')}
            </label>
            <input
              type="number"
              value={config.atr_period ?? 14}
              onChange={(e) =>
                updateField('atr_period', parseInt(e.target.value) || 14)
              }
              disabled={disabled || !atrEnabled}
              min={5}
              max={200}
              className="w-24 px-3 py-2 rounded"
              style={{
                background: '#1E2329',
                border: '1px solid #2B3139',
                color: '#EAECEF',
              }}
            />
          </div>

          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('atrTimeframe')}
            </label>
            <select
              value={config.atr_timeframe ?? '1d'}
              onChange={(e) => updateField('atr_timeframe', e.target.value)}
              disabled={disabled || !atrEnabled}
              className="w-full px-3 py-2 rounded"
              style={{
                background: '#1E2329',
                border: '1px solid #2B3139',
                color: '#EAECEF',
              }}
            >
              <option value="1h">1h</option>
              <option value="4h">4h</option>
              <option value="1d">1d</option>
            </select>
          </div>
        </div>
        {/* Position Value Ratio (Risk Control - CODE ENFORCED) */}
        <div className="mb-2">
          <p className="text-xs font-medium" style={{ color: '#0ECB81' }}>
            {t('positionValueRatio')}
          </p>
          <p className="text-xs mt-1" style={{ color: '#848E9C' }}>
            {t('positionValueRatioDesc')}
          </p>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #0ECB81' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('btcEthPositionValueRatio')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('btcEthPositionValueRatioDesc')}
            </p>
            <div className="flex items-center gap-2">
              <input
                type="range"
                value={config.btc_eth_max_position_value_ratio ?? 5}
                onChange={(e) =>
                  updateField('btc_eth_max_position_value_ratio', parseFloat(e.target.value))
                }
                disabled={disabled}
                min={0.5}
                max={10}
                step={0.5}
                className="flex-1 accent-green-500"
              />
              <span
                className="w-12 text-center font-mono"
                style={{ color: '#0ECB81' }}
              >
                {config.btc_eth_max_position_value_ratio ?? 5}x
              </span>
            </div>
          </div>

          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #0ECB81' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('altcoinPositionValueRatio')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('altcoinPositionValueRatioDesc')}
            </p>
            <div className="flex items-center gap-2">
              <input
                type="range"
                value={config.altcoin_max_position_value_ratio ?? 1}
                onChange={(e) =>
                  updateField('altcoin_max_position_value_ratio', parseFloat(e.target.value))
                }
                disabled={disabled}
                min={0.5}
                max={10}
                step={0.5}
                className="flex-1 accent-green-500"
              />
              <span
                className="w-12 text-center font-mono"
                style={{ color: '#0ECB81' }}
              >
                {config.altcoin_max_position_value_ratio ?? 1}x
              </span>
            </div>
          </div>

          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('minOIValue')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('minOIValueDesc')}
            </p>
            <div className="flex items-center">
              <input
                type="number"
                value={config.min_open_interest_value_millions ?? 15}
                onChange={(e) => {
                  const raw = e.target.value.trim()
                  if (raw === '') {
                    updateField('min_open_interest_value_millions', 15)
                    return
                  }
                  const next = Number(raw)
                  updateField('min_open_interest_value_millions', Number.isFinite(next) ? next : 15)
                }}
                disabled={disabled}
                min={0}
                max={500}
                step={1}
                className="w-28 px-3 py-2 rounded"
                style={{
                  background: '#1E2329',
                  border: '1px solid #2B3139',
                  color: '#EAECEF',
                }}
              />
              <span className="ml-2" style={{ color: '#848E9C' }}>
                M
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Risk Parameters */}
      <div>
        <div className="flex items-center gap-2 mb-4">
          <AlertTriangle className="w-5 h-5" style={{ color: '#F6465D' }} />
          <h3 className="font-medium" style={{ color: '#EAECEF' }}>
            {t('riskParameters')}
          </h3>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('minRiskReward')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('minRiskRewardDesc')}
            </p>
            <div className="flex items-center">
              <span style={{ color: '#848E9C' }}>1:</span>
              <input
                type="number"
                value={config.min_risk_reward_ratio ?? 3}
                onChange={(e) =>
                  updateField('min_risk_reward_ratio', parseFloat(e.target.value) || 3)
                }
                disabled={disabled}
                min={1}
                max={10}
                step={0.5}
                className="w-20 px-3 py-2 rounded ml-2"
                style={{
                  background: '#1E2329',
                  border: '1px solid #2B3139',
                  color: '#EAECEF',
                }}
              />
            </div>
          </div>

          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #0ECB81' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('maxMarginUsage')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('maxMarginUsageDesc')}
            </p>
            <div className="flex items-center gap-2">
              <input
                type="range"
                value={(config.max_margin_usage ?? 0.9) * 100}
                onChange={(e) =>
                  updateField('max_margin_usage', parseInt(e.target.value) / 100)
                }
                disabled={disabled}
                min={10}
                max={100}
                className="flex-1 accent-green-500"
              />
              <span className="w-12 text-center font-mono" style={{ color: '#0ECB81' }}>
                {Math.round((config.max_margin_usage ?? 0.9) * 100)}%
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Entry Requirements */}
      <div>
        <div className="flex items-center gap-2 mb-4">
          <Shield className="w-5 h-5" style={{ color: '#0ECB81' }} />
          <h3 className="font-medium" style={{ color: '#EAECEF' }}>
            {t('entryRequirements')}
          </h3>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('minPositionSize')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('minPositionSizeDesc')}
            </p>
            <div className="flex items-center gap-2 mb-2">
              <input
                type="checkbox"
                checked={enforceMinPositionSize}
                onChange={(e) =>
                  updateField('enforce_min_position_size', e.target.checked)
                }
                disabled={disabled}
                className="accent-green-500"
              />
              <span className="text-xs" style={{ color: '#EAECEF' }}>
                {t('enforceMinPositionSize')}
              </span>
            </div>
            {!enforceMinPositionSize && (
              <p className="text-xs mb-2" style={{ color: '#F0B90B' }}>
                {t('enforceMinPositionSizeDesc')}
              </p>
            )}
            <div className="flex items-center">
              <input
                type="number"
                value={config.min_position_size ?? 12}
                onChange={(e) =>
                  updateField('min_position_size', parseFloat(e.target.value) || 12)
                }
                disabled={disabled || !enforceMinPositionSize}
                min={10}
                max={1000}
                className="w-24 px-3 py-2 rounded"
                style={{
                  background: '#1E2329',
                  border: '1px solid #2B3139',
                  color: '#EAECEF',
                }}
              />
              <span className="ml-2" style={{ color: '#848E9C' }}>
                USDT
              </span>
            </div>
          </div>

          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('minConfidence')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('minConfidenceDesc')}
            </p>
            <div className="flex items-center gap-2">
              <input
                type="range"
                value={config.min_confidence ?? 75}
                onChange={(e) =>
                  updateField('min_confidence', parseInt(e.target.value))
                }
                disabled={disabled}
                min={50}
                max={100}
                className="flex-1 accent-green-500"
              />
              <span className="w-12 text-center font-mono" style={{ color: '#0ECB81' }}>
                {config.min_confidence ?? 75}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Stop-loss Flip */}
      <div>
        <div className="flex items-center gap-2 mb-4">
          <AlertTriangle className="w-5 h-5" style={{ color: '#F0B90B' }} />
          <div>
            <h3 className="font-medium" style={{ color: '#EAECEF' }}>
              {t('stopLossFlip')}
            </h3>
            <p className="text-xs" style={{ color: '#848E9C' }}>
              {t('stopLossFlipDesc')}
            </p>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <div className="flex items-center gap-2 mb-2">
              <input
                type="checkbox"
                checked={stopLossFlipEnabled}
                onChange={(e) =>
                  updateField('stop_loss_flip_enabled', e.target.checked)
                }
                disabled={disabled}
                className="accent-yellow-500"
              />
              <span className="text-sm font-medium" style={{ color: '#EAECEF' }}>
                {t('stopLossFlipEnabled')}
              </span>
            </div>

            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('stopLossFlipRunnerRatio')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('stopLossFlipRunnerRatioDesc')}
            </p>
            <div className="flex items-center gap-2">
              <input
                type="range"
                value={Math.round(stopLossFlipRunnerRatio * 100)}
                onChange={(e) => {
                  const raw = parseInt(e.target.value)
                  const next = Number.isFinite(raw) ? raw / 100 : 0.3
                  const clamped = Math.max(0, Math.min(1, next))
                  updateField('stop_loss_flip_runner_ratio', clamped)
                }}
                disabled={stopLossFlipInputDisabled}
                min={0}
                max={100}
                step={1}
                className="flex-1 accent-yellow-500"
              />
              <span
                className="w-12 text-center font-mono"
                style={{ color: '#F0B90B' }}
              >
                {Math.round(stopLossFlipRunnerRatio * 100)}%
              </span>
            </div>
          </div>

          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('stopLossFlipTrailPct')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('stopLossFlipTrailPctDesc')}
            </p>
            <div className="flex items-center">
              <input
                type="number"
                value={Number((stopLossFlipTrailPct * 100).toFixed(3))}
                onChange={(e) => {
                  const raw = e.target.value.trim()
                  if (raw === '') {
                    updateField('stop_loss_flip_trail_pct', 0.003)
                    return
                  }
                  const nextPct = Number(raw)
                  const pct = Number.isFinite(nextPct) ? nextPct : 0.3
                  const clampedPct = Math.max(0, Math.min(100, pct))
                  updateField('stop_loss_flip_trail_pct', clampedPct / 100)
                }}
                disabled={stopLossFlipInputDisabled}
                min={0}
                max={100}
                step={0.1}
                className="w-28 px-3 py-2 rounded"
                style={{
                  background: '#1E2329',
                  border: '1px solid #2B3139',
                  color: '#EAECEF',
                }}
              />
              <span className="ml-2" style={{ color: '#848E9C' }}>
                %
              </span>
            </div>

            <div className="mt-4">
              <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
                {t('stopLossFlipPollSecs')}
              </label>
              <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
                {t('stopLossFlipPollSecsDesc')}
              </p>
              <div className="flex items-center">
                <input
                  type="number"
                  value={config.stop_loss_flip_poll_secs ?? 10}
                  onChange={(e) => {
                    const raw = e.target.value.trim()
                    if (raw === '') {
                      updateField('stop_loss_flip_poll_secs', 10)
                      return
                    }
                    const next = parseInt(raw)
                    const n = Number.isFinite(next) ? next : 10
                    const clamped = Math.max(1, Math.min(3600, n))
                    updateField('stop_loss_flip_poll_secs', clamped)
                  }}
                  disabled={stopLossFlipInputDisabled}
                  min={1}
                  max={3600}
                  step={1}
                  className="w-28 px-3 py-2 rounded"
                  style={{
                    background: '#1E2329',
                    border: '1px solid #2B3139',
                    color: '#EAECEF',
                  }}
                />
                <span className="ml-2" style={{ color: '#848E9C' }}>
                  s
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Profit Drawdown Protection */}
      <div>
        <div className="flex items-center gap-2 mb-4">
          <AlertTriangle className="w-5 h-5" style={{ color: '#F0B90B' }} />
          <div>
            <h3 className="font-medium" style={{ color: '#EAECEF' }}>
              {t('drawdownClose')}
            </h3>
            <p className="text-xs" style={{ color: '#848E9C' }}>
              {t('drawdownCloseDesc')}
            </p>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <div className="flex items-center gap-2 mb-2">
              <input
                type="checkbox"
                checked={drawdownCloseEnabled}
                onChange={(e) =>
                  updateField('drawdown_close_enabled', e.target.checked)
                }
                disabled={disabled}
                className="accent-yellow-500"
              />
              <span className="text-sm font-medium" style={{ color: '#EAECEF' }}>
                {t('drawdownCloseEnabled')}
              </span>
            </div>

            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('drawdownCloseMinProfitPct')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('drawdownCloseMinProfitPctDesc')}
            </p>
            <div className="flex items-center">
              <input
                type="number"
                value={drawdownCloseMinProfitPct}
                onChange={(e) => {
                  const raw = e.target.value.trim()
                  if (raw === '') {
                    updateField('drawdown_close_min_profit_pct', 5)
                    return
                  }
                  const next = Number(raw)
                  const n = Number.isFinite(next) ? next : 5
                  const clamped = Math.max(0, Math.min(1000, n))
                  updateField('drawdown_close_min_profit_pct', clamped)
                }}
                disabled={drawdownCloseInputDisabled}
                min={0}
                max={1000}
                step={0.1}
                className="w-28 px-3 py-2 rounded"
                style={{
                  background: '#1E2329',
                  border: '1px solid #2B3139',
                  color: '#EAECEF',
                }}
              />
              <span className="ml-2" style={{ color: '#848E9C' }}>
                %
              </span>
            </div>
          </div>

          <div
            className="p-4 rounded-lg"
            style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
          >
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('drawdownClosePct')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('drawdownClosePctDesc')}
            </p>
            <div className="flex items-center">
              <input
                type="number"
                value={drawdownClosePct}
                onChange={(e) => {
                  const raw = e.target.value.trim()
                  if (raw === '') {
                    updateField('drawdown_close_pct', 40)
                    return
                  }
                  const next = Number(raw)
                  const n = Number.isFinite(next) ? next : 40
                  const clamped = Math.max(0, Math.min(1000, n))
                  updateField('drawdown_close_pct', clamped)
                }}
                disabled={drawdownCloseInputDisabled}
                min={0}
                max={1000}
                step={0.1}
                className="w-28 px-3 py-2 rounded"
                style={{
                  background: '#1E2329',
                  border: '1px solid #2B3139',
                  color: '#EAECEF',
                }}
              />
              <span className="ml-2" style={{ color: '#848E9C' }}>
                %
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}



