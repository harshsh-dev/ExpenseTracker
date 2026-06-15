import { jsPDF } from 'jspdf'
import autoTable from 'jspdf-autotable'
import { renderAllocationPie, renderTrendBarChart } from './pdf-charts'
import type { ReportData } from './report'

const PERIOD_LABEL: Record<ReportData['period'], string> = {
  weekly: 'Weekly',
  monthly: 'Monthly',
  annual: 'Annual',
}

export type PdfFeatures = {
  income?: boolean
  expenses?: boolean
  investments?: boolean
}

// Standard PDF fonts can't render the ₹ glyph, so use the ASCII currency code.
function money(n: number, currency = 'INR'): string {
  try {
    return new Intl.NumberFormat('en-IN', {
      style: 'currency',
      currency,
      currencyDisplay: 'code',
      maximumFractionDigits: 2,
    }).format(n)
  } catch {
    return `${currency} ${n.toFixed(2)}`
  }
}

function finalY(doc: jsPDF): number {
  return (doc as unknown as { lastAutoTable: { finalY: number } }).lastAutoTable.finalY
}

function ensureSpace(doc: jsPDF, y: number, needed: number, margin: number): number {
  const pageH = doc.internal.pageSize.getHeight()
  if (y + needed > pageH - margin) {
    doc.addPage()
    return margin
  }
  return y
}

function sectionTitle(doc: jsPDF, y: number, title: string, margin: number): number {
  doc.setFont('helvetica', 'bold')
  doc.setFontSize(12)
  doc.setTextColor(20, 20, 20)
  doc.text(title, margin, y)
  return y + 14
}

function addChartImage(
  doc: jsPDF,
  dataUrl: string,
  x: number,
  y: number,
  w: number,
  h: number,
): void {
  doc.addImage(dataUrl, 'PNG', x, y, w, h)
}

export async function generateReportPdf(
  data: ReportData,
  opts: { currency?: string; features?: PdfFeatures } = {},
): Promise<void> {
  const currency = opts.currency ?? 'INR'
  const feats: PdfFeatures = {
    income: true,
    expenses: true,
    investments: true,
    ...opts.features,
  }

  const doc = new jsPDF({ unit: 'pt', format: 'a4' })
  const pageW = doc.internal.pageSize.getWidth()
  const pageH = doc.internal.pageSize.getHeight()
  const margin = 40
  const contentW = pageW - margin * 2
  let y = margin

  // Header band
  doc.setFillColor(28, 28, 26)
  doc.rect(0, 0, pageW, 72, 'F')
  doc.setFont('helvetica', 'bold')
  doc.setFontSize(18)
  doc.setTextColor(255, 255, 255)
  doc.text(`${PERIOD_LABEL[data.period]} Report`, margin, 32)
  doc.setFont('helvetica', 'normal')
  doc.setFontSize(11)
  doc.setTextColor(200, 200, 198)
  doc.text(data.range.label, margin, 50)
  doc.setFontSize(9)
  doc.text(`Generated ${new Date().toLocaleString('en-IN')}`, pageW - margin, 32, { align: 'right' })
  doc.text('Money Tracker', pageW - margin, 50, { align: 'right' })
  y = 88

  // Summary rows (only relevant features)
  const summaryRows: string[][] = []
  if (feats.income) summaryRows.push(['Total income', money(data.income, currency)])
  if (feats.expenses) summaryRows.push(['Total expenses', money(data.expense, currency)])
  if (feats.income && feats.expenses) {
    summaryRows.push(['Net savings', money(data.net, currency)])
    summaryRows.push(['Savings rate', `${data.savingsRate.toFixed(1)}%`])
  }
  if (feats.investments && (data.investedInPeriod > 0 || data.investedCount > 0)) {
    summaryRows.push([
      'Invested this period',
      `${money(data.investedInPeriod, currency)} (${data.investedCount})`,
    ])
  }
  if (feats.investments && data.portfolioValue > 0) {
    summaryRows.push(['Portfolio value', money(data.portfolioValue, currency)])
    summaryRows.push(['Portfolio P/L', money(data.portfolioPnl, currency)])
  }

  if (summaryRows.length > 0) {
    autoTable(doc, {
      startY: y,
      margin: { left: margin, right: margin },
      head: [['Summary', 'Amount']],
      body: summaryRows,
      theme: 'striped',
      headStyles: { fillColor: [28, 28, 26], fontSize: 10, fontStyle: 'bold' },
      bodyStyles: { fontSize: 10, textColor: [50, 50, 50] },
      columnStyles: { 1: { halign: 'right', fontStyle: 'bold' } },
      alternateRowStyles: { fillColor: [248, 248, 246] },
    })
    y = finalY(doc) + 20
  }

  // Charts section
  const showBar = feats.income || feats.expenses
  const showPie = feats.income && data.income > 0
  const barPng = showBar
    ? renderTrendBarChart(data.trend, {
        width: Math.round(contentW * 2),
        height: 440,
        showIncome: !!feats.income,
        showExpense: !!feats.expenses,
      })
    : null
  const piePng = showPie
    ? renderAllocationPie(
        data.income,
        data.byCategory.map((c) => ({ name: c.name, value: c.value, color: c.color })),
        { width: Math.round(contentW * 2), height: 520 },
      )
    : null

  if (barPng || piePng) {
    y = ensureSpace(doc, y, 40, margin)
    y = sectionTitle(doc, y, 'Visualizations', margin)

    if (barPng && piePng) {
      const gap = 16
      const halfW = (contentW - gap) / 2
      const barH = 200
      const pieH = 240
      const rowH = Math.max(barH, pieH) + 8
      y = ensureSpace(doc, y, rowH, margin)

      doc.setFont('helvetica', 'bold')
      doc.setFontSize(10)
      doc.setTextColor(80, 80, 78)
      doc.text('Income vs Expense (trend)', margin, y)
      doc.text('Income allocation', margin + halfW + gap, y)
      y += 8

      addChartImage(doc, barPng, margin, y, halfW, barH)
      addChartImage(doc, piePng, margin + halfW + gap, y, halfW, pieH)
      y += rowH + 12
    } else if (barPng) {
      const barH = 210
      y = ensureSpace(doc, y, barH + 20, margin)
      doc.setFont('helvetica', 'bold')
      doc.setFontSize(10)
      doc.setTextColor(80, 80, 78)
      doc.text('Income vs Expense (trend)', margin, y)
      y += 8
      addChartImage(doc, barPng, margin, y, contentW, barH)
      y += barH + 16
    } else if (piePng) {
      const pieH = 260
      y = ensureSpace(doc, y, pieH + 20, margin)
      doc.setFont('helvetica', 'bold')
      doc.setFontSize(10)
      doc.setTextColor(80, 80, 78)
      doc.text('Income allocation', margin, y)
      y += 8
      addChartImage(doc, piePng, margin, y, contentW * 0.65, pieH)
      y += pieH + 16
    }
  }

  // Spending by category
  if (feats.expenses && data.byCategory.length > 0) {
    y = ensureSpace(doc, y, 60, margin)
    y = sectionTitle(doc, y, 'Spending by category', margin)
    autoTable(doc, {
      startY: y,
      margin: { left: margin, right: margin },
      head: [['Category', 'Amount', '% of spend']],
      body: data.byCategory.map((c) => [
        c.name,
        money(c.value, currency),
        data.expense > 0 ? `${((c.value / data.expense) * 100).toFixed(1)}%` : '—',
      ]),
      theme: 'striped',
      headStyles: { fillColor: [28, 28, 26], fontSize: 10 },
      bodyStyles: { fontSize: 9 },
      columnStyles: { 1: { halign: 'right' }, 2: { halign: 'right' } },
      alternateRowStyles: { fillColor: [248, 248, 246] },
    })
    y = finalY(doc) + 16
  }

  // Top expenses
  if (feats.expenses && data.topExpenses.length > 0) {
    y = ensureSpace(doc, y, 60, margin)
    y = sectionTitle(doc, y, 'Top expenses', margin)
    autoTable(doc, {
      startY: y,
      margin: { left: margin, right: margin },
      head: [['Date', 'Category', 'Note', 'Amount']],
      body: data.topExpenses.map((e) => [
        e.date,
        e.category,
        e.note || '—',
        money(e.amount, currency),
      ]),
      theme: 'striped',
      headStyles: { fillColor: [28, 28, 26], fontSize: 10 },
      bodyStyles: { fontSize: 9 },
      columnStyles: { 3: { halign: 'right' } },
      alternateRowStyles: { fillColor: [248, 248, 246] },
    })
    y = finalY(doc) + 16
  }

  // Income sources
  if (feats.income && data.sources.length > 0) {
    y = ensureSpace(doc, y, 60, margin)
    y = sectionTitle(doc, y, 'Income sources', margin)
    autoTable(doc, {
      startY: y,
      margin: { left: margin, right: margin },
      head: [['Source', 'Amount']],
      body: data.sources.map((s) => [s.source, money(s.value, currency)]),
      theme: 'striped',
      headStyles: { fillColor: [28, 28, 26], fontSize: 10 },
      bodyStyles: { fontSize: 9 },
      columnStyles: { 1: { halign: 'right' } },
      alternateRowStyles: { fillColor: [248, 248, 246] },
    })
  }

  // Page numbers
  const pages = doc.getNumberOfPages()
  for (let p = 1; p <= pages; p++) {
    doc.setPage(p)
    doc.setFont('helvetica', 'normal')
    doc.setFontSize(8)
    doc.setTextColor(160, 160, 158)
    doc.text(`Page ${p} of ${pages}`, pageW / 2, pageH - 20, { align: 'center' })
  }

  const stamp = data.range.label.replace(/[^\w]+/g, '-').toLowerCase()
  doc.save(`report-${data.period}-${stamp}.pdf`)
}
