export function formatMoney(amount: number, currency = 'INR'): string {
  try {
    return new Intl.NumberFormat('en-IN', {
      style: 'currency',
      currency,
      maximumFractionDigits: 2,
    }).format(amount)
  } catch {
    return `${currency} ${amount.toFixed(2)}`
  }
}

export function formatDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' })
}

export function monthName(m: number): string {
  return (
    [
      'Jan',
      'Feb',
      'Mar',
      'Apr',
      'May',
      'Jun',
      'Jul',
      'Aug',
      'Sep',
      'Oct',
      'Nov',
      'Dec',
    ][m - 1] ?? String(m)
  )
}

export function todayISO(): string {
  return new Date().toISOString().slice(0, 10)
}

// isoDateParts parses YYYY-MM-DD without timezone shifts (unlike new Date(iso)).
export function isoDateParts(iso: string): { year: number; month: number; day: number } | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso)
  if (!m) return null
  return { year: Number(m[1]), month: Number(m[2]), day: Number(m[3]) }
}

// addDaysISO shifts an ISO date by delta days (UTC arithmetic, so no
// timezone-driven off-by-one).
export function addDaysISO(iso: string, delta: number): string {
  const p = isoDateParts(iso)
  if (!p) return iso
  return new Date(Date.UTC(p.year, p.month - 1, p.day + delta)).toISOString().slice(0, 10)
}

// weekdayIndex returns 0 (Sunday) .. 6 (Saturday) for an ISO date.
export function weekdayIndex(iso: string): number {
  const p = isoDateParts(iso)
  if (!p) return 0
  return new Date(Date.UTC(p.year, p.month - 1, p.day)).getUTCDay()
}

export function formatDayMonth(iso: string): string {
  const p = isoDateParts(iso)
  if (!p) return iso
  return `${p.day} ${monthName(p.month)}`
}
