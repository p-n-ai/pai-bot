import { DownloadIcon } from 'lucide-react'

import { adminExportLinks } from '@/lib/export-links'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

export function ExportPanel() {
  return (
    <section aria-label='School record downloads' className='mt-6'>
      <div className='overflow-hidden rounded-2xl border border-[var(--admin-line)] bg-[var(--admin-surface)]'>
        <ul className='m-0 list-none divide-y divide-[var(--admin-line)] p-0'>
          {adminExportLinks.map((item) => (
            <li
              className='grid gap-4 px-4 py-5 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center sm:px-6'
              key={item.href}
            >
              <div className='flex min-w-0 items-start gap-4'>
                <span className='flex size-10 shrink-0 items-center justify-center rounded-xl bg-[var(--admin-surface-muted)] text-muted-foreground'>
                  <item.icon aria-hidden className='size-4' />
                </span>
                <div className='min-w-0'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <h2 className='m-0 text-sm font-semibold text-foreground'>
                      {item.title}
                    </h2>
                    <Badge variant='secondary'>Ready</Badge>
                  </div>
                  <p className='mt-1 mb-0 text-sm leading-6 text-muted-foreground'>
                    {item.description}
                  </p>
                </div>
              </div>
              <div className='flex items-center justify-between gap-4 sm:justify-end'>
                <span className='text-xs font-medium tracking-wide text-muted-foreground uppercase'>
                  Current school
                </span>
                <Button asChild size='sm' variant='outline'>
                  <a
                    aria-label={`Download ${item.title}`}
                    download
                    href={item.href}
                  >
                    <DownloadIcon data-icon='inline-start' />
                    Download file
                  </a>
                </Button>
              </div>
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}
