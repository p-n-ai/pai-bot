import { PlusIcon } from 'lucide-react'
import { useCallback, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useSubmitStatus } from '@/hooks/use-submit-status'
import { createGroup } from '@/lib/admin-api'

interface ClassCreateFormState {
  cadence: string
  error: string
  handleCadenceChange: (event: ChangeEvent<HTMLInputElement>) => void
  handleNameChange: (event: ChangeEvent<HTMLInputElement>) => void
  handleSubmit: (event: FormEvent<HTMLFormElement>) => void
  handleSyllabusChange: (value: string) => void
  isPending: boolean
  name: string
  syllabus: string
}

interface CreateClassAction {
  beginSubmit: () => void
  cadence: string
  finishSubmit: () => void
  name: string
  onCreated: () => void
  setCadence: (value: string) => void
  setError: (value: string) => void
  setName: (value: string) => void
  syllabus: string
}

export function ClassCreateForm({ onCreated }: { onCreated: () => void }) {
  const form = useClassCreateForm(onCreated)
  const submitDisabled = isCreateClassDisabled(form)

  return (
    <form
      className='grid grid-cols-[minmax(180px,1fr)_minmax(150px,0.7fr)_minmax(150px,0.7fr)_auto] items-end gap-3'
      onSubmit={form.handleSubmit}
    >
      <ClassCreateFields form={form} />
      {form.error ? (
        <p className='text-muted-foreground'>{form.error}</p>
      ) : null}
      <Button disabled={submitDisabled} type='submit'>
        <PlusIcon data-icon='inline-start' />
        {getCreateClassLabel(form)}
      </Button>
    </form>
  )
}

function isCreateClassDisabled(form: ClassCreateFormState): boolean {
  return form.isPending || !form.name.trim()
}

function getCreateClassLabel(form: ClassCreateFormState): string {
  return form.isPending ? 'Creating...' : 'Create class'
}

function useClassCreateForm(onCreated: () => void): ClassCreateFormState {
  const [name, setName] = useState('')
  const [syllabus, setSyllabus] = useState('KSSM Form 1')
  const [cadence, setCadence] = useState('')
  const { beginSubmit, error, finishSubmit, isPending, setError } =
    useSubmitStatus()

  const handleSubmit = useCreateClassSubmit({
    beginSubmit,
    cadence,
    finishSubmit,
    name,
    onCreated,
    setCadence,
    setError,
    setName,
    syllabus,
  })
  const handleNameChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      setName(event.target.value)
    },
    [],
  )
  const handleCadenceChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      setCadence(event.target.value)
    },
    [],
  )

  return {
    cadence,
    error,
    handleCadenceChange,
    handleNameChange,
    handleSubmit,
    handleSyllabusChange: setSyllabus,
    isPending,
    name,
    syllabus,
  }
}

function ClassCreateFields({ form }: { form: ClassCreateFormState }) {
  return (
    <>
      <ClassNameField form={form} />
      <SyllabusField form={form} />
      <CadenceField form={form} />
    </>
  )
}

function ClassNameField({ form }: { form: ClassCreateFormState }) {
  return (
    <div className='flex flex-col gap-2'>
      <Label htmlFor='class-name'>Class name</Label>
      <Input
        id='class-name'
        onChange={form.handleNameChange}
        placeholder='Form 1 Algebra A'
        required
        value={form.name}
      />
    </div>
  )
}

function SyllabusField({ form }: { form: ClassCreateFormState }) {
  return (
    <div className='flex flex-col gap-2'>
      <Label>Syllabus</Label>
      <Select onValueChange={form.handleSyllabusChange} value={form.syllabus}>
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value='KSSM Form 1'>KSSM Form 1</SelectItem>
          <SelectItem value='KSSM Form 2'>KSSM Form 2</SelectItem>
          <SelectItem value='KSSM Form 3'>KSSM Form 3</SelectItem>
        </SelectContent>
      </Select>
    </div>
  )
}

function CadenceField({ form }: { form: ClassCreateFormState }) {
  return (
    <div className='flex flex-col gap-2'>
      <Label htmlFor='class-cadence'>Cadence</Label>
      <Input
        id='class-cadence'
        onChange={form.handleCadenceChange}
        placeholder='Mon, Wed, Fri'
        value={form.cadence}
      />
    </div>
  )
}

function useCreateClassSubmit({
  beginSubmit,
  cadence,
  finishSubmit,
  name,
  onCreated,
  setCadence,
  setError,
  setName,
  syllabus,
}: CreateClassAction) {
  return useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      beginSubmit()
      createClass({
        cadence,
        name,
        onCreated,
        setCadence,
        setError,
        setName,
        syllabus,
      }).finally(finishSubmit)
    },
    [
      beginSubmit,
      cadence,
      finishSubmit,
      name,
      onCreated,
      setCadence,
      setError,
      setName,
      syllabus,
    ],
  )
}

async function createClass({
  cadence,
  name,
  onCreated,
  setCadence,
  setError,
  setName,
  syllabus,
}: Omit<CreateClassAction, 'beginSubmit' | 'finishSubmit'>) {
  try {
    await createGroup({
      cadence: cadence.trim(),
      name: name.trim(),
      subject: 'Mathematics',
      syllabus,
      type: 'class',
    })
    setName('')
    setCadence('')
    onCreated()
  } catch (caught) {
    setError(caught instanceof Error ? caught.message : 'Class create failed')
  }
}
