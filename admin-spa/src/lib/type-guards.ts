export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

export function isString(value: unknown): value is string {
  return typeof value === 'string'
}

export function isNumber(value: unknown): value is number {
  return typeof value === 'number'
}

export function hasStringProps(
  record: Record<string, unknown>,
  keys: Array<string>,
): boolean {
  return keys.every((key) => isString(record[key]))
}

export function hasNumberProps(
  record: Record<string, unknown>,
  keys: Array<string>,
): boolean {
  return keys.every((key) => isNumber(record[key]))
}

export function optionalStringOrNull(value: unknown): boolean {
  return value === null || isString(value)
}
