import type { Category, Expense, Income, Investment, Loan, Recurring, Snapshot } from '../types'

const BASE = import.meta.env.VITE_API_URL ?? ''

// Session token in localStorage + Authorization header. Cookies alone don't
// survive cross-site (Hosting ↔ Render) — Safari et al. block them.
const TOKEN_KEY = 'mt_session_token'
let authToken: string | null = localStorage.getItem(TOKEN_KEY)

function setAuthToken(token: string | null) {
  authToken = token
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method,
    credentials: 'include', // session cookie (same-origin setups)
    headers: {
      ...(body ? { 'Content-Type': 'application/json' } : {}),
      ...(authToken ? { Authorization: `Bearer ${authToken}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`
    try {
      const data = await res.json()
      if (data?.error) msg = data.error
    } catch {
      /* ignore */
    }
    // Session expired mid-use: tell AuthProvider to re-check and show login.
    if (res.status === 401) window.dispatchEvent(new Event('auth:unauthorized'))
    throw new ApiError(res.status, msg)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

// Generic CRUD endpoint factory: one line per resource.
function resource<T extends { id: string }>(path: string) {
  return {
    list: () => request<T[]>('GET', path),
    create: (body: Partial<T>) => request<T>('POST', path, body),
    update: (id: string, body: Partial<T>) => request<T>('PUT', `${path}/${id}`, body),
    remove: (id: string) => request<void>('DELETE', `${path}/${id}`),
  }
}

export interface SymbolHit {
  symbol: string
  name: string
}

export interface AppConfig {
  app: string
  features: string[]
}

export interface RefreshResult {
  investments: Investment[]
  results: { id: string; symbol: string; ok: boolean; price?: number; error?: string }[]
  refreshedAt: string
}

export interface AuthMe {
  enabled: boolean
  authenticated: boolean
  mode?: 'notion' | 'password'
  user?: { name: string; email: string; avatarUrl: string; workspaceName: string }
}

export interface NotionSyncResult {
  startedAt: string
  finishedAt: string
  error?: string
  created: number
  updated: number
  archived: number
  expenses: number
  incomes: number
  investments: number
}

export interface NotionPullResult {
  startedAt: string
  finishedAt: string
  error?: string
  created: number
  updated: number
  unchanged: number
  skipped: number
  skipReasons?: string[]
}

export interface NotionStatus {
  configured: boolean
  connected?: boolean
  workspaceName?: string
  pageUrl?: string
  running?: boolean
  lastSyncedAt?: string
  last?: NotionSyncResult
  lastPull?: NotionPullResult
}

// Full-page navigation target for "Continue with Notion" (OAuth redirect).
export const notionLoginUrl = `${BASE}/api/auth/notion/login`

export const api = {
  getMe: () => request<AuthMe>('GET', '/api/auth/me'),
  login: async (password: string) => {
    const { token } = await request<{ token: string }>('POST', '/api/auth/login', { password })
    setAuthToken(token)
  },
  logout: async () => {
    await request<void>('POST', '/api/auth/logout')
    setAuthToken(null)
  },
  notionStatus: () => request<NotionStatus>('GET', '/api/notion/status'),
  notionSync: () => request<{ status: string }>('POST', '/api/notion/sync'),
  notionPull: () => request<{ status: string }>('POST', '/api/notion/pull'),
  getConfig: () => request<AppConfig>('GET', '/api/config'),
  incomes: resource<Income>('/api/incomes'),
  expenses: resource<Expense>('/api/expenses'),
  investments: resource<Investment>('/api/investments'),
  categories: resource<Category>('/api/categories'),
  recurring: resource<Recurring>('/api/recurring'),
  loans: resource<Loan>('/api/loans'),
  exportSnapshot: () => request<Snapshot>('GET', '/api/backup/export'),
  importSnapshot: (snap: Snapshot) =>
    request<{ status: string }>('POST', '/api/backup/import', snap),
  refreshPrices: () => request<RefreshResult>('POST', '/api/quotes/refresh'),
  searchSymbols: (kind: 'mf' | 'stock' | 'bse', q: string) =>
    request<SymbolHit[]>('GET', `/api/quotes/search/${kind}?q=${encodeURIComponent(q)}`),
}

export type Resource = keyof Pick<
  typeof api,
  'incomes' | 'expenses' | 'investments' | 'categories'
>
