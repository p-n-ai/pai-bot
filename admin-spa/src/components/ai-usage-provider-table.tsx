import { useMemo } from 'react'

import type { AIUsageSummary } from '@/lib/ai-usage-types'
import { DataTable } from '@/components/data-table'
import { StatePanel } from '@/components/state-panel'
import { formatCompactNumber } from '@/lib/ai-usage-view'

const providerColumns = ['Provider', 'Model', 'Messages', 'Tokens']

export function AIUsageProviderTable({
  providers,
}: {
  providers: AIUsageSummary['providers']
}) {
  const rows = useMemo(() => getProviderRows(providers), [providers])

  if (providers.length === 0) {
    return (
      <StatePanel title='No provider traffic recorded'>
        Provider rows will populate after the first successful AI requests for
        this tenant.
      </StatePanel>
    )
  }

  return <DataTable columns={providerColumns} rows={rows} />
}

function getProviderRows(providers: AIUsageSummary['providers']) {
  return providers.map((provider) => ({
    id: `${provider.provider}-${provider.model}`,
    cells: [
      { key: 'provider', value: provider.provider },
      { key: 'model', value: provider.model },
      { key: 'messages', value: formatCompactNumber(provider.messages) },
      { key: 'tokens', value: formatCompactNumber(provider.total_tokens) },
    ],
  }))
}
