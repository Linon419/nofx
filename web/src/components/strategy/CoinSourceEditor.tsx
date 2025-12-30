import { useState } from 'react'
import { Plus, X, Database, TrendingUp, List, Link, AlertCircle } from 'lucide-react'
import type { CoinSourceConfig } from '../../types'

// Default API URLs for data sources
const DEFAULT_COIN_POOL_API_URL = 'http://nofxaios.com:30006/api/ai500/list?auth=cm_568c67eae410d912c54c'
const DEFAULT_OI_TOP_API_URL = 'http://nofxaios.com:30006/api/oi/top-ranking?limit=20&duration=1h&auth=cm_568c67eae410d912c54c'
const DEFAULT_OTC_TOP_API_URL = 'http://168.138.207.11:3080/api/public/top-otc-crypto'

interface CoinSourceEditorProps {
  config: CoinSourceConfig
  onChange: (config: CoinSourceConfig) => void
  disabled?: boolean
  language: string
}

export function CoinSourceEditor({
  config,
  onChange,
  disabled,
  language,
}: CoinSourceEditorProps) {
  const [newCoin, setNewCoin] = useState('')

  const t = (key: string) => {
    const translations: Record<string, Record<string, string>> = {
      sourceType: { zh: '鏁版嵁鏉ユ簮绫诲瀷', en: 'Source Type' },
      static: { zh: '闈欐€佸垪琛?, en: 'Static List' },
      coinpool: { zh: 'AI500 鏁版嵁婧?, en: 'AI500 Data Provider' },
      oi_top: { zh: 'OI Top', en: 'OI Top' },

      otc_top: { zh: 'OTC Top', en: 'OTC Top' },
      mixed: { zh: '娣峰悎妯″紡', en: 'Mixed Mode' },
      staticCoins: { zh: '鑷畾涔夊竵绉?, en: 'Custom Coins' },
      addCoin: { zh: '娣诲姞甯佺', en: 'Add Coin' },
      useCoinPool: { zh: '鍚敤 AI500 鏁版嵁婧?, en: 'Enable AI500 Data Provider' },
      coinPoolLimit: { zh: '鏁版嵁婧愭暟閲忎笂闄?, en: 'Data Provider Limit' },
      coinPoolApiUrl: { zh: 'AI500 API URL', en: 'AI500 API URL' },
      coinPoolApiUrlPlaceholder: { zh: '杈撳叆 AI500 鏁版嵁婧?API 鍦板潃...', en: 'Enter AI500 data provider API URL...' },
      useOITop: { zh: '鍚敤 OI Top 鏁版嵁', en: 'Enable OI Top' },
      oiTopLimit: { zh: 'OI Top 鏁伴噺涓婇檺', en: 'OI Top Limit' },
      oiTopApiUrl: { zh: 'OI Top API URL', en: 'OI Top API URL' },
      oiTopApiUrlPlaceholder: { zh: 'Enter OI Top API URL...', en: 'Enter OI Top API URL...' },

      useOTCTop: { zh: 'Enable OTC Top', en: 'Enable OTC Top' },
      otcTopApiUrl: { zh: 'OTC Top API URL', en: 'OTC Top API URL' },
      otcTopApiUrlPlaceholder: { zh: 'Enter OTC Top API URL...', en: 'Enter OTC Top API URL...' },

      staticDesc: { zh: 'Manually specify trading coins', en: 'Manually specify trading coins' },
      coinpoolDesc: {
        zh: 'Use AI500 smart-filtered popular coins',
        en: 'Use AI500 smart-filtered popular coins',
      },

      oiTopDesc: {
        zh: 'Use coins with fastest OI growth',
        en: 'Use coins with fastest OI growth',
      },

      otcTopDesc: {
        zh: 'Use coins with highest OTC index',
        en: 'Use coins with highest OTC index',
      },
      mixedDesc: {
        zh: '缁勫悎澶氱鏁版嵁婧愶紝AI500 + OI Top + OTC Top + 鑷畾涔?,
        en: 'Combine multiple sources: AI500 + OI Top + OTC Top + Custom',
      },
      apiUrlRequired: { zh: '闇€瑕佸～鍐?API URL 鎵嶈兘鑾峰彇鏁版嵁', en: 'API URL required to fetch data' },
      dataSourceConfig: { zh: '鏁版嵁婧愰厤缃?, en: 'Data Source Configuration' },
      fillDefault: { zh: '濉叆榛樿', en: 'Fill Default' },
    }
    return translations[key]?.[language] || key
  }

  const sourceTypes = [
    { value: 'static', icon: List, color: '#848E9C' },
    { value: 'coinpool', icon: Database, color: '#F0B90B' },
    { value: 'oi_top', icon: TrendingUp, color: '#0ECB81' },
    { value: 'otc_top', icon: TrendingUp, color: '#f97316' },
    { value: 'mixed', icon: Database, color: '#60a5fa' },
  ] as const

  // xyz dex assets (stocks, forex, commodities) - should NOT get USDT suffix
  const xyzDexAssets = new Set([
    // Stocks
    'TSLA', 'NVDA', 'AAPL', 'MSFT', 'META', 'AMZN', 'GOOGL', 'AMD', 'COIN', 'NFLX',
    'PLTR', 'HOOD', 'INTC', 'MSTR', 'TSM', 'ORCL', 'MU', 'RIVN', 'COST', 'LLY',
    'CRCL', 'SKHX', 'SNDK',
    // Forex
    'EUR', 'JPY',
    // Commodities
    'GOLD', 'SILVER',
    // Index
    'XYZ100',
  ])

  const isXyzDexAsset = (symbol: string): boolean => {
    const base = symbol.toUpperCase().replace(/^XYZ:/, '').replace(/USDT$|USD$|-USDC$/, '')
    return xyzDexAssets.has(base)
  }

  const handleAddCoin = () => {
    if (!newCoin.trim()) return
    const symbol = newCoin.toUpperCase().trim()

    // For xyz dex assets (stocks, forex, commodities), use xyz: prefix without USDT
    let formattedSymbol: string
    if (isXyzDexAsset(symbol)) {
      // Remove xyz: prefix (case-insensitive) and any USD suffixes
      const base = symbol.replace(/^xyz:/i, '').replace(/USDT$|USD$|-USDC$/i, '')
      formattedSymbol = `xyz:${base}`
    } else {
      formattedSymbol = symbol.endsWith('USDT') ? symbol : `${symbol}USDT`
    }

    const currentCoins = config.static_coins || []
    if (!currentCoins.includes(formattedSymbol)) {
      onChange({
        ...config,
        static_coins: [...currentCoins, formattedSymbol],
      })
    }
    setNewCoin('')
  }

  const handleRemoveCoin = (coin: string) => {
    onChange({
      ...config,
      static_coins: (config.static_coins || []).filter((c) => c !== coin),
    })
  }

  return (
    <div className="space-y-6">
      {/* Source Type Selector */}
      <div>
        <label className="block text-sm font-medium mb-3" style={{ color: '#EAECEF' }}>
          {t('sourceType')}
        </label>
        <div className="grid grid-cols-4 gap-3">
          {sourceTypes.map(({ value, icon: Icon, color }) => (
            <button
              key={value}
              onClick={() =>
                !disabled &&
                onChange({ ...config, source_type: value as CoinSourceConfig['source_type'] })
              }
              disabled={disabled}
              className={`p-4 rounded-lg border transition-all ${
                config.source_type === value
                  ? 'ring-2 ring-yellow-500'
                  : 'hover:bg-white/5'
              }`}
              style={{
                background:
                  config.source_type === value
                    ? 'rgba(240, 185, 11, 0.1)'
                    : '#0B0E11',
                borderColor: '#2B3139',
              }}
            >
              <Icon className="w-6 h-6 mx-auto mb-2" style={{ color }} />
              <div className="text-sm font-medium" style={{ color: '#EAECEF' }}>
                {t(value)}
              </div>
              <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
                {t(`${value}Desc`)}
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* Static Coins */}
      {(config.source_type === 'static' || config.source_type === 'mixed') && (
        <div>
          <label className="block text-sm font-medium mb-3" style={{ color: '#EAECEF' }}>
            {t('staticCoins')}
          </label>
          <div className="flex flex-wrap gap-2 mb-3">
            {(config.static_coins || []).map((coin) => (
              <span
                key={coin}
                className="flex items-center gap-1 px-3 py-1.5 rounded-full text-sm"
                style={{ background: '#2B3139', color: '#EAECEF' }}
              >
                {coin}
                {!disabled && (
                  <button
                    onClick={() => handleRemoveCoin(coin)}
                    className="ml-1 hover:text-red-400 transition-colors"
                  >
                    <X className="w-3 h-3" />
                  </button>
                )}
              </span>
            ))}
          </div>
          {!disabled && (
            <div className="flex gap-2">
              <input
                type="text"
                value={newCoin}
                onChange={(e) => setNewCoin(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleAddCoin()}
                placeholder="BTC, ETH, SOL..."
                className="flex-1 px-4 py-2 rounded-lg"
                style={{
                  background: '#0B0E11',
                  border: '1px solid #2B3139',
                  color: '#EAECEF',
                }}
              />
              <button
                onClick={handleAddCoin}
                className="px-4 py-2 rounded-lg flex items-center gap-2 transition-colors"
                style={{ background: '#F0B90B', color: '#0B0E11' }}
              >
                <Plus className="w-4 h-4" />
                {t('addCoin')}
              </button>
            </div>
          )}
        </div>
      )}

      {/* Coin Pool Options */}
      {(config.source_type === 'coinpool' || config.source_type === 'mixed') && (
        <div className="space-y-4">
          <div className="flex items-center gap-2 mb-2">
            <Link className="w-4 h-4" style={{ color: '#F0B90B' }} />
            <span className="text-sm font-medium" style={{ color: '#EAECEF' }}>
              {t('dataSourceConfig')} - AI500
            </span>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="flex items-center gap-3 mb-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={config.use_coin_pool}
                  onChange={(e) =>
                    !disabled && onChange({ ...config, use_coin_pool: e.target.checked })
                  }
                  disabled={disabled}
                  className="w-5 h-5 rounded accent-yellow-500"
                />
                <span style={{ color: '#EAECEF' }}>{t('useCoinPool')}</span>
              </label>
              {config.use_coin_pool && (
                <div className="flex items-center gap-3">
                  <span className="text-sm" style={{ color: '#848E9C' }}>
                    {t('coinPoolLimit')}:
                  </span>
                  <input
                    type="number"
                    value={config.coin_pool_limit || 10}
                    onChange={(e) =>
                      !disabled &&
                      onChange({ ...config, coin_pool_limit: parseInt(e.target.value) || 10 })
                    }
                    disabled={disabled}
                    min={1}
                    max={100}
                    className="w-20 px-3 py-1.5 rounded"
                    style={{
                      background: '#0B0E11',
                      border: '1px solid #2B3139',
                      color: '#EAECEF',
                    }}
                  />
                </div>
              )}
            </div>
          </div>

          {config.use_coin_pool && (
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="text-sm" style={{ color: '#848E9C' }}>
                  {t('coinPoolApiUrl')}
                </label>
                {!disabled && !config.coin_pool_api_url && (
                  <button
                    type="button"
                    onClick={() => onChange({ ...config, coin_pool_api_url: DEFAULT_COIN_POOL_API_URL })}
                    className="text-xs px-2 py-1 rounded"
                    style={{ background: '#F0B90B20', color: '#F0B90B' }}
                  >
                    {t('fillDefault')}
                  </button>
                )}
              </div>
              <input
                type="url"
                value={config.coin_pool_api_url || ''}
                onChange={(e) =>
                  !disabled && onChange({ ...config, coin_pool_api_url: e.target.value })
                }
                disabled={disabled}
                placeholder={t('coinPoolApiUrlPlaceholder')}
                className="w-full px-4 py-2.5 rounded-lg font-mono text-sm"
                style={{
                  background: '#0B0E11',
                  border: '1px solid #2B3139',
                  color: '#EAECEF',
                }}
              />
              {!config.coin_pool_api_url && (
                <div className="flex items-center gap-2 mt-2">
                  <AlertCircle className="w-4 h-4" style={{ color: '#F0B90B' }} />
                  <span className="text-xs" style={{ color: '#F0B90B' }}>
                    {t('apiUrlRequired')}
                  </span>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* OI Top Options */}
      {(config.source_type === 'oi_top' || config.source_type === 'mixed') && (
        <div className="space-y-4">
          <div className="flex items-center gap-2 mb-2">
            <Link className="w-4 h-4" style={{ color: '#0ECB81' }} />
            <span className="text-sm font-medium" style={{ color: '#EAECEF' }}>
              {t('dataSourceConfig')} - OI Top
            </span>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="flex items-center gap-3 mb-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={config.use_oi_top}
                  onChange={(e) =>
                    !disabled && onChange({ ...config, use_oi_top: e.target.checked })
                  }
                  disabled={disabled}
                  className="w-5 h-5 rounded accent-yellow-500"
                />
                <span style={{ color: '#EAECEF' }}>{t('useOITop')}</span>
              </label>
              {config.use_oi_top && (
                <div className="flex items-center gap-3">
                  <span className="text-sm" style={{ color: '#848E9C' }}>
                    {t('oiTopLimit')}:
                  </span>
                  <input
                    type="number"
                    value={config.oi_top_limit || 20}
                    onChange={(e) =>
                      !disabled &&
                      onChange({ ...config, oi_top_limit: parseInt(e.target.value) || 20 })
                    }
                    disabled={disabled}
                    min={1}
                    max={50}
                    className="w-20 px-3 py-1.5 rounded"
                    style={{
                      background: '#0B0E11',
                      border: '1px solid #2B3139',
                      color: '#EAECEF',
                    }}
                  />
                </div>
              )}
            </div>
          </div>

          {config.use_oi_top && (
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="text-sm" style={{ color: '#848E9C' }}>
                  {t('oiTopApiUrl')}
                </label>
                {!disabled && !config.oi_top_api_url && (
                  <button
                    type="button"
                    onClick={() => onChange({ ...config, oi_top_api_url: DEFAULT_OI_TOP_API_URL })}
                    className="text-xs px-2 py-1 rounded"
                    style={{ background: '#0ECB8120', color: '#0ECB81' }}
                  >
                    {t('fillDefault')}
                  </button>
                )}
              </div>
              <input
                type="url"
                value={config.oi_top_api_url || ''}
                onChange={(e) =>
                  !disabled && onChange({ ...config, oi_top_api_url: e.target.value })
                }
                disabled={disabled}
                placeholder={t('oiTopApiUrlPlaceholder')}
                className="w-full px-4 py-2.5 rounded-lg font-mono text-sm"
                style={{
                  background: '#0B0E11',
                  border: '1px solid #2B3139',
                  color: '#EAECEF',
                }}
              />
              {!config.oi_top_api_url && (
                <div className="flex items-center gap-2 mt-2">
                  <AlertCircle className="w-4 h-4" style={{ color: '#F0B90B' }} />
                  <span className="text-xs" style={{ color: '#F0B90B' }}>
                    {t('apiUrlRequired')}
                  </span>
                </div>
              )}
            </div>
          )}
        </div>
      )}
    
      {/* OTC Top Options */}
      {(config.source_type === 'otc_top' || config.source_type === 'mixed') && (
        <div className="space-y-4">
          <div className="flex items-center gap-2 mb-2">
            <Link className="w-4 h-4" style={{ color: '#f97316' }} />
            <span className="text-sm font-medium" style={{ color: '#EAECEF' }}>
              {t('dataSourceConfig')} - OTC Top
            </span>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="flex items-center gap-3 mb-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={config.use_otc_top}
                  onChange={(e) =>
                    !disabled && onChange({ ...config, use_otc_top: e.target.checked })
                  }
                  disabled={disabled}
                  className="w-5 h-5 rounded accent-yellow-500"
                />
                <span style={{ color: '#EAECEF' }}>{t('useOTCTop')}</span>
              </label>
            </div>
          </div>

          {config.use_otc_top && (
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="text-sm" style={{ color: '#848E9C' }}>
                  {t('otcTopApiUrl')}
                </label>
                {!disabled && !config.otc_top_api_url && (
                  <button
                    type="button"
                    onClick={() => onChange({ ...config, otc_top_api_url: DEFAULT_OTC_TOP_API_URL })}
                    className="text-xs px-2 py-1 rounded"
                    style={{ background: '#f9731620', color: '#f97316' }}
                  >
                    {t('fillDefault')}
                  </button>
                )}
              </div>
              <input
                type="url"
                value={config.otc_top_api_url || ''}
                onChange={(e) =>
                  !disabled && onChange({ ...config, otc_top_api_url: e.target.value })
                }
                disabled={disabled}
                placeholder={t('otcTopApiUrlPlaceholder')}
                className="w-full px-4 py-2.5 rounded-lg font-mono text-sm"
                style={{
                  background: '#0B0E11',
                  border: '1px solid #2B3139',
                  color: '#EAECEF',
                }}
              />
              {!config.otc_top_api_url && (
                <div className="flex items-center gap-2 mt-2">
                  <AlertCircle className="w-4 h-4" style={{ color: '#F0B90B' }} />
                  <span className="text-xs" style={{ color: '#F0B90B' }}>
                    {t('apiUrlRequired')}
                  </span>
                </div>
              )}
            </div>
          )}
        </div>
      )}</div>
  )
}













