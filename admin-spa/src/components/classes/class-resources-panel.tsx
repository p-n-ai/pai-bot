import { useCallback, useEffect, useReducer, useRef } from 'react'
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

type UploadStatus = 'idle' | 'uploading' | 'complete' | 'error'

type ResourcesState = {
  resources: Array<TeacherResource>
  loading: boolean
  loadError: string
  actionError: string
  fileError: string
  title: string
  classIDs: Array<string>
  uploadStatus: UploadStatus
}

type ResourcesAction =
  | { type: 'loaded'; resources: Array<TeacherResource> }
  | { type: 'loadFailed'; error: string }
  | { type: 'fileSelected'; error: string }
  | { type: 'titleChanged'; title: string }
  | { type: 'classToggled'; classID: string; checked: boolean }
  | { type: 'uploadStarted' }
  | {
      type: 'uploadSucceeded'
      resource: TeacherResource
      selectedClassID: string
    }
  | { type: 'uploadFailed'; error: string }
  | { type: 'actionStarted' }
  | { type: 'activeChanged'; resourceID: string; active: boolean }
  | { type: 'resourceDeleted'; resourceID: string }
  | { type: 'actionFailed'; error: string }

const uploadedAtFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short',
})

export function ClassResourcesPanel({
  groups,
  selectedClass,
}: {
  groups: Array<GroupRecord>
  selectedClass: GroupRecord
}) {
  const [state, dispatch] = useReducer(
    resourcesReducer,
    selectedClass.id,
    createInitialState,
  )
  const fileRef = useRef<File | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    let cancelled = false
    listTeacherResources(selectedClass.id)
      .then((items) => {
        if (!cancelled) {
          dispatch({ type: 'loaded', resources: items })
        }
      })
      .catch((caught: unknown) => {
        if (!cancelled) {
          dispatch({ type: 'loadFailed', error: readResourceError(caught) })
        }
      })

    return () => {
      cancelled = true
    }
  }, [selectedClass.id])

  const selectFile = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const nextFile = event.target.files?.[0] ?? null
    if (nextFile && !isAllowedTeacherResourceFile(nextFile)) {
      fileRef.current = null
      dispatch({
        type: 'fileSelected',
        error: 'Choose a PDF, DOCX, or PPTX file.',
      })
      event.target.value = ''
      return
    }

    fileRef.current = nextFile
    dispatch({ type: 'fileSelected', error: '' })
  }, [])

  const changeTitle = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    dispatch({ type: 'titleChanged', title: event.target.value })
  }, [])

  const toggleClass = useCallback(
    (classID: string, checked: boolean) => {
      if (classID === selectedClass.id) {
        return
      }
      dispatch({ type: 'classToggled', classID, checked })
    },
    [selectedClass.id],
  )

  const submitUpload = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      const file = fileRef.current
      if (!file) {
        dispatch({
          type: 'fileSelected',
          error: 'Choose a PDF, DOCX, or PPTX file.',
        })
        return
      }

      dispatch({ type: 'uploadStarted' })
      try {
        const resource = await uploadTeacherResource({
          file,
          title: state.title,
          classIDs: state.classIDs,
        })
        fileRef.current = null
        if (fileInputRef.current) {
          fileInputRef.current.value = ''
        }
        dispatch({
          type: 'uploadSucceeded',
          resource,
          selectedClassID: selectedClass.id,
        })
      } catch (caught: unknown) {
        dispatch({ type: 'uploadFailed', error: readResourceError(caught) })
      }
    },
    [selectedClass.id, state.classIDs, state.title],
  )

  const changeActive = useCallback(
    async (resource: TeacherResource) => {
      dispatch({ type: 'actionStarted' })
      try {
        await setTeacherResourceActive(
          resource.id,
          selectedClass.id,
          !resource.active,
        )
        dispatch({
          type: 'activeChanged',
          resourceID: resource.id,
          active: !resource.active,
        })
      } catch (caught: unknown) {
        dispatch({ type: 'actionFailed', error: readResourceError(caught) })
      }
    },
    [selectedClass.id],
  )

  const removeResource = useCallback(
    async (resource: TeacherResource) => {
      dispatch({ type: 'actionStarted' })
      try {
        await deleteTeacherResource(resource.id, selectedClass.id)
        dispatch({ type: 'resourceDeleted', resourceID: resource.id })
      } catch (caught: unknown) {
        dispatch({ type: 'actionFailed', error: readResourceError(caught) })
      }
    },
    [selectedClass.id],
  )

  const selectedClassIDs = new Set(state.classIDs)

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
            aria-describedby={
              state.fileError ? 'resource-file-error' : undefined
            }
            aria-invalid={Boolean(state.fileError)}
            className='min-h-11 sm:min-h-8'
            id='resource-file'
            name='resource-file'
            onChange={selectFile}
            ref={fileInputRef}
            type='file'
          />
          {state.fileError ? (
            <p className='text-sm text-destructive' id='resource-file-error'>
              {state.fileError}
            </p>
          ) : null}
        </div>
        <div className='grid gap-2'>
          <Label htmlFor='resource-title'>Display title (optional)</Label>
          <Input
            autoComplete='off'
            className='min-h-11 sm:min-h-8'
            id='resource-title'
            name='resource-title'
            onChange={changeTitle}
            placeholder='Week 3 revision'
            value={state.title}
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
                <ClassResourceGroupOption
                  checked={selectedClassIDs.has(group.id)}
                  group={group}
                  key={group.id}
                  onToggle={toggleClass}
                  required={required}
                />
              )
            })}
          </div>
        </fieldset>
        <div>
          <Button
            className='min-h-11 sm:min-h-8'
            disabled={state.uploadStatus === 'uploading'}
            type='submit'
          >
            {state.uploadStatus === 'uploading'
              ? 'Uploading resource…'
              : 'Upload resource'}
          </Button>
        </div>
        <UploadFeedback error={state.actionError} status={state.uploadStatus} />
      </form>

      <div className='mt-6 border-t border-border pt-6'>
        <ResourceList
          error={state.loadError}
          loading={state.loading}
          onChangeActive={changeActive}
          onDelete={removeResource}
          resources={state.resources}
        />
      </div>
    </SurfaceSection>
  )
}

function ClassResourceGroupOption({
  checked,
  group,
  onToggle,
  required,
}: {
  checked: boolean
  group: GroupRecord
  onToggle: (classID: string, checked: boolean) => void
  required: boolean
}) {
  const toggle = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      onToggle(group.id, event.target.checked)
    },
    [group.id, onToggle],
  )

  return (
    <div className='flex min-h-11 items-center gap-2 sm:min-h-8'>
      <input
        className='size-4 rounded border-input accent-primary'
        checked={checked}
        disabled={required}
        id={`resource-class-${group.id}`}
        onChange={toggle}
        type='checkbox'
      />
      <Label
        className='flex min-h-11 flex-1 items-center sm:min-h-8'
        htmlFor={`resource-class-${group.id}`}
      >
        {group.name}
        {required ? ' (required)' : ''}
      </Label>
    </div>
  )
}

function UploadFeedback({
  error,
  status,
}: {
  error: string
  status: 'idle' | 'uploading' | 'complete' | 'error'
}) {
  if (status === 'error') {
    return (
      <p className='text-sm text-destructive' role='alert'>
        Unable to prepare this file: {error}
      </p>
    )
  }
  if (error) {
    return (
      <p className='text-sm text-destructive' role='alert'>
        {error}
      </p>
    )
  }
  if (status === 'complete') {
    return <p role='status'>Resource is ready for tutor search.</p>
  }
  if (status === 'uploading') {
    return <p role='status'>Preparing the file for tutor search…</p>
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
    return <LoadingStatus>Loading class resources…</LoadingStatus>
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
  const changeActive = useCallback(() => {
    ignorePromise(onChangeActive(resource))
  }, [onChangeActive, resource])
  const remove = useCallback(() => {
    ignorePromise(onDelete(resource))
  }, [onDelete, resource])

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
          <dt className='font-medium text-foreground'>Searchable sections</dt>
          <dd>{formatSectionCount(resource.chunk_count)}</dd>
        </div>
      </dl>
      {resource.extraction_error ? (
        <p className='text-sm text-destructive' role='alert'>
          Unable to prepare this file: {resource.extraction_error}
        </p>
      ) : (
        <p className='text-sm text-muted-foreground'>Ready for tutor search</p>
      )}
      <div className='flex flex-wrap gap-2'>
        <Button
          className='min-h-11 sm:min-h-8'
          onClick={changeActive}
          type='button'
          variant='outline'
        >
          {resource.active ? 'Hide from tutor' : 'Show to tutor'}
        </Button>
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button
              className='min-h-11 sm:min-h-8'
              type='button'
              variant='destructive'
            >
              Delete resource
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete {displayTitle}?</AlertDialogTitle>
              <AlertDialogDescription>
                This permanently removes the file and its searchable content
                from every class where it is shared.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction onClick={remove} variant='destructive'>
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
  return caught instanceof Error
    ? caught.message
    : 'Unable to update this resource. Check your connection and try again.'
}

function formatSectionCount(count: number): string {
  return `${count} searchable ${count === 1 ? 'section' : 'sections'}`
}

function formatUploadedAt(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : uploadedAtFormatter.format(date)
}

function ignorePromise(promise: Promise<void>): void {
  promise.catch(() => undefined)
}

function createInitialState(selectedClassID: string): ResourcesState {
  return {
    resources: [],
    loading: true,
    loadError: '',
    actionError: '',
    fileError: '',
    title: '',
    classIDs: [selectedClassID],
    uploadStatus: 'idle',
  }
}

function resourcesReducer(
  state: ResourcesState,
  action: ResourcesAction,
): ResourcesState {
  switch (action.type) {
    case 'loaded':
      return { ...state, resources: action.resources, loading: false }
    case 'loadFailed':
      return { ...state, loadError: action.error, loading: false }
    case 'fileSelected':
      return {
        ...state,
        fileError: action.error,
        uploadStatus: 'idle',
      }
    case 'titleChanged':
      return { ...state, title: action.title }
    case 'classToggled':
      return {
        ...state,
        classIDs: action.checked
          ? Array.from(new Set([...state.classIDs, action.classID]))
          : state.classIDs.filter((id) => id !== action.classID),
      }
    case 'uploadStarted':
      return { ...state, actionError: '', uploadStatus: 'uploading' }
    case 'uploadSucceeded':
      return {
        ...state,
        resources: [action.resource, ...state.resources],
        actionError: '',
        fileError: '',
        title: '',
        classIDs: [action.selectedClassID],
        uploadStatus: 'complete',
      }
    case 'uploadFailed':
      return { ...state, actionError: action.error, uploadStatus: 'error' }
    case 'actionStarted':
      return { ...state, actionError: '' }
    case 'activeChanged':
      return {
        ...state,
        resources: state.resources.map((resource) =>
          resource.id === action.resourceID
            ? { ...resource, active: action.active }
            : resource,
        ),
      }
    case 'resourceDeleted':
      return {
        ...state,
        resources: state.resources.filter(
          (resource) => resource.id !== action.resourceID,
        ),
      }
    case 'actionFailed':
      return { ...state, actionError: action.error }
  }
}
