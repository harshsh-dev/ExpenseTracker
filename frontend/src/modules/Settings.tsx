import { useEffect, useRef, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  IconBrandNotion,
  IconCloudDownload,
  IconDownload,
  IconExternalLink,
  IconUpload,
} from '@tabler/icons-react'
import { api } from '../api/client'
import { useAuth } from '../auth'
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

      <NotionSyncCard />

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

// One-way export of all data into databases on a "Money Tracker" page in the
// signed-in user's Notion workspace. Runs server-side; we poll while it runs.
function NotionSyncCard() {
  const { enabled } = useAuth()
  const qc = useQueryClient()
  const [err, setErr] = useState<string | null>(null)

  const { data: status, refetch } = useQuery({
    queryKey: ['notion', 'status'],
    queryFn: () => api.notionStatus(),
    enabled,
    refetchInterval: (query) => (query.state.data?.running ? 2000 : false),
  })

  // A finished pull may have changed app data; refresh the other pages.
  const pullStamp = status?.lastPull?.finishedAt
  const pullChanged = (status?.lastPull?.created ?? 0) + (status?.lastPull?.updated ?? 0) > 0
  useEffect(() => {
    if (pullStamp && pullChanged) {
      void qc.invalidateQueries({ queryKey: ['expenses'] })
      void qc.invalidateQueries({ queryKey: ['incomes'] })
      void qc.invalidateQueries({ queryKey: ['investments'] })
      void qc.invalidateQueries({ queryKey: ['categories'] })
    }
  }, [pullStamp, pullChanged, qc])

  if (!enabled) return null

  async function run(op: () => Promise<{ status: string }>) {
    setErr(null)
    try {
      await op()
      await refetch()
    } catch (e) {
      setErr((e as Error).message)
    }
  }

  const last = status?.last
  const lastPull = status?.lastPull
  const running = status?.running ?? false

  return (
    <Card className="mt-4">
      <div className="card-title">
        <IconBrandNotion size={16} stroke={1.75} style={{ verticalAlign: 'text-bottom' }} />{' '}
        Notion sync
      </div>
      <p className="muted" style={{ fontSize: 12, marginBottom: 16 }}>
        Mirror expenses, income, and investments to databases in your Notion
        workspace{status?.workspaceName ? ` (${status.workspaceName})` : ''}. Pull brings
        rows added or edited in Notion back into the app (deletions never cross over).
      </p>
      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
        <Button onClick={() => void run(api.notionSync)} disabled={running}>
          <IconBrandNotion size={15} stroke={1.75} />
          {running ? 'Working…' : 'Sync to Notion'}
        </Button>
        <Button variant="ghost" onClick={() => void run(api.notionPull)} disabled={running}>
          <IconCloudDownload size={15} stroke={1.75} /> Pull from Notion
        </Button>
        {status?.pageUrl && (
          <a className="btn btn-ghost" href={status.pageUrl} target="_blank" rel="noreferrer">
            <IconExternalLink size={15} stroke={1.75} /> Open in Notion
          </a>
        )}
      </div>
      <p className="muted" style={{ fontSize: 12, marginTop: 12, marginBottom: 0 }}>
        {running
          ? 'Working — large datasets take a few minutes (Notion rate limits).'
          : err
            ? err
            : last?.error
              ? `Last sync failed: ${last.error}`
              : status?.lastSyncedAt
                ? `Last synced ${new Date(status.lastSyncedAt).toLocaleString()}` +
                  (last ? ` — ${last.created} created, ${last.updated} updated.` : '.')
                : 'Never synced yet.'}
      </p>
      {lastPull && !running && (
        <p className="muted" style={{ fontSize: 12, marginTop: 6, marginBottom: 0 }}>
          {lastPull.error
            ? `Last pull failed: ${lastPull.error}`
            : `Last pull: ${lastPull.created} added, ${lastPull.updated} updated, ` +
              `${lastPull.unchanged} unchanged, ${lastPull.skipped} skipped.`}
          {!!lastPull.skipReasons?.length && (
            <span> ({lastPull.skipReasons.join('; ')})</span>
          )}
        </p>
      )}
    </Card>
  )
}
