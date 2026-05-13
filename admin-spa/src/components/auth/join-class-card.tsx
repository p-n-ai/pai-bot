import type { JoinClassView } from '@/lib/join-types'

export function JoinClassCard({ joinClass }: { joinClass: JoinClassView }) {
  return (
    <section className='join-card' aria-label='Class join link'>
      <p className='eyebrow'>Class join link</p>
      <h1>{joinClass.class_name}</h1>
      <p className='lede'>
        {joinClass.school_name} is ready to use this join route for{' '}
        <strong>{joinClass.class_slug}</strong>. Student enrollment and invite
        completion still land in the next slice.
      </p>
      <dl className='detail-list'>
        <div>
          <dt>School</dt>
          <dd>{joinClass.school_name}</dd>
        </div>
        <div>
          <dt>Curriculum</dt>
          <dd>{joinClass.curriculum_label}</dd>
        </div>
      </dl>
    </section>
  )
}
