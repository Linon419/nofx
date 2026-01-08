import { useState } from 'react'
import { SlidersHorizontal, AlertTriangle, ChevronDown, ChevronRight, RotateCcw } from 'lucide-react'
import type { PromptTogglesConfig, PromptModulesConfig } from '../../types'

interface PromptTogglesEditorProps {
  config?: PromptTogglesConfig
  onChange: (config: PromptTogglesConfig) => void
  modules?: PromptModulesConfig
  onModulesChange: (config: PromptModulesConfig) => void
  disabled?: boolean
  language: string
  visionEnabled?: boolean
}

export function PromptTogglesEditor({
  config,
  onChange,
  modules,
  onModulesChange,
  disabled,
  language,
  visionEnabled,
}: PromptTogglesEditorProps) {
  const t = (key: string) => {
    const translations: Record<string, Record<string, string>> = {
      title: { zh: 'Prompt 模块开关', en: 'Prompt Module Toggles' },
      desc: { zh: '控制内置 Prompt 段落是否参与实际请求（影响 token 与行为）', en: 'Control which built-in prompt modules are sent (affects tokens & behavior).' },
      schema: { zh: 'Schema / 数据字典', en: 'Schema / Data Dictionary' },
      modeVariant: { zh: '模式变体段落', en: 'Mode Variant Block' },
      hardConstraints: { zh: 'Hard Constraints 文本', en: 'Hard Constraints Text' },
      outputFormat: { zh: '输出格式要求', en: 'Output Format Requirements' },
      analysisStage: { zh: '分析层（预分析）', en: 'Analysis Stage (Pre-analysis)' },
      visionVerbose: { zh: '读图层提示词（详细）', en: 'Vision Prompt (Verbose)' },
      warningHard: {
        zh: '关闭后模型可能忽略风控含义；但后台仍会强制校验风控规则。',
        en: 'Turning this off may reduce risk-awareness in the model; backend risk validation still applies.',
      },
      warningFormat: {
        zh: '关闭后更容易输出不可解析内容，导致交易失败。',
        en: 'Turning this off increases parse failures and may break trading.',
      },
      warningAnalysis: {
        zh: '关闭后即使配置了分析层模型，也不会运行预分析阶段。',
        en: 'Turning this off disables pre-analysis even if an analysis model is configured.',
      },
      hintVision: {
        zh: '仅影响读图阶段的 system prompt；读图功能是否启用仍由 Vision 开关决定。',
        en: 'Only affects the vision stage system prompt; Vision enablement is controlled by the Vision switch.',
      },
      enabled: { zh: '启用', en: 'Enable' },
      disabledByVision: { zh: '（Vision 未启用）', en: '(Vision is disabled)' },
    }
    return translations[key]?.[language] || key
  }

  const current: PromptTogglesConfig = {
    include_schema_prompt: config?.include_schema_prompt ?? true,
    include_mode_variant: config?.include_mode_variant ?? true,
    include_hard_constraints: config?.include_hard_constraints ?? true,
    include_output_format: config?.include_output_format ?? true,
    enable_analysis_stage: config?.enable_analysis_stage ?? true,
    use_verbose_vision_system_prompt: config?.use_verbose_vision_system_prompt ?? true,
  }

  const currentModules: PromptModulesConfig = {
    schema_prompt: modules?.schema_prompt ?? '',
    schema_prompt_lite: modules?.schema_prompt_lite ?? '',
    mode_variant_aggressive: modules?.mode_variant_aggressive ?? '',
    mode_variant_conservative: modules?.mode_variant_conservative ?? '',
    mode_variant_scalping: modules?.mode_variant_scalping ?? '',
    hard_constraints: modules?.hard_constraints ?? '',
    output_format: modules?.output_format ?? '',
    analysis_core_prompt: modules?.analysis_core_prompt ?? '',
    vision_system_prompt: modules?.vision_system_prompt ?? '',
  }

  const set = (patch: Partial<PromptTogglesConfig>) => {
    if (disabled) return
    onChange({ ...current, ...patch })
  }

  const setModules = (patch: Partial<PromptModulesConfig>) => {
    if (disabled) return
    onModulesChange({ ...currentModules, ...patch })
  }

  const placeholder =
    language === 'zh'
      ? '留空=使用系统默认。支持 Go template 变量，例如：{{.AccountEquity}} / {{.MaxPositions}} / {{.ExitPlanID}} / {{.ExitPlanExample}}'
      : 'Empty = use system default. Supports Go template vars, e.g. {{.AccountEquity}} / {{.MaxPositions}} / {{.ExitPlanID}} / {{.ExitPlanExample}}'

  const [expandedEditors, setExpandedEditors] = useState<Record<string, boolean>>({
    schema: false,
    mode: false,
    hard: false,
    format: false,
    analysis: false,
    vision: false,
  })

  const toggleEditor = (key: keyof typeof expandedEditors) => {
    setExpandedEditors((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const SwitchRow = ({
    label,
    checked,
    onCheckedChange,
    description,
    warning,
    disabledReason,
  }: {
    label: string
    checked: boolean
    onCheckedChange: (next: boolean) => void
    description?: string
    warning?: string
    disabledReason?: string
  }) => {
    const isDisabled = Boolean(disabled)
    return (
      <div
        className="rounded-lg px-3 py-3 space-y-2"
        style={{ background: '#0B0E11', border: '1px solid #2B3139', opacity: isDisabled ? 0.6 : 1 }}
      >
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <div className="text-sm font-medium" style={{ color: '#EAECEF' }}>
                {label}
              </div>
              {disabledReason && (
                <div className="text-[11px]" style={{ color: '#848E9C' }}>
                  {disabledReason}
                </div>
              )}
            </div>
            {description && (
              <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
                {description}
              </div>
            )}
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              className="sr-only peer"
              checked={checked}
              onChange={(e) => !isDisabled && onCheckedChange(e.target.checked)}
              disabled={isDisabled}
            />
            <div className="w-10 h-5 bg-[#2B3139] peer-focus:outline-none rounded-full peer peer-checked:bg-[#0ECB81] transition-colors" />
            <div className="absolute left-0.5 top-0.5 w-4 h-4 bg-white rounded-full transition-transform peer-checked:translate-x-5" />
          </label>
        </div>
        {warning && !checked && (
          <div className="flex items-start gap-2 text-xs" style={{ color: '#F0B90B' }}>
            <AlertTriangle className="w-4 h-4 mt-0.5 flex-shrink-0" />
            <div>{warning}</div>
          </div>
        )}
      </div>
    )
  }

  const visionDisabled = !(visionEnabled ?? false)

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-2">
        <div className="p-2 rounded-lg" style={{ background: 'rgba(96, 165, 250, 0.12)' }}>
          <SlidersHorizontal className="w-4 h-4" style={{ color: '#60a5fa' }} />
        </div>
        <div className="flex-1">
          <div className="text-sm font-semibold" style={{ color: '#EAECEF' }}>
            {t('title')}
          </div>
          <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
            {t('desc')}
          </div>
        </div>
      </div>

      <SwitchRow
        label={t('schema')}
        checked={Boolean(current.include_schema_prompt)}
        onCheckedChange={(v) => set({ include_schema_prompt: v })}
      />
      <div className="flex justify-end">
        <button
          type="button"
          onClick={() => toggleEditor('schema')}
          className="flex items-center gap-1 px-2 py-1 rounded text-xs hover:bg-white/5"
          style={{ color: '#848E9C' }}
          disabled={Boolean(disabled)}
        >
          {expandedEditors.schema ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
          {language === 'zh' ? '编辑文本' : 'Edit text'}
        </button>
      </div>
      {expandedEditors.schema && (
        <div className="space-y-3">
          <div className="rounded-lg px-3 py-3" style={{ background: '#0B0E11', border: '1px solid #2B3139' }}>
            <div className="flex items-center justify-between">
              <div className="text-sm font-medium" style={{ color: '#EAECEF' }}>
                {language === 'zh' ? 'Schema（系统）' : 'Schema (System)'}
              </div>
              <button
                type="button"
                onClick={() => setModules({ schema_prompt: '' })}
                disabled={Boolean(disabled) || !currentModules.schema_prompt}
                className="flex items-center gap-1 px-2 py-1 rounded text-xs hover:bg-white/5 disabled:opacity-30"
                style={{ color: '#848E9C' }}
              >
                <RotateCcw className="w-3 h-3" />
                {language === 'zh' ? '清空' : 'Clear'}
              </button>
            </div>
            <textarea
              value={currentModules.schema_prompt || ''}
              onChange={(e) => setModules({ schema_prompt: e.target.value })}
              disabled={Boolean(disabled)}
              placeholder={placeholder}
              className="w-full h-40 mt-2 px-3 py-2 rounded-lg resize-none font-mono text-xs"
              style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
            />
          </div>

          <div className="rounded-lg px-3 py-3" style={{ background: '#0B0E11', border: '1px solid #2B3139' }}>
            <div className="flex items-center justify-between">
              <div className="text-sm font-medium" style={{ color: '#EAECEF' }}>
                {language === 'zh' ? 'Schema Lite（分析层）' : 'Schema Lite (Analysis)'}
              </div>
              <button
                type="button"
                onClick={() => setModules({ schema_prompt_lite: '' })}
                disabled={Boolean(disabled) || !currentModules.schema_prompt_lite}
                className="flex items-center gap-1 px-2 py-1 rounded text-xs hover:bg-white/5 disabled:opacity-30"
                style={{ color: '#848E9C' }}
              >
                <RotateCcw className="w-3 h-3" />
                {language === 'zh' ? '清空' : 'Clear'}
              </button>
            </div>
            <textarea
              value={currentModules.schema_prompt_lite || ''}
              onChange={(e) => setModules({ schema_prompt_lite: e.target.value })}
              disabled={Boolean(disabled)}
              placeholder={placeholder}
              className="w-full h-32 mt-2 px-3 py-2 rounded-lg resize-none font-mono text-xs"
              style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
            />
          </div>
        </div>
      )}

      <SwitchRow
        label={t('modeVariant')}
        checked={Boolean(current.include_mode_variant)}
        onCheckedChange={(v) => set({ include_mode_variant: v })}
      />
      <div className="flex justify-end">
        <button
          type="button"
          onClick={() => toggleEditor('mode')}
          className="flex items-center gap-1 px-2 py-1 rounded text-xs hover:bg-white/5"
          style={{ color: '#848E9C' }}
          disabled={Boolean(disabled)}
        >
          {expandedEditors.mode ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
          {language === 'zh' ? '编辑文本' : 'Edit text'}
        </button>
      </div>
      {expandedEditors.mode && (
        <div className="space-y-3">
          {(
            [
              ['mode_variant_aggressive', language === 'zh' ? 'Aggressive' : 'Aggressive'] as const,
              ['mode_variant_conservative', language === 'zh' ? 'Conservative' : 'Conservative'] as const,
              ['mode_variant_scalping', language === 'zh' ? 'Scalping' : 'Scalping'] as const,
            ] as const
          ).map(([key, label]) => (
            <div key={key} className="rounded-lg px-3 py-3" style={{ background: '#0B0E11', border: '1px solid #2B3139' }}>
              <div className="flex items-center justify-between">
                <div className="text-sm font-medium" style={{ color: '#EAECEF' }}>
                  {label}
                </div>
                <button
                  type="button"
                  onClick={() => setModules({ [key]: '' } as Partial<PromptModulesConfig>)}
                  disabled={Boolean(disabled) || !currentModules[key]}
                  className="flex items-center gap-1 px-2 py-1 rounded text-xs hover:bg-white/5 disabled:opacity-30"
                  style={{ color: '#848E9C' }}
                >
                  <RotateCcw className="w-3 h-3" />
                  {language === 'zh' ? '清空' : 'Clear'}
                </button>
              </div>
              <textarea
                value={currentModules[key] || ''}
                onChange={(e) => setModules({ [key]: e.target.value } as Partial<PromptModulesConfig>)}
                disabled={Boolean(disabled)}
                placeholder={placeholder}
                className="w-full h-28 mt-2 px-3 py-2 rounded-lg resize-none font-mono text-xs"
                style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
              />
            </div>
          ))}
        </div>
      )}

      <SwitchRow
        label={t('hardConstraints')}
        checked={Boolean(current.include_hard_constraints)}
        onCheckedChange={(v) => set({ include_hard_constraints: v })}
        warning={t('warningHard')}
      />
      <div className="flex justify-end">
        <button
          type="button"
          onClick={() => toggleEditor('hard')}
          className="flex items-center gap-1 px-2 py-1 rounded text-xs hover:bg-white/5"
          style={{ color: '#848E9C' }}
          disabled={Boolean(disabled)}
        >
          {expandedEditors.hard ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
          {language === 'zh' ? '编辑文本' : 'Edit text'}
        </button>
      </div>
      {expandedEditors.hard && (
        <div className="rounded-lg px-3 py-3" style={{ background: '#0B0E11', border: '1px solid #2B3139' }}>
          <div className="flex items-center justify-between">
            <div className="text-sm font-medium" style={{ color: '#EAECEF' }}>
              {language === 'zh' ? 'Hard Constraints（替换整段）' : 'Hard Constraints (Replace whole block)'}
            </div>
            <button
              type="button"
              onClick={() => setModules({ hard_constraints: '' })}
              disabled={Boolean(disabled) || !currentModules.hard_constraints}
              className="flex items-center gap-1 px-2 py-1 rounded text-xs hover:bg-white/5 disabled:opacity-30"
              style={{ color: '#848E9C' }}
            >
              <RotateCcw className="w-3 h-3" />
              {language === 'zh' ? '清空' : 'Clear'}
            </button>
          </div>
          <textarea
            value={currentModules.hard_constraints || ''}
            onChange={(e) => setModules({ hard_constraints: e.target.value })}
            disabled={Boolean(disabled)}
            placeholder={placeholder}
            className="w-full h-40 mt-2 px-3 py-2 rounded-lg resize-none font-mono text-xs"
            style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
          />
        </div>
      )}

      <SwitchRow
        label={t('outputFormat')}
        checked={Boolean(current.include_output_format)}
        onCheckedChange={(v) => set({ include_output_format: v })}
        warning={t('warningFormat')}
      />
      <div className="flex justify-end">
        <button
          type="button"
          onClick={() => toggleEditor('format')}
          className="flex items-center gap-1 px-2 py-1 rounded text-xs hover:bg-white/5"
          style={{ color: '#848E9C' }}
          disabled={Boolean(disabled)}
        >
          {expandedEditors.format ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
          {language === 'zh' ? '编辑文本' : 'Edit text'}
        </button>
      </div>
      {expandedEditors.format && (
        <div className="rounded-lg px-3 py-3" style={{ background: '#0B0E11', border: '1px solid #2B3139' }}>
          <div className="flex items-center justify-between">
            <div className="text-sm font-medium" style={{ color: '#EAECEF' }}>
              {language === 'zh' ? 'Output Format（替换整段）' : 'Output Format (Replace whole block)'}
            </div>
            <button
              type="button"
              onClick={() => setModules({ output_format: '' })}
              disabled={Boolean(disabled) || !currentModules.output_format}
              className="flex items-center gap-1 px-2 py-1 rounded text-xs hover:bg-white/5 disabled:opacity-30"
              style={{ color: '#848E9C' }}
            >
              <RotateCcw className="w-3 h-3" />
              {language === 'zh' ? '清空' : 'Clear'}
            </button>
          </div>
          <textarea
            value={currentModules.output_format || ''}
            onChange={(e) => setModules({ output_format: e.target.value })}
            disabled={Boolean(disabled)}
            placeholder={placeholder}
            className="w-full h-48 mt-2 px-3 py-2 rounded-lg resize-none font-mono text-xs"
            style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
          />
        </div>
      )}

      <SwitchRow
        label={t('analysisStage')}
        checked={Boolean(current.enable_analysis_stage)}
        onCheckedChange={(v) => set({ enable_analysis_stage: v })}
        warning={t('warningAnalysis')}
      />
      <div className="flex justify-end">
        <button
          type="button"
          onClick={() => toggleEditor('analysis')}
          className="flex items-center gap-1 px-2 py-1 rounded text-xs hover:bg-white/5"
          style={{ color: '#848E9C' }}
          disabled={Boolean(disabled)}
        >
          {expandedEditors.analysis ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
          {language === 'zh' ? '编辑文本' : 'Edit text'}
        </button>
      </div>
      {expandedEditors.analysis && (
        <div className="rounded-lg px-3 py-3" style={{ background: '#0B0E11', border: '1px solid #2B3139' }}>
          <div className="flex items-center justify-between">
            <div className="text-sm font-medium" style={{ color: '#EAECEF' }}>
              {language === 'zh' ? '分析层 Core Prompt' : 'Analysis Core Prompt'}
            </div>
            <button
              type="button"
              onClick={() => setModules({ analysis_core_prompt: '' })}
              disabled={Boolean(disabled) || !currentModules.analysis_core_prompt}
              className="flex items-center gap-1 px-2 py-1 rounded text-xs hover:bg-white/5 disabled:opacity-30"
              style={{ color: '#848E9C' }}
            >
              <RotateCcw className="w-3 h-3" />
              {language === 'zh' ? '清空' : 'Clear'}
            </button>
          </div>
          <textarea
            value={currentModules.analysis_core_prompt || ''}
            onChange={(e) => setModules({ analysis_core_prompt: e.target.value })}
            disabled={Boolean(disabled)}
            placeholder={placeholder}
            className="w-full h-40 mt-2 px-3 py-2 rounded-lg resize-none font-mono text-xs"
            style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
          />
        </div>
      )}

      <SwitchRow
        label={t('visionVerbose')}
        checked={Boolean(current.use_verbose_vision_system_prompt)}
        onCheckedChange={(v) => set({ use_verbose_vision_system_prompt: v })}
        description={
          visionDisabled ? `${t('hintVision')} ${t('disabledByVision')}` : t('hintVision')
        }
      />
      <div className="flex justify-end">
        <button
          type="button"
          onClick={() => toggleEditor('vision')}
          className="flex items-center gap-1 px-2 py-1 rounded text-xs hover:bg-white/5"
          style={{ color: '#848E9C' }}
          disabled={Boolean(disabled)}
        >
          {expandedEditors.vision ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
          {language === 'zh' ? '编辑文本' : 'Edit text'}
        </button>
      </div>
      {expandedEditors.vision && (
        <div className="rounded-lg px-3 py-3" style={{ background: '#0B0E11', border: '1px solid #2B3139', opacity: visionDisabled ? 0.6 : 1 }}>
          <div className="flex items-center justify-between">
            <div className="text-sm font-medium" style={{ color: '#EAECEF' }}>
              {language === 'zh' ? '读图层 System Prompt' : 'Vision System Prompt'}
            </div>
            <button
              type="button"
              onClick={() => setModules({ vision_system_prompt: '' })}
              disabled={Boolean(disabled) || !currentModules.vision_system_prompt}
              className="flex items-center gap-1 px-2 py-1 rounded text-xs hover:bg-white/5 disabled:opacity-30"
              style={{ color: '#848E9C' }}
            >
              <RotateCcw className="w-3 h-3" />
              {language === 'zh' ? '清空' : 'Clear'}
            </button>
          </div>
          <textarea
            value={currentModules.vision_system_prompt || ''}
            onChange={(e) => setModules({ vision_system_prompt: e.target.value })}
            disabled={Boolean(disabled)}
            placeholder={placeholder}
            className="w-full h-48 mt-2 px-3 py-2 rounded-lg resize-none font-mono text-xs"
            style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
          />
        </div>
      )}
    </div>
  )
}
