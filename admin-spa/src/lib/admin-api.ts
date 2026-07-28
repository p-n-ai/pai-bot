import { readClassProgress } from './dashboard-types'
import { readAISettings } from './ai-settings-types'
import { readCodexAuthStatus } from './codex-auth-types'
import { isAIUsageSummary } from './ai-usage-types'
import { isGroupDetail, isGroupRecord } from './group-types'
import { isInviteRecord, isUserManagementView } from './user-management-types'
import { isJoinClassView } from './join-types'
import { isOnboardingView, isSubmitOnboardingResult } from './onboarding-types'
import { isParentSummary } from './parent-summary-types'
import { isStudentConversations, isStudentDetail } from './student-detail-types'
import { isWhatsAppStatus } from './whatsapp-types'
import { readEmbedConfig } from './embed-config-types'
import { isTeacherResource } from './teacher-resource-types'
import type { ClassProgress } from './dashboard-types'
import type { AISettings, UpdateAISettingsInput } from './ai-settings-types'
import type { CodexAuthStatus } from './codex-auth-types'
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
import type {
  TeacherResource,
  UploadTeacherResourceInput,
} from './teacher-resource-types'

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

export async function getAISettings(
  fetcher: typeof fetch = fetch,
): Promise<AISettings> {
  const payload = await fetchJSON('/api/admin/ai/settings', fetcher)
  const settings = readAISettings(payload)

  if (!settings) {
    throw new APIContractError('Invalid AI settings response')
  }

  return settings
}

export async function updateAISettings(
  input: UpdateAISettingsInput,
  fetcher: typeof fetch = fetch,
): Promise<AISettings> {
  const payload = await fetchJSON('/api/admin/ai/settings', fetcher, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
  const settings = readAISettings(payload)

  if (!settings) {
    throw new APIContractError('Invalid AI settings response')
  }

  return settings
}

/** Returns the server-owned Codex device authorization state. */
export async function getCodexAuthStatus(
  fetcher: typeof fetch = fetch,
): Promise<CodexAuthStatus> {
  const payload = await fetchJSON('/api/admin/ai/codex/auth', fetcher)
  const status = readCodexAuthStatus(payload)
  if (!status) {
    throw new APIContractError('Invalid Codex auth response')
  }
  return status
}

/** Starts Codex device authorization on the server. */
export async function startCodexDeviceAuth(
  fetcher: typeof fetch = fetch,
): Promise<CodexAuthStatus> {
  const payload = await fetchJSON('/api/admin/ai/codex/auth/device', fetcher, {
    method: 'POST',
  })
  const status = readCodexAuthStatus(payload)
  if (!status) {
    throw new APIContractError('Invalid Codex auth response')
  }
  return status
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

export async function listTeacherResources(
  classID: string,
  fetcher: typeof fetch = fetch,
): Promise<Array<TeacherResource>> {
  const payload = await fetchJSON(
    `/api/admin/teacher-resources?class_id=${encodeURIComponent(classID)}&include_inactive=true`,
    fetcher,
  )

  if (
    !Array.isArray(payload) ||
    !payload.every(isTeacherResource) ||
    !payload.every((resource) => resource.class_ids.includes(classID))
  ) {
    throw new APIContractError('Invalid class resources response')
  }

  return payload
}

export async function uploadTeacherResource(
  input: UploadTeacherResourceInput,
  fetcher: typeof fetch = fetch,
): Promise<TeacherResource> {
  const formData = new FormData()
  formData.append('file', input.file)
  if (input.title.trim()) {
    formData.append('title', input.title.trim())
  }
  for (const classID of input.classIDs) {
    formData.append('class_id', classID)
  }

  const payload = await fetchMultipart(
    '/api/admin/teacher-resources',
    formData,
    fetcher,
  )

  if (!isTeacherResource(payload)) {
    throw new APIContractError('Invalid class resource upload response')
  }

  const responseClassIDs = new Set(payload.class_ids)
  if (!input.classIDs.every((classID) => responseClassIDs.has(classID))) {
    throw new APIContractError('Invalid class resource upload response')
  }

  return payload
}

export async function setTeacherResourceActive(
  resourceID: string,
  classID: string,
  active: boolean,
  fetcher: typeof fetch = fetch,
): Promise<void> {
  const action = active ? 'activate' : 'deactivate'
  await fetchEmpty(
    `/api/admin/teacher-resources/${encodeURIComponent(resourceID)}/${action}?class_id=${encodeURIComponent(classID)}`,
    fetcher,
    { method: 'POST' },
  )
}

export async function deleteTeacherResource(
  resourceID: string,
  classID: string,
  fetcher: typeof fetch = fetch,
): Promise<void> {
  await fetchEmpty(
    `/api/admin/teacher-resources/${encodeURIComponent(resourceID)}?class_id=${encodeURIComponent(classID)}`,
    fetcher,
    { method: 'DELETE' },
  )
}

export async function getJoinClass(
  slug: string,
  fetcher: typeof fetch = fetch,
): Promise<JoinClassView> {
  const response = await fetcher(`/api/join/${encodeURIComponent(slug)}`, {
    cache: 'no-store',
  })

  if (!response.ok) {
    throw new AdminAPIError(await readErrorMessage(response), response.status)
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
    throw new AdminAPIError(await readErrorMessage(response), response.status)
  }

  const payload: unknown = await response.json()
  return payload
}

async function fetchMultipart(
  path: string,
  body: FormData,
  fetcher: typeof fetch,
): Promise<unknown> {
  const response = await fetcher(path, {
    body,
    method: 'POST',
    credentials: 'include',
    cache: 'no-store',
  })

  if (!response.ok) {
    throw new AdminAPIError(await readErrorMessage(response), response.status)
  }

  const payload: unknown = await response.json()
  return payload
}

async function fetchEmpty(
  path: string,
  fetcher: typeof fetch,
  init: RequestInit,
): Promise<void> {
  const response = await fetcher(path, {
    ...init,
    credentials: 'include',
    cache: 'no-store',
  })

  if (!response.ok) {
    throw new AdminAPIError(await readErrorMessage(response), response.status)
  }
}

export class AdminAPIError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'AdminAPIError'
    this.status = status
  }
}

async function readErrorMessage(response: Response): Promise<string> {
  const body = await response.text()

  if (!body.trim()) {
    return `Request failed: ${response.status}`
  }

  return body
}
