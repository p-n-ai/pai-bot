import { useCallback, useState } from 'react'

import type { SubmitOnboardingResult } from '@/lib/onboarding-types'
import {
  CopyIcon,
  ExternalLinkIcon,
  PencilIcon,
} from '@/components/ui/pandai-icons'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { OnboardingTeacherInvites } from '@/components/onboarding/onboarding-teacher-invites'
import { StatePanel } from '@/components/shared/state-panel'

export function OnboardingResultActions({
  onEditSetup,
  result,
}: {
  onEditSetup: () => void
  result: SubmitOnboardingResult
}) {
  const [copyFeedback, setCopyFeedback] = useState('')
  const handleCopy = useCallback(() => {
    navigator.clipboard
      .writeText(result.join_link)
      .then(() => {
        setCopyFeedback('Join link copied.')
      })
      .catch(() => {
        setCopyFeedback(
          'Unable to copy the join link. Copy it from the field and try again.',
        )
      })
  }, [result.join_link])
  const handleOpen = useCallback(() => {
    window.open(result.join_link, '_blank', 'noopener,noreferrer')
  }, [result.join_link])

  return (
    <StatePanel title='Your first class is ready'>
      <div className='grid gap-3.5'>
        <p>Share this link with students joining {result.class_name}.</p>
        <p>{result.class_name}</p>
        <p>{result.school_name}</p>
        <Input
          aria-label='Student join link'
          readOnly
          value={result.join_link}
        />
        <div className='mt-6 flex flex-wrap gap-3'>
          <Button onClick={handleCopy} type='button' variant='outline'>
            <CopyIcon data-icon='inline-start' />
            Copy join link
          </Button>
          <Button onClick={handleOpen} type='button' variant='outline'>
            <ExternalLinkIcon data-icon='inline-start' />
            Open join page
          </Button>
          <Button onClick={onEditSetup} type='button' variant='ghost'>
            <PencilIcon data-icon='inline-start' />
            Edit setup
          </Button>
        </div>
        {copyFeedback ? (
          <p className='text-muted-foreground'>{copyFeedback}</p>
        ) : null}
        <OnboardingTeacherInvites />
      </div>
    </StatePanel>
  )
}
