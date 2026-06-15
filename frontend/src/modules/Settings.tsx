import { useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { IconDownload, IconUpload } from '@tabler/icons-react'
import { api } from '../api/client'
import type { Snapshot } from '../types'
import { Button, Card, PageHeader } from '../components/ui'

export default function SettingsPage() {
  const qc = useQueryClient()
  const fileRef = useRef<HTMLInputElement>(null)
  const [msg, setMsg] = useState<{ kind: 'ok' | 'err'; text: string } | null>(null)
  const [busy, setBusy] = useState(false)

  async function handleExport() {
    setBusy(true)
    setMsg(null)
    try {
      const snap = await api.exportSnapshot()
      const blob = new Blob([JSON.stringify(snap, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      const stamp = new Date().toISOString().slice(0, 10)
      a.href = url
      a.download = `moneytracker-backup-${stamp}.json`
      a.click()
      URL.revokeObjectURL(url)
      setMsg({ kind: 'ok', text: 'Backup downloaded.' })
    } catch (e) {
      setMsg({ kind: 'err', text: (e as Error).message })
    } finally {
      setBusy(false)
    }
  }

  async function handleImport(file: File) {
    setBusy(true)
    setMsg(null)
    try {
      const snap = JSON.parse(await file.text()) as Snapshot
      await api.importSnapshot(snap)
      await qc.invalidateQueries()
      setMsg({ kind: 'ok', text: 'Data restored from backup.' })
    } catch (e) {
      setMsg({ kind: 'err', text: (e as Error).message })
    } finally {
      setBusy(false)
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  return (
    <>
      <PageHeader
        title="Backup & Restore"
        subtitle="Download a snapshot quarterly. Re-upload it on any device, or after a redeploy, to resume where you left off."
      />

      <div className="grid gap-4 sm:grid-cols-2">
        <Card>
          <div className="card-title">Export</div>
          <p className="muted" style={{ fontSize: 12, marginBottom: 16 }}>
            Save all income, expenses, investments, and categories as a single JSON file.
          </p>
          <Button onClick={handleExport} disabled={busy}>
            <IconDownload size={15} stroke={1.75} /> Download backup
          </Button>
        </Card>

        <Card>
          <div className="card-title">Import</div>
          <p className="muted" style={{ fontSize: 12, marginBottom: 16 }}>
            Replace current data with a previously downloaded backup file.
          </p>
          <input
            ref={fileRef}
            type="file"
            accept="application/json"
            style={{ display: 'none' }}
            onChange={(e) => {
              const f = e.target.files?.[0]
              if (f) void handleImport(f)
            }}
          />
          <Button variant="ghost" onClick={() => fileRef.current?.click()} disabled={busy}>
            <IconUpload size={15} stroke={1.75} /> Choose backup file…
          </Button>
        </Card>
      </div>

      {msg && (
        <div
          className="card"
          style={{
            fontSize: 13,
            color: msg.kind === 'ok' ? 'var(--color-positive)' : 'var(--color-negative)',
            borderColor:
              msg.kind === 'ok' ? 'var(--color-accent-border)' : 'var(--color-danger-border)',
            background: msg.kind === 'ok' ? 'var(--color-accent-soft)' : 'var(--color-danger-soft)',
          }}
        >
          {msg.text}
        </div>
      )}
    </>
  )
}
