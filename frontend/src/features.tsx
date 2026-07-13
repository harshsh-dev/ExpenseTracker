import { createContext, useContext, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from './api/client'

export type Feature =
  | 'dashboard'
  | 'income'
  | 'expenses'
  | 'investments'
  | 'recurring'
  | 'loans'
  | 'categories'
  | 'report'
  | 'backup'

const ALL_FEATURES: Feature[] = [
  'dashboard',
  'income',
  'expenses',
  'investments',
  'recurring',
  'loans',
  'categories',
  'report',
  'backup',
]

// Optional build-time override (VITE_FEATURES) lets the static frontend be
// trimmed independently of the backend, e.g. VITE_FEATURES="income,expenses".
// When unset, the backend's /api/config is the source of truth.
function buildOverride(): Feature[] | null {
  const raw = import.meta.env.VITE_FEATURES as string | undefined
  if (!raw) return null
  const t = raw.trim().toLowerCase()
  if (t === '' || t === 'all' || t === '*') return ALL_FEATURES
  const wanted = new Set(t.split(/[\s,;]+/).filter(Boolean))
  const out = ALL_FEATURES.filter((f) => wanted.has(f))
  return out.length ? out : ALL_FEATURES
}

const FeaturesContext = createContext<Set<Feature>>(new Set(ALL_FEATURES))

export function FeaturesProvider({ children }: { children: ReactNode }) {
  const override = buildOverride()

  const { data, isLoading, isError } = useQuery({
    queryKey: ['config'],
    queryFn: () => api.getConfig(),
    enabled: !override,
    staleTime: Infinity,
    retry: 1,
  })

  if (!override && isLoading) {
    return <div className="app-loading">Loading…</div>
  }

  // Fall back to all features if config can't be loaded, so the app is never
  // left blank by a transient backend hiccup.
  const resolved: Feature[] = override
    ? override
    : isError || !data
      ? ALL_FEATURES
      : ALL_FEATURES.filter((f) => data.features.includes(f))

  return (
    <FeaturesContext.Provider value={new Set(resolved)}>{children}</FeaturesContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useFeatures(): Set<Feature> {
  return useContext(FeaturesContext)
}

// eslint-disable-next-line react-refresh/only-export-components
export function useFeature(feature: Feature): boolean {
  return useContext(FeaturesContext).has(feature)
}
