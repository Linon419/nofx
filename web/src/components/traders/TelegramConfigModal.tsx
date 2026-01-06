import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import type { TelegramConfig, UpdateTelegramConfigRequest } from '../../types'
import { api } from '../../lib/api'
import { CryptoService } from '../../lib/crypto'
import { httpClient } from '../../lib/httpClient'
import { t, type Language } from '../../i18n/translations'

interface TelegramConfigModalProps {
  isOpen: boolean
  onClose: () => void
  language: Language
}

export function TelegramConfigModal({
  isOpen,
  onClose,
  language,
}: TelegramConfigModalProps) {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [config, setConfig] = useState<TelegramConfig | null>(null)

  const [enabled, setEnabled] = useState(false)
  const [botToken, setBotToken] = useState('')
  const [chatId, setChatId] = useState('')
  const [notifyOnOpen, setNotifyOnOpen] = useState(true)
  const [notifyOnClose, setNotifyOnClose] = useState(true)
  const [notifyOnError, setNotifyOnError] = useState(true)

  useEffect(() => {
    if (!isOpen) return
    setLoading(true)
    api
      .getTelegramConfig()
      .then((cfg) => {
        setConfig(cfg)
        setEnabled(!!cfg.enabled)
        setBotToken('')
        setChatId(cfg.chat_id || '')
        setNotifyOnOpen(cfg.notify_on_open ?? true)
        setNotifyOnClose(cfg.notify_on_close ?? true)
        setNotifyOnError(cfg.notify_on_error ?? true)
      })
      .catch((err) => {
        console.error('Failed to load telegram config:', err)
        toast.error(t('operationFailed', language))
      })
      .finally(() => setLoading(false))
  }, [isOpen, language])

  if (!isOpen) return null

  const handleTest = async () => {
    const requestBody = {
      bot_token: botToken,
      chat_id: chatId,
    }

    setTesting(true)
    try {
      await toast.promise(
        (async () => {
          const cfg = await CryptoService.fetchCryptoConfig()

          if (!cfg.transport_encryption) {
            const result = await httpClient.post(
              `/api/notifications/telegram/test`,
              requestBody
            )
            if (!result.success)
              throw new Error(result.message || 'Telegram test failed')
            return
          }

          const publicKey = await CryptoService.fetchPublicKey()
          await CryptoService.initialize(publicKey)

          const userId = localStorage.getItem('user_id') || ''
          const sessionId = sessionStorage.getItem('session_id') || ''
          const encryptedPayload = await CryptoService.encryptSensitiveData(
            JSON.stringify(requestBody),
            userId,
            sessionId
          )

          const result = await httpClient.post(
            `/api/notifications/telegram/test`,
            encryptedPayload
          )
          if (!result.success)
            throw new Error(result.message || 'Telegram test failed')
        })(),
        {
          loading: language === 'zh' ? '正在发送测试消息…' : 'Sending test message...',
          success: language === 'zh' ? '测试消息已发送' : 'Test message sent',
          error: (err) =>
            language === 'zh'
              ? `发送测试消息失败：${(err as Error)?.message || err}`
              : `Failed to send test message: ${(err as Error)?.message || err}`,
        }
      )
    } finally {
      setTesting(false)
    }
  }

  const handleSave = async () => {
    const req: UpdateTelegramConfigRequest = {
      enabled,
      bot_token: botToken,
      chat_id: chatId,
      notify_on_open: notifyOnOpen,
      notify_on_close: notifyOnClose,
      notify_on_error: notifyOnError,
    }

    setSaving(true)
    try {
      await toast.promise(api.updateTelegramConfig(req), {
        loading: language === 'zh' ? '正在更新 Telegram 配置…' : 'Updating Telegram config...',
        success: language === 'zh' ? 'Telegram 配置已更新' : 'Telegram config updated',
        error: language === 'zh' ? '更新 Telegram 配置失败' : 'Failed to update Telegram config',
      })
      onClose()
    } finally {
      setSaving(false)
    }
  }

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
              {t('telegramNotifications', language)}
            </h2>
            <div className="text-xs text-[#848E9C] mt-1">
              {language === 'zh'
                ? '按用户维度配置，交易开/平仓和错误通知'
                : 'Per-user settings for trade and error notifications'}
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-[#848E9C] hover:text-[#EAECEF] transition-colors text-2xl leading-none"
            aria-label="Close"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-5">
          {loading ? (
            <div className="text-sm text-[#848E9C]">
              {language === 'zh' ? '加载中…' : 'Loading...'}
            </div>
          ) : (
            <>
              <div className="flex items-center justify-between bg-[#0B0E11] border border-[#2B3139] rounded-lg px-4 py-3">
                <div className="text-sm text-[#EAECEF]">
                  {t('telegramEnabled', language)}
                </div>
                <label className="inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={enabled}
                    onChange={(e) => setEnabled(e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="relative w-11 h-6 bg-[#2B3139] peer-focus:outline-none rounded-full peer peer-checked:bg-[#0ECB81]">
                    <div className="absolute top-[2px] left-[2px] bg-white w-5 h-5 rounded-full transition-all peer-checked:translate-x-full" />
                  </div>
                </label>
              </div>

              <div className="space-y-2">
                <label className="text-sm text-[#848E9C] block">
                  {t('telegramBotToken', language)}
                </label>
                <input
                  type="password"
                  value={botToken}
                  onChange={(e) => setBotToken(e.target.value)}
                  placeholder={
                    config?.has_bot_token
                      ? t('telegramBotTokenAlreadySet', language)
                      : t('telegramBotTokenPlaceholder', language)
                  }
                  className="w-full px-3 py-2 rounded bg-[#0B0E11] border border-[#2B3139] text-[#EAECEF] placeholder:text-[#5A6472] focus:outline-none focus:border-[#F0B90B]/50"
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm text-[#848E9C] block">
                  {t('telegramChatId', language)}
                </label>
                <input
                  type="text"
                  value={chatId}
                  onChange={(e) => setChatId(e.target.value)}
                  placeholder={t('telegramChatIdPlaceholder', language)}
                  className="w-full px-3 py-2 rounded bg-[#0B0E11] border border-[#2B3139] text-[#EAECEF] placeholder:text-[#5A6472] focus:outline-none focus:border-[#F0B90B]/50"
                />
              </div>

              <div className="bg-[#0B0E11] border border-[#2B3139] rounded-lg p-4 space-y-3">
                <label className="flex items-center justify-between text-sm text-[#EAECEF]">
                  <span>{t('telegramNotifyOnOpen', language)}</span>
                  <input
                    type="checkbox"
                    checked={notifyOnOpen}
                    onChange={(e) => setNotifyOnOpen(e.target.checked)}
                    className="w-4 h-4 accent-[#0ECB81]"
                  />
                </label>
                <label className="flex items-center justify-between text-sm text-[#EAECEF]">
                  <span>{t('telegramNotifyOnClose', language)}</span>
                  <input
                    type="checkbox"
                    checked={notifyOnClose}
                    onChange={(e) => setNotifyOnClose(e.target.checked)}
                    className="w-4 h-4 accent-[#0ECB81]"
                  />
                </label>
                <label className="flex items-center justify-between text-sm text-[#EAECEF]">
                  <span>{t('telegramNotifyOnError', language)}</span>
                  <input
                    type="checkbox"
                    checked={notifyOnError}
                    onChange={(e) => setNotifyOnError(e.target.checked)}
                    className="w-4 h-4 accent-[#0ECB81]"
                  />
                </label>
              </div>
            </>
          )}
        </div>

        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-[#2B3139] bg-[#1E2329] rounded-b-xl">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded bg-[#2B3139] text-[#EAECEF] hover:bg-[#3A4250] transition-colors"
            disabled={saving || testing}
          >
            {language === 'zh' ? '取消' : 'Cancel'}
          </button>
          <button
            onClick={handleTest}
            className="px-4 py-2 rounded bg-[#0ECB81] text-black font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
            disabled={
              saving ||
              testing ||
              loading ||
              !chatId.trim() ||
              (!botToken.trim() && !config?.has_bot_token)
            }
          >
            {language === 'zh' ? '发送测试' : 'Send test'}
          </button>
          <button
            onClick={handleSave}
            className="px-4 py-2 rounded bg-[#F0B90B] text-black font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
            disabled={saving || testing || loading}
          >
            {language === 'zh' ? '保存' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  )
}
