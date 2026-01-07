import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { api } from '../../lib/api'
import type { MaintenanceCleanupConfig } from '../../types'
import { t, type Language } from '../../i18n/translations'

interface MaintenanceCleanupModalProps {
  isOpen: boolean
  onClose: () => void
  language: Language
}

export function MaintenanceCleanupModal({
  isOpen,
  onClose,
  language,
}: MaintenanceCleanupModalProps) {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [config, setConfig] = useState<MaintenanceCleanupConfig | null>(null)
  const [enabled, setEnabled] = useState(true)

  useEffect(() => {
    if (!isOpen) return
    setLoading(true)
    api
      .getMaintenanceCleanupConfig()
      .then((cfg) => {
        setConfig(cfg)
        setEnabled(!!cfg.enabled)
      })
      .catch((err) => {
        console.error('Failed to load cleanup config:', err)
        toast.error(language === 'zh' ? '获取清理配置失败' : 'Failed to load cleanup config')
      })
      .finally(() => setLoading(false))
  }, [isOpen, language])

  if (!isOpen) return null

  const handleSave = async () => {
    setSaving(true)
    try {
      const updated = await api.updateMaintenanceCleanupConfig({ enabled })
      setConfig(updated)
      toast.success(language === 'zh' ? '已保存' : 'Saved')
      onClose()
    } catch (err) {
      console.error('Failed to save cleanup config:', err)
      toast.error(language === 'zh' ? '保存失败' : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  const supported = config?.supported ?? true
  const days = config?.days ?? 3

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm p-4 overflow-y-auto">
      <div
        className="bg-[#1E2329] border border-[#2B3139] rounded-xl shadow-2xl max-w-xl w-full my-8"
        style={{ maxHeight: 'calc(100vh - 4rem)' }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between p-6 border-b border-[#2B3139] bg-gradient-to-r from-[#1E2329] to-[#252B35] sticky top-0 z-10 rounded-t-xl">
          <div>
            <h2 className="text-xl font-bold text-[#EAECEF]">
              {language === 'zh' ? '自动清理(SQLite)' : 'Auto Cleanup (SQLite)'}
            </h2>
            <p className="text-sm text-[#848E9C] mt-1">
              {language === 'zh'
                ? `仅保留最近 ${days} 天的 AI 决策记录与权益快照`
                : `Keep only the last ${days} days of AI decisions and equity snapshots`}
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-[#848E9C] hover:text-[#EAECEF] transition-colors"
            disabled={saving}
          >
            ✕
          </button>
        </div>

        <div className="p-6 space-y-4 overflow-y-auto">
          <div className="flex items-center justify-between p-4 bg-[#0B0E11] border border-[#2B3139] rounded-lg">
            <div>
              <div className="text-[#EAECEF] font-medium">
                {language === 'zh' ? '启用自动清理' : 'Enable auto cleanup'}
              </div>
              <div className="text-xs text-[#848E9C] mt-1">
                {supported
                  ? language === 'zh'
                    ? '默认开启；关闭后数据库会持续增长'
                    : 'Enabled by default; disabling will allow DB growth'
                  : language === 'zh'
                    ? '当前数据库不是 SQLite，自动清理不生效'
                    : 'Current DB is not SQLite; auto cleanup is unavailable'}
              </div>
            </div>

            <label className={`inline-flex items-center cursor-pointer ${!supported ? 'opacity-50 cursor-not-allowed' : ''}`}>
              <input
                type="checkbox"
                checked={enabled}
                onChange={(e) => setEnabled(e.target.checked)}
                disabled={!supported || loading || saving}
                className="sr-only peer"
              />
              <div className="relative w-11 h-6 bg-[#2B3139] peer-focus:outline-none rounded-full peer peer-checked:bg-[#0ECB81]">
                <div className="absolute top-[2px] left-[2px] bg-white w-5 h-5 rounded-full transition-all peer-checked:translate-x-full" />
              </div>
            </label>
          </div>

          <div className="bg-[#0B0E11] border border-[#2B3139] rounded-lg p-4">
            <div className="text-sm text-[#EAECEF] font-medium">
              {language === 'zh' ? '说明' : 'Notes'}
            </div>
            <div className="text-xs text-[#848E9C] mt-2 space-y-1">
              <div>
                {language === 'zh'
                  ? '• 清理内容：AI 决策记录(decision_records) + 权益快照(trader_equity_snapshots)'
                  : '• Scope: AI decision records (decision_records) + equity snapshots (trader_equity_snapshots)'}
              </div>
              <div>
                {language === 'zh'
                  ? '• 清理频率：后台每小时检查一次；有删除时会不定期执行 VACUUM'
                  : '• Frequency: checks hourly; may VACUUM after deletions'}
              </div>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-[#2B3139] bg-[#1E2329] rounded-b-xl">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded bg-[#2B3139] text-[#EAECEF] hover:bg-[#3A4250] transition-colors"
            disabled={saving}
          >
            {t('cancel', language)}
          </button>
          <button
            onClick={handleSave}
            className="px-4 py-2 rounded bg-[#F0B90B] text-black font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
            disabled={saving || loading || !supported}
          >
            {t('save', language)}
          </button>
        </div>
      </div>
    </div>
  )
}
