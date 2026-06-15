import type { Category, Expense, Income, Investment, Snapshot } from '../types'

const BASE = import.meta.env.VITE_API_URL ?? ''

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
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
    throw new Error(msg)
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

export interface RefreshResult {
  investments: Investment[]
  results: { id: string; symbol: string; ok: boolean; price?: number; error?: string }[]
  refreshedAt: string
}

export const api = {
  incomes: resource<Income>('/api/incomes'),
  expenses: resource<Expense>('/api/expenses'),
  investments: resource<Investment>('/api/investments'),
  categories: resource<Category>('/api/categories'),
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
