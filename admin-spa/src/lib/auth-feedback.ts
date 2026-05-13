const authErrorMessages = {
  tenant_required:
    'Multiple school accounts match this Google email. Sign in with email once, then link Google from inside the admin UI.',
  link_required:
    'We found no Google-linked admin account yet. Sign in with email once, then link Google from inside the admin UI.',
  already_linked:
    'That Google account is already linked to a different admin account.',
  flow_invalid: 'Your Google sign-in session expired. Start again.',
  domain_not_allowed:
    'That Google account is outside the allowed workspace domain for this admin.',
  provider_unavailable:
    'Google sign-in is not configured in this environment yet.',
  google_auth_failed: 'Google sign-in failed. Please try again.',
} as const

export function getAuthErrorMessage(code: string | undefined): string {
  return isAuthErrorCode(code) ? authErrorMessages[code] : ''
}

function isAuthErrorCode(
  code: string | undefined,
): code is keyof typeof authErrorMessages {
  return Boolean(code && code in authErrorMessages)
}
