import { SlidersHorizontal, AlertTriangle } from 'lucide-react'
import type { PromptTogglesConfig } from '../../types'

interface PromptTogglesEditorProps {
  config?: PromptTogglesConfig
  onChange: (config: PromptTogglesConfig) => void
  disabled?: boolean
  language: string
  visionEnabled?: boolean
}

export function PromptTogglesEditor({
  config,
  onChange,
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

  const set = (patch: Partial<PromptTogglesConfig>) => {
    if (disabled) return
    onChange({ ...current, ...patch })
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

      <SwitchRow
        label={t('modeVariant')}
        checked={Boolean(current.include_mode_variant)}
        onCheckedChange={(v) => set({ include_mode_variant: v })}
      />

      <SwitchRow
        label={t('hardConstraints')}
        checked={Boolean(current.include_hard_constraints)}
        onCheckedChange={(v) => set({ include_hard_constraints: v })}
        warning={t('warningHard')}
      />

      <SwitchRow
        label={t('outputFormat')}
        checked={Boolean(current.include_output_format)}
        onCheckedChange={(v) => set({ include_output_format: v })}
        warning={t('warningFormat')}
      />

      <SwitchRow
        label={t('analysisStage')}
        checked={Boolean(current.enable_analysis_stage)}
        onCheckedChange={(v) => set({ enable_analysis_stage: v })}
        warning={t('warningAnalysis')}
      />

      <SwitchRow
        label={t('visionVerbose')}
        checked={Boolean(current.use_verbose_vision_system_prompt)}
        onCheckedChange={(v) => set({ use_verbose_vision_system_prompt: v })}
        description={
          visionDisabled ? `${t('hintVision')} ${t('disabledByVision')}` : t('hintVision')
        }
      />
    </div>
  )
}
