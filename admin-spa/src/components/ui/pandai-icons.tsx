import type { ComponentType, SVGProps } from 'react'

import type { PandaiIconName } from '@/components/ui/pandai-icon'
import { PandaiIcon } from '@/components/ui/pandai-icon'
import { cn } from '@/lib/utils'

type PandaiIconProps = Omit<SVGProps<SVGSVGElement>, 'children' | 'name'>

/** Component contract for an icon backed by the Pandai Student outline sprite. */
export type PandaiIconComponent = ComponentType<PandaiIconProps>

function createPandaiIcon(
  name: PandaiIconName,
  defaultClassName?: string,
): PandaiIconComponent {
  function MappedPandaiIcon({ className, ...props }: PandaiIconProps) {
    return (
      <PandaiIcon
        className={cn(defaultClassName, className)}
        name={name}
        {...props}
      />
    )
  }

  MappedPandaiIcon.displayName = `PandaiIcon(${name})`
  return MappedPandaiIcon
}

/** Pandai equivalents for the semantic icon names used throughout the admin SPA. */
export const {
  AlertCircleIcon,
  ArrowLeftIcon,
  BoltIcon,
  Building2Icon,
  Check,
  CheckIcon,
  ChevronDownIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronUpIcon,
  CircleCheckIcon,
  Copy,
  CopyIcon,
  DatabaseBackupIcon,
  DownloadIcon,
  ExternalLinkIcon,
  FileSpreadsheetIcon,
  GaugeIcon,
  GraduationCapIcon,
  HeartHandshakeIcon,
  InfoIcon,
  Loader2Icon,
  MailCheckIcon,
  MailPlusIcon,
  MailWarningIcon,
  MessageCircle,
  MessageCircleIcon,
  MessagesSquareIcon,
  MinusIcon,
  MoreHorizontalIcon,
  OctagonXIcon,
  PencilIcon,
  PlusIcon,
  RotateCcwIcon,
  SchoolIcon,
  SearchIcon,
  ShieldCheckIcon,
  SparklesIcon,
  TriangleAlertIcon,
  TrophyIcon,
  UsersIcon,
  UsersRoundIcon,
  X,
  XIcon,
} = {
  AlertCircleIcon: createPandaiIcon('alert-triangle'),
  ArrowLeftIcon: createPandaiIcon('arrow-left'),
  BoltIcon: createPandaiIcon('zap'),
  Building2Icon: createPandaiIcon('home'),
  Check: createPandaiIcon('check'),
  CheckIcon: createPandaiIcon('check'),
  ChevronDownIcon: createPandaiIcon('chevron-down'),
  ChevronLeftIcon: createPandaiIcon('chevron-left'),
  ChevronRightIcon: createPandaiIcon('chevron-right'),
  ChevronUpIcon: createPandaiIcon('chevron-down', 'rotate-180'),
  CircleCheckIcon: createPandaiIcon('check-circle'),
  Copy: createPandaiIcon('copy'),
  CopyIcon: createPandaiIcon('copy'),
  DatabaseBackupIcon: createPandaiIcon('layers'),
  DownloadIcon: createPandaiIcon('download'),
  ExternalLinkIcon: createPandaiIcon('external-link'),
  FileSpreadsheetIcon: createPandaiIcon('file-text'),
  GaugeIcon: createPandaiIcon('bar-chart-3'),
  GraduationCapIcon: createPandaiIcon('mortar-board'),
  HeartHandshakeIcon: createPandaiIcon('heart'),
  InfoIcon: createPandaiIcon('info'),
  Loader2Icon: createPandaiIcon('refresh-cw'),
  MailCheckIcon: createPandaiIcon('mail'),
  MailPlusIcon: createPandaiIcon('mail'),
  MailWarningIcon: createPandaiIcon('alert-triangle'),
  MessageCircle: createPandaiIcon('message-square'),
  MessageCircleIcon: createPandaiIcon('message-square'),
  MessagesSquareIcon: createPandaiIcon('message-square'),
  MinusIcon: createPandaiIcon('minus'),
  MoreHorizontalIcon: createPandaiIcon('more-horizontal'),
  OctagonXIcon: createPandaiIcon('x-circle'),
  PencilIcon: createPandaiIcon('edit-3'),
  PlusIcon: createPandaiIcon('plus'),
  RotateCcwIcon: createPandaiIcon('rotate-ccw'),
  SchoolIcon: createPandaiIcon('home'),
  SearchIcon: createPandaiIcon('search'),
  ShieldCheckIcon: createPandaiIcon('shield'),
  SparklesIcon: createPandaiIcon('star'),
  TriangleAlertIcon: createPandaiIcon('alert-triangle'),
  TrophyIcon: createPandaiIcon('award'),
  UsersIcon: createPandaiIcon('users'),
  UsersRoundIcon: createPandaiIcon('users'),
  X: createPandaiIcon('x'),
  XIcon: createPandaiIcon('x'),
}
