export function parseActivationSearch(search: Record<string, unknown>) {
  const token = search.token

  return typeof token === 'string' ? { token } : {}
}
