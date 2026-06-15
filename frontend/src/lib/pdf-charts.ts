import type { CategorySlice, TrendBucket } from './report'

const INCOME_COLOR = '#1d9e75'
const EXPENSE_COLOR = '#d4537e'
const UNSPENT_COLOR = '#1d9e75'
const AXIS_COLOR = '#888780'
const GRID_COLOR = '#e8e8e6'
const TEXT_COLOR = '#1c1c1a'
const MUTED_COLOR = '#6f6f6a'

function hexToRgb(hex: string): [number, number, number] {
  const h = hex.replace('#', '')
  const n = parseInt(h.length === 3 ? h.split('').map((c) => c + c).join('') : h, 16)
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255]
}

function compactMoney(n: number): string {
  if (n >= 100_000) return `INR ${(n / 100_000).toFixed(1)}L`
  if (n >= 1_000) return `INR ${(n / 1_000).toFixed(1)}K`
  return `INR ${Math.round(n)}`
}

function fullMoney(n: number): string {
  try {
    return new Intl.NumberFormat('en-IN', {
      style: 'currency',
      currency: 'INR',
      currencyDisplay: 'code',
      maximumFractionDigits: 0,
    }).format(n)
  } catch {
    return `INR ${n.toFixed(0)}`
  }
}

function canvasToPng(canvas: HTMLCanvasElement): string {
  return canvas.toDataURL('image/png')
}

// renderTrendBarChart draws the income vs expense trend as a PNG for PDF embed.
export function renderTrendBarChart(
  trend: TrendBucket[],
  opts: { width: number; height: number; showIncome: boolean; showExpense: boolean },
): string | null {
  if (trend.length === 0) return null
  const { width, height, showIncome, showExpense } = opts
  if (!showIncome && !showExpense) return null

  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) return null

  ctx.fillStyle = '#ffffff'
  ctx.fillRect(0, 0, width, height)

  const pad = { top: 28, right: 16, bottom: 44, left: 56 }
  const chartW = width - pad.left - pad.right
  const chartH = height - pad.top - pad.bottom

  const maxVal = Math.max(
    1,
    ...trend.flatMap((b) => [
      showIncome ? b.income : 0,
      showExpense ? b.expense : 0,
    ]),
  )

  // Grid + Y labels
  ctx.strokeStyle = GRID_COLOR
  ctx.fillStyle = MUTED_COLOR
  ctx.font = '11px Helvetica, Arial, sans-serif'
  ctx.textAlign = 'right'
  ctx.textBaseline = 'middle'
  for (let i = 0; i <= 4; i++) {
    const val = (maxVal * i) / 4
    const y = pad.top + chartH - (chartH * i) / 4
    ctx.beginPath()
    ctx.moveTo(pad.left, y)
    ctx.lineTo(pad.left + chartW, y)
    ctx.stroke()
    ctx.fillText(compactMoney(val), pad.left - 8, y)
  }

  const groupW = chartW / trend.length
  const barCount = (showIncome ? 1 : 0) + (showExpense ? 1 : 0)
  const barW = Math.min(18, (groupW - 12) / Math.max(barCount, 1))

  trend.forEach((bucket, i) => {
    const gx = pad.left + i * groupW + groupW / 2
    let offset = -((barCount - 1) * (barW + 4)) / 2

    ctx.fillStyle = TEXT_COLOR
    ctx.font = '10px Helvetica, Arial, sans-serif'
    ctx.textAlign = 'center'
    ctx.textBaseline = 'top'
    const label =
      bucket.label.length > 8 ? bucket.label.slice(0, 7) + '…' : bucket.label
    ctx.fillText(label, gx, pad.top + chartH + 10)

    const drawBar = (value: number, color: string) => {
      if (value <= 0) {
        offset += barW + 4
        return
      }
      const h = (value / maxVal) * chartH
      const x = gx + offset - barW / 2
      const y = pad.top + chartH - h
      ctx.fillStyle = color
      ctx.fillRect(x, y, barW, h)
      offset += barW + 4
    }

    if (showIncome) drawBar(bucket.income, INCOME_COLOR)
    if (showExpense) drawBar(bucket.expense, EXPENSE_COLOR)
  })

  // Legend
  let lx = pad.left
  const ly = 12
  ctx.font = '11px Helvetica, Arial, sans-serif'
  ctx.textAlign = 'left'
  ctx.textBaseline = 'middle'
  if (showIncome) {
    ctx.fillStyle = INCOME_COLOR
    ctx.fillRect(lx, ly - 5, 10, 10)
    ctx.fillStyle = TEXT_COLOR
    ctx.fillText('Income', lx + 14, ly)
    lx += 70
  }
  if (showExpense) {
    ctx.fillStyle = EXPENSE_COLOR
    ctx.fillRect(lx, ly - 5, 10, 10)
    ctx.fillStyle = TEXT_COLOR
    ctx.fillText('Expense', lx + 14, ly)
  }

  return canvasToPng(canvas)
}

type PieSlice = { name: string; value: number; color: string }

function buildPieSlices(income: number, categories: CategorySlice[]): PieSlice[] {
  const expenseTotal = categories.reduce((s, c) => s + c.value, 0)
  const unspent = Math.max(0, income - expenseTotal)
  const slices = categories.filter((c) => c.value > 0).map((c) => ({
    name: c.name,
    value: c.value,
    color: c.color,
  }))
  if (unspent > 0) slices.push({ name: 'Unspent', value: unspent, color: UNSPENT_COLOR })
  return slices
}

// renderAllocationPie draws the income allocation donut + legend as a PNG.
export function renderAllocationPie(
  income: number,
  categories: CategorySlice[],
  opts: { width: number; height: number },
): string | null {
  if (income <= 0) return null
  const slices = buildPieSlices(income, categories)
  if (slices.length === 0) return null

  const { width, height } = opts
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) return null

  ctx.fillStyle = '#ffffff'
  ctx.fillRect(0, 0, width, height)

  const cx = width / 2
  const cy = 108
  const outerR = 88
  const innerR = 58
  const total = slices.reduce((s, sl) => s + sl.value, 0)

  let start = -Math.PI / 2
  for (const sl of slices) {
    const angle = (sl.value / total) * Math.PI * 2
    const end = start + angle
    const [r, g, b] = hexToRgb(sl.color)
    ctx.fillStyle = `rgb(${r},${g},${b})`
    ctx.beginPath()
    ctx.arc(cx, cy, outerR, start, end)
    ctx.arc(cx, cy, innerR, end, start, true)
    ctx.closePath()
    ctx.fill()
    start = end
  }

  // Center label
  ctx.textAlign = 'center'
  ctx.fillStyle = MUTED_COLOR
  ctx.font = '11px Helvetica, Arial, sans-serif'
  ctx.fillText('Total received', cx, cy - 10)
  ctx.fillStyle = TEXT_COLOR
  ctx.font = 'bold 13px Helvetica, Arial, sans-serif'
  ctx.fillText(fullMoney(income), cx, cy + 10)

  // Legend
  const legendTop = 200
  ctx.textAlign = 'left'
  ctx.font = '11px Helvetica, Arial, sans-serif'
  slices.forEach((sl, i) => {
    const y = legendTop + i * 20
    const [r, g, b] = hexToRgb(sl.color)
    ctx.fillStyle = `rgb(${r},${g},${b})`
    ctx.beginPath()
    ctx.arc(24, y, 5, 0, Math.PI * 2)
    ctx.fill()
    ctx.fillStyle = TEXT_COLOR
    const name = sl.name.length > 22 ? sl.name.slice(0, 21) + '…' : sl.name
    ctx.fillText(name, 36, y + 4)
    ctx.textAlign = 'right'
    ctx.fillStyle = sl.name === 'Unspent' ? INCOME_COLOR : TEXT_COLOR
    ctx.font = sl.name === 'Unspent' ? 'bold 11px Helvetica, Arial, sans-serif' : '11px Helvetica, Arial, sans-serif'
    ctx.fillText(fullMoney(sl.value), width - 24, y + 4)
    ctx.textAlign = 'left'
    ctx.font = '11px Helvetica, Arial, sans-serif'
  })

  return canvasToPng(canvas)
}
