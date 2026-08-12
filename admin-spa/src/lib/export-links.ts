import type { PandaiIconComponent } from '@/components/ui/pandai-icons'
import {
  DatabaseBackupIcon,
  FileSpreadsheetIcon,
  MessagesSquareIcon,
} from '@/components/ui/pandai-icons'

export interface AdminExportLink {
  description: string
  href: string
  icon: PandaiIconComponent
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
      'Download conversation transcripts with message details for audit and review.',
    href: '/api/admin/export/conversations',
    icon: MessagesSquareIcon,
    title: 'Conversations JSON',
  },
  {
    description:
      'Download topic mastery and review schedules for spreadsheet analysis.',
    href: '/api/admin/export/progress',
    icon: DatabaseBackupIcon,
    title: 'Progress CSV',
  },
]
