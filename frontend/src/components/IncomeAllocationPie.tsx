import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'
import { formatMoney } from '../lib/format'

const UNSPENT_COLOR = '#1d9e75'

export type CategorySlice = { name: string; value: number; color: string }

type Slice = CategorySlice & { kind: 'category' | 'unspent' }

function buildSlices(income: number, categories: CategorySlice[]): {
  slices: Slice[]
  overspent: number
} {
  const expenseTotal = categories.reduce((s, c) => s + c.value, 0)
  const unspent = Math.max(0, income - expenseTotal)
  const overspent = Math.max(0, expenseTotal - income)

  const slices: Slice[] = categories
    .filter((c) => c.value > 0)
    .map((c) => ({ name: c.name, value: c.value, color: c.color, kind: 'category' as const }))

  if (unspent > 0) {
    slices.push({ name: 'Unspent', value: unspent, color: UNSPENT_COLOR, kind: 'unspent' })
  }

  return { slices, overspent }
}

// IncomeAllocationPie: full donut = total income. Slices = expense categories + unspent.
export function IncomeAllocationPie({
  income,
  categories,
  tooltipStyle,
  labelStyle,
}: {
  income: number
  categories: CategorySlice[]
  tooltipStyle?: React.CSSProperties
  labelStyle?: React.CSSProperties
}) {
  if (income <= 0) return null

  const { slices, overspent } = buildSlices(income, categories)
  if (slices.length === 0) return null

  return (
    <div>
      <div style={{ position: 'relative', width: '100%', height: 240 }}>
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={slices}
              dataKey="value"
              nameKey="name"
              cx="50%"
              cy="50%"
              innerRadius={68}
              outerRadius={100}
              paddingAngle={slices.length > 1 ? 2 : 0}
              stroke="none"
            >
              {slices.map((entry) => (
                <Cell key={entry.name} fill={entry.color} />
              ))}
            </Pie>
            <Tooltip
              contentStyle={tooltipStyle}
              labelStyle={labelStyle}
              formatter={(v, name) => [formatMoney(Number(v)), String(name)]}
            />
          </PieChart>
        </ResponsiveContainer>
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            textAlign: 'center',
            pointerEvents: 'none',
            maxWidth: 120,
          }}
        >
          <div className="muted" style={{ fontSize: 11 }}>
            Total received
          </div>
          <div style={{ fontWeight: 600, fontSize: 15, color: 'var(--color-text-primary)' }}>
            {formatMoney(income)}
          </div>
        </div>
      </div>

      <div style={{ marginTop: 12, display: 'flex', flexDirection: 'column', gap: 6 }}>
        {slices.map((s) => (
          <div
            key={s.name}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              fontSize: 12,
              gap: 8,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 0 }}>
              <span
                className="pill-dot"
                style={{ background: s.color, flexShrink: 0 }}
              />
              <span
                style={{
                  color: 'var(--color-text-primary)',
                  fontWeight: s.kind === 'unspent' ? 600 : 400,
                }}
              >
                {s.name}
              </span>
            </div>
            <span
              style={{
                fontWeight: 500,
                color: s.kind === 'unspent' ? 'var(--color-positive)' : 'var(--color-text-primary)',
                flexShrink: 0,
              }}
            >
              {formatMoney(s.value)}
            </span>
          </div>
        ))}
      </div>

      {overspent > 0 && (
        <p className="negative" style={{ fontSize: 12, marginTop: 10, textAlign: 'center' }}>
          Expenses exceed income by {formatMoney(overspent)}
        </p>
      )}
    </div>
  )
}
