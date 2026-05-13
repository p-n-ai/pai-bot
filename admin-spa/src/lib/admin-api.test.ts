import { describe, expect, it, vi } from 'vitest'

import {
  addEmbedOrigin,
  createGroup,
  disconnectWhatsApp,
  getAIUsage,
  getClassProgress,
  getEmbedConfig,
  getGroupDetail,
  getJoinClass,
  getOnboarding,
  getParentSummary,
  getStudentConversations,
  getStudentDetail,
  getUserManagement,
  getWhatsAppStatus,
  issueInvite,
  listGroups,
  reissueInvite,
  removeEmbedOrigin,
  sendStudentNudge,
  submitOnboarding,
  updateEmbedConfig,
  upsertTokenBudgetWindow,
} from './admin-api'
import { parentSummaryFixture } from './parent-summary-types.test'
import {
  studentConversationFixture,
  studentDetailFixture,
} from './student-detail-types.test'

describe('admin dashboard API', () => {
  it('reads class progress with cookie credentials', async () => {
    const progress = {
      topic_ids: ['linear-equations'],
      students: [
        {
          id: 'student_1',
          name: 'Alya',
          topics: {
            'linear-equations': 0.83,
          },
        },
      ],
    }
    const fetcher = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(progress), {
        status: 200,
      }),
    )

    await expect(getClassProgress('all-students', fetcher)).resolves.toEqual(
      progress,
    )

    expect(fetcher).toHaveBeenCalledWith(
      '/api/admin/classes/all-students/progress',
      {
        credentials: 'include',
        cache: 'no-store',
        headers: {},
      },
    )
  })

  it('posts student nudges without a GET fallback', async () => {
    const fetcher = vi.fn().mockResolvedValue(
      new Response('{}', {
        status: 200,
      }),
    )

    await expect(
      sendStudentNudge('student_1', fetcher),
    ).resolves.toBeUndefined()

    expect(fetcher).toHaveBeenCalledWith(
      '/api/admin/students/student_1/nudge',
      {
        method: 'POST',
        credentials: 'include',
        cache: 'no-store',
        headers: {},
      },
    )
  })

  it('lists class groups through the admin groups endpoint', async () => {
    const groups = [
      {
        id: 'class_1',
        name: 'Form 1 Algebra',
        type: 'class',
        join_code: 'ABC123',
        member_count: 4,
      },
    ]
    const fetcher = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(groups), {
        status: 200,
      }),
    )

    await expect(listGroups('class', fetcher)).resolves.toEqual(groups)

    expect(fetcher).toHaveBeenCalledWith('/api/admin/groups?type=class', {
      credentials: 'include',
      cache: 'no-store',
      headers: {},
    })
  })

  it('creates class groups with JSON body and cookie credentials', async () => {
    const group = {
      id: 'class_1',
      name: 'Form 1 Algebra',
      type: 'class',
      join_code: 'ABC123',
      member_count: 0,
    }
    const fetcher = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(group), {
        status: 200,
      }),
    )

    await expect(
      createGroup(
        {
          name: 'Form 1 Algebra',
          type: 'class',
          syllabus: 'KSSM Form 1',
          subject: 'Mathematics',
        },
        fetcher,
      ),
    ).resolves.toEqual(group)

    expect(fetcher).toHaveBeenCalledWith('/api/admin/groups', {
      method: 'POST',
      body: JSON.stringify({
        name: 'Form 1 Algebra',
        type: 'class',
        syllabus: 'KSSM Form 1',
        subject: 'Mathematics',
      }),
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      cache: 'no-store',
    })
  })

  it('reads selected group roster details', async () => {
    const detail = {
      id: 'class_1',
      name: 'Form 1 Algebra',
      type: 'class',
      join_code: 'ABC123',
      member_count: 1,
      members: [
        {
          id: 'student_1',
          name: 'Alya',
          role: 'member',
          channel: 'telegram',
          mastery: 0.72,
        },
      ],
    }
    const fetcher = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(detail), {
        status: 200,
      }),
    )

    await expect(getGroupDetail('class_1', fetcher)).resolves.toEqual(detail)
  })

  it('reads AI usage through the admin API with a typed contract', async () => {
    const usage = {
      total_messages: 12,
      total_input_tokens: 3000,
      total_output_tokens: 2000,
      providers: [
        {
          provider: 'openai',
          model: 'gpt-4.1-mini',
          messages: 12,
          input_tokens: 3000,
          output_tokens: 2000,
          total_tokens: 5000,
        },
      ],
    }
    const fetcher = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(usage), {
        status: 200,
      }),
    )

    await expect(getAIUsage(fetcher)).resolves.toEqual(usage)

    expect(fetcher).toHaveBeenCalledWith('/api/admin/ai/usage', {
      credentials: 'include',
      cache: 'no-store',
      headers: {},
    })
  })

  it('upserts token budget windows with JSON body and cookie credentials', async () => {
    const usage = {
      total_messages: 12,
      total_input_tokens: 3000,
      total_output_tokens: 2000,
      providers: [],
      budget_limit_tokens: 300000,
      budget_used_tokens: 5000,
      budget_remaining_tokens: 295000,
      budget_period_start: '2026-04-01',
      budget_period_end: '2026-04-30',
    }
    const fetcher = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(usage), {
        status: 200,
      }),
    )

    await expect(
      upsertTokenBudgetWindow(
        {
          budget_tokens: 300000,
          period_start: '2026-04-01',
          period_end: '2026-04-30',
        },
        fetcher,
      ),
    ).resolves.toEqual(usage)

    expect(fetcher).toHaveBeenCalledWith('/api/admin/ai/budget-window', {
      method: 'POST',
      body: JSON.stringify({
        budget_tokens: 300000,
        period_start: '2026-04-01',
        period_end: '2026-04-30',
      }),
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      cache: 'no-store',
    })
  })

  it('reads user management data with a typed contract', async () => {
    const view = {
      summary: {
        teachers: 1,
        parents: 1,
        pending_invites: 1,
        students: 0,
        total_users: 3,
      },
      active_users: [
        {
          id: 'teacher_1',
          name: 'Teacher One',
          email: 'teacher@example.com',
          role: 'teacher',
          status: 'active',
          created_at: '2026-05-08T00:00:00Z',
        },
      ],
      pending_invites: [
        {
          id: 'invite_1',
          email: 'parent@example.com',
          role: 'parent',
          status: 'pending',
          expires_at: '2026-05-15T00:00:00Z',
          created_at: '2026-05-08T00:00:00Z',
          invited_by: 'Admin',
        },
      ],
      students: [],
    }
    const fetcher = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify(view)))

    await expect(getUserManagement(fetcher)).resolves.toEqual(view)
  })

  it('issues and reissues invites through admin endpoints', async () => {
    const invite = {
      email: 'teacher@example.com',
      invite_token: 'token_1',
      role: 'teacher',
    }
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(invite)))
      .mockResolvedValueOnce(new Response(JSON.stringify(invite)))

    await expect(
      issueInvite(
        {
          email: 'teacher@example.com',
          role: 'teacher',
        },
        fetcher,
      ),
    ).resolves.toEqual(invite)
    await expect(reissueInvite('invite_1', fetcher)).resolves.toEqual(invite)

    expect(fetcher).toHaveBeenNthCalledWith(1, '/api/admin/invites', {
      method: 'POST',
      body: JSON.stringify({
        email: 'teacher@example.com',
        role: 'teacher',
      }),
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      cache: 'no-store',
    })
    expect(fetcher).toHaveBeenNthCalledWith(
      2,
      '/api/admin/invites/invite_1/reissue',
      {
        method: 'POST',
        credentials: 'include',
        cache: 'no-store',
        headers: {},
      },
    )
  })

  it('loads public join class data without admin credentials', async () => {
    const joinClass = {
      class_id: 'class_1',
      class_name: 'Form 1 Algebra',
      class_slug: 'form-1-algebra',
      curriculum_label: 'KSSM Form 1',
      school_name: 'Sekolah Harapan',
    }
    const fetcher = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(joinClass), {
        status: 200,
      }),
    )

    await expect(getJoinClass('form 1/algebra', fetcher)).resolves.toEqual(
      joinClass,
    )
    expect(fetcher).toHaveBeenCalledWith('/api/join/form%201%2Falgebra', {
      cache: 'no-store',
    })
  })

  it('reads and submits onboarding through cookie-backed admin endpoints', async () => {
    const view = {
      tenant_id: 'tenant_1',
      tenant_name: 'Sekolah Harapan',
      onboarding: null,
    }
    const result = {
      class_id: 'class_1',
      school_name: 'Sekolah Harapan',
      class_name: 'Form 1 Mathematics',
      join_link: 'https://app.test/join/form-1-mathematics',
      save_status: 'saved',
    }
    const input = {
      school_name: 'Sekolah Harapan',
      curriculum: {
        syllabus_id: 'kssm-algebra',
        label: 'KSSM Algebra',
      },
      first_class: {
        name: 'Form 1 Mathematics',
        slug: 'form-1-mathematics',
      },
      bot_setup: {
        preset: 'guided-practice',
      },
    }
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(view)))
      .mockResolvedValueOnce(new Response(JSON.stringify(result)))

    await expect(getOnboarding(fetcher)).resolves.toEqual(view)
    await expect(submitOnboarding(input, fetcher)).resolves.toEqual(result)
    expect(fetcher).toHaveBeenNthCalledWith(1, '/api/admin/onboarding', {
      credentials: 'include',
      cache: 'no-store',
      headers: {},
    })
    expect(fetcher).toHaveBeenNthCalledWith(2, '/api/admin/onboarding', {
      method: 'POST',
      body: JSON.stringify(input),
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      cache: 'no-store',
    })
  })

  it('reads and disconnects WhatsApp through admin endpoints', async () => {
    const status = {
      connected: false,
      qr_image: 'data:image/png;base64,abc',
    }
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(status)))
      .mockResolvedValueOnce(new Response('{}'))

    await expect(getWhatsAppStatus(fetcher)).resolves.toEqual(status)
    await expect(disconnectWhatsApp(fetcher)).resolves.toBeUndefined()
    expect(fetcher).toHaveBeenNthCalledWith(1, '/api/admin/whatsapp/status', {
      credentials: 'include',
      cache: 'no-store',
      headers: {},
    })
    expect(fetcher).toHaveBeenNthCalledWith(
      2,
      '/api/admin/whatsapp/disconnect',
      {
        method: 'POST',
        credentials: 'include',
        cache: 'no-store',
        headers: {},
      },
    )
  })

  it('manages embed config through tenant admin endpoints', async () => {
    const config = {
      id: '',
      tenant_id: 'tenant_1',
      enabled: false,
      allowed_origins: ['https://school.example'],
      theme_config: {
        color: '#0f172a',
      },
      created_at: undefined,
      updated_at: undefined,
    }
    const updated = {
      ...config,
      enabled: true,
    }
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(config)))
      .mockResolvedValueOnce(new Response(JSON.stringify(updated)))
      .mockResolvedValueOnce(new Response('{}'))
      .mockResolvedValueOnce(new Response('{}'))

    await expect(getEmbedConfig(fetcher)).resolves.toEqual(config)
    await expect(
      updateEmbedConfig(
        {
          enabled: true,
          theme_config: config.theme_config,
        },
        fetcher,
      ),
    ).resolves.toEqual(updated)
    await expect(
      addEmbedOrigin('https://staging.school.example', fetcher),
    ).resolves.toBeUndefined()
    await expect(
      removeEmbedOrigin('https://school.example', fetcher),
    ).resolves.toBeUndefined()

    expect(fetcher).toHaveBeenNthCalledWith(1, '/api/admin/embed/config', {
      credentials: 'include',
      cache: 'no-store',
      headers: {},
    })
    expect(fetcher).toHaveBeenNthCalledWith(2, '/api/admin/embed/config', {
      method: 'PUT',
      body: JSON.stringify({
        enabled: true,
        theme_config: config.theme_config,
      }),
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      cache: 'no-store',
    })
    expect(fetcher).toHaveBeenNthCalledWith(3, '/api/admin/embed/origins', {
      method: 'POST',
      body: JSON.stringify({
        origin: 'https://staging.school.example',
      }),
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      cache: 'no-store',
    })
    expect(fetcher).toHaveBeenNthCalledWith(4, '/api/admin/embed/origins', {
      method: 'DELETE',
      body: JSON.stringify({
        origin: 'https://school.example',
      }),
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      cache: 'no-store',
    })
  })

  it('reads parent summaries through the admin parent endpoint', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify(parentSummaryFixture)))

    await expect(getParentSummary('parent 1', fetcher)).resolves.toEqual(
      parentSummaryFixture,
    )
    expect(fetcher).toHaveBeenCalledWith('/api/admin/parents/parent%201', {
      credentials: 'include',
      cache: 'no-store',
      headers: {},
    })
  })

  it('reads student detail and conversations through admin endpoints', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(studentDetailFixture)))
      .mockResolvedValueOnce(
        new Response(JSON.stringify(studentConversationFixture)),
      )

    await expect(getStudentDetail('student 1', fetcher)).resolves.toEqual(
      studentDetailFixture,
    )
    await expect(
      getStudentConversations('student 1', fetcher),
    ).resolves.toEqual(studentConversationFixture)
    expect(fetcher).toHaveBeenNthCalledWith(
      1,
      '/api/admin/students/student%201',
      {
        credentials: 'include',
        cache: 'no-store',
        headers: {},
      },
    )
    expect(fetcher).toHaveBeenNthCalledWith(
      2,
      '/api/admin/students/student%201/conversations',
      {
        credentials: 'include',
        cache: 'no-store',
        headers: {},
      },
    )
  })
})
