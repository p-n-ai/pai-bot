import { readClassProgress } from './dashboard-types'
import { isAIUsageSummary } from './ai-usage-types'
import { isGroupDetail, isGroupRecord } from './group-types'
import { isInviteRecord, isUserManagementView } from './user-management-types'
import { isJoinClassView } from './join-types'
import { isOnboardingView, isSubmitOnboardingResult } from './onboarding-types'
import { isParentSummary } from './parent-summary-types'
import { isStudentConversations, isStudentDetail } from './student-detail-types'
import { isWhatsAppStatus } from './whatsapp-types'
import { readEmbedConfig } from './embed-config-types'
import type { ClassProgress } from './dashboard-types'
import type { EmbedConfig, UpdateEmbedConfigInput } from './embed-config-types'
import type {
  AIUsageSummary,
  UpsertTokenBudgetWindowInput,
} from './ai-usage-types'
import type { CreateGroupInput, GroupDetail, GroupRecord } from './group-types'
import type { JoinClassView } from './join-types'
import type {
  OnboardingView,
  SubmitOnboardingInput,
  SubmitOnboardingResult,
} from './onboarding-types'
import type { ParentSummary } from './parent-summary-types'
import type { StudentConversation, StudentDetail } from './student-detail-types'
import type {
  InviteIssueRole,
  InviteRecord,
  UserManagementView,
} from './user-management-types'
import type { WhatsAppStatus } from './whatsapp-types'

class APIContractError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'APIContractError'
  }
}

export async function getClassProgress(
  classID = 'all-students',
  fetcher: typeof fetch = fetch,
): Promise<ClassProgress> {
  const payload = await fetchJSON(
    `/api/admin/classes/${classID}/progress`,
    fetcher,
  )

  const progress = readClassProgress(payload)

  if (!progress) {
    throw new APIContractError('Invalid class progress response')
  }

  return progress
}

export async function sendStudentNudge(
  studentID: string,
  fetcher: typeof fetch = fetch,
): Promise<void> {
  await fetchJSON(`/api/admin/students/${studentID}/nudge`, fetcher, {
    method: 'POST',
  })
}

export async function listGroups(
  type = 'class',
  fetcher: typeof fetch = fetch,
): Promise<Array<GroupRecord>> {
  const payload = await fetchJSON(
    `/api/admin/groups?type=${encodeURIComponent(type)}`,
    fetcher,
  )

  if (!Array.isArray(payload) || !payload.every(isGroupRecord)) {
    throw new APIContractError('Invalid groups response')
  }

  return payload
}

export async function getGroupDetail(
  id: string,
  fetcher: typeof fetch = fetch,
): Promise<GroupDetail> {
  const payload = await fetchJSON(`/api/admin/groups/${id}`, fetcher)

  if (!isGroupDetail(payload)) {
    throw new APIContractError('Invalid group detail response')
  }

  return payload
}

export async function createGroup(
  input: CreateGroupInput,
  fetcher: typeof fetch = fetch,
): Promise<GroupRecord> {
  const payload = await fetchJSON('/api/admin/groups', fetcher, {
    method: 'POST',
    body: JSON.stringify(input),
  })

  if (!isGroupRecord(payload)) {
    throw new APIContractError('Invalid group create response')
  }

  return payload
}

export async function getAIUsage(
  fetcher: typeof fetch = fetch,
): Promise<AIUsageSummary> {
  const payload = await fetchJSON('/api/admin/ai/usage', fetcher)

  if (!isAIUsageSummary(payload)) {
    throw new APIContractError('Invalid AI usage response')
  }

  return payload
}

export async function upsertTokenBudgetWindow(
  input: UpsertTokenBudgetWindowInput,
  fetcher: typeof fetch = fetch,
): Promise<AIUsageSummary> {
  const payload = await fetchJSON('/api/admin/ai/budget-window', fetcher, {
    method: 'POST',
    body: JSON.stringify(input),
  })

  if (!isAIUsageSummary(payload)) {
    throw new APIContractError('Invalid AI budget response')
  }

  return payload
}

export async function getUserManagement(
  fetcher: typeof fetch = fetch,
): Promise<UserManagementView> {
  const payload = await fetchJSON('/api/admin/users', fetcher)

  if (!isUserManagementView(payload)) {
    throw new APIContractError('Invalid user management response')
  }

  return payload
}

export async function getOnboarding(
  fetcher: typeof fetch = fetch,
): Promise<OnboardingView> {
  const payload = await fetchJSON('/api/admin/onboarding', fetcher)

  if (!isOnboardingView(payload)) {
    throw new APIContractError('Invalid onboarding response')
  }

  return payload
}

export async function submitOnboarding(
  input: SubmitOnboardingInput,
  fetcher: typeof fetch = fetch,
): Promise<SubmitOnboardingResult> {
  const payload = await fetchJSON('/api/admin/onboarding', fetcher, {
    method: 'POST',
    body: JSON.stringify(input),
  })

  if (!isSubmitOnboardingResult(payload)) {
    throw new APIContractError('Invalid onboarding submit response')
  }

  return payload
}

export async function getWhatsAppStatus(
  fetcher: typeof fetch = fetch,
): Promise<WhatsAppStatus> {
  const payload = await fetchJSON('/api/admin/whatsapp/status', fetcher)

  if (!isWhatsAppStatus(payload)) {
    throw new APIContractError('Invalid WhatsApp status response')
  }

  return payload
}

export async function getParentSummary(
  parentID: string,
  fetcher: typeof fetch = fetch,
): Promise<ParentSummary> {
  const payload = await fetchJSON(
    `/api/admin/parents/${encodeURIComponent(parentID)}`,
    fetcher,
  )

  if (!isParentSummary(payload)) {
    throw new APIContractError('Invalid parent summary response')
  }

  return payload
}

export async function getStudentDetail(
  studentID: string,
  fetcher: typeof fetch = fetch,
): Promise<StudentDetail> {
  const payload = await fetchJSON(
    `/api/admin/students/${encodeURIComponent(studentID)}`,
    fetcher,
  )

  if (!isStudentDetail(payload)) {
    throw new APIContractError('Invalid student detail response')
  }

  return payload
}

export async function getStudentConversations(
  studentID: string,
  fetcher: typeof fetch = fetch,
): Promise<Array<StudentConversation>> {
  const payload = await fetchJSON(
    `/api/admin/students/${encodeURIComponent(studentID)}/conversations`,
    fetcher,
  )

  if (!isStudentConversations(payload)) {
    throw new APIContractError('Invalid student conversations response')
  }

  return payload
}

export async function disconnectWhatsApp(
  fetcher: typeof fetch = fetch,
): Promise<void> {
  await fetchJSON('/api/admin/whatsapp/disconnect', fetcher, {
    method: 'POST',
  })
}

export async function getEmbedConfig(
  fetcher: typeof fetch = fetch,
): Promise<EmbedConfig> {
  const payload = await fetchJSON('/api/admin/embed/config', fetcher)
  const config = readEmbedConfig(payload)

  if (!config) {
    throw new APIContractError('Invalid embed config response')
  }

  return config
}

export async function updateEmbedConfig(
  input: UpdateEmbedConfigInput,
  fetcher: typeof fetch = fetch,
): Promise<EmbedConfig> {
  const payload = await fetchJSON('/api/admin/embed/config', fetcher, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
  const config = readEmbedConfig(payload)

  if (!config) {
    throw new APIContractError('Invalid embed config response')
  }

  return config
}

export async function addEmbedOrigin(
  origin: string,
  fetcher: typeof fetch = fetch,
): Promise<void> {
  await fetchJSON('/api/admin/embed/origins', fetcher, {
    method: 'POST',
    body: JSON.stringify({ origin }),
  })
}

export async function removeEmbedOrigin(
  origin: string,
  fetcher: typeof fetch = fetch,
): Promise<void> {
  await fetchJSON('/api/admin/embed/origins', fetcher, {
    method: 'DELETE',
    body: JSON.stringify({ origin }),
  })
}

export async function issueInvite(
  input: { email: string; role: InviteIssueRole },
  fetcher: typeof fetch = fetch,
): Promise<InviteRecord> {
  const payload = await fetchJSON('/api/admin/invites', fetcher, {
    method: 'POST',
    body: JSON.stringify(input),
  })

  if (!isInviteRecord(payload)) {
    throw new APIContractError('Invalid invite response')
  }

  return payload
}

export async function reissueInvite(
  inviteID: string,
  fetcher: typeof fetch = fetch,
): Promise<InviteRecord> {
  if (!inviteID.trim()) {
    throw new Error('Invite ID is required')
  }

  const payload = await fetchJSON(
    `/api/admin/invites/${encodeURIComponent(inviteID)}/reissue`,
    fetcher,
    {
      method: 'POST',
    },
  )

  if (!isInviteRecord(payload)) {
    throw new APIContractError('Invalid invite response')
  }

  return payload
}

export async function getJoinClass(
  slug: string,
  fetcher: typeof fetch = fetch,
): Promise<JoinClassView> {
  const response = await fetcher(`/api/join/${encodeURIComponent(slug)}`, {
    cache: 'no-store',
  })

  if (!response.ok) {
    throw new Error(await readErrorMessage(response))
  }

  const payload: unknown = await response.json()

  if (!isJoinClassView(payload)) {
    throw new APIContractError('Invalid join class response')
  }

  return payload
}

// fallow-ignore-next-line complexity
async function fetchJSON(
  path: string,
  fetcher: typeof fetch,
  init?: RequestInit,
): Promise<unknown> {
  const response = await fetcher(path, {
    ...init,
    headers: {
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
    credentials: 'include',
    cache: 'no-store',
  })

  if (!response.ok) {
    throw new Error(await readErrorMessage(response))
  }

  return response.json() as Promise<unknown>
}

async function readErrorMessage(response: Response): Promise<string> {
  const body = await response.text()

  if (!body.trim()) {
    return `Request failed: ${response.status}`
  }

  return body
}
