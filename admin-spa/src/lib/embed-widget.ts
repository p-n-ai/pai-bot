import type { EmbedThemeConfig } from './embed-config-types'

export type EmbedLanguage = 'en' | 'ms' | 'zh'
export type EmbedPosition = 'bottom-left' | 'bottom-right'

export interface EmbedTheme {
  color: string
  language: EmbedLanguage
  position: EmbedPosition
}

const embedForegroundDark = 'oklch(0.371 0 0)'
const embedForegroundLight = 'oklch(1 0 0)'

// The native color input and published embed contract require six-digit hex.
export const defaultEmbedTheme: EmbedTheme = {
  color: '#00cc85',
  language: 'en',
  position: 'bottom-right',
}

const embedCopy = {
  en: {
    greeting: 'Hi! What would you like to learn today?',
    placeholder: 'Ask a question…',
    send: 'Send',
  },
  ms: {
    greeting: 'Hai! Apakah yang ingin anda pelajari hari ini?',
    placeholder: 'Tanya soalan…',
    send: 'Hantar',
  },
  zh: {
    greeting: '你好！你今天想学习什么？',
    placeholder: '输入问题…',
    send: '发送',
  },
} as const

export function getEmbedCopy(language: EmbedLanguage) {
  return embedCopy[language]
}

export function readableForeground(color: string) {
  const match = /^#([\dA-Fa-f]{6})$/.exec(color)
  if (!match) {
    return embedForegroundLight
  }
  const channels = [0, 2, 4].map((offset) => {
    const value = Number.parseInt(match[1].slice(offset, offset + 2), 16) / 255
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4
  })
  const luminance =
    0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2]
  const darkLuminance = 0.0513
  const darkContrast =
    (Math.max(luminance, darkLuminance) + 0.05) /
    (Math.min(luminance, darkLuminance) + 0.05)
  const lightContrast = 1.05 / (luminance + 0.05)
  return darkContrast >= lightContrast
    ? embedForegroundDark
    : embedForegroundLight
}

export function readEmbedTheme(config: EmbedThemeConfig): EmbedTheme {
  return {
    color:
      config.color !== undefined && /^#[\dA-Fa-f]{6}$/.test(config.color)
        ? config.color
        : defaultEmbedTheme.color,
    language:
      config.language === 'ms' || config.language === 'zh'
        ? config.language
        : 'en',
    position:
      config.position === 'bottom-left' ? 'bottom-left' : 'bottom-right',
  }
}

export function buildEmbedSnippet({
  apiBase,
  tenantSlug,
  theme,
}: {
  apiBase: string
  tenantSlug: string
  theme: EmbedTheme
}) {
  const attributes = {
    src: `${apiBase.replace(/\/+$/, '')}/embed/pai-chat.js`,
    'data-tenant': tenantSlug,
    'data-color': theme.color,
    'data-language': theme.language,
    'data-position': theme.position,
  }

  return `<script ${Object.entries(attributes)
    .map(([name, value]) => `${name}="${escapeAttribute(value)}"`)
    .join(' ')} async></script>`
}

function escapeAttribute(value: string) {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('"', '&quot;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
}
