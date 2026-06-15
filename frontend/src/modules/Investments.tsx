import { useState } from 'react'
import {
  IconPlus,
  IconPencil,
  IconTrash,
  IconArrowUpRight,
  IconArrowDownRight,
  IconRefresh,
} from '@tabler/icons-react'
import { useInvestmentCrud, useInvestments, useRefreshPrices } from '../api/hooks'
import {
  investmentCurrentValue,
  investmentPnl,
  type Investment,
  type InvestmentType,
} from '../types'
import { Button, Card, Empty, Field, Input, Modal, PageHeader, Select } from '../components/ui'
import { SymbolSearch } from '../components/SymbolSearch'
import { formatMoney, todayISO } from '../lib/format'

const types: InvestmentType[] = [
  'stocks',
  'mutual_fund',
  'fd',
  'rd',
  'gold',
  'crypto',
  'bonds',
  'real_estate',
  'other',
]

type Provider = 'manual' | 'mfapi' | 'stock' | 'bse' | 'coingecko'
const providers: { id: Provider; label: string }[] = [
  { id: 'manual', label: 'Manual (enter value)' },
  { id: 'mfapi', label: 'Mutual fund (auto NAV)' },
  { id: 'stock', label: 'Indian stock (NSE)' },
  { id: 'bse', label: 'Indian stock (BSE)' },
  { id: 'coingecko', label: 'Crypto (CoinGecko)' },
]

function searchKind(provider: Provider): 'mf' | 'stock' | 'bse' {
  if (provider === 'mfapi') return 'mf'
  if (provider === 'bse') return 'bse'
  return 'stock'
}

function emptyForm(): Partial<Investment> {
  return {
    name: '',
    type: 'mutual_fund',
    provider: 'manual',
    amountInvested: undefined,
    currentValue: undefined,
    investedOn: todayISO(),
    note: '',
  }
}

export default function InvestmentsPage() {
  const { data: investments = [], isLoading } = useInvestments()
  const { create, update, remove } = useInvestmentCrud()
  const refresh = useRefreshPrices()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Investment | null>(null)
  const [form, setForm] = useState<Partial<Investment>>(emptyForm())

  const totalInvested = investments.reduce((s, i) => s + i.amountInvested, 0)
  const totalCurrent = investments.reduce(
    (s, i) => s + (investmentCurrentValue(i) ?? i.amountInvested),
    0,
  )
  const totalPnl = totalCurrent - totalInvested
  const lastUpdated = investments
    .map((i) => i.lastPriceAt)
    .filter(Boolean)
    .sort()
    .pop()

  const provider = (form.provider ?? 'manual') as Provider

  function startCreate() {
    setEditing(null)
    setForm(emptyForm())
    setOpen(true)
  }
  function startEdit(inv: Investment) {
    setEditing(inv)
    setForm(inv)
    setOpen(true)
  }
  async function save() {
    const auto = provider !== 'manual'
    const body = {
      ...form,
      amountInvested: Number(form.amountInvested),
      currentValue: !auto && form.currentValue != null ? Number(form.currentValue) : undefined,
      quantity: auto && form.quantity != null ? Number(form.quantity) : undefined,
      symbol: auto ? form.symbol : undefined,
    }
    if (editing) await update.mutateAsync({ id: editing.id, body })
    else await create.mutateAsync(body)
    setOpen(false)
  }

  return (
    <>
      <PageHeader
        title="Investments"
        subtitle={
          lastUpdated
            ? `Prices updated ${new Date(lastUpdated).toLocaleString('en-IN')}`
            : 'Portfolio & profit / loss'
        }
        action={
          <div style={{ display: 'flex', gap: 8 }}>
            <Button variant="ghost" onClick={() => refresh.mutate()} disabled={refresh.isPending}>
              <IconRefresh
                size={15}
                stroke={1.75}
                style={refresh.isPending ? { animation: 'spin 1s linear infinite' } : undefined}
              />
              {refresh.isPending ? 'Refreshing…' : 'Refresh prices'}
            </Button>
            <Button onClick={startCreate}>
              <IconPlus size={15} stroke={2} /> Add investment
            </Button>
          </div>
        }
      />

      <div className="stats" style={{ gridTemplateColumns: 'repeat(3,1fr)' }}>
        <Card>
          <div className="stat-label">Invested</div>
          <div className="stat-value">{formatMoney(totalInvested)}</div>
        </Card>
        <Card>
          <div className="stat-label">Current value</div>
          <div className="stat-value">{formatMoney(totalCurrent)}</div>
        </Card>
        <Card>
          <div className="stat-label">Profit / Loss</div>
          <div className={`stat-value ${totalPnl >= 0 ? 'positive' : 'negative'}`}>
            {formatMoney(totalPnl)}
          </div>
        </Card>
      </div>

      {refresh.isError && (
        <div className="card negative" style={{ fontSize: 13 }}>
          Refresh failed: {(refresh.error as Error).message}
        </div>
      )}

      {isLoading ? (
        <Empty>Loading…</Empty>
      ) : investments.length === 0 ? (
        <Empty>No investments yet.</Empty>
      ) : (
        <Card>
          {investments.map((inv) => {
            const cur = investmentCurrentValue(inv)
            const { value: pnl, pct } = investmentPnl(inv)
            const positive = (pnl ?? 0) >= 0
            return (
              <div className="txn" key={inv.id}>
                <div className="txn-info">
                  <div className="txn-name">
                    {inv.name}{' '}
                    <span className="muted" style={{ fontSize: 11, fontWeight: 400 }}>
                      ({inv.type.replace('_', ' ')})
                    </span>
                  </div>
                  <div className="txn-cat">
                    Invested {formatMoney(inv.amountInvested, inv.currency)}
                    {inv.quantity != null ? ` · ${inv.quantity} units` : ''}
                    {inv.lastPrice != null ? ` · @ ${formatMoney(inv.lastPrice, inv.currency)}` : ''}
                  </div>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <div className="txn-amt" style={{ color: 'var(--color-text-primary)' }}>
                    {cur != null ? formatMoney(cur, inv.currency) : '—'}
                  </div>
                  {pnl != null && (
                    <div
                      className={positive ? 'positive' : 'negative'}
                      style={{
                        fontSize: 11,
                        display: 'flex',
                        alignItems: 'center',
                        gap: 2,
                        justifyContent: 'flex-end',
                      }}
                    >
                      {positive ? (
                        <IconArrowUpRight size={13} stroke={2} />
                      ) : (
                        <IconArrowDownRight size={13} stroke={2} />
                      )}
                      {formatMoney(pnl, inv.currency)} ({pct?.toFixed(1)}%)
                    </div>
                  )}
                </div>
                <div className="row-actions">
                  <button className="icon-btn" onClick={() => startEdit(inv)} title="Edit">
                    <IconPencil size={16} stroke={1.75} />
                  </button>
                  <button
                    className="icon-btn danger"
                    onClick={() => remove.mutate(inv.id)}
                    title="Delete"
                  >
                    <IconTrash size={16} stroke={1.75} />
                  </button>
                </div>
              </div>
            )
          })}
        </Card>
      )}

      <Modal
        title={editing ? 'Edit investment' : 'Add investment'}
        open={open}
        onClose={() => setOpen(false)}
      >
        <div className="grid grid-cols-2 gap-4">
          <Field label="Name">
            <Input value={form.name ?? ''} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </Field>
          <Field label="Type">
            <Select
              value={form.type ?? 'mutual_fund'}
              onChange={(e) => setForm({ ...form, type: e.target.value as InvestmentType })}
            >
              {types.map((t) => (
                <option key={t} value={t}>
                  {t.replace('_', ' ')}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Amount invested">
            <Input
              type="number"
              value={form.amountInvested ?? ''}
              onChange={(e) => setForm({ ...form, amountInvested: Number(e.target.value) })}
            />
          </Field>
          <Field label="Invested on">
            <Input
              type="date"
              value={form.investedOn ?? ''}
              onChange={(e) => setForm({ ...form, investedOn: e.target.value })}
            />
          </Field>
          <Field label="Price source">
            <Select
              value={provider}
              onChange={(e) =>
                setForm({ ...form, provider: e.target.value as Provider, symbol: '' })
              }
            >
              {providers.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.label}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Platform (optional)">
            <Input
              value={form.platform ?? ''}
              onChange={(e) => setForm({ ...form, platform: e.target.value })}
            />
          </Field>
        </div>

        {provider === 'manual' ? (
          <div className="mt-4">
            <Field label="Current value (manual)">
              <Input
                type="number"
                value={form.currentValue ?? ''}
                onChange={(e) =>
                  setForm({
                    ...form,
                    currentValue: e.target.value === '' ? undefined : Number(e.target.value),
                  })
                }
              />
            </Field>
          </div>
        ) : (
          <div className="mt-4 grid grid-cols-2 gap-4">
            <div>
              <div className="field-label" style={{ marginBottom: 5 }}>
                {provider === 'coingecko' ? 'Coin id' : 'Symbol'}
              </div>
              {provider === 'coingecko' ? (
                <Input
                  placeholder="e.g. bitcoin"
                  value={form.symbol ?? ''}
                  onChange={(e) => setForm({ ...form, symbol: e.target.value })}
                />
              ) : (
                <SymbolSearch
                  kind={searchKind(provider)}
                  value={form.symbol}
                  selectedName={form.name}
                  onPick={(hit) =>
                    setForm((f) => ({
                      ...f,
                      symbol: hit.symbol,
                      name: f.name ? f.name : hit.name,
                    }))
                  }
                />
              )}
            </div>
            <Field label="Quantity / units">
              <Input
                type="number"
                value={form.quantity ?? ''}
                onChange={(e) =>
                  setForm({
                    ...form,
                    quantity: e.target.value === '' ? undefined : Number(e.target.value),
                  })
                }
              />
            </Field>
          </div>
        )}

        <p className="muted" style={{ fontSize: 11, marginTop: 12 }}>
          {provider === 'manual'
            ? 'Enter the current value yourself to track profit/loss.'
            : 'Current value & P/L are fetched automatically from the price source. Hit “Refresh prices” after saving.'}
        </p>

        <div className="modal-actions">
          <Button variant="ghost" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button onClick={save} disabled={!form.name || !form.amountInvested}>
            {editing ? 'Save' : 'Add investment'}
          </Button>
        </div>
      </Modal>
    </>
  )
}
