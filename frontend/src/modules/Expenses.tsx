import { useMemo, useState } from 'react'
import { IconPlus, IconPencil, IconTrash } from '@tabler/icons-react'
import { useCategories, useExpenseCrud, useExpenses } from '../api/hooks'
import type { Category, Expense, PaymentMethod } from '../types'
import { Button, Card, Empty, Field, Input, Modal, PageHeader, Select } from '../components/ui'
import { formatDate, formatMoney, todayISO } from '../lib/format'

const methods: PaymentMethod[] = ['upi', 'cash', 'card', 'netbanking', 'wallet', 'other']

function emptyForm(categoryId: string): Partial<Expense> {
  return {
    amount: undefined,
    categoryId,
    subcategory: '',
    date: todayISO(),
    paymentMethod: 'upi',
    note: '',
  }
}

export default function ExpensesPage() {
  const { data: expenses = [], isLoading } = useExpenses()
  const { data: categories = [] } = useCategories()
  const { create, update, remove } = useExpenseCrud()

  const catById = useMemo(
    () => Object.fromEntries(categories.map((c) => [c.id, c])) as Record<string, Category>,
    [categories],
  )

  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Expense | null>(null)
  const [form, setForm] = useState<Partial<Expense>>(emptyForm(''))
  const [filterCat, setFilterCat] = useState('')

  const filtered = filterCat ? expenses.filter((e) => e.categoryId === filterCat) : expenses
  const total = filtered.reduce((s, e) => s + e.amount, 0)
  const selectedCat = categories.find((c) => c.id === form.categoryId)

  function startCreate() {
    setEditing(null)
    setForm(emptyForm(categories[0]?.id ?? ''))
    setOpen(true)
  }
  function startEdit(exp: Expense) {
    setEditing(exp)
    setForm(exp)
    setOpen(true)
  }
  async function save() {
    const body = { ...form, amount: Number(form.amount) }
    if (editing) await update.mutateAsync({ id: editing.id, body })
    else await create.mutateAsync(body)
    setOpen(false)
  }

  return (
    <>
      <PageHeader
        title="Expenses"
        subtitle={`${filtered.length} entries · ${formatMoney(total)}`}
        action={
          <Button onClick={startCreate} disabled={categories.length === 0}>
            <IconPlus size={15} stroke={2} /> Add expense
          </Button>
        }
      />

      <div style={{ maxWidth: 240 }}>
        <Select value={filterCat} onChange={(e) => setFilterCat(e.target.value)}>
          <option value="">All categories</option>
          {categories.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </Select>
      </div>

      {isLoading ? (
        <Empty>Loading…</Empty>
      ) : filtered.length === 0 ? (
        <Empty>No expenses. Add your daily spends here.</Empty>
      ) : (
        <Card>
          {filtered.map((exp) => {
            const cat = catById[exp.categoryId]
            return (
              <div className="txn" key={exp.id}>
                <div className="txn-icon" style={{ background: (cat?.color ?? '#888780') + '22' }}>
                  <span className="pill-dot" style={{ background: cat?.color ?? '#888780' }} />
                </div>
                <div className="txn-info">
                  <div className="txn-name">{exp.note || cat?.name || 'Expense'}</div>
                  <div className="txn-cat">
                    {cat?.name ?? 'Unknown'}
                    {exp.subcategory ? ` · ${exp.subcategory}` : ''} · {formatDate(exp.date)} ·{' '}
                    {exp.paymentMethod}
                  </div>
                </div>
                <div className="txn-amt negative">−{formatMoney(exp.amount, exp.currency)}</div>
                <div className="row-actions">
                  <button className="icon-btn" onClick={() => startEdit(exp)} title="Edit">
                    <IconPencil size={16} stroke={1.75} />
                  </button>
                  <button
                    className="icon-btn danger"
                    onClick={() => remove.mutate(exp.id)}
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

      <Modal title={editing ? 'Edit expense' : 'Add expense'} open={open} onClose={() => setOpen(false)}>
        <div className="grid grid-cols-2 gap-4">
          <Field label="Amount">
            <Input
              type="number"
              autoFocus
              value={form.amount ?? ''}
              onChange={(e) => setForm({ ...form, amount: Number(e.target.value) })}
            />
          </Field>
          <Field label="Date">
            <Input
              type="date"
              value={form.date ?? ''}
              onChange={(e) => setForm({ ...form, date: e.target.value })}
            />
          </Field>
        </div>

        <div className="mt-4">
          <div className="field-label mb-2">Category</div>
          <div className="flex flex-wrap gap-1.5">
            {categories.map((c) => (
              <span
                key={c.id}
                className={`pill pill-select ${form.categoryId === c.id ? 'active' : ''}`}
                onClick={() => setForm({ ...form, categoryId: c.id, subcategory: '' })}
              >
                <span className="pill-dot" style={{ background: c.color }} />
                {c.name}
              </span>
            ))}
          </div>
        </div>

        <div className="mt-4 grid grid-cols-2 gap-4">
          <Field label="Subcategory">
            <Select
              value={form.subcategory ?? ''}
              onChange={(e) => setForm({ ...form, subcategory: e.target.value })}
            >
              <option value="">—</option>
              {selectedCat?.subcategories.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Payment method">
            <Select
              value={form.paymentMethod ?? 'upi'}
              onChange={(e) => setForm({ ...form, paymentMethod: e.target.value as PaymentMethod })}
            >
              {methods.map((m) => (
                <option key={m} value={m}>
                  {m}
                </option>
              ))}
            </Select>
          </Field>
        </div>

        <div className="mt-4">
          <Field label="Note (optional)">
            <Input
              value={form.note ?? ''}
              placeholder="What did you spend on?"
              onChange={(e) => setForm({ ...form, note: e.target.value })}
            />
          </Field>
        </div>

        <div className="modal-actions">
          <Button variant="ghost" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button onClick={save} disabled={!form.amount || !form.categoryId}>
            {editing ? 'Save' : 'Add expense'}
          </Button>
        </div>
      </Modal>
    </>
  )
}
