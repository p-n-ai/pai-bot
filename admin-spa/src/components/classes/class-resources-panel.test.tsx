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

import { ClassResourcesPanel } from './class-resources-panel'
import type * as AdminApi from '@/lib/admin-api'
import { AdminAPIError } from '@/lib/admin-api'

const apiMocks = vi.hoisted(() => ({
  deleteTeacherResource: vi.fn(),
  listTeacherResources: vi.fn(),
  setTeacherResourceActive: vi.fn(),
  uploadTeacherResource: vi.fn(),
}))

vi.mock('@/lib/admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof AdminApi>()
  return { ...actual, ...apiMocks }
})

describe('ClassResourcesPanel', () => {
  beforeEach(() => {
    for (const mock of Object.values(apiMocks)) {
      mock.mockReset()
    }
    apiMocks.listTeacherResources.mockResolvedValue([teacherResourceFixture])
    apiMocks.uploadTeacherResource.mockResolvedValue({
      ...teacherResourceFixture,
      id: 'resource-2',
      filename: 'revision.pptx',
      title: 'Revision slides',
      source_type: 'pptx',
      class_ids: ['class-1', 'class-2'],
    })
    apiMocks.setTeacherResourceActive.mockResolvedValue(undefined)
    apiMocks.deleteTeacherResource.mockResolvedValue(undefined)
  })

  afterEach(() => {
    cleanup()
  })

  it('lists indexed resources and uploads to selected classes', async () => {
    renderPanel()

    expect(await screen.findByText('Algebra revision')).toBeInTheDocument()
    expect(screen.getByText('algebra.pdf')).toBeInTheDocument()
    expect(screen.getByText('12 page/slide chunks')).toBeInTheDocument()
    expect(screen.getByText('Ms Lim')).toBeInTheDocument()
    expect(screen.getByText('Extraction: Indexed')).toBeInTheDocument()

    fireEvent.click(screen.getByLabelText('Form 2 Algebra B'))
    fireEvent.change(screen.getByLabelText('Resource file'), {
      target: {
        files: [new File(['ppt'], 'revision.pptx')],
      },
    })
    fireEvent.change(screen.getByLabelText('Display title (optional)'), {
      target: { value: 'Revision slides' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Upload resource' }))

    await waitFor(() =>
      expect(apiMocks.uploadTeacherResource).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'Revision slides',
          classIDs: ['class-1', 'class-2'],
        }),
      ),
    )
    expect(
      await screen.findByText('Resource extracted and indexed.'),
    ).toBeInTheDocument()
  })

  it('rejects an unsupported format before upload', async () => {
    renderPanel()
    await screen.findByText('Algebra revision')

    fireEvent.change(screen.getByLabelText('Resource file'), {
      target: { files: [new File(['text'], 'notes.txt')] },
    })

    expect(
      screen.getByText('Choose a PDF, DOCX, or PPTX file.'),
    ).toBeInTheDocument()
    expect(apiMocks.uploadTeacherResource).not.toHaveBeenCalled()
  })

  it('shows upload extraction failure text', async () => {
    apiMocks.uploadTeacherResource.mockRejectedValue(
      new Error('image-only teacher resource file'),
    )
    renderPanel()
    await screen.findByText('Algebra revision')

    fireEvent.change(screen.getByLabelText('Resource file'), {
      target: { files: [new File(['pdf'], 'scan.pdf')] },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Upload resource' }))

    expect(
      await screen.findByText(
        'Extraction/indexing failed: image-only teacher resource file',
      ),
    ).toBeInTheDocument()
  })

  it('shows a clear session expiry error', async () => {
    apiMocks.listTeacherResources.mockRejectedValue(
      new AdminAPIError('Unauthorized', 401),
    )
    renderPanel()

    expect(
      await screen.findByText('Your session expired. Sign in again.'),
    ).toBeInTheDocument()
  })

  it('keeps the selected class required', async () => {
    renderPanel()
    await screen.findByText('Algebra revision')

    const selectedClass = screen.getByLabelText('Form 1 Algebra A (required)')
    expect(selectedClass).toBeChecked()
    expect(selectedClass).toBeDisabled()
  })

  it('deactivates and deletes with confirmation', async () => {
    renderPanel()
    await screen.findByText('Algebra revision')

    fireEvent.click(screen.getByRole('button', { name: 'Deactivate' }))
    await waitFor(() =>
      expect(apiMocks.setTeacherResourceActive).toHaveBeenCalledWith(
        'resource-1',
        'class-1',
        false,
      ),
    )
    expect(
      await screen.findByRole('button', { name: 'Reactivate' }),
    ).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(
      await screen.findByText('Delete Algebra revision?'),
    ).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Delete resource' }))

    await waitFor(() =>
      expect(apiMocks.deleteTeacherResource).toHaveBeenCalledWith(
        'resource-1',
        'class-1',
      ),
    )
    await waitFor(() =>
      expect(screen.queryByText('Algebra revision')).not.toBeInTheDocument(),
    )
  })

  it('shows lifecycle failures instead of stale upload success', async () => {
    apiMocks.setTeacherResourceActive.mockRejectedValue(
      new Error('Could not deactivate resource'),
    )
    apiMocks.deleteTeacherResource.mockRejectedValue(
      new Error('Could not delete resource'),
    )
    renderPanel()
    await screen.findByText('Algebra revision')

    fireEvent.change(screen.getByLabelText('Resource file'), {
      target: { files: [new File(['ppt'], 'revision.pptx')] },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Upload resource' }))
    expect(
      await screen.findByText('Resource extracted and indexed.'),
    ).toBeInTheDocument()

    fireEvent.click(screen.getAllByRole('button', { name: 'Deactivate' })[0])
    expect(
      await screen.findByText('Could not deactivate resource'),
    ).toBeInTheDocument()
    expect(
      screen.queryByText('Resource extracted and indexed.'),
    ).not.toBeInTheDocument()

    fireEvent.click(screen.getAllByRole('button', { name: 'Delete' })[0])
    fireEvent.click(
      await screen.findByRole('button', { name: 'Delete resource' }),
    )
    expect(
      await screen.findByText('Could not delete resource'),
    ).toBeInTheDocument()
    expect(
      screen.queryByText('Resource extracted and indexed.'),
    ).not.toBeInTheDocument()
  })
})

const groups = [
  {
    id: 'class-1',
    join_code: 'ALG-1',
    member_count: 1,
    name: 'Form 1 Algebra A',
    type: 'class',
  },
  {
    id: 'class-2',
    join_code: 'ALG-2',
    member_count: 1,
    name: 'Form 2 Algebra B',
    type: 'class',
  },
] as const

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

function renderPanel() {
  return render(
    <ClassResourcesPanel groups={[...groups]} selectedClass={groups[0]} />,
  )
}
