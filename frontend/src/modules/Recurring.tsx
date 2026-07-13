import { useState } from 'react'
import {
  IconPlus,
  IconPencil,
  IconTrash,
  IconPlayerPause,
  IconPlayerPlay,
  IconRepeat,
  IconTrendingUp,
} from '@tabler/icons-react'
import {
  useCategories,
  useInvestments,
  useRecurring,
  useRecurringCrud,
} from '../api/hooks'
import type { Recurring } from '../types'
import { Button, Card, Empty, Field, Input, Modal, PageHeader, Pill, Select } from '../components/ui'
import { formatMoney, todayISO } from '../lib/format'

function emptyForm(): Partial<Recurring> {
  return {
    kind: 'expense',
    name: '',
    amount: undefined,
    cadence: 'monthly',
    startDate: todayISO(),
    paymentMethod: 'upi',
  }
}

export default function RecurringPage() {
  const { data: rules = [], isLoading } = useRecurring()
  const { data: categories = [] } = useCategories()
  const { data: investments = [] } = useInvestments()
  const { create, update, remove } = useRecurringCrud()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Recurring | null>(null)
  const [form, setForm] = useState<Partial<Recurring>>(emptyForm())

  const catName = (id?: string) => categories.find((c) => c.id === id)?.name ?? '—'
  const invName = (id?: string) => investments.find((i) => i.id === id)?.name ?? '(deleted)'

  const monthlyTotal = rules
    .filter((r) => !r.paused && r.cadence === 'monthly')
    .reduce((s, r) => s + r.amount, 0)

  function startCreate() {
    setEditing(null)
    setForm(emptyForm())
    setOpen(true)
  }
  function startEdit(r: Recurring) {
    setEditing(r)
    setForm(r)
    setOpen(true)
  }
  async function save() {
    const body = { ...form, amount: Number(form.amount) }
    if (editing) await update.mutateAsync({ id: editing.id, body })
    else await create.mutateAsync(body)
    setOpen(false)
  }
  function togglePause(r: Recurring) {
    update.mutate({ id: r.id, body: { ...r, paused: !r.paused } })
  }

  const isExpense = form.kind !== 'sip'

  return (
    <>
      <PageHeader
        title="Recurring"
        subtitle={`Subscriptions, EMIs & SIPs — ${formatMoney(monthlyTotal)}/month across active monthly rules`}
        action={
          <Button onClick={startCreate}>
            <IconPlus size={15} stroke={2} /> Add rule
          </Button>
        }
      />

      {isLoading ? (
        <Empty>Loading…</Empty>
      ) : rules.length === 0 ? (
        <Empty>
          No recurring rules yet. Add your subscriptions, EMIs, or SIPs — entries are created
          automatically when due.
        </Empty>
      ) : (
        <Card>
          {rules.map((r) => (
            <div className="txn" key={r.id} style={r.paused ? { opacity: 0.55 } : undefined}>
              <div className="txn-info">
                <div className="txn-name" style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                  {r.kind === 'sip' ? (
                    <IconTrendingUp size={15} stroke={1.75} />
                  ) : (
                    <IconRepeat size={15} stroke={1.75} />
                  )}
                  {r.name}
                  {r.paused && <Pill>paused</Pill>}
                </div>
                <div className="txn-cat">
                  {r.cadence}
                  {' · '}
                  {r.kind === 'sip' ? `SIP → ${invName(r.investmentId)}` : catName(r.categoryId)}
                  {r.nextRunOn && !r.paused && (r.endDate ? r.nextRunOn <= r.endDate : true)
                    ? ` · next on ${r.nextRunOn}`
                    : ''}
                  {r.endDate ? ` · ends ${r.endDate}` : ''}
                </div>
              </div>
              <div className="txn-amt">{formatMoney(r.amount, r.currency)}</div>
              <div className="row-actions">
                <button
                  className="icon-btn"
                  onClick={() => togglePause(r)}
                  title={r.paused ? 'Resume' : 'Pause'}
                >
                  {r.paused ? (
                    <IconPlayerPlay size={16} stroke={1.75} />
                  ) : (
                    <IconPlayerPause size={16} stroke={1.75} />
                  )}
                </button>
                <button className="icon-btn" onClick={() => startEdit(r)} title="Edit">
                  <IconPencil size={16} stroke={1.75} />
                </button>
                <button className="icon-btn danger" onClick={() => remove.mutate(r.id)} title="Delete">
                  <IconTrash size={16} stroke={1.75} />
                </button>
              </div>
            </div>
          ))}
        </Card>
      )}

      <Modal
        title={editing ? 'Edit rule' : 'Add recurring rule'}
        open={open}
        onClose={() => setOpen(false)}
      >
        <div className="grid grid-cols-2 gap-4">
          <Field label="Type">
            <Select
              value={form.kind ?? 'expense'}
              onChange={(e) => setForm({ ...form, kind: e.target.value as Recurring['kind'] })}
            >
              <option value="expense">Recurring expense (subscription, EMI)</option>
              <option value="sip">SIP (recurring investment)</option>
            </Select>
          </Field>
          <Field label="Name">
            <Input
              placeholder={isExpense ? 'YouTube Premium' : 'Nifty 50 SIP'}
              value={form.name ?? ''}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
          </Field>
          <Field label="Amount">
            <Input
              type="number"
              value={form.amount ?? ''}
              onChange={(e) => setForm({ ...form, amount: Number(e.target.value) })}
            />
          </Field>
          <Field label="Repeats">
            <Select
              value={form.cadence ?? 'monthly'}
              onChange={(e) => setForm({ ...form, cadence: e.target.value as Recurring['cadence'] })}
            >
              <option value="monthly">Monthly</option>
              <option value="weekly">Weekly</option>
              <option value="yearly">Yearly</option>
            </Select>
          </Field>

          {isExpense ? (
            <>
              <Field label="Category">
                <Select
                  value={form.categoryId ?? ''}
                  onChange={(e) => setForm({ ...form, categoryId: e.target.value })}
                >
                  <option value="">Select…</option>
                  {categories
                    .filter((c) => !c.archived)
                    .map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                </Select>
              </Field>
              <Field label="Payment method">
                <Select
                  value={form.paymentMethod ?? 'upi'}
                  onChange={(e) =>
                    setForm({ ...form, paymentMethod: e.target.value as Recurring['paymentMethod'] })
                  }
                >
                  {['upi', 'card', 'netbanking', 'cash', 'wallet', 'other'].map((m) => (
                    <option key={m} value={m}>
                      {m}
                    </option>
                  ))}
                </Select>
              </Field>
            </>
          ) : (
            <Field label="Investment">
              <Select
                value={form.investmentId ?? ''}
                onChange={(e) => setForm({ ...form, investmentId: e.target.value })}
              >
                <option value="">Select…</option>
                {investments.map((i) => (
                  <option key={i.id} value={i.id}>
                    {i.name}
                  </option>
                ))}
              </Select>
            </Field>
          )}

          <Field label="First payment">
            <Input
              type="date"
              value={form.startDate ?? ''}
              onChange={(e) => setForm({ ...form, startDate: e.target.value })}
            />
          </Field>
          <Field label="Ends on (optional)">
            <Input
              type="date"
              value={form.endDate ?? ''}
              onChange={(e) => setForm({ ...form, endDate: e.target.value })}
            />
          </Field>
        </div>
        <p className="muted" style={{ fontSize: 11, marginTop: 10 }}>
          Entries are created automatically when due — including catch-up for days the server was
          asleep. {isExpense ? 'Each occurrence appears on the Expenses page.' : 'Each occurrence adds to the investment’s invested amount (and units, when price-tracked).'}
        </p>
        <div className="modal-actions">
          <Button variant="ghost" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button
            onClick={save}
            disabled={
              !form.name ||
              !form.amount ||
              !form.startDate ||
              (isExpense ? !form.categoryId : !form.investmentId)
            }
          >
            {editing ? 'Save' : 'Add rule'}
          </Button>
        </div>
      </Modal>
    </>
  )
}
