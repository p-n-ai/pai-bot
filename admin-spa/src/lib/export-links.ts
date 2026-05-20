import {
  DatabaseBackupIcon,
  FileSpreadsheetIcon,
  MessagesSquareIcon,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

export interface AdminExportLink {
  description: string
  href: string
  icon: LucideIcon
  title: string
}

export const adminExportLinks: Array<AdminExportLink> = [
  {
    description:
      'Download student roster details with form, channel, tracked topics, and average mastery.',
    href: '/api/admin/export/students',
    icon: FileSpreadsheetIcon,
    title: 'Students CSV',
  },
  {
    description:
      'Download structured conversation transcripts with message metadata for audit and review workflows.',
    href: '/api/admin/export/conversations',
    icon: MessagesSquareIcon,
    title: 'Conversations JSON',
  },
  {
    description:
      'Download per-topic mastery records with scheduling fields for spreadsheet or warehouse import.',
    href: '/api/admin/export/progress',
    icon: DatabaseBackupIcon,
    title: 'Progress CSV',
  },
]
