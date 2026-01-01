import { useState } from 'react'
import { ChevronDown, ChevronRight, RotateCcw, FileText } from 'lucide-react'
import type { PromptSectionsConfig } from '../../types'

interface PromptSectionsEditorProps {
  config: PromptSectionsConfig | undefined
  onChange: (config: PromptSectionsConfig) => void
  disabled?: boolean
  language: string
}

// Default prompt sections (same as backend defaults)
const defaultSections: PromptSectionsConfig = {
  role_definition: `# 你是专业的加密货币交易AI

你专注于技术分析和风险管理，基于市场数据做出理性的交易决策。
你的目标是在控制风险的前提下，捕捉高概率的交易机会。`,

  trading_frequency: `# ⏱️ 交易频率认知

- 优秀交易员：每天2-4笔 ≈ 每小时0.1-0.2笔
- 每小时>2笔 = 过度交易
- 单笔持仓时间≥30-60分钟
如果你发现自己每个周期都在交易 → 标准过低；若持仓<30分钟就平仓 → 过于急躁。`,

  entry_standards: `# 🎯 开仓标准（严格）

只在多重信号共振时开仓：
- 趋势方向明确（EMA排列、价格位置）
- 动量确认（MACD、RSI协同）
- 波动率适中（ATR合理范围）
- 量价配合（成交量支持方向）

避免：单一指标、信号矛盾、横盘震荡、刚平仓即重启。`,

  decision_process: `# 📋 决策流程

1. 检查持仓 → 是否该止盈/止损
2. 扫描候选币 + 多时间框 → 是否存在强信号
3. 评估风险回报比 → 是否满足最小要求
4. 先写思维链，再输出结构化JSON`,
  exit_strategy_plan: 'plan_tp_tiers_sl_single',
  recent_trades_limit: 3,
}

export function PromptSectionsEditor({
  config,
  onChange,
  disabled,
  language,
}: PromptSectionsEditorProps) {
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({
    role_definition: false,
    trading_frequency: false,
    entry_standards: false,
    decision_process: false,
  })

  const t = (key: string) => {
    const translations: Record<string, Record<string, string>> = {
      promptSections: { zh: 'System Prompt 自定义', en: 'System Prompt Customization' },
      promptSectionsDesc: { zh: '自定义 AI 行为和决策逻辑（输出格式和风控规则不可修改）', en: 'Customize AI behavior and decision logic (output format and risk rules are fixed)' },
      roleDefinition: { zh: '角色定义', en: 'Role Definition' },
      roleDefinitionDesc: { zh: '定义 AI 的身份和核心目标', en: 'Define AI identity and core objectives' },
      tradingFrequency: { zh: '交易频率', en: 'Trading Frequency' },
      tradingFrequencyDesc: { zh: '设定交易频率预期和过度交易警告', en: 'Set trading frequency expectations and overtrading warnings' },
      entryStandards: { zh: '开仓标准', en: 'Entry Standards' },
      entryStandardsDesc: { zh: '定义开仓信号条件和避免事项', en: 'Define entry signal conditions and avoidances' },
      decisionProcess: { zh: '决策流程', en: 'Decision Process' },
      decisionProcessDesc: { zh: '设定决策步骤和思考流程', en: 'Set decision steps and thinking process' },
      exitStrategyPlan: { zh: '止盈止损方案', en: 'Exit Strategy Plan' },
      exitStrategyPlanDesc: { zh: '选择 exit_plan 模板，影响 LLM 输出结构', en: 'Select the exit_plan template to shape LLM output' },
      exitPlanHint: { zh: '提示', en: 'Hint' },
      exitPlanTpTiersSlSingle: { zh: '分段止盈 + 单止损（默认）', en: 'Tiered TP + single SL (default)' },
      exitPlanTpTiersSlSingleHint: { zh: 'children = tp_tiers + sl_single', en: 'children = tp_tiers + sl_single' },
      exitPlanTpSingleSlSingle: { zh: '单止盈 + 单止损', en: 'Single TP + single SL' },
      exitPlanTpSingleSlSingleHint: { zh: 'children = tp_single + sl_single', en: 'children = tp_single + sl_single' },
      exitPlanSlAtrTpSingle: { zh: 'ATR 止损 + 单止盈', en: 'ATR SL + single TP' },
      exitPlanSlAtrTpSingleHint: { zh: 'children = sl_atr + tp_single', en: 'children = sl_atr + tp_single' },
      exitPlanTpAtrSlSingle: { zh: 'ATR 止盈 + 单止损', en: 'ATR TP + single SL' },
      exitPlanTpAtrSlSingleHint: { zh: 'children = tp_atr + sl_single', en: 'children = tp_atr + sl_single' },
      recentTradesLimit: { zh: '最近平仓条数', en: 'Recent Trades Count' },
      recentTradesLimitDesc: { zh: '追加到 User Prompt 的最近平仓记录数量（默认 3）', en: 'Number of recent closed trades appended to the User Prompt (default 3).' },
      resetToDefault: { zh: '重置为默认', en: 'Reset to Default' },
      chars: { zh: '字符', en: 'chars' },
    }
    return translations[key]?.[language] || key
  }

  const sections = [
    { key: 'role_definition', label: t('roleDefinition'), desc: t('roleDefinitionDesc') },
    { key: 'trading_frequency', label: t('tradingFrequency'), desc: t('tradingFrequencyDesc') },
    { key: 'entry_standards', label: t('entryStandards'), desc: t('entryStandardsDesc') },
    { key: 'decision_process', label: t('decisionProcess'), desc: t('decisionProcessDesc') },
  ]

  const exitPlanOptions = [
    { value: 'plan_tp_tiers_sl_single', label: t('exitPlanTpTiersSlSingle'), hint: t('exitPlanTpTiersSlSingleHint') },
    { value: 'plan_tp_single_sl_single', label: t('exitPlanTpSingleSlSingle'), hint: t('exitPlanTpSingleSlSingleHint') },
    { value: 'plan_sl_atr_tp_single', label: t('exitPlanSlAtrTpSingle'), hint: t('exitPlanSlAtrTpSingleHint') },
    { value: 'plan_tp_atr_sl_single', label: t('exitPlanTpAtrSlSingle'), hint: t('exitPlanTpAtrSlSingleHint') },
  ]

  const currentConfig = config || {}
  const defaultExitPlan = defaultSections.exit_strategy_plan ?? 'plan_tp_tiers_sl_single'
  const exitPlanValue = currentConfig.exit_strategy_plan ?? defaultExitPlan
  const selectedExitPlan = exitPlanOptions.find((option) => option.value === exitPlanValue)
  const isExitPlanModified =
    currentConfig.exit_strategy_plan !== undefined && currentConfig.exit_strategy_plan !== defaultExitPlan

  const defaultRecentTradesLimit = defaultSections.recent_trades_limit ?? 3
  const recentTradesLimit = currentConfig.recent_trades_limit ?? defaultRecentTradesLimit
  const isRecentTradesLimitModified =
    currentConfig.recent_trades_limit !== undefined &&
    currentConfig.recent_trades_limit !== defaultRecentTradesLimit

  const updateExitPlan = (value: string) => {
    if (!disabled) {
      onChange({ ...currentConfig, exit_strategy_plan: value })
    }
  }

  const resetExitPlan = () => {
    if (!disabled) {
      onChange({ ...currentConfig, exit_strategy_plan: defaultExitPlan })
    }
  }

  const updateRecentTradesLimit = (value: string) => {
    if (disabled) {
      return
    }
    if (value.trim() === '') {
      onChange({ ...currentConfig, recent_trades_limit: defaultRecentTradesLimit })
      return
    }
    const parsed = Number(value)
    if (!Number.isFinite(parsed)) {
      return
    }
    const normalized = Math.max(1, Math.floor(parsed))
    onChange({ ...currentConfig, recent_trades_limit: normalized })
  }

  const resetRecentTradesLimit = () => {
    if (!disabled) {
      onChange({ ...currentConfig, recent_trades_limit: defaultRecentTradesLimit })
    }
  }

  const updateSection = (key: keyof PromptSectionsConfig, value: string) => {
    if (!disabled) {
      onChange({ ...currentConfig, [key]: value })
    }
  }

  const resetSection = (key: keyof PromptSectionsConfig) => {
    if (!disabled) {
      onChange({ ...currentConfig, [key]: defaultSections[key] })
    }
  }

  const toggleSection = (key: string) => {
    setExpandedSections((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const getValue = (key: keyof PromptSectionsConfig): string => {
    const value = currentConfig[key] ?? defaultSections[key]
    return typeof value === 'string' ? value : ''
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-2 mb-4">
        <FileText className="w-5 h-5 mt-0.5" style={{ color: '#a855f7' }} />
        <div>
          <h3 className="font-medium" style={{ color: '#EAECEF' }}>
            {t('promptSections')}
          </h3>
          <p className="text-xs mt-1" style={{ color: '#848E9C' }}>
            {t('promptSectionsDesc')}
          </p>
        </div>
      </div>

      <div
        className="rounded-lg px-3 py-3"
        style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-sm font-medium" style={{ color: '#EAECEF' }}>
              {t('exitStrategyPlan')}
            </p>
            <p className="text-xs mt-1" style={{ color: '#848E9C' }}>
              {t('exitStrategyPlanDesc')}
            </p>
          </div>
          <select
            value={exitPlanValue}
            onChange={(e) => updateExitPlan(e.target.value)}
            disabled={disabled}
            className="px-2 py-1 rounded text-sm"
            style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
          >
            {exitPlanOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>
        {selectedExitPlan?.hint && (
          <p className="text-xs mt-2" style={{ color: '#848E9C' }}>
            <span className="font-medium" style={{ color: '#AEB4C0' }}>
              {t('exitPlanHint')}:
            </span>{' '}
            {selectedExitPlan.hint}
          </p>
        )}
        <div className="flex justify-end mt-2">
          <button
            onClick={resetExitPlan}
            disabled={disabled || !isExitPlanModified}
            className="flex items-center gap-1 px-2 py-1 rounded text-xs transition-colors hover:bg-white/5 disabled:opacity-30"
            style={{ color: '#848E9C' }}
          >
            <RotateCcw className="w-3 h-3" />
            {t('resetToDefault')}
          </button>
        </div>
      </div>

      <div
        className="rounded-lg px-3 py-3"
        style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-sm font-medium" style={{ color: '#EAECEF' }}>
              {t('recentTradesLimit')}
            </p>
            <p className="text-xs mt-1" style={{ color: '#848E9C' }}>
              {t('recentTradesLimitDesc')}
            </p>
          </div>
          <input
            type="number"
            min={1}
            step={1}
            value={recentTradesLimit}
            onChange={(e) => updateRecentTradesLimit(e.target.value)}
            disabled={disabled}
            className="w-20 px-2 py-1 rounded text-sm text-right"
            style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
          />
        </div>
        <div className="flex justify-end mt-2">
          <button
            onClick={resetRecentTradesLimit}
            disabled={disabled || !isRecentTradesLimitModified}
            className="flex items-center gap-1 px-2 py-1 rounded text-xs transition-colors hover:bg-white/5 disabled:opacity-30"
            style={{ color: '#848E9C' }}
          >
            <RotateCcw className="w-3 h-3" />
            {t('resetToDefault')}
          </button>
        </div>
      </div>

      <div className="space-y-2">
        {sections.map(({ key, label, desc }) => {
          const sectionKey = key as keyof PromptSectionsConfig
          const isExpanded = expandedSections[key]
          const value = getValue(sectionKey)
          const isModified = currentConfig[sectionKey] !== undefined && currentConfig[sectionKey] !== defaultSections[sectionKey]

          return (
            <div
              key={key}
              className="rounded-lg overflow-hidden"
              style={{ background: '#0B0E11', border: '1px solid #2B3139' }}
            >
              <button
                onClick={() => toggleSection(key)}
                className="w-full flex items-center justify-between px-3 py-2.5 hover:bg-white/5 transition-colors text-left"
              >
                <div className="flex items-center gap-2">
                  {isExpanded ? (
                    <ChevronDown className="w-4 h-4" style={{ color: '#848E9C' }} />
                  ) : (
                    <ChevronRight className="w-4 h-4" style={{ color: '#848E9C' }} />
                  )}
                  <span className="text-sm font-medium" style={{ color: '#EAECEF' }}>
                    {label}
                  </span>
                  {isModified && (
                    <span
                      className="px-1.5 py-0.5 text-[10px] rounded"
                      style={{ background: 'rgba(168, 85, 247, 0.15)', color: '#a855f7' }}
                    >
                      {language === 'zh' ? '已修改' : 'Modified'}
                    </span>
                  )}
                </div>
                <span className="text-[10px]" style={{ color: '#848E9C' }}>
                  {value.length} {t('chars')}
                </span>
              </button>

              {isExpanded && (
                <div className="px-3 pb-3">
                  <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
                    {desc}
                  </p>
                  <textarea
                    value={value}
                    onChange={(e) => updateSection(sectionKey, e.target.value)}
                    disabled={disabled}
                    rows={6}
                    className="w-full px-3 py-2 rounded-lg resize-y font-mono text-xs"
                    style={{
                      background: '#1E2329',
                      border: '1px solid #2B3139',
                      color: '#EAECEF',
                      minHeight: '120px',
                    }}
                  />
                  <div className="flex justify-end mt-2">
                    <button
                      onClick={() => resetSection(sectionKey)}
                      disabled={disabled || !isModified}
                      className="flex items-center gap-1 px-2 py-1 rounded text-xs transition-colors hover:bg-white/5 disabled:opacity-30"
                      style={{ color: '#848E9C' }}
                    >
                      <RotateCcw className="w-3 h-3" />
                      {t('resetToDefault')}
                    </button>
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
