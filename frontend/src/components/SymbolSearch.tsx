import { useEffect, useRef, useState } from 'react'
import { api, type SymbolHit } from '../api/client'

export function SymbolSearch({
  kind,
  value,
  selectedName,
  onPick,
}: {
  kind: 'mf' | 'stock' | 'bse'
  value?: string
  selectedName?: string
  onPick: (hit: SymbolHit) => void
}) {
  const placeholder =
    kind === 'mf' ? 'Search mutual fund…' : kind === 'bse' ? 'Search BSE stock…' : 'Search NSE stock…'
  const [query, setQuery] = useState('')
  const [hits, setHits] = useState<SymbolHit[]>([])
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const boxRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let active = true
    const q = query.trim()
    const t = setTimeout(async () => {
      if (q.length < 2) {
        if (active) setHits([])
        return
      }
      if (active) setLoading(true)
      try {
        const res = await api.searchSymbols(kind, q)
        if (active) setHits(res.slice(0, 12))
      } catch {
        if (active) setHits([])
      } finally {
        if (active) setLoading(false)
      }
    }, 300)
    return () => {
      active = false
      clearTimeout(t)
    }
  }, [query, kind])

  useEffect(() => {
    function onDoc(e: MouseEvent) {
      if (boxRef.current && !boxRef.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [])

  return (
    <div ref={boxRef} style={{ position: 'relative' }}>
      <input
        className="input"
        placeholder={placeholder}
        value={query}
        onChange={(e) => {
          setQuery(e.target.value)
          setOpen(true)
        }}
        onFocus={() => setOpen(true)}
      />
      {value && (
        <div className="muted" style={{ fontSize: 11, marginTop: 4 }}>
          Selected: <strong>{selectedName || value}</strong> ({value})
        </div>
      )}
      {open && (query.trim().length >= 2 || hits.length > 0) && (
        <div
          className="card"
          style={{
            position: 'absolute',
            zIndex: 10,
            top: '100%',
            left: 0,
            right: 0,
            marginTop: 4,
            padding: 6,
            maxHeight: 240,
            overflowY: 'auto',
            boxShadow: '0 8px 24px rgba(0,0,0,0.12)',
          }}
        >
          {loading && <div className="muted" style={{ fontSize: 12, padding: 8 }}>Searching…</div>}
          {!loading && hits.length === 0 && (
            <div className="muted" style={{ fontSize: 12, padding: 8 }}>No matches</div>
          )}
          {hits.map((h) => (
            <div
              key={h.symbol}
              onClick={() => {
                onPick(h)
                setQuery('')
                setHits([])
                setOpen(false)
              }}
              style={{ padding: '7px 8px', borderRadius: 6, cursor: 'pointer', fontSize: 12 }}
              onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--color-background-secondary)')}
              onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
            >
              <div style={{ color: 'var(--color-text-primary)' }}>{h.name}</div>
              <div className="muted" style={{ fontSize: 11 }}>{h.symbol}</div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
