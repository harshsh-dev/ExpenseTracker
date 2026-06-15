import { useMemo, useRef, useState } from 'react'
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
import {
  IconChevronLeft,
  IconChevronRight,
  IconDownload,
  IconCalendar,
} from '@tabler/icons-react'
import { useCategories, useExpenses, useIncomes, useInvestments } from '../api/hooks'
import { IncomeAllocationPie } from '../components/IncomeAllocationPie'
import { Button, Card, Empty, PageHeader, Select } from '../components/ui'
import { formatMoney, monthName } from '../lib/format'
import { buildReport, shift, type ReportPeriod } from '../lib/report'
import { useTheme } from '../theme'
import { useFeatures } from '../features'

const periods: { id: ReportPeriod; label: string }[] = [
  { id: 'weekly', label: 'Weekly' },
  { id: 'monthly', label: 'Monthly' },
  { id: 'annual', label: 'Annual' },
]

export default function ReportsPage() {
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

  const [period, setPeriod] = useState<ReportPeriod>('monthly')
  const [anchor, setAnchor] = useState(() => new Date())
  const [busy, setBusy] = useState(false)
  const chartRef = useRef<HTMLDivElement>(null)

  const report = useMemo(
    () => buildReport({ period, anchor, incomes, expenses, investments, categories }),
    [period, anchor, incomes, expenses, investments, categories],
  )

  async function downloadPdf() {
    const svg = chartRef.current?.querySelector('svg') as SVGElement | null
    setBusy(true)
    try {
      // Lazy-loaded so jsPDF stays out of the initial bundle.
      const { generateReportPdf } = await import('../lib/pdf')
      await generateReportPdf(report, { chartSvg: svg })
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <PageHeader
        title="Reports"
        subtitle="Generate weekly, monthly or annual reports — visualize and download as PDF."
        action={
          <Button onClick={downloadPdf} disabled={busy || !report.hasData}>
            <IconDownload size={15} stroke={1.75} />
            {busy ? 'Generating…' : 'Download PDF'}
          </Button>
        }
      />

      <Card>
        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 12,
          }}
        >
          <div style={{ width: 160 }}>
            <Select value={period} onChange={(e) => setPeriod(e.target.value as ReportPeriod)}>
              {periods.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.label}
                </option>
              ))}
            </Select>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <button
              className="icon-btn"
              onClick={() => setAnchor((a) => shift(period, a, -1))}
              title="Previous"
            >
              <IconChevronLeft size={18} stroke={1.75} />
            </button>
            <div
              style={{
                minWidth: 200,
                textAlign: 'center',
                fontWeight: 600,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                gap: 6,
              }}
            >
              <IconCalendar size={15} stroke={1.75} />
              {report.range.label}
            </div>
            <button
              className="icon-btn"
              onClick={() => setAnchor((a) => shift(period, a, 1))}
              title="Next"
            >
              <IconChevronRight size={18} stroke={1.75} />
            </button>
            <Button variant="ghost" onClick={() => setAnchor(new Date())}>
              Today
            </Button>
          </div>
        </div>
      </Card>

      <div className="stats">
        {hasIncome && <Stat label="Income" value={formatMoney(report.income)} />}
        {hasExpenses && <Stat label="Expenses" value={formatMoney(report.expense)} />}
        {hasIncome && hasExpenses && (
          <Stat
            label="Net savings"
            value={formatMoney(report.net)}
            sub={`${report.savingsRate.toFixed(0)}% savings rate`}
            tone={report.net >= 0 ? 'positive' : 'negative'}
          />
        )}
        {hasInvestments && (
          <Stat
            label="Invested this period"
            value={formatMoney(report.investedInPeriod)}
            sub={`${report.investedCount} new`}
          />
        )}
      </div>

      {!report.hasData ? (
        <Empty>No data for this period. Try another range or add some entries.</Empty>
      ) : (
        <>
          {(hasIncome || hasExpenses) && (
            <div className="grid gap-5 lg:grid-cols-2">
              <Card>
                <div className="card-title">Income vs Expense (trend)</div>
                <div ref={chartRef} style={{ width: '100%', height: 280 }}>
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={report.trend} barGap={6}>
                      <CartesianGrid strokeDasharray="3 3" stroke={chart.grid} vertical={false} />
                      <XAxis dataKey="label" stroke={chart.axis} fontSize={12} tickLine={false} />
                      <YAxis
                        stroke={chart.axis}
                        fontSize={12}
                        width={52}
                        tickLine={false}
                        axisLine={false}
                      />
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

              <Card>
                <div className="card-title">Income allocation</div>
                {hasIncome && report.income > 0 ? (
                  <IncomeAllocationPie
                    income={report.income}
                    categories={report.byCategory.map((c) => ({
                      name: c.name,
                      value: c.value,
                      color: c.color,
                    }))}
                    tooltipStyle={{
                      background: chart.tooltipBg,
                      border: `0.5px solid ${chart.tooltipBorder}`,
                      borderRadius: 8,
                      fontSize: 12,
                      color: chart.tooltipText,
                    }}
                    labelStyle={{ color: chart.tooltipText }}
                  />
                ) : (
                  <Empty>No income recorded for this period.</Empty>
                )}
              </Card>
            </div>
          )}

          {hasExpenses && (
            <div className="grid gap-5 lg:grid-cols-2">
              <Card>
                <div className="card-title">Spending by category</div>
                {report.byCategory.length === 0 ? (
                  <Empty>No expenses in this period.</Empty>
                ) : (
                  report.byCategory.slice(0, 8).map((c) => (
                    <div className="bar-row" key={c.name}>
                      <div className="bar-label">
                        <span className="pill-dot" style={{ background: c.color }} />
                        {c.name}
                      </div>
                      <div className="bar-track">
                        <div className="bar-fill" style={{ width: `${c.pct}%`, background: c.color }} />
                      </div>
                      <div className="bar-amt">{formatMoney(c.value)}</div>
                    </div>
                  ))
                )}
              </Card>

              <Card>
                <div className="card-title">Top expenses</div>
                {report.topExpenses.length === 0 ? (
                  <Empty>No expenses in this period.</Empty>
                ) : (
                  report.topExpenses.slice(0, 7).map((e, i) => (
                    <div className="txn" key={i}>
                      <div className="txn-info">
                        <div className="txn-name">{e.note || e.category}</div>
                        <div className="txn-cat">
                          {e.category} · {e.date}
                        </div>
                      </div>
                      <div className="txn-amt negative">−{formatMoney(e.amount)}</div>
                    </div>
                  ))
                )}
              </Card>
            </div>
          )}

          {hasIncome && report.sources.length > 0 && (
            <Card>
              <div className="card-title">Income sources</div>
              {report.sources.map((s) => (
                <div className="txn" key={s.source}>
                  <div className="txn-info">
                    <div className="txn-name">{s.source}</div>
                  </div>
                  <div className="txn-amt positive">{formatMoney(s.value)}</div>
                </div>
              ))}
            </Card>
          )}

          {hasInvestments && (
            <Card>
              <div className="card-title">Portfolio snapshot</div>
              <div className="txn">
                <div className="txn-info">
                  <div className="txn-name">Current value</div>
                  <div className="txn-cat">As of {monthName(new Date().getMonth() + 1)} {new Date().getFullYear()}</div>
                </div>
                <div className="txn-amt">{formatMoney(report.portfolioValue)}</div>
              </div>
              <div className="txn">
                <div className="txn-info">
                  <div className="txn-name">Total P/L</div>
                </div>
                <div className={`txn-amt ${report.portfolioPnl >= 0 ? 'positive' : 'negative'}`}>
                  {formatMoney(report.portfolioPnl)}
                </div>
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
