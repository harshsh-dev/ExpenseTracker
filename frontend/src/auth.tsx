import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { IconBrandNotion, IconLogin2, IconWallet } from '@tabler/icons-react'
import { api, notionLoginUrl, type AuthMe } from './api/client'
import { Button, Card, Input } from './components/ui'

interface AuthState {
  enabled: boolean
  user: AuthMe['user'] | null
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthState>({
  enabled: false,
  user: null,
  logout: async () => {},
})

// AuthProvider gates the whole app: when the backend has Notion login
// configured and there is no session, it renders the login screen instead of
// the app. When auth is disabled server-side, it is a passthrough.
export function AuthProvider({ children }: { children: ReactNode }) {
  const qc = useQueryClient()

  const { data: me, isLoading } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: () => api.getMe(),
    staleTime: 60_000,
    retry: 1,
  })

  // client.ts fires this on any 401 (e.g. session expired mid-use).
  useEffect(() => {
    const onUnauthorized = () => void qc.invalidateQueries({ queryKey: ['auth', 'me'] })
    window.addEventListener('auth:unauthorized', onUnauthorized)
    return () => window.removeEventListener('auth:unauthorized', onUnauthorized)
  }, [qc])

  if (isLoading) {
    return <div className="app-loading">Loading…</div>
  }

  // If /api/auth/me itself failed (backend down), let the app render; the
  // feature/data queries will surface the real error.
  if (me?.enabled && !me.authenticated) {
    return <Login mode={me.mode} />
  }

  const logout = async () => {
    await api.logout()
    await qc.invalidateQueries()
  }

  return (
    <AuthContext.Provider
      value={{ enabled: me?.enabled ?? false, user: me?.user ?? null, logout }}
    >
      {children}
    </AuthContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthState {
  return useContext(AuthContext)
}

function Login({ mode }: { mode?: AuthMe['mode'] }) {
  const qc = useQueryClient()
  const urlError = new URLSearchParams(window.location.search).get('authError')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(urlError)
  const [busy, setBusy] = useState(false)

  async function submitPassword() {
    setBusy(true)
    setError(null)
    try {
      await api.login(password)
      await qc.invalidateQueries({ queryKey: ['auth', 'me'] })
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="app-loading" style={{ padding: 16 }}>
      <Card className="text-center">
        <div style={{ maxWidth: 340, display: 'grid', gap: 16, padding: 16 }}>
          <div className="logo" style={{ justifyContent: 'center' }}>
            <IconWallet size={22} stroke={1.75} /> Money Tracker
          </div>

          {mode === 'password' ? (
            <form
              style={{ display: 'grid', gap: 12 }}
              onSubmit={(e) => {
                e.preventDefault()
                void submitPassword()
              }}
            >
              <p className="muted" style={{ fontSize: 13 }}>
                Enter the app password to access your money tracker.
              </p>
              <Input
                type="password"
                placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoFocus
              />
              {error && (
                <p style={{ fontSize: 13, color: 'var(--color-negative)' }}>{error}</p>
              )}
              <Button type="submit" disabled={busy || password === ''}>
                <IconLogin2 size={17} stroke={1.75} /> {busy ? 'Signing in…' : 'Sign in'}
              </Button>
            </form>
          ) : (
            <>
              <p className="muted" style={{ fontSize: 13 }}>
                Sign in with your Notion account to access your money tracker
                and sync your data to a Notion workspace.
              </p>
              {error && (
                <p style={{ fontSize: 13, color: 'var(--color-negative)' }}>{error}</p>
              )}
              <Button onClick={() => (window.location.href = notionLoginUrl)}>
                <IconBrandNotion size={17} stroke={1.75} /> Continue with Notion
              </Button>
              <p className="muted" style={{ fontSize: 11 }}>
                On the Notion consent screen, share at least one page — the
                sync creates a “Money Tracker” page inside it.
              </p>
            </>
          )}
        </div>
      </Card>
    </div>
  )
}
