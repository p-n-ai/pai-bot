interface LoginEnv {
  NEXT_PUBLIC_PAI_AUTH_GOOGLE_LOGIN_ENABLED?: string
  VITE_PAI_AUTH_GOOGLE_LOGIN_ENABLED?: string
}

export function isGoogleLoginEnabled(env?: LoginEnv): boolean {
  const source = env ?? (import.meta.env as LoginEnv)

  return (
    source.VITE_PAI_AUTH_GOOGLE_LOGIN_ENABLED === 'true' ||
    source.NEXT_PUBLIC_PAI_AUTH_GOOGLE_LOGIN_ENABLED === 'true'
  )
}
