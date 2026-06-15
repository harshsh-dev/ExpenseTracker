import type { Category, Expense, Income, Investment } from '../types'
import { investmentCurrentValue } from '../types'
import { monthName } from './format'

export type ReportPeriod = 'weekly' | 'monthly' | 'annual'

export interface ReportRange {
  start: Date
  end: Date
  startISO: string
  endISO: string
  label: string
}

export interface CategorySlice {
  name: string
  color: string
  value: number
  pct: number
}

export interface SourceSlice {
  source: string
  value: number
}

export interface TrendBucket {
  label: string
  income: number
  expense: number
}

export interface TopExpense {
  date: string
  category: string
  note: string
  amount: number
}

export interface ReportData {
  period: ReportPeriod
  range: ReportRange
  income: number
  expense: number
  net: number
  savingsRate: number
  byCategory: CategorySlice[]
  sources: SourceSlice[]
  topExpenses: TopExpense[]
  trend: TrendBucket[]
  investedInPeriod: number
  investedCount: number
  portfolioValue: number
  portfolioPnl: number
  hasData: boolean
}

function iso(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function addDays(d: Date, n: number): Date {
  const r = new Date(d)
  r.setDate(r.getDate() + n)
  return r
}

// monday returns the Monday of the week containing d (local time).
function monday(d: Date): Date {
  const r = new Date(d.getFullYear(), d.getMonth(), d.getDate())
  const dow = (r.getDay() + 6) % 7 // 0 = Monday
  return addDays(r, -dow)
}

// rangeFor computes the inclusive date window + a human label for a period.
export function rangeFor(period: ReportPeriod, anchor: Date): ReportRange {
  if (period === 'weekly') {
    const start = monday(anchor)
    const end = addDays(start, 6)
    const sameMonth = start.getMonth() === end.getMonth()
    const label = sameMonth
      ? `${monthName(start.getMonth() + 1)} ${start.getDate()}–${end.getDate()}, ${end.getFullYear()}`
      : `${monthName(start.getMonth() + 1)} ${start.getDate()} – ${monthName(end.getMonth() + 1)} ${end.getDate()}, ${end.getFullYear()}`
    return { start, end, startISO: iso(start), endISO: iso(end), label }
  }
  if (period === 'monthly') {
    const start = new Date(anchor.getFullYear(), anchor.getMonth(), 1)
    const end = new Date(anchor.getFullYear(), anchor.getMonth() + 1, 0)
    return {
      start,
      end,
      startISO: iso(start),
      endISO: iso(end),
      label: `${monthName(start.getMonth() + 1)} ${start.getFullYear()}`,
    }
  }
  const start = new Date(anchor.getFullYear(), 0, 1)
  const end = new Date(anchor.getFullYear(), 11, 31)
  return { start, end, startISO: iso(start), endISO: iso(end), label: String(start.getFullYear()) }
}

// shift moves the anchor by one period in the given direction (+1 / -1).
export function shift(period: ReportPeriod, anchor: Date, dir: number): Date {
  if (period === 'weekly') return addDays(anchor, dir * 7)
  if (period === 'monthly') return new Date(anchor.getFullYear(), anchor.getMonth() + dir, 1)
  return new Date(anchor.getFullYear() + dir, anchor.getMonth(), 1)
}

function inRange(dateISO: string, range: ReportRange): boolean {
  return dateISO >= range.startISO && dateISO <= range.endISO
}

// incomeInBucket places each income entry in the trend bucket where it was
// actually received (receivedOn), matching how expenses use their date field.
// Annual view groups by the income's month/year (same as the Dashboard).
function incomeInBucket(inc: Income, bStart: Date, bEnd: Date, period: ReportPeriod): number {
  if (period === 'annual') {
    return inc.year === bStart.getFullYear() && inc.month === bStart.getMonth() + 1 ? inc.amount : 0
  }
  if (!inc.receivedOn) return 0
  const bStartISO = iso(bStart)
  const bEndISO = iso(bEnd)
  return inc.receivedOn >= bStartISO && inc.receivedOn <= bEndISO ? inc.amount : 0
}

// incomeForPeriod totals income for the selected window. Monthly/annual use the
// income's month+year (how users record salary); weekly uses receivedOn.
function incomeForPeriod(inc: Income, period: ReportPeriod, range: ReportRange): number {
  if (period === 'weekly') {
    return inc.receivedOn && inRange(inc.receivedOn, range) ? inc.amount : 0
  }
  if (period === 'monthly') {
    return inc.year === range.start.getFullYear() && inc.month === range.start.getMonth() + 1
      ? inc.amount
      : 0
  }
  return inc.year === range.start.getFullYear() ? inc.amount : 0
}

function buildTrend(
  period: ReportPeriod,
  range: ReportRange,
  incomes: Income[],
  expenses: Expense[],
): TrendBucket[] {
  const buckets: { label: string; start: Date; end: Date }[] = []

  if (period === 'weekly') {
    const names = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
    for (let i = 0; i < 7; i++) {
      const d = addDays(range.start, i)
      buckets.push({ label: names[i], start: d, end: d })
    }
  } else if (period === 'monthly') {
    const last = range.end.getDate()
    for (let day = 1; day <= last; day += 7) {
      const start = new Date(range.start.getFullYear(), range.start.getMonth(), day)
      const end = new Date(range.start.getFullYear(), range.start.getMonth(), Math.min(day + 6, last))
      buckets.push({ label: `${day}–${Math.min(day + 6, last)}`, start, end })
    }
  } else {
    for (let m = 0; m < 12; m++) {
      const start = new Date(range.start.getFullYear(), m, 1)
      const end = new Date(range.start.getFullYear(), m + 1, 0)
      buckets.push({ label: monthName(m + 1), start, end })
    }
  }

  return buckets.map((b) => {
    const bStartISO = iso(b.start)
    const bEndISO = iso(b.end)
    const expense = expenses
      .filter((e) => e.date >= bStartISO && e.date <= bEndISO)
      .reduce((s, e) => s + e.amount, 0)
    const income = incomes.reduce((s, i) => s + incomeInBucket(i, b.start, b.end, period), 0)
    return { label: b.label, income, expense }
  })
}

export function buildReport(args: {
  period: ReportPeriod
  anchor: Date
  incomes: Income[]
  expenses: Expense[]
  investments: Investment[]
  categories: Category[]
}): ReportData {
  const { period, anchor, incomes, expenses, investments, categories } = args
  const range = rangeFor(period, anchor)

  const income = incomes.reduce((s, i) => s + incomeForPeriod(i, period, range), 0)

  const periodExpenses = expenses.filter((e) => inRange(e.date, range))
  const expense = periodExpenses.reduce((s, e) => s + e.amount, 0)
  const net = income - expense
  const savingsRate = income > 0 ? (net / income) * 100 : 0

  const catMeta = Object.fromEntries(categories.map((c) => [c.id, c]))
  const catMap = new Map<string, CategorySlice>()
  for (const e of periodExpenses) {
    const meta = catMeta[e.categoryId]
    const cur = catMap.get(e.categoryId) ?? {
      name: meta?.name ?? 'Uncategorized',
      color: meta?.color ?? '#888780',
      value: 0,
      pct: 0,
    }
    cur.value += e.amount
    catMap.set(e.categoryId, cur)
  }
  const byCategory = [...catMap.values()].sort((a, b) => b.value - a.value)
  const max = byCategory[0]?.value ?? 1
  for (const c of byCategory) c.pct = (c.value / max) * 100

  const srcMap = new Map<string, number>()
  for (const i of incomes) {
    const v = incomeForPeriod(i, period, range)
    if (v <= 0) continue
    srcMap.set(i.source, (srcMap.get(i.source) ?? 0) + v)
  }
  const sources = [...srcMap.entries()]
    .map(([source, value]) => ({ source, value }))
    .sort((a, b) => b.value - a.value)

  const topExpenses: TopExpense[] = [...periodExpenses]
    .sort((a, b) => b.amount - a.amount)
    .slice(0, 10)
    .map((e) => ({
      date: e.date,
      category: catMeta[e.categoryId]?.name ?? 'Uncategorized',
      note: e.note ?? '',
      amount: e.amount,
    }))

  const periodInvestments = investments.filter((i) => inRange(i.investedOn, range))
  const investedInPeriod = periodInvestments.reduce((s, i) => s + i.amountInvested, 0)
  const portfolioValue = investments.reduce(
    (s, i) => s + (investmentCurrentValue(i) ?? i.amountInvested),
    0,
  )
  const portfolioInvested = investments.reduce((s, i) => s + i.amountInvested, 0)

  return {
    period,
    range,
    income,
    expense,
    net,
    savingsRate,
    byCategory,
    sources,
    topExpenses,
    trend: buildTrend(period, range, incomes, periodExpenses),
    investedInPeriod,
    investedCount: periodInvestments.length,
    portfolioValue,
    portfolioPnl: portfolioValue - portfolioInvested,
    hasData: income > 0 || periodExpenses.length > 0 || periodInvestments.length > 0,
  }
}
