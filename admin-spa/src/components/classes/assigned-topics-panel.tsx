import { Button } from '@/components/ui/button'

const assignedTopics = [
  {
    id: 'algebraic-expressions',
    progress: 0.76,
    status: 'Live',
    title: 'Algebraic Expressions',
  },
  {
    id: 'linear-equations',
    progress: 0.42,
    status: 'Starting next',
    title: 'Linear Equations',
  },
  {
    id: 'fractions',
    progress: 0,
    status: 'Upcoming',
    title: 'Fractions',
  },
] as const

export function AssignedTopicsPanel() {
  return (
    <section className='state-panel'>
      <div>
        <p className='eyebrow'>Assigned topics</p>
        <h2>Topic assignment</h2>
        <p>Topic assignment (coming soon)</p>
      </div>
      <div className='topic-list'>
        {assignedTopics.map((topic) => (
          <TopicProgressItem key={topic.id} topic={topic} />
        ))}
      </div>
      <Button disabled type='button' variant='outline'>
        Assign topics to class (coming soon)
      </Button>
    </section>
  )
}

function TopicProgressItem({
  topic,
}: {
  topic: (typeof assignedTopics)[number]
}) {
  return (
    <div className='topic-progress-row'>
      <div>
        <strong>{topic.title}</strong>
        <span>{topic.status}</span>
      </div>
      <span>{Math.round(topic.progress * 100)}%</span>
    </div>
  )
}
