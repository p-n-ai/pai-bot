import { describe, expect, it } from 'vitest'

import { isInviteRecord, isUserManagementView } from './user-management-types'

describe('user management type guards', () => {
  it('accepts the user management API shape', () => {
    expect(
      isUserManagementView({
        summary: {
          parents: 1,
          pending_invites: 1,
          students: 1,
          teachers: 1,
          total_users: 3,
        },
        active_users: [
          {
            created_at: '2026-05-08T00:00:00Z',
            email: 'teacher@example.com',
            id: 'teacher_1',
            name: 'Teacher One',
            role: 'teacher',
            status: 'active',
          },
        ],
        students: [
          {
            channel: 'telegram',
            created_at: '2026-05-08T00:00:00Z',
            external_id: 'tg_1',
            form: 'Form 1',
            id: 'student_1',
            name: 'Student One',
          },
        ],
        pending_invites: [
          {
            created_at: '2026-05-08T00:00:00Z',
            email: 'parent@example.com',
            expires_at: '2026-05-15T00:00:00Z',
            id: 'invite_1',
            invited_by: 'Admin',
            role: 'parent',
            status: 'pending',
          },
        ],
      }),
    ).toBe(true)
  })

  it('rejects malformed user management API shapes', () => {
    expect(
      isUserManagementView({
        summary: {
          parents: '1',
          pending_invites: 1,
          students: 1,
          teachers: 1,
          total_users: 3,
        },
        active_users: [],
        students: [],
        pending_invites: [],
      }),
    ).toBe(false)
  })

  it('accepts invite responses with optional delivery fields', () => {
    expect(
      isInviteRecord({
        delivery_status: 'sent',
        email: 'teacher@example.com',
        invite_token: 'token_1',
        role: 'teacher',
      }),
    ).toBe(true)
  })
})
