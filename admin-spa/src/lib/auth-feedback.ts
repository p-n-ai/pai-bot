const authErrorMessages = {
  tenant_required:
    'This Google email matches more than one school. Sign in with email and choose your school.',
  link_required:
    'No admin account is linked to this Google email. Sign in with email instead.',
  already_linked:
    'This Google account is linked to another admin account. Sign in with email or contact your platform administrator.',
  flow_invalid: 'Your Google sign-in session expired. Start again.',
  domain_not_allowed:
    'Use a Google account from your school’s approved domain, or sign in with email.',
  provider_unavailable:
    'Google sign-in is unavailable. Sign in with email instead.',
  google_auth_failed:
    'Unable to sign in with Google. Try again or sign in with email instead.',
} as const

export function getAuthErrorMessage(code: string | undefined): string {
  return isAuthErrorCode(code) ? authErrorMessages[code] : ''
}

function isAuthErrorCode(
  code: string | undefined,
): code is keyof typeof authErrorMessages {
  return Boolean(code && code in authErrorMessages)
}
