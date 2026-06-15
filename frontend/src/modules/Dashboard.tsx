import { useMemo } from 'react'
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { IconCalendar } from '@tabler/icons-react'
import { useCategories, useExpenses, useIncomes, useInvestments } from '../api/hooks'
import { investmentCurrentValue } from '../types'
import { Card, Empty } from '../components/ui'
import { formatMoney, isoDateParts, monthName } from '../lib/format'
import { useTheme } from '../theme'
import { useFeatures } from '../features'

export default function Dashboard() {
  const { theme } = useTheme()
  const dark = theme === 'dark'
  const features = useFeatures()
  const hasIncome = features.has('income')
  const hasExpenses = features.has('expenses')
  const hasInvestments = features.has('investments')
  const chart = {
    grid: dark ? 'rgba(255,255,255,0.08)' : 'rgba(20,20,20,0.06)',
    axis: dark ? '#6f6f6a' : '#a0a09a',
    tooltipBg: dark ? '#1b1b1a' : '#ffffff',
    tooltipBorder: dark ? 'rgba(255,255,255,0.16)' : 'rgba(20,20,20,0.12)',
    cursor: dark ? 'rgba(255,255,255,0.05)' : 'rgba(20,20,20,0.04)',
    tooltipText: dark ? '#ededec' : '#1c1c1a',
  }

  const { data: incomes = [] } = useIncomes()
  const { data: expenses = [] } = useExpenses()
  const { data: investments = [] } = useInvestments()
  const { data: categories = [] } = useCategories()

  const totalIncome = incomes.reduce((s, i) => s + i.amount, 0)
  const totalExpense = expenses.reduce((s, e) => s + e.amount, 0)
  const net = totalIncome - totalExpense
  const savingsRate = totalIncome > 0 ? (net / totalIncome) * 100 : 0

  const invested = investments.reduce((s, i) => s + i.amountInvested, 0)
  const portfolio = investments.reduce(
    (s, i) => s + (investmentCurrentValue(i) ?? i.amountInvested),
    0,
  )
  const portfolioPnl = portfolio - invested

  const monthly = useMemo(() => {
    const map = new Map<string, { key: string; label: string; income: number; expense: number }>()
    const touch = (y: number, m: number) => {
      const key = `${y}-${String(m).padStart(2, '0')}`
      if (!map.has(key))
        map.set(key, { key, label: `${monthName(m)} ${String(y).slice(2)}`, income: 0, expense: 0 })
      return map.get(key)!
    }
    for (const i of incomes) touch(i.year, i.month).income += i.amount
    for (const e of expenses) {
      const parts = isoDateParts(e.date)
      if (!parts) continue
      touch(parts.year, parts.month).expense += e.amount
    }
    return [...map.values()].sort((a, b) => a.key.localeCompare(b.key)).slice(-6)
  }, [incomes, expenses])

  const byCategory = useMemo(() => {
    const meta = Object.fromEntries(categories.map((c) => [c.id, c]))
    const map = new Map<string, { name: string; value: number; color: string }>()
    for (const e of expenses) {
      const cat = meta[e.categoryId]
      const cur = map.get(e.categoryId) ?? {
        name: cat?.name ?? 'Unknown',
        value: 0,
        color: cat?.color ?? '#888780',
      }
      cur.value += e.amount
      map.set(e.categoryId, cur)
    }
    const arr = [...map.values()].sort((a, b) => b.value - a.value)
    const max = arr[0]?.value ?? 1
    return arr.map((c) => ({ ...c, pct: (c.value / max) * 100 }))
  }, [expenses, categories])

  const recent = useMemo(() => {
    const meta = Object.fromEntries(categories.map((c) => [c.id, c]))
    return [...expenses]
      .sort((a, b) => (a.date < b.date ? 1 : -1))
      .slice(0, 6)
      .map((e) => ({ ...e, cat: meta[e.categoryId] }))
  }, [expenses, categories])

  const now = new Date()
  const hasData = incomes.length + expenses.length + investments.length > 0

  return (
    <>
      <div className="topbar">
        <div>
          <h1>Dashboard</h1>
          <div className="subtitle">
            <IconCalendar size={14} stroke={1.75} />
            {monthName(now.getMonth() + 1)} {now.getFullYear()}
          </div>
        </div>
      </div>

      <div className="stats">
        {hasIncome && <Stat label="Total income" value={formatMoney(totalIncome)} />}
        {hasExpenses && <Stat label="Total expenses" value={formatMoney(totalExpense)} />}
        {hasIncome && hasExpenses && (
          <Stat
            label="Net savings"
            value={formatMoney(net)}
            sub={`${savingsRate.toFixed(0)}% savings rate`}
            tone={net >= 0 ? 'positive' : 'negative'}
          />
        )}
        {hasInvestments && (
          <Stat
            label="Portfolio"
            value={formatMoney(portfolio)}
            sub={`${portfolioPnl >= 0 ? '+' : ''}${formatMoney(portfolioPnl)} P/L`}
            tone={portfolioPnl >= 0 ? 'positive' : 'negative'}
          />
        )}
      </div>

      {!hasData ? (
        <Empty>Add some data to see your dashboard come alive.</Empty>
      ) : (
        <>
          {hasExpenses && (
            <div className="grid gap-5 lg:grid-cols-2">
              <Card>
                <div className="card-title">Spending by category</div>
                {byCategory.length === 0 ? (
                  <Empty>No expenses yet.</Empty>
                ) : (
                  byCategory.slice(0, 6).map((c) => (
                    <div className="bar-row" key={c.name}>
                      <div className="bar-label">
                        <span className="pill-dot" style={{ background: c.color }} />
                        {c.name}
                      </div>
                      <div className="bar-track">
                        <div
                          className="bar-fill"
                          style={{ width: `${c.pct}%`, background: c.color }}
                        />
                      </div>
                      <div className="bar-amt">{formatMoney(c.value)}</div>
                    </div>
                  ))
                )}
              </Card>

              <Card>
                <div className="card-title">Recent expenses</div>
                {recent.length === 0 ? (
                  <Empty>No expenses yet.</Empty>
                ) : (
                  recent.map((e) => (
                    <div className="txn" key={e.id}>
                      <div
                        className="txn-icon"
                        style={{ background: (e.cat?.color ?? '#888780') + '22' }}
                      >
                        <span
                          className="pill-dot"
                          style={{ background: e.cat?.color ?? '#888780' }}
                        />
                      </div>
                      <div className="txn-info">
                        <div className="txn-name">{e.note || e.cat?.name || 'Expense'}</div>
                        <div className="txn-cat">
                          {e.cat?.name ?? 'Unknown'} · {e.date}
                        </div>
                      </div>
                      <div className="txn-amt negative">−{formatMoney(e.amount, e.currency)}</div>
                    </div>
                  ))
                )}
              </Card>
            </div>
          )}

          {(hasIncome || hasExpenses) && (
            <Card>
            <div className="card-title">Income vs Expense (last 6 months)</div>
            <div style={{ width: '100%', height: 280 }}>
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={monthly} barGap={6}>
                  <CartesianGrid strokeDasharray="3 3" stroke={chart.grid} vertical={false} />
                  <XAxis dataKey="label" stroke={chart.axis} fontSize={12} tickLine={false} />
                  <YAxis stroke={chart.axis} fontSize={12} width={52} tickLine={false} axisLine={false} />
                  <Tooltip
                    cursor={{ fill: chart.cursor }}
                    contentStyle={{
                      background: chart.tooltipBg,
                      border: `0.5px solid ${chart.tooltipBorder}`,
                      borderRadius: 8,
                      fontSize: 12,
                      color: chart.tooltipText,
                    }}
                    labelStyle={{ color: chart.tooltipText }}
                    formatter={(v) => formatMoney(Number(v))}
                  />
                  <Legend wrapperStyle={{ fontSize: 12 }} />
                  {hasIncome && (
                    <Bar dataKey="income" name="Income" fill="#1d9e75" radius={[4, 4, 0, 0]} />
                  )}
                  {hasExpenses && (
                    <Bar dataKey="expense" name="Expense" fill="#d4537e" radius={[4, 4, 0, 0]} />
                  )}
              </BarChart>
              </ResponsiveContainer>
            </div>
          </Card>
          )}
        </>
      )}
    </>
  )
}

function Stat({
  label,
  value,
  sub,
  tone,
}: {
  label: string
  value: string
  sub?: string
  tone?: 'positive' | 'negative'
}) {
  return (
    <Card>
      <div className="stat-label">{label}</div>
      <div className="stat-value">{value}</div>
      {sub && <div className={`stat-sub ${tone ?? ''}`}>{sub}</div>}
    </Card>
  )
}
