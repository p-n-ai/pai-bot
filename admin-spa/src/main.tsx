import { Agentation } from 'agentation'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { AdminApp } from './app'
import { AuthProvider } from './auth-provider'
import './styles.css'
import { TooltipProvider } from '@/components/ui/tooltip'

const rootElement = document.getElementById('root')
const isPublicStatusPage =
  window.location.pathname === '/health' ||
  window.location.pathname === '/health/'

if (!rootElement) {
  throw new Error('Root element not found')
}

createRoot(rootElement).render(
  <StrictMode>
    <AuthProvider readSession={!isPublicStatusPage}>
      <TooltipProvider>
        <AdminApp />
      </TooltipProvider>
    </AuthProvider>
    {import.meta.env.DEV ? (
      <Agentation endpoint='http://localhost:4747' />
    ) : null}
  </StrictMode>,
)
