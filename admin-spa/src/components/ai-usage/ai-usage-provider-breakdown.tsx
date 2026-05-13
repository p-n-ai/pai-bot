import type { AIUsageView } from '@/lib/ai-usage-view'
import { AIUsageProviderSummary } from '@/components/ai-usage/ai-usage-provider-summary'
import { AIUsageProviderTable } from '@/components/ai-usage/ai-usage-provider-table'
import { SurfaceSection } from '@/components/shared/surface-section'

export function AIUsageProviderBreakdown({ view }: { view: AIUsageView }) {
  return (
    <SurfaceSection
      description='Provider and model mix for the recorded AI traffic in this workspace.'
      title='Provider breakdown'
    >
      <div className='dashboard-stack'>
        <AIUsageProviderTable providers={view.providers} />
        <AIUsageProviderSummary view={view} />
      </div>
    </SurfaceSection>
  )
}
