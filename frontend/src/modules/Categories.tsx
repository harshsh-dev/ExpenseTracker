import { useState } from 'react'
import { IconPlus, IconPencil, IconTrash } from '@tabler/icons-react'
import { useCategories, useCategoryCrud } from '../api/hooks'
import type { Category } from '../types'
import { Button, Card, Empty, Field, Input, Modal, PageHeader, Pill } from '../components/ui'

function emptyForm(): Partial<Category> & { subsText?: string } {
  return { name: '', color: '#1d9e75', subsText: '' }
}

export default function CategoriesPage() {
  const { data: categories = [], isLoading } = useCategories()
  const { create, update, remove } = useCategoryCrud()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<Category | null>(null)
  const [form, setForm] = useState<Partial<Category> & { subsText?: string }>(emptyForm())

  function startCreate() {
    setEditing(null)
    setForm(emptyForm())
    setOpen(true)
  }
  function startEdit(cat: Category) {
    setEditing(cat)
    setForm({ ...cat, subsText: cat.subcategories.join(', ') })
    setOpen(true)
  }
  async function save() {
    const subcategories = (form.subsText ?? '')
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
    const body = {
      name: form.name,
      color: form.color,
      subcategories,
      archived: form.archived ?? false,
    }
    if (editing) await update.mutateAsync({ id: editing.id, body })
    else await create.mutateAsync(body)
    setOpen(false)
  }

  return (
    <>
      <PageHeader
        title="Categories"
        subtitle="Customize your expense taxonomy"
        action={
          <Button onClick={startCreate}>
            <IconPlus size={15} stroke={2} /> Add category
          </Button>
        }
      />

      {isLoading ? (
        <Empty>Loading…</Empty>
      ) : categories.length === 0 ? (
        <Empty>No categories.</Empty>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2">
          {categories.map((cat) => (
            <Card key={cat.id}>
              <div className="flex items-start justify-between">
                <div>
                  <Pill color={cat.color}>{cat.name}</Pill>
                  {cat.subcategories.length > 0 && (
                    <div className="muted mt-2" style={{ fontSize: 11 }}>
                      {cat.subcategories.join(' · ')}
                    </div>
                  )}
                </div>
                <div className="row-actions">
                  <button className="icon-btn" onClick={() => startEdit(cat)} title="Edit">
                    <IconPencil size={16} stroke={1.75} />
                  </button>
                  <button
                    className="icon-btn danger"
                    onClick={() => remove.mutate(cat.id)}
                    title="Delete"
                  >
                    <IconTrash size={16} stroke={1.75} />
                  </button>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      <Modal
        title={editing ? 'Edit category' : 'Add category'}
        open={open}
        onClose={() => setOpen(false)}
      >
        <div className="flex flex-col gap-4">
          <Field label="Name">
            <Input value={form.name ?? ''} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </Field>
          <Field label="Color">
            <Input
              type="color"
              style={{ height: 40, width: 80, padding: 4 }}
              value={form.color ?? '#1d9e75'}
              onChange={(e) => setForm({ ...form, color: e.target.value })}
            />
          </Field>
          <Field label="Subcategories (comma-separated)">
            <Input
              value={form.subsText ?? ''}
              onChange={(e) => setForm({ ...form, subsText: e.target.value })}
              placeholder="e.g. Restaurants, Cafes, Takeout"
            />
          </Field>
        </div>
        <div className="modal-actions">
          <Button variant="ghost" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button onClick={save} disabled={!form.name}>
            {editing ? 'Save' : 'Add category'}
          </Button>
        </div>
      </Modal>
    </>
  )
}
