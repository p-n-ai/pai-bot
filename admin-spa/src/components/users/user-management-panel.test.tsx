/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { UserManagementPanel } from './user-management-panel'
import type { UserManagementView } from '@/lib/user-management-types'
import type * as AdminAPI from '@/lib/admin-api'

const getUserManagement = vi.hoisted(() => vi.fn())
const issueInvite = vi.hoisted(() => vi.fn())
const reissueInvite = vi.hoisted(() => vi.fn())

vi.mock('@/lib/admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof AdminAPI>()

  return {
    ...actual,
    getUserManagement,
    issueInvite,
    reissueInvite,
  }
})

const data: UserManagementView = {
  summary: {
    parents: 1,
    pending_invites: 1,
    students: 2,
    teachers: 1,
    total_users: 3,
  },
  active_users: [
    {
      created_at: '2026-04-05T10:00:00Z',
      email: 'teacher@example.com',
      id: 'teacher-1',
      name: 'Teacher One',
      role: 'teacher',
      status: 'active',
      tenant_name: 'Sekolah Merdeka',
    },
  ],
  students: [
    {
      channel: 'telegram',
      created_at: '2026-04-04T10:00:00Z',
      external_id: 'tg-student-1',
      form: 'Form 1',
      id: 'student-1',
      name: 'Student One',
    },
    {
      channel: 'whatsapp',
      created_at: '2026-04-03T10:00:00Z',
      external_id: 'wa-student-2',
      form: 'Form 2',
      id: 'student-2',
      name: 'Student Two',
    },
  ],
  pending_invites: [
    {
      created_at: '2026-04-06T10:00:00Z',
      delivery_status: 'sent',
      email: 'newteacher@example.com',
      expires_at: '2026-04-13T10:00:00Z',
      id: 'invite-1',
      invited_by: 'Admin User',
      role: 'teacher',
      status: 'pending',
      tenant_name: 'Sekolah Merdeka',
    },
  ],
}

async function openInviteSheet() {
  await screen.findByRole('tab', { name: 'Admin users' })
  fireEvent.click(screen.getByRole('button', { name: 'Invite' }))
}

describe('UserManagementPanel', () => {
  const writeText = vi.fn()

  beforeEach(() => {
    getUserManagement.mockResolvedValue(data)
    issueInvite.mockReset()
    reissueInvite.mockReset()
    writeText.mockReset()
    Object.assign(navigator, {
      clipboard: {
        writeText,
      },
    })
  })

  afterEach(() => {
    cleanup()
  })

  it('copies the latest issued activation link', async () => {
    issueInvite.mockResolvedValue({
      email: 'newteacher@example.com',
      invite_token: 'invite-token',
      delivery_status: 'sent',
      role: 'teacher',
    })
    writeText.mockResolvedValue(undefined)

    render(<UserManagementPanel />)
    await openInviteSheet()

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'newteacher@example.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Send invite' }))

    await waitFor(() => {
      expect(issueInvite).toHaveBeenCalledWith({
        email: 'newteacher@example.com',
        role: 'teacher',
      })
    })

    fireEvent.click(screen.getByRole('button', { name: 'Copy link' }))

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith(
        'http://localhost:3000/activate?token=invite-token',
      )
    })
    expect(screen.getByText('Copied')).toBeInTheDocument()
  }, 10_000)

  it('explains when the latest invite email was sent', async () => {
    issueInvite.mockResolvedValue({
      email: 'newteacher@example.com',
      invite_token: 'invite-token',
      delivery_status: 'sent',
      role: 'teacher',
    })

    render(<UserManagementPanel />)
    await openInviteSheet()

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'newteacher@example.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Send invite' }))

    expect(await screen.findByText('Invite email sent.')).toBeInTheDocument()
  })

  it('labels the latest invite activation link like the source admin form', async () => {
    issueInvite.mockResolvedValue({
      email: 'newteacher@example.com',
      invite_token: 'invite-token',
      delivery_status: 'sent',
      role: 'teacher',
    })

    render(<UserManagementPanel />)
    await openInviteSheet()

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'newteacher@example.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Send invite' }))

    expect(await screen.findByLabelText('Activation link')).toHaveValue(
      'http://localhost:3000/activate?token=invite-token',
    )
  })

  it('renders and searches tenant names for active users and invites', async () => {
    render(<UserManagementPanel />)

    await screen.findByRole('tab', { name: 'Admin users' })

    expect(screen.getAllByText(/Sekolah Merdeka/u)).toHaveLength(2)

    fireEvent.change(screen.getByLabelText('Search users'), {
      target: { value: 'merdeka' },
    })

    expect(screen.getByText(/Teacher One/u)).toBeInTheDocument()
    expect(
      screen.getByText('newteacher@example.com (Sekolah Merdeka)'),
    ).toBeInTheDocument()
  })

  it('renders the pending invite lifecycle status like the source admin table', async () => {
    render(<UserManagementPanel />)

    await screen.findByRole('tab', { name: 'Admin users' })

    screen.getByText('newteacher@example.com (Sekolah Merdeka)')

    expect(screen.getAllByText('Status')).toHaveLength(2)
    expect(screen.getByText('pending')).toBeInTheDocument()
  })

  it('exposes the invite role selector through its label', async () => {
    render(<UserManagementPanel />)

    await openInviteSheet()

    expect(screen.getByLabelText('Role')).toBeInTheDocument()
  })

  it('separates admin users from the full user directory', async () => {
    render(<UserManagementPanel />)

    expect(await screen.findByText(/Teacher One/u)).toBeInTheDocument()
    expect(screen.queryByText('Student One')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('tab', { name: 'All users' }))

    expect(screen.getByText('Student One')).toBeInTheDocument()
    expect(screen.getByText('telegram')).toBeInTheDocument()
    expect(screen.getByText(/Teacher One/u)).toBeInTheDocument()
  })

  it('shows the source-admin generic copy when user management loading fails', async () => {
    getUserManagement.mockRejectedValue(new Error('database timeout'))

    render(<UserManagementPanel />)

    expect(
      await screen.findByText(
        "User management data isn't available right now.",
      ),
    ).toBeInTheDocument()
    expect(screen.queryByText('database timeout')).not.toBeInTheDocument()
  })
})
