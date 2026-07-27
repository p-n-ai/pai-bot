import { CopyIcon, MailPlusIcon } from 'lucide-react'
import { useCallback } from 'react'

import { resolveInviteLink } from '@/components/users/user-management-invite-form'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useInviteWorkflow } from '@/hooks/use-invite-workflow'

export function ClassInvitePanel() {
  const invite = useInviteWorkflow()
  const { handleRoleChange: updateRole } = invite
  const handleRoleChange = useCallback(
    (value: string) => {
      if (value === 'teacher' || value === 'parent' || value === 'admin') {
        updateRole(value)
      }
    },
    [updateRole],
  )

  return (
    <form className='grid gap-4' onSubmit={invite.handleSubmitInvite}>
      <div className='grid gap-4 sm:grid-cols-[minmax(0,1fr)_minmax(150px,0.4fr)]'>
        <div className='grid gap-2'>
          <Label htmlFor='class-invite-email'>Email address</Label>
          <Input
            autoComplete='email'
            className='min-h-11'
            id='class-invite-email'
            name='email'
            onChange={invite.handleUpdateEmail}
            placeholder='teacher@example.com'
            required
            spellCheck={false}
            type='email'
            value={invite.email}
          />
        </div>
        <div className='grid gap-2'>
          <Label htmlFor='class-invite-role'>School role</Label>
          <Select onValueChange={handleRoleChange} value={invite.role}>
            <SelectTrigger className='min-h-11 w-full' id='class-invite-role'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='teacher'>Teacher</SelectItem>
              <SelectItem value='parent'>Parent</SelectItem>
              <SelectItem value='admin'>Admin</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
      {invite.error ? (
        <p className='text-sm text-destructive' role='alert'>
          {invite.error} Try again.
        </p>
      ) : null}
      {invite.latestInvite ? (
        <div
          className='grid gap-3 rounded-lg border border-border bg-muted/30 p-4'
          role='status'
        >
          <div>
            <p className='font-medium'>
              Invite ready for {invite.latestInvite.email}
            </p>
            <p className='text-sm text-muted-foreground'>
              Send this private activation link to the recipient.
            </p>
          </div>
          <Label htmlFor='class-invite-link'>Activation link</Label>
          <Input
            className='min-h-11 font-mono text-xs'
            id='class-invite-link'
            readOnly
            value={resolveInviteLink(invite.latestInvite)}
          />
          <div className='flex flex-wrap items-center gap-3'>
            <Button
              className='min-h-11'
              onClick={invite.handleCopyLatestInvite}
              type='button'
              variant='outline'
            >
              <CopyIcon data-icon='inline-start' />
              Copy link
            </Button>
            {invite.copyFeedback ? (
              <span className='text-sm text-muted-foreground'>
                {invite.copyFeedback}
              </span>
            ) : null}
          </div>
        </div>
      ) : null}
      <Button
        className='min-h-11 w-full sm:w-fit'
        disabled={invite.isSubmitting || !invite.email.trim()}
        type='submit'
      >
        <MailPlusIcon data-icon='inline-start' />
        {invite.isSubmitting ? 'Creating invite…' : 'Create invite'}
      </Button>
    </form>
  )
}
