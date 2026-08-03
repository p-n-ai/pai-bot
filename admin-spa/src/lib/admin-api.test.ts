import { describe, expect, it, vi } from 'vitest'

import {
  addEmbedOrigin,
  createGroup,
  deleteTeacherResource,
  disconnectWhatsApp,
  getAISettings,
  getAIUsage,
  getClassProgress,
  getCodexAuthStatus,
  getEmbedConfig,
  getGroupDetail,
  getGroupLeaderboard,
  getJoinClass,
  getOnboarding,
  getParentSummary,
  getStudentConversations,
  getStudentDetail,
  getUserManagement,
  getWhatsAppStatus,
  issueInvite,
  listGroups,
  listTeacherResources,
  reissueInvite,
  removeEmbedOrigin,
  sendStudentNudge,
  setTeacherResourceActive,
  startCodexDeviceAuth,
  submitOnboarding,
  updateAISettings,
  updateEmbedConfig,
  uploadTeacherResource,
  upsertTokenBudgetWindow,
} from './admin-api'
import { aiSettingsFixture } from './ai-settings-types.test'
import { parentSummaryFixture } from './parent-summary-types.test'
import {
  studentConversationFixture,
  studentDetailFixture,
} from './student-detail-types.test'

describe('admin dashboard API', () => {
  it('uploads class resources as multipart without a JSON content type', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify(teacherResourceFixture), { status: 201 }),
      )
    const file = new File(['pdf'], 'algebra.pdf', {
      type: 'application/pdf',
    })

    await expect(
      uploadTeacherResource(
        {
          file,
          title: 'Algebra revision',
          classIDs: ['class-1'],
        },
        fetcher,
      ),
    ).resolves.toEqual(teacherResourceFixture)

    expect(fetcher).toHaveBeenCalledOnce()
    const [path, init] = fetcher.mock.calls[0]
    expect(path).toBe('/api/admin/teacher-resources')
    expect(init).toMatchObject({
      method: 'POST',
      credentials: 'include',
      cache: 'no-store',
    })
    expect(init.headers).toBeUndefined()
    expect(init.body).toBeInstanceOf(FormData)
    expect(init.body.get('file')).toBe(file)
    expect(init.body.get('title')).toBe('Algebra revision')
    expect(init.body.getAll('class_id')).toEqual(['class-1'])
  })

  it('rejects resources outside the requested class scope', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(
        new Response(
          JSON.stringify([
            { ...teacherResourceFixture, class_ids: ['another-class'] },
          ]),
        ),
      )

    await expect(listTeacherResources('class-1', fetcher)).rejects.toThrow(
      'Invalid class resources response',
    )
  })

  it('deactivates, reactivates, and deletes resources in class scope', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }))

    await setTeacherResourceActive('resource/1', 'class 1', false, fetcher)
    await setTeacherResourceActive('resource/1', 'class 1', true, fetcher)
    await deleteTeacherResource('resource/1', 'class 1', fetcher)

    expect(fetcher).toHaveBeenNthCalledWith(
      1,
      '/api/admin/teacher-resources/resource%2F1/deactivate?class_id=class%201',
      { method: 'POST', credentials: 'include', cache: 'no-store' },
    )
    expect(fetcher).toHaveBeenNthCalledWith(
      2,
      '/api/admin/teacher-resources/resource%2F1/activate?class_id=class%201',
      { method: 'POST', credentials: 'include', cache: 'no-store' },
    )
    expect(fetcher).toHaveBeenNthCalledWith(
      3,
      '/api/admin/teacher-resources/resource%2F1?class_id=class%201',
      { method: 'DELETE', credentials: 'include', cache: 'no-store' },
    )
  })

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
        closed: false,
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

  it('reads the encoded class leaderboard and rejects malformed entries', async () => {
    const entries = [
      { user_id: 'student_2', user_name: 'Hakim', mastery_gain: 0.12, rank: 1 },
      { user_id: 'student_1', user_name: 'Alya', mastery_gain: 0.08, rank: 2 },
    ]
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(entries)))
      .mockResolvedValueOnce(
        new Response(JSON.stringify([{ ...entries[0], rank: 'first' }])),
      )

    await expect(getGroupLeaderboard('class/1', fetcher)).resolves.toEqual(
      entries,
    )
    expect(fetcher).toHaveBeenNthCalledWith(
      1,
      '/api/admin/groups/class%2F1/leaderboard',
      { credentials: 'include', cache: 'no-store', headers: {} },
    )
    await expect(getGroupLeaderboard('class/1', fetcher)).rejects.toThrow(
      'Invalid group leaderboard response',
    )
  })

  it('creates class groups with JSON body and cookie credentials', async () => {
    const group = {
      id: 'class_1',
      name: 'Form 1 Algebra',
      type: 'class',
      join_code: 'ABC123',
      member_count: 0,
      closed: false,
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
      closed: false,
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

  it('reads and updates AI settings without echoing the key', async () => {
    const updated = {
      ...aiSettingsFixture,
      providers: aiSettingsFixture.providers.map((provider) =>
        provider.type === 'api_key' && provider.name === 'openrouter'
          ? {
              ...provider,
              credential: {
                ...provider.credential,
                effective: { set: true, last4: 'z9y8' },
              },
            }
          : provider,
      ),
    }
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(aiSettingsFixture)))
      .mockResolvedValueOnce(new Response(JSON.stringify(updated)))

    await expect(getAISettings(fetcher)).resolves.toEqual(aiSettingsFixture)
    await expect(
      updateAISettings(
        {
          expectedRevision: 3,
          defaultProvider: { type: 'api_key', name: 'openrouter' },
          provider: {
            type: 'api_key',
            name: 'openrouter',
            apiKey: 'sk-or-secret',
          },
        },
        fetcher,
      ),
    ).resolves.toEqual(updated)

    expect(fetcher).toHaveBeenNthCalledWith(1, '/api/admin/ai/settings', {
      credentials: 'include',
      cache: 'no-store',
      headers: {},
    })
    expect(fetcher).toHaveBeenNthCalledWith(2, '/api/admin/ai/settings', {
      method: 'PUT',
      body: JSON.stringify({
        expectedRevision: 3,
        defaultProvider: { type: 'api_key', name: 'openrouter' },
        provider: {
          type: 'api_key',
          name: 'openrouter',
          apiKey: 'sk-or-secret',
        },
      }),
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      cache: 'no-store',
    })
  })

  it('rejects AI settings responses that break the contract', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ defaultProvider: 'openai' })),
      )

    await expect(getAISettings(fetcher)).rejects.toThrow(
      'Invalid AI settings response',
    )
  })

  it('reads and starts Codex device authorization', async () => {
    const awaiting = {
      state: 'awaiting_authorization',
      verificationUrl: 'https://auth.openai.com/codex/device',
      userCode: 'ABCD-1234',
    }
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ state: 'disconnected' })),
      )
      .mockResolvedValueOnce(new Response(JSON.stringify(awaiting)))

    await expect(getCodexAuthStatus(fetcher)).resolves.toEqual({
      state: 'disconnected',
      verificationUrl: '',
      userCode: '',
      message: '',
    })
    await expect(startCodexDeviceAuth(fetcher)).resolves.toEqual({
      ...awaiting,
      message: '',
    })

    expect(fetcher).toHaveBeenNthCalledWith(1, '/api/admin/ai/codex/auth', {
      credentials: 'include',
      cache: 'no-store',
      headers: {},
    })
    expect(fetcher).toHaveBeenNthCalledWith(
      2,
      '/api/admin/ai/codex/auth/device',
      {
        method: 'POST',
        credentials: 'include',
        cache: 'no-store',
        headers: {},
      },
    )
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
      public_embed_base_url: 'https://chat.example',
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

const teacherResourceFixture = {
  id: 'resource-1',
  filename: 'algebra.pdf',
  title: 'Algebra revision',
  source_type: 'pdf',
  media_type: 'application/pdf',
  byte_size: 2048,
  chunk_count: 12,
  active: true,
  class_ids: ['class-1'],
  created_at: '2026-07-27T12:00:00Z',
  updated_at: '2026-07-27T12:00:00Z',
  uploader_name: 'Ms Lim',
}
