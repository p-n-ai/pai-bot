export interface GroupRecord {
  id: string
  name: string
  type: 'class' | 'study_group'
  description?: string
  syllabus?: string
  subject?: string
  cadence?: string
  join_code: string
  member_count: number
  created_at?: string
  updated_at?: string
}

export interface GroupMemberRecord {
  id: string
  name: string
  role: 'member' | 'leader' | 'teacher'
  channel: string
  mastery: number
}

export interface GroupDetail extends GroupRecord {
  members: Array<GroupMemberRecord>
}

export interface CreateGroupInput {
  name: string
  type: 'class'
  syllabus: string
  cadence?: string
  subject: 'Mathematics'
}

export function isGroupRecord(value: unknown): value is GroupRecord {
  return isRecord(value) && hasGroupShape(value)
}

export function isGroupDetail(value: unknown): value is GroupDetail {
  return isGroupRecord(value) && hasMembers(value)
}

function hasGroupShape(value: Record<string, unknown>): boolean {
  return [hasID, hasName, hasType, hasJoinCode, hasMemberCount].every((check) =>
    check(value),
  )
}

function hasMembers(value: GroupRecord): value is GroupDetail {
  return (
    'members' in value &&
    Array.isArray(value.members) &&
    value.members.every(isGroupMember)
  )
}

function isGroupMember(value: unknown): value is GroupMemberRecord {
  return isRecord(value) && hasMemberShape(value)
}

function hasMemberShape(value: Record<string, unknown>): boolean {
  return [hasID, hasName, hasMemberRole, hasChannel, hasMastery].every(
    (check) => check(value),
  )
}

function hasID(value: Record<string, unknown>): boolean {
  return typeof value.id === 'string'
}

function hasName(value: Record<string, unknown>): boolean {
  return typeof value.name === 'string'
}

function hasType(value: Record<string, unknown>): boolean {
  return value.type === 'class' || value.type === 'study_group'
}

function hasJoinCode(value: Record<string, unknown>): boolean {
  return typeof value.join_code === 'string'
}

function hasMemberCount(value: Record<string, unknown>): boolean {
  return typeof value.member_count === 'number'
}

function hasMemberRole(value: Record<string, unknown>): boolean {
  return (
    value.role === 'member' ||
    value.role === 'leader' ||
    value.role === 'teacher'
  )
}

function hasChannel(value: Record<string, unknown>): boolean {
  return typeof value.channel === 'string'
}

function hasMastery(value: Record<string, unknown>): boolean {
  return typeof value.mastery === 'number'
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
