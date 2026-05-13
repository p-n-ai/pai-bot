import { DownloadIcon } from 'lucide-react'

import { adminExportLinks } from '@/lib/export-links'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/data-table'

export function ExportPanel() {
  const rows = adminExportLinks.map((item) => ({
    id: item.href,
    cells: [
      {
        key: 'name',
        value: (
          <span className='flex items-center gap-3'>
            <span className='flex size-9 items-center justify-center rounded-md border border-border bg-muted'>
              <item.icon className='size-4 text-muted-foreground' />
            </span>
            <span className='grid gap-0.5'>
              <h2 className='m-0 text-sm font-medium'>{item.title}</h2>
              <span className='text-sm text-muted-foreground'>
                {item.description}
              </span>
            </span>
          </span>
        ),
      },
      {
        key: 'status',
        value: <Badge variant='secondary'>Ready</Badge>,
      },
      {
        key: 'scope',
        value: 'Current tenant',
      },
      {
        key: 'action',
        value: (
          <Button asChild size='sm' variant='outline'>
            <a aria-label={`Download ${item.title}`} download href={item.href}>
              <DownloadIcon data-icon='inline-start' />
              Download
            </a>
          </Button>
        ),
      },
    ],
  }))

  return (
    <section aria-label='Tenant export downloads' className='mt-6'>
      <div className='overflow-x-auto rounded-lg border border-border bg-card'>
        <DataTable
          className='min-w-[760px]'
          columns={['Export', 'Status', 'Scope', '']}
          rows={rows}
        />
      </div>
    </section>
  )
}
