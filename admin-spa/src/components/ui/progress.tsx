'use client'

import * as React from 'react'
import { Progress as ProgressPrimitive } from 'radix-ui'

import { cn } from '@/lib/utils'

function Progress({
  className,
  value,
  ...props
}: React.ComponentProps<typeof ProgressPrimitive.Root>) {
  const indicatorStyle = React.useMemo(
    () => ({ transform: `translateX(-${100 - (value || 0)}%)` }),
    [value],
  )

  return (
    <ProgressPrimitive.Root
      data-slot='progress'
      className={cn(
        'relative flex h-1 w-full items-center overflow-x-hidden rounded-full bg-muted',
        className,
      )}
      {...props}
    >
      <ProgressPrimitive.Indicator
        data-slot='progress-indicator'
        className='size-full flex-1 bg-primary transition-transform duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]'
        style={indicatorStyle}
      />
    </ProgressPrimitive.Root>
  )
}

export { Progress }
