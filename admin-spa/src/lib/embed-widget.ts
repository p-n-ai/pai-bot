export type EmbedLanguage = 'en' | 'ms' | 'zh'
export type EmbedPosition = 'bottom-left' | 'bottom-right'

export interface EmbedTheme {
  color: string
  language: EmbedLanguage
  position: EmbedPosition
}

export const defaultEmbedTheme: EmbedTheme = {
  color: '#b45a1a',
  language: 'en',
  position: 'bottom-right',
}

export function readEmbedTheme(config: Record<string, unknown>): EmbedTheme {
  return {
    color:
      typeof config.color === 'string' && /^#[\dA-Fa-f]{6}$/.test(config.color)
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
