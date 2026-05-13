import { CopyIcon, ExternalLinkIcon, PencilIcon } from 'lucide-react'
import { useCallback, useState } from 'react'

import type { SubmitOnboardingResult } from '@/lib/onboarding-types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { OnboardingTeacherInvites } from '@/components/onboarding-teacher-invites'
import { StatePanel } from '@/components/state-panel'

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
        setCopyFeedback('Copied')
      })
      .catch(() => {
        setCopyFeedback('Could not copy the join link.')
      })
  }, [result.join_link])
  const handleOpen = useCallback(() => {
    window.open(result.join_link, '_blank', 'noopener,noreferrer')
  }, [result.join_link])

  return (
    <StatePanel title='Welcome to your classroom.'>
      <div className='grid gap-3.5'>
        <p>Your first class is ready to share.</p>
        <p>{result.class_name}</p>
        <p>{result.school_name}</p>
        <p>Students</p>
        <Input aria-label='Join link' readOnly value={result.join_link} />
        <div className='mt-6 flex flex-wrap gap-3'>
          <Button onClick={handleCopy} type='button' variant='outline'>
            <CopyIcon data-icon='inline-start' />
            Copy
          </Button>
          <Button onClick={handleOpen} type='button' variant='outline'>
            <ExternalLinkIcon data-icon='inline-start' />
            Open
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
