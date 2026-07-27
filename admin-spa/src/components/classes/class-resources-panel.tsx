import { useCallback, useEffect, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'

import type { GroupRecord } from '@/lib/group-types'
import type { TeacherResource } from '@/lib/teacher-resource-types'
import {
  AdminAPIError,
  deleteTeacherResource,
  listTeacherResources,
  setTeacherResourceActive,
  uploadTeacherResource,
} from '@/lib/admin-api'
import { isAllowedTeacherResourceFile } from '@/lib/teacher-resource-types'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { LoadingStatus, StatePanel } from '@/components/shared/state-panel'
import { SurfaceSection } from '@/components/shared/surface-section'

export function ClassResourcesPanel({
  groups,
  selectedClass,
}: {
  groups: Array<GroupRecord>
  selectedClass: GroupRecord
}) {
  const [resources, setResources] = useState<Array<TeacherResource>>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [actionError, setActionError] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [fileError, setFileError] = useState('')
  const [title, setTitle] = useState('')
  const [classIDs, setClassIDs] = useState<Array<string>>([selectedClass.id])
  const [uploadStatus, setUploadStatus] = useState<
    'idle' | 'uploading' | 'complete' | 'error'
  >('idle')

  useEffect(() => {
    let cancelled = false
    setClassIDs([selectedClass.id])
    setResources([])
    setFile(null)
    setFileError('')
    setTitle('')
    setUploadStatus('idle')
    setActionError('')
    setLoading(true)
    setLoadError('')
    listTeacherResources(selectedClass.id)
      .then((items) => {
        if (!cancelled) {
          setResources(items)
        }
      })
      .catch((caught: unknown) => {
        if (!cancelled) {
          setLoadError(readResourceError(caught))
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [selectedClass.id])

  const selectFile = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const nextFile = event.target.files?.[0] ?? null
    if (nextFile && !isAllowedTeacherResourceFile(nextFile)) {
      setFile(null)
      setFileError('Choose a PDF, DOCX, or PPTX file.')
      event.target.value = ''
      return
    }

    setFile(nextFile)
    setFileError('')
    setUploadStatus('idle')
  }, [])

  const toggleClass = useCallback(
    (classID: string, checked: boolean) => {
      if (classID === selectedClass.id) {
        return
      }
      setClassIDs((current) =>
        checked
          ? Array.from(new Set([...current, classID]))
          : current.filter((id) => id !== classID),
      )
    },
    [selectedClass.id],
  )

  const submitUpload = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      const form = event.currentTarget
      if (!file) {
        setFileError('Choose a PDF, DOCX, or PPTX file.')
        return
      }

      setActionError('')
      setUploadStatus('uploading')
      try {
        const resource = await uploadTeacherResource({
          file,
          title,
          classIDs,
        })
        setResources((current) => [resource, ...current])
        setFile(null)
        setTitle('')
        setClassIDs([selectedClass.id])
        setUploadStatus('complete')
        const input = form.elements.namedItem('resource-file')
        if (input instanceof HTMLInputElement) {
          input.value = ''
        }
      } catch (caught: unknown) {
        setActionError(readResourceError(caught))
        setUploadStatus('error')
      }
    },
    [classIDs, file, selectedClass.id, title],
  )

  const changeActive = useCallback(
    async (resource: TeacherResource) => {
      setActionError('')
      try {
        await setTeacherResourceActive(
          resource.id,
          selectedClass.id,
          !resource.active,
        )
        setResources((current) =>
          current.map((item) =>
            item.id === resource.id
              ? { ...item, active: !resource.active }
              : item,
          ),
        )
      } catch (caught: unknown) {
        setActionError(readResourceError(caught))
      }
    },
    [selectedClass.id],
  )

  const removeResource = useCallback(
    async (resource: TeacherResource) => {
      setActionError('')
      try {
        await deleteTeacherResource(resource.id, selectedClass.id)
        setResources((current) =>
          current.filter((item) => item.id !== resource.id),
        )
      } catch (caught: unknown) {
        setActionError(readResourceError(caught))
      }
    },
    [selectedClass.id],
  )

  return (
    <SurfaceSection
      description='Upload lesson material for this class. Supported formats: PDF, DOCX, and PPTX.'
      title='Class resources'
    >
      <form className='grid gap-4' onSubmit={submitUpload}>
        <div className='grid gap-2'>
          <Label htmlFor='resource-file'>Resource file</Label>
          <Input
            accept='.pdf,.docx,.pptx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.openxmlformats-officedocument.presentationml.presentation'
            aria-describedby={fileError ? 'resource-file-error' : undefined}
            aria-invalid={Boolean(fileError)}
            id='resource-file'
            name='resource-file'
            onChange={selectFile}
            type='file'
          />
          {fileError ? (
            <p className='text-sm text-destructive' id='resource-file-error'>
              {fileError}
            </p>
          ) : null}
        </div>
        <div className='grid gap-2'>
          <Label htmlFor='resource-title'>Display title (optional)</Label>
          <Input
            id='resource-title'
            onChange={(event) => setTitle(event.target.value)}
            placeholder='e.g. Week 3 revision'
            value={title}
          />
        </div>
        <fieldset className='grid gap-2'>
          <legend className='text-sm font-medium'>Available to classes</legend>
          <p className='text-sm text-muted-foreground'>
            The selected class is required. You can share this resource with
            additional classes.
          </p>
          <div className='grid gap-2 sm:grid-cols-2'>
            {groups.map((group) => {
              const required = group.id === selectedClass.id
              return (
                <div className='flex items-center gap-2' key={group.id}>
                  <input
                    className='size-4 rounded border-input accent-primary'
                    checked={classIDs.includes(group.id)}
                    disabled={required}
                    id={`resource-class-${group.id}`}
                    onChange={(event) =>
                      toggleClass(group.id, event.target.checked)
                    }
                    type='checkbox'
                  />
                  <Label htmlFor={`resource-class-${group.id}`}>
                    {group.name}
                    {required ? ' (required)' : ''}
                  </Label>
                </div>
              )
            })}
          </div>
        </fieldset>
        <div>
          <Button disabled={uploadStatus === 'uploading'} type='submit'>
            {uploadStatus === 'uploading'
              ? 'Uploading resource…'
              : 'Upload resource'}
          </Button>
        </div>
        <UploadFeedback error={actionError} status={uploadStatus} />
      </form>

      <div className='mt-6 border-t border-border pt-6'>
        <ResourceList
          error={loadError}
          loading={loading}
          onChangeActive={changeActive}
          onDelete={removeResource}
          resources={resources}
        />
      </div>
    </SurfaceSection>
  )
}

function UploadFeedback({
  error,
  status,
}: {
  error: string
  status: 'idle' | 'uploading' | 'complete' | 'error'
}) {
  if (status === 'complete') {
    return <p role='status'>Resource extracted and indexed.</p>
  }
  if (status === 'error') {
    return (
      <p className='text-sm text-destructive' role='alert'>
        Extraction/indexing failed: {error}
      </p>
    )
  }
  if (status === 'uploading') {
    return <p role='status'>Extracting pages or slides and indexing chunks.</p>
  }
  if (error) {
    return (
      <p className='text-sm text-destructive' role='alert'>
        {error}
      </p>
    )
  }
  return null
}

function ResourceList({
  error,
  loading,
  onChangeActive,
  onDelete,
  resources,
}: {
  error: string
  loading: boolean
  onChangeActive: (resource: TeacherResource) => Promise<void>
  onDelete: (resource: TeacherResource) => Promise<void>
  resources: Array<TeacherResource>
}) {
  if (loading) {
    return <LoadingStatus>Loading class resources...</LoadingStatus>
  }
  if (error) {
    return (
      <StatePanel role='alert' title='Class resources unavailable'>
        {error}
      </StatePanel>
    )
  }
  if (resources.length === 0) {
    return (
      <StatePanel title='No resources yet'>
        Upload material to make it available to this class.
      </StatePanel>
    )
  }

  return (
    <ul className='grid gap-3' aria-label='Uploaded class resources'>
      {resources.map((resource) => (
        <ResourceItem
          key={resource.id}
          onChangeActive={onChangeActive}
          onDelete={onDelete}
          resource={resource}
        />
      ))}
    </ul>
  )
}

function ResourceItem({
  onChangeActive,
  onDelete,
  resource,
}: {
  onChangeActive: (resource: TeacherResource) => Promise<void>
  onDelete: (resource: TeacherResource) => Promise<void>
  resource: TeacherResource
}) {
  const displayTitle = resource.title.trim() || resource.filename
  const uploader =
    resource.uploader_name?.trim() || resource.uploader_id?.trim()

  return (
    <li className='grid gap-3 rounded-lg border border-border p-4'>
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div>
          <h3 className='font-semibold'>{displayTitle}</h3>
          {displayTitle !== resource.filename ? (
            <p className='text-sm text-muted-foreground'>{resource.filename}</p>
          ) : null}
        </div>
        <div className='flex gap-2'>
          <Badge variant='outline'>{resource.source_type.toUpperCase()}</Badge>
          <Badge
            variant={
              resource.extraction_error
                ? 'destructive'
                : resource.active
                  ? 'default'
                  : 'secondary'
            }
          >
            {resource.extraction_error
              ? 'Error'
              : resource.active
                ? 'Active'
                : 'Inactive'}
          </Badge>
        </div>
      </div>
      <dl className='grid gap-1 text-sm text-muted-foreground sm:grid-cols-3'>
        <div>
          <dt className='font-medium text-foreground'>Uploaded</dt>
          <dd>{formatUploadedAt(resource.created_at)}</dd>
        </div>
        <div>
          <dt className='font-medium text-foreground'>Uploader</dt>
          <dd>{uploader || 'Not available'}</dd>
        </div>
        <div>
          <dt className='font-medium text-foreground'>Indexed content</dt>
          <dd>{resource.chunk_count} page/slide chunks</dd>
        </div>
      </dl>
      {resource.extraction_error ? (
        <p className='text-sm text-destructive' role='alert'>
          Extraction error: {resource.extraction_error}
        </p>
      ) : (
        <p className='text-sm text-muted-foreground'>Extraction: Indexed</p>
      )}
      <div className='flex flex-wrap gap-2'>
        <Button
          onClick={() => ignorePromise(onChangeActive(resource))}
          type='button'
          variant='outline'
        >
          {resource.active ? 'Deactivate' : 'Reactivate'}
        </Button>
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button type='button' variant='destructive'>
              Delete
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete {displayTitle}?</AlertDialogTitle>
              <AlertDialogDescription>
                This permanently removes the resource and its indexed chunks
                from every class where it is shared.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction
                onClick={() => ignorePromise(onDelete(resource))}
                variant='destructive'
              >
                Delete resource
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </li>
  )
}

function readResourceError(caught: unknown): string {
  if (caught instanceof AdminAPIError && caught.status === 401) {
    return 'Your session expired. Sign in again.'
  }
  return caught instanceof Error ? caught.message : 'Resource request failed'
}

function formatUploadedAt(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime())
    ? value
    : new Intl.DateTimeFormat(undefined, {
        dateStyle: 'medium',
        timeStyle: 'short',
      }).format(date)
}

function ignorePromise(promise: Promise<void>): void {
  promise.catch(() => undefined)
}
