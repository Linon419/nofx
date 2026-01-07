import { Eye, Image as ImageIcon, Info } from 'lucide-react'
import type { VisionConfig } from '../../types'

interface VisionEditorProps {
  config?: VisionConfig
  onChange: (config: VisionConfig) => void
  disabled?: boolean
  language: string
}

type VisionIndicators = NonNullable<VisionConfig['indicators']>

export function VisionEditor({ config, onChange, disabled, language }: VisionEditorProps) {
  const t = (key: string) => {
    const translations: Record<string, Record<string, string>> = {
      title: { zh: '视觉读图', en: 'Vision (Chart Reading)' },
      desc: {
        zh: '为支持视觉的模型附加 K 线图（会增加延迟与成本）',
        en: 'Attach chart images for vision-capable models (higher latency & cost)',
      },
      enabled: { zh: '启用', en: 'Enable' },
      maxSymbols: { zh: '币种数量', en: 'Symbols' },
      timeframes: { zh: '时间周期', en: 'Timeframes' },
      imageSize: { zh: '图片尺寸', en: 'Image Size' },
      width: { zh: '宽', en: 'Width' },
      height: { zh: '高', en: 'Height' },
      concurrency: { zh: '渲染并发', en: 'Render Concurrency' },
      indicators: { zh: '图中指标', en: 'Indicators in Image' },
      showEMA: { zh: 'EMA', en: 'EMA' },
      showMACD: { zh: 'MACD', en: 'MACD' },
      showCVD: { zh: 'CVD', en: 'CVD' },
      showWaveTrend: { zh: 'WaveTrend', en: 'WaveTrend' },
      showSqueeze: { zh: '挤压预警 (Squeeze)', en: 'Squeeze' },
      showDivergence: { zh: '背离 (Divergence)', en: 'Divergence' },
      note: {
        zh: '模式：每个币种按所选时间周期附加 K 线图；最后一次汇总决策不带图。',
        en: 'Mode: per-symbol image calls use the selected timeframes; final decision call is text-only.',
      },
    }
    return translations[key]?.[language] || key
  }

  const current: VisionConfig = {
    enabled: config?.enabled ?? false,
    max_symbols: config?.max_symbols ?? 5,
    timeframes: config?.timeframes ?? ['1h', '15m'],
    image_width: config?.image_width ?? 1600,
    image_height: config?.image_height ?? 1396,
    render_concurrency: config?.render_concurrency ?? 1,
    indicators: {
      show_ema: config?.indicators?.show_ema ?? true,
      show_macd: config?.indicators?.show_macd ?? true,
      show_cvd: config?.indicators?.show_cvd ?? true,
      show_wavetrend: config?.indicators?.show_wavetrend ?? true,
      show_squeeze: config?.indicators?.show_squeeze ?? false,
      show_divergence: config?.indicators?.show_divergence ?? true,
    },
  }

  const set = (patch: Partial<VisionConfig>) => {
    onChange({ ...current, ...patch })
  }

  const setIndicators = (patch: Partial<VisionIndicators>) => {
    set({ indicators: { ...(current.indicators || {}), ...patch } })
  }

  const toggleTimeframe = (tf: string) => {
    const list = current.timeframes || []
    const next = list.includes(tf) ? list.filter((x) => x !== tf) : [...list, tf]
    set({ timeframes: next })
  }

  const indicatorItems: Array<{ key: keyof VisionIndicators; label: string }> = [
    { key: 'show_ema', label: t('showEMA') },
    { key: 'show_macd', label: t('showMACD') },
    { key: 'show_cvd', label: t('showCVD') },
    { key: 'show_wavetrend', label: t('showWaveTrend') },
    { key: 'show_squeeze', label: t('showSqueeze') },
    { key: 'show_divergence', label: t('showDivergence') },
  ]

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-2">
        <div className="p-2 rounded-lg" style={{ background: 'rgba(240, 185, 11, 0.12)' }}>
          <Eye className="w-4 h-4" style={{ color: '#F0B90B' }} />
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

      <div
        className="flex items-center justify-between px-3 py-2 rounded-lg"
        style={{ border: '1px solid #2B3139', background: '#0B0E11' }}
      >
        <div className="flex items-center gap-2">
          <Info className="w-4 h-4" style={{ color: '#848E9C' }} />
          <span className="text-xs" style={{ color: '#EAECEF' }}>
            {t('enabled')}
          </span>
        </div>
        <label className="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            className="sr-only peer"
            checked={current.enabled}
            onChange={(e) => !disabled && set({ enabled: e.target.checked })}
            disabled={disabled}
          />
          <div className="w-10 h-5 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:bg-yellow-500 transition-colors" />
          <div className="absolute left-0.5 top-0.5 w-4 h-4 bg-white rounded-full transition-transform peer-checked:translate-x-5" />
        </label>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div className="px-3 py-2 rounded-lg" style={{ border: '1px solid #2B3139', background: '#0B0E11' }}>
          <div className="text-xs mb-2" style={{ color: '#848E9C' }}>
            {t('maxSymbols')}
          </div>
          <input
            type="number"
            min={1}
            max={20}
            value={current.max_symbols}
            onChange={(e) => !disabled && set({ max_symbols: Number(e.target.value) || 1 })}
            disabled={disabled}
            className="w-full px-2 py-1 rounded text-xs"
            style={{ background: '#111827', border: '1px solid #2B3139', color: '#EAECEF' }}
          />
        </div>

        <div className="px-3 py-2 rounded-lg" style={{ border: '1px solid #2B3139', background: '#0B0E11' }}>
          <div className="text-xs mb-2" style={{ color: '#848E9C' }}>
            {t('concurrency')}
          </div>
          <input
            type="number"
            min={1}
            max={8}
            value={current.render_concurrency}
            onChange={(e) => !disabled && set({ render_concurrency: Number(e.target.value) || 1 })}
            disabled={disabled}
            className="w-full px-2 py-1 rounded text-xs"
            style={{ background: '#111827', border: '1px solid #2B3139', color: '#EAECEF' }}
          />
        </div>
      </div>

      <div className="px-3 py-2 rounded-lg" style={{ border: '1px solid #2B3139', background: '#0B0E11' }}>
        <div className="flex items-center gap-2 mb-2">
          <ImageIcon className="w-4 h-4" style={{ color: '#848E9C' }} />
          <div className="text-xs" style={{ color: '#848E9C' }}>
            {t('imageSize')}
          </div>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <div className="text-[11px] mb-1" style={{ color: '#848E9C' }}>
              {t('width')}
            </div>
            <input
              type="number"
              min={320}
              max={2048}
              value={current.image_width}
              onChange={(e) => !disabled && set({ image_width: Number(e.target.value) || 1600 })}
              disabled={disabled}
              className="w-full px-2 py-1 rounded text-xs"
              style={{ background: '#111827', border: '1px solid #2B3139', color: '#EAECEF' }}
            />
          </div>
          <div>
            <div className="text-[11px] mb-1" style={{ color: '#848E9C' }}>
              {t('height')}
            </div>
            <input
              type="number"
              min={240}
              max={2048}
              value={current.image_height}
              onChange={(e) => !disabled && set({ image_height: Number(e.target.value) || 1396 })}
              disabled={disabled}
              className="w-full px-2 py-1 rounded text-xs"
              style={{ background: '#111827', border: '1px solid #2B3139', color: '#EAECEF' }}
            />
          </div>
        </div>
      </div>

      <div className="px-3 py-2 rounded-lg" style={{ border: '1px solid #2B3139', background: '#0B0E11' }}>
        <div className="text-xs mb-2" style={{ color: '#848E9C' }}>
          {t('timeframes')}
        </div>
        <div className="flex gap-2 flex-wrap">
          {['5m', '15m', '1h'].map((tf) => (
            <button
              key={tf}
              type="button"
              onClick={() => !disabled && toggleTimeframe(tf)}
              className="px-2 py-1 rounded text-xs transition-colors"
              style={{
                border: '1px solid #2B3139',
                background: (current.timeframes || []).includes(tf) ? 'rgba(240,185,11,0.15)' : '#111827',
                color: '#EAECEF',
              }}
              disabled={disabled}
            >
              {tf}
            </button>
          ))}
        </div>
      </div>

      <div className="px-3 py-2 rounded-lg" style={{ border: '1px solid #2B3139', background: '#0B0E11' }}>
        <div className="text-xs mb-2" style={{ color: '#848E9C' }}>
          {t('indicators')}
        </div>
        <div className="grid grid-cols-2 gap-2">
          {indicatorItems.map((item) => (
            <label
              key={String(item.key)}
              className="flex items-center gap-2 px-2 py-2 rounded-md cursor-pointer"
              style={{ border: '1px solid #2B3139', background: '#111827' }}
            >
              <input
                type="checkbox"
                className="w-4 h-4 rounded border-gray-600 text-yellow-500 focus:ring-2 focus:ring-yellow-500/50"
                checked={Boolean(current.indicators?.[item.key])}
                onChange={(e) => !disabled && setIndicators({ [item.key]: e.target.checked })}
                disabled={disabled}
              />
              <span className="text-xs" style={{ color: '#EAECEF' }}>
                {item.label}
              </span>
            </label>
          ))}
        </div>
      </div>

      <div className="text-xs flex items-center gap-2" style={{ color: '#848E9C' }}>
        <Info className="w-4 h-4" />
        {t('note')}
      </div>
    </div>
  )
}
