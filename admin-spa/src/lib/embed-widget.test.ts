import { describe, expect, it } from 'vitest'

import {
  buildEmbedSnippet,
  defaultEmbedTheme,
  readEmbedTheme,
} from './embed-widget'

describe('embed widget helpers', () => {
  it('builds a deterministic current-origin snippet', () => {
    expect(
      buildEmbedSnippet({
        apiBase: 'https://admin.example/',
        tenantSlug: 'school-a',
        theme: {
          color: '#123456',
          language: 'ms',
          position: 'bottom-left',
        },
      }),
    ).toBe(
      '<script src="https://admin.example/embed/pai-chat.js" data-tenant="school-a" data-color="#123456" data-language="ms" data-position="bottom-left" async></script>',
    )
  })

  it('normalizes unsupported theme values to safe defaults', () => {
    expect(
      readEmbedTheme({
        color: 'red',
        language: 'xx',
        position: 'top',
      }),
    ).toEqual(defaultEmbedTheme)
  })

  it('escapes tenant attributes', () => {
    expect(
      buildEmbedSnippet({
        apiBase: 'https://admin.example',
        tenantSlug: '" onload="bad',
        theme: defaultEmbedTheme,
      }),
    ).toContain('data-tenant="&quot; onload=&quot;bad"')
  })
})
