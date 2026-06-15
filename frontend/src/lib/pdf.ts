import { jsPDF } from 'jspdf'
import autoTable from 'jspdf-autotable'
import type { ReportData } from './report'

const PERIOD_LABEL: Record<ReportData['period'], string> = {
  weekly: 'Weekly',
  monthly: 'Monthly',
  annual: 'Annual',
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

// svgToPng rasterizes a (Recharts) SVG node to a PNG data URL on a white canvas.
async function svgToPng(
  svg: SVGElement,
  scale = 2,
): Promise<{ dataUrl: string; width: number; height: number }> {
  const rect = svg.getBoundingClientRect()
  const width = Math.max(1, rect.width || 600)
  const height = Math.max(1, rect.height || 280)
  const clone = svg.cloneNode(true) as SVGElement
  clone.setAttribute('width', String(width))
  clone.setAttribute('height', String(height))
  clone.setAttribute('xmlns', 'http://www.w3.org/2000/svg')
  const xml = new XMLSerializer().serializeToString(clone)
  const src = 'data:image/svg+xml;base64,' + btoa(unescape(encodeURIComponent(xml)))

  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => {
      const canvas = document.createElement('canvas')
      canvas.width = width * scale
      canvas.height = height * scale
      const ctx = canvas.getContext('2d')
      if (!ctx) return reject(new Error('no canvas context'))
      ctx.fillStyle = '#ffffff'
      ctx.fillRect(0, 0, canvas.width, canvas.height)
      ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
      resolve({ dataUrl: canvas.toDataURL('image/png'), width, height })
    }
    img.onerror = () => reject(new Error('svg render failed'))
    img.src = src
  })
}

export async function generateReportPdf(
  data: ReportData,
  opts: { chartSvg?: SVGElement | null; currency?: string } = {},
): Promise<void> {
  const currency = opts.currency ?? 'INR'
  const doc = new jsPDF({ unit: 'pt', format: 'a4' })
  const pageW = doc.internal.pageSize.getWidth()
  const pageH = doc.internal.pageSize.getHeight()
  const margin = 40
  const contentW = pageW - margin * 2
  let y = margin

  // Header
  doc.setFont('helvetica', 'bold')
  doc.setFontSize(18)
  doc.setTextColor(20, 20, 20)
  doc.text(`${PERIOD_LABEL[data.period]} Report`, margin, y + 4)
  doc.setFont('helvetica', 'normal')
  doc.setFontSize(11)
  doc.setTextColor(110, 110, 110)
  doc.text(data.range.label, margin, y + 22)
  doc.setFontSize(9)
  doc.text(`Generated ${new Date().toLocaleString('en-IN')}`, pageW - margin, y + 4, {
    align: 'right',
  })
  y += 40

  // Summary
  autoTable(doc, {
    startY: y,
    margin: { left: margin, right: margin },
    head: [['Summary', '']],
    body: [
      ['Total income', money(data.income, currency)],
      ['Total expenses', money(data.expense, currency)],
      ['Net savings', money(data.net, currency)],
      ['Savings rate', `${data.savingsRate.toFixed(1)}%`],
      ['Invested this period', `${money(data.investedInPeriod, currency)} (${data.investedCount})`],
      ['Portfolio value', money(data.portfolioValue, currency)],
      ['Portfolio P/L', money(data.portfolioPnl, currency)],
    ],
    theme: 'plain',
    headStyles: { fontStyle: 'bold', fontSize: 12, textColor: [20, 20, 20] },
    bodyStyles: { fontSize: 10, textColor: [50, 50, 50] },
    columnStyles: { 1: { halign: 'right' } },
  })
  y = finalY(doc) + 18

  // Trend chart (best-effort: skip if rasterization fails)
  if (opts.chartSvg) {
    try {
      const png = await svgToPng(opts.chartSvg)
      const imgW = contentW
      const imgH = (png.height / png.width) * imgW
      if (y + imgH > pageH - margin) {
        doc.addPage()
        y = margin
      }
      doc.setFont('helvetica', 'bold')
      doc.setFontSize(11)
      doc.setTextColor(20, 20, 20)
      doc.text('Income vs Expense', margin, y)
      y += 10
      doc.addImage(png.dataUrl, 'PNG', margin, y, imgW, imgH)
      y += imgH + 18
    } catch {
      /* chart is optional; tables still render */
    }
  }

  // Spending by category
  if (data.byCategory.length > 0) {
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
    })
    y = finalY(doc) + 16
  }

  // Top expenses
  if (data.topExpenses.length > 0) {
    autoTable(doc, {
      startY: y,
      margin: { left: margin, right: margin },
      head: [['Date', 'Category', 'Note', 'Amount']],
      body: data.topExpenses.map((e) => [e.date, e.category, e.note || '—', money(e.amount, currency)]),
      theme: 'striped',
      headStyles: { fillColor: [28, 28, 26], fontSize: 10 },
      bodyStyles: { fontSize: 9 },
      columnStyles: { 3: { halign: 'right' } },
    })
    y = finalY(doc) + 16
  }

  // Income sources
  if (data.sources.length > 0) {
    autoTable(doc, {
      startY: y,
      margin: { left: margin, right: margin },
      head: [['Income source', 'Amount']],
      body: data.sources.map((s) => [s.source, money(s.value, currency)]),
      theme: 'striped',
      headStyles: { fillColor: [28, 28, 26], fontSize: 10 },
      bodyStyles: { fontSize: 9 },
      columnStyles: { 1: { halign: 'right' } },
    })
  }

  const stamp = data.range.label.replace(/[^\w]+/g, '-').toLowerCase()
  doc.save(`report-${data.period}-${stamp}.pdf`)
}
