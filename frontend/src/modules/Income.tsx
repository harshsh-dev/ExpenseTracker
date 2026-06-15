import { useMemo, useState } from 'react'
import { IconPlus, IconPencil, IconTrash } from '@tabler/icons-react'
import { useIncomes, useIncomeCrud } from '../api/hooks'
import type { Income } from '../types'
import { Button, Card, Empty, Field, Input, Modal, PageHeader } from '../components/ui'
import { formatMoney, monthName, todayISO } from '../lib/format'

const now = new Date()

function emptyForm(): Partial<Income> {
  return {
    source: 'Salary',
    amount: undefined,
    month: now.getMonth() + 1,
    year: now.getFullYear(),
    receivedOn: todayISO(),
    note: '',
  }
}

export default function IncomePage() {
  const { data: incomes = [], isLoading } = useIncomes()
  const { create, update, remove } = useIncomeCrud()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Income | null>(null)
  const [form, setForm] = useState<Partial<Income>>(emptyForm())

  const grouped = useMemo(() => {
    const map = new Map<string, Income[]>()
    for (const inc of incomes) {
      const key = `${inc.year}-${String(inc.month).padStart(2, '0')}`
      map.set(key, [...(map.get(key) ?? []), inc])
    }
    return [...map.entries()].sort((a, b) => b[0].localeCompare(a[0]))
  }, [incomes])

  const total = incomes.reduce((s, i) => s + i.amount, 0)

  function startCreate() {
    setEditing(null)
    setForm(emptyForm())
    setOpen(true)
  }
  function startEdit(inc: Income) {
    setEditing(inc)
    setForm(inc)
    setOpen(true)
  }
  async function save() {
    const body = {
      ...form,
      amount: Number(form.amount),
      month: Number(form.month),
      year: Number(form.year),
    }
    if (editing) await update.mutateAsync({ id: editing.id, body })
    else await create.mutateAsync(body)
    setOpen(false)
  }

  return (
    <>
      <PageHeader
        title="Income"
        subtitle={`Total recorded: ${formatMoney(total)}`}
        action={
          <Button onClick={startCreate}>
            <IconPlus size={15} stroke={2} /> Add income
          </Button>
        }
      />

      {isLoading ? (
        <Empty>Loading…</Empty>
      ) : grouped.length === 0 ? (
        <Empty>No income yet. Add your first monthly income.</Empty>
      ) : (
        <div className="flex flex-col gap-4">
          {grouped.map(([key, items]) => {
            const [y, m] = key.split('-')
            const monthTotal = items.reduce((s, i) => s + i.amount, 0)
            return (
              <Card key={key}>
                <div className="mb-1 flex items-center justify-between">
                  <div className="card-title" style={{ margin: 0 }}>
                    {monthName(Number(m))} {y}
                  </div>
                  <span className="text-sm font-semibold positive">{formatMoney(monthTotal)}</span>
                </div>
                {items.map((inc) => (
                  <div className="txn" key={inc.id}>
                    <div className="txn-info">
                      <div className="txn-name">{inc.source}</div>
                      <div className="txn-cat">{inc.note || 'Received ' + inc.receivedOn}</div>
                    </div>
                    <div className="txn-amt positive">+{formatMoney(inc.amount, inc.currency)}</div>
                    <div className="row-actions">
                      <button className="icon-btn" onClick={() => startEdit(inc)} title="Edit">
                        <IconPencil size={16} stroke={1.75} />
                      </button>
                      <button
                        className="icon-btn danger"
                        onClick={() => remove.mutate(inc.id)}
                        title="Delete"
                      >
                        <IconTrash size={16} stroke={1.75} />
                      </button>
                    </div>
                  </div>
                ))}
              </Card>
            )
          })}
        </div>
      )}

      <Modal title={editing ? 'Edit income' : 'Add income'} open={open} onClose={() => setOpen(false)}>
        <div className="grid grid-cols-2 gap-4">
          <Field label="Source">
            <Input value={form.source ?? ''} onChange={(e) => setForm({ ...form, source: e.target.value })} />
          </Field>
          <Field label="Amount">
            <Input
              type="number"
              value={form.amount ?? ''}
              onChange={(e) => setForm({ ...form, amount: Number(e.target.value) })}
            />
          </Field>
          <Field label="Month">
            <Input
              type="number"
              min={1}
              max={12}
              value={form.month ?? ''}
              onChange={(e) => setForm({ ...form, month: Number(e.target.value) })}
            />
          </Field>
          <Field label="Year">
            <Input
              type="number"
              value={form.year ?? ''}
              onChange={(e) => setForm({ ...form, year: Number(e.target.value) })}
            />
          </Field>
          <Field label="Received on">
            <Input
              type="date"
              value={form.receivedOn ?? ''}
              onChange={(e) => setForm({ ...form, receivedOn: e.target.value })}
            />
          </Field>
          <Field label="Note (optional)">
            <Input value={form.note ?? ''} onChange={(e) => setForm({ ...form, note: e.target.value })} />
          </Field>
        </div>
        <div className="modal-actions">
          <Button variant="ghost" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button onClick={save} disabled={!form.source || !form.amount}>
            {editing ? 'Save' : 'Add income'}
          </Button>
        </div>
      </Modal>
    </>
  )
}
