export function formatTopicLabel(topicID: string): string {
  return topicID.split('-').filter(Boolean).map(capitalize).join(' ')
}

function capitalize(value: string): string {
  return `${value.charAt(0).toUpperCase()}${value.slice(1)}`
}
