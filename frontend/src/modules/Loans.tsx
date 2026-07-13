import { useState } from 'react'
import { IconPlus, IconPencil, IconTrash, IconCashBanknote } from '@tabler/icons-react'
import { useLoanCrud, useLoans } from '../api/hooks'
import { loanOutstanding, loanRepaid, type Loan } from '../types'
import { Button, Card, Empty, Field, Input, Modal, PageHeader, Pill } from '../components/ui'
import { formatMoney, todayISO } from '../lib/format'

function emptyForm(): Partial<Loan> {
  return { borrower: '', principal: undefined, lentOn: todayISO(), dueOn: '', note: '' }
}

export default function LoansPage() {
  const { data: loans = [], isLoading } = useLoans()
  const { create, update, remove } = useLoanCrud()

  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Loan | null>(null)
  const [form, setForm] = useState<Partial<Loan>>(emptyForm())

  const [repayFor, setRepayFor] = useState<Loan | null>(null)
  const [repayAmount, setRepayAmount] = useState('')
  const [repayDate, setRepayDate] = useState(todayISO())
  const [repayNote, setRepayNote] = useState('')

  const totalOut = loans.reduce((s, l) => s + Math.max(0, loanOutstanding(l)), 0)

  function startCreate() {
    setEditing(null)
    setForm(emptyForm())
    setOpen(true)
  }
  function startEdit(l: Loan) {
    setEditing(l)
    setForm(l)
    setOpen(true)
  }
  async function save() {
    const body = { ...form, principal: Number(form.principal) }
    if (editing) await update.mutateAsync({ id: editing.id, body })
    else await create.mutateAsync({ ...body, repayments: [] })
    setOpen(false)
  }

  function startRepay(l: Loan) {
    setRepayFor(l)
    setRepayAmount('')
    setRepayDate(todayISO())
    setRepayNote('')
  }
  async function saveRepayment() {
    if (!repayFor) return
    const body: Partial<Loan> = {
      ...repayFor,
      repayments: [
        ...repayFor.repayments,
        { id: '', amount: Number(repayAmount), date: repayDate, note: repayNote || undefined },
      ],
    }
    await update.mutateAsync({ id: repayFor.id, body })
    setRepayFor(null)
  }
  function removeRepayment(l: Loan, repaymentId: string) {
    update.mutate({
      id: l.id,
      body: { ...l, repayments: l.repayments.filter((r) => r.id !== repaymentId) },
    })
  }

  return (
    <>
      <PageHeader
        title="Loans given"
        subtitle={`Outstanding across ${loans.length} loan${loans.length === 1 ? '' : 's'}: ${formatMoney(totalOut)}`}
        action={
          <Button onClick={startCreate}>
            <IconPlus size={15} stroke={2} /> Add loan
          </Button>
        }
      />

      {isLoading ? (
        <Empty>Loading…</Empty>
      ) : loans.length === 0 ? (
        <Empty>No loans recorded. Track money you've lent and how it comes back.</Empty>
      ) : (
        <div className="flex flex-col gap-4">
          {loans.map((l) => {
            const repaid = loanRepaid(l)
            const out = loanOutstanding(l)
            const pct = Math.min(100, Math.max(0, (repaid / l.principal) * 100))
            const settled = out <= 0
            return (
              <Card key={l.id}>
                <div className="mb-1 flex items-center justify-between">
                  <div className="card-title" style={{ margin: 0, display: 'flex', gap: 8 }}>
                    {l.borrower}
                    {settled && <Pill color="#22c55e">settled</Pill>}
                    {!settled && l.dueOn && l.dueOn < todayISO() && <Pill color="#ef4444">overdue</Pill>}
                  </div>
                  <div className="row-actions">
                    <button className="icon-btn" onClick={() => startEdit(l)} title="Edit loan">
                      <IconPencil size={16} stroke={1.75} />
                    </button>
                    <button className="icon-btn danger" onClick={() => remove.mutate(l.id)} title="Delete loan">
                      <IconTrash size={16} stroke={1.75} />
                    </button>
                  </div>
                </div>

                <div className="muted" style={{ fontSize: 12 }}>
                  Lent {formatMoney(l.principal, l.currency)} on {l.lentOn}
                  {l.dueOn ? ` · due ${l.dueOn}` : ''}
                  {l.note ? ` · ${l.note}` : ''}
                </div>

                <div
                  style={{
                    height: 6,
                    borderRadius: 3,
                    background: 'var(--color-accent-soft, #8887801f)',
                    margin: '10px 0',
                    overflow: 'hidden',
                  }}
                >
                  <div
                    style={{
                      width: `${pct}%`,
                      height: '100%',
                      borderRadius: 3,
                      background: settled ? '#22c55e' : 'var(--color-positive, #16a34a)',
                    }}
                  />
                </div>

                <div className="mb-2 flex items-center justify-between" style={{ fontSize: 13 }}>
                  <span className="positive">Repaid {formatMoney(repaid, l.currency)}</span>
                  <span className={settled ? 'muted' : ''}>
                    {settled ? 'Fully repaid' : `Outstanding ${formatMoney(out, l.currency)}`}
                  </span>
                </div>

                {l.repayments.map((r) => (
                  <div className="txn" key={r.id}>
                    <div className="txn-info">
                      <div className="txn-name">{r.note || 'Repayment'}</div>
                      <div className="txn-cat">{r.date}</div>
                    </div>
                    <div className="txn-amt positive">+{formatMoney(r.amount, l.currency)}</div>
                    <div className="row-actions">
                      <button
                        className="icon-btn danger"
                        onClick={() => removeRepayment(l, r.id)}
                        title="Remove repayment"
                      >
                        <IconTrash size={16} stroke={1.75} />
                      </button>
                    </div>
                  </div>
                ))}

                {!settled && (
                  <Button variant="ghost" onClick={() => startRepay(l)}>
                    <IconCashBanknote size={15} stroke={1.75} /> Record repayment
                  </Button>
                )}
              </Card>
            )
          })}
        </div>
      )}

      <Modal title={editing ? 'Edit loan' : 'Add loan'} open={open} onClose={() => setOpen(false)}>
        <div className="grid grid-cols-2 gap-4">
          <Field label="Borrower">
            <Input
              placeholder="Who did you lend to?"
              value={form.borrower ?? ''}
              onChange={(e) => setForm({ ...form, borrower: e.target.value })}
            />
          </Field>
          <Field label="Amount lent">
            <Input
              type="number"
              value={form.principal ?? ''}
              onChange={(e) => setForm({ ...form, principal: Number(e.target.value) })}
            />
          </Field>
          <Field label="Lent on">
            <Input
              type="date"
              value={form.lentOn ?? ''}
              onChange={(e) => setForm({ ...form, lentOn: e.target.value })}
            />
          </Field>
          <Field label="Expected back by (optional)">
            <Input
              type="date"
              value={form.dueOn ?? ''}
              onChange={(e) => setForm({ ...form, dueOn: e.target.value })}
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
          <Button onClick={save} disabled={!form.borrower || !form.principal || !form.lentOn}>
            {editing ? 'Save' : 'Add loan'}
          </Button>
        </div>
      </Modal>

      <Modal
        title={repayFor ? `Repayment from ${repayFor.borrower}` : 'Repayment'}
        open={repayFor !== null}
        onClose={() => setRepayFor(null)}
      >
        <div className="grid grid-cols-2 gap-4">
          <Field label="Amount">
            <Input type="number" value={repayAmount} onChange={(e) => setRepayAmount(e.target.value)} />
          </Field>
          <Field label="Date">
            <Input type="date" value={repayDate} onChange={(e) => setRepayDate(e.target.value)} />
          </Field>
          <Field label="Note (optional)">
            <Input
              placeholder="EMI #3, partial, full & final…"
              value={repayNote}
              onChange={(e) => setRepayNote(e.target.value)}
            />
          </Field>
        </div>
        {repayFor && (
          <p className="muted" style={{ fontSize: 12, marginTop: 8 }}>
            Outstanding: {formatMoney(loanOutstanding(repayFor), repayFor.currency)} — partial amounts,
            EMIs, or a full settlement all work; any order is fine.
          </p>
        )}
        <div className="modal-actions">
          <Button variant="ghost" onClick={() => setRepayFor(null)}>
            Cancel
          </Button>
          <Button onClick={saveRepayment} disabled={!repayAmount || Number(repayAmount) <= 0}>
            Record
          </Button>
        </div>
      </Modal>
    </>
  )
}
