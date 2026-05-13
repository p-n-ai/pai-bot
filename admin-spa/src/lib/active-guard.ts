export function runWhenActive(isActive: boolean, action: () => void) {
  if (isActive) {
    action()
  }
}
