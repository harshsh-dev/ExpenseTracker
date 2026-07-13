export interface Base {
  id: string
  createdAt: string
  updatedAt: string
}

export interface Income extends Base {
  source: string
  amount: number
  currency: string
  month: number
  year: number
  receivedOn: string
  note?: string
}

export interface Expense extends Base {
  amount: number
  currency: string
  categoryId: string
  subcategory?: string
  date: string
  paymentMethod: PaymentMethod
  note?: string
  recurringId?: string
}

export type PaymentMethod = 'cash' | 'card' | 'upi' | 'netbanking' | 'wallet' | 'other'

export type InvestmentType =
  | 'stocks'
  | 'mutual_fund'
  | 'fd'
  | 'rd'
  | 'gold'
  | 'crypto'
  | 'bonds'
  | 'real_estate'
  | 'other'

export interface Investment extends Base {
  name: string
  type: InvestmentType
  platform?: string
  symbol?: string
  provider: 'coingecko' | 'mfapi' | 'stock' | 'bse' | 'manual'
  quantity?: number
  amountInvested: number
  currentValue?: number
  currency: string
  investedOn: string
  note?: string
  lastPrice?: number
  lastPriceAt?: string
}

export interface Category extends Base {
  name: string
  color: string
  icon?: string
  subcategories: string[]
  archived: boolean
}

export type RecurringKind = 'expense' | 'sip'
export type Cadence = 'monthly' | 'weekly' | 'yearly'

export interface Recurring extends Base {
  kind: RecurringKind
  name: string
  amount: number
  currency: string
  cadence: Cadence
  startDate: string
  endDate?: string
  paused: boolean
  note?: string
  // kind === 'expense'
  categoryId?: string
  subcategory?: string
  paymentMethod?: PaymentMethod
  // kind === 'sip'
  investmentId?: string
  nextRunOn?: string
}

export interface Repayment {
  id: string
  amount: number
  date: string
  note?: string
}

export interface Loan extends Base {
  borrower: string
  principal: number
  currency: string
  lentOn: string
  dueOn?: string
  note?: string
  repayments: Repayment[]
}

export interface Snapshot {
  schemaVersion: number
  exportedAt: string
  app: string
  data: {
    incomes: Income[]
    expenses: Expense[]
    investments: Investment[]
    categories: Category[]
    recurring?: Recurring[]
    loans?: Loan[]
  }
}

// Derived (computed in the UI, never stored)
export function investmentCurrentValue(inv: Investment): number | undefined {
  if (inv.quantity != null && inv.lastPrice != null) return inv.quantity * inv.lastPrice
  return inv.currentValue
}

export function loanRepaid(loan: Loan): number {
  return loan.repayments.reduce((s, r) => s + r.amount, 0)
}

export function loanOutstanding(loan: Loan): number {
  return loan.principal - loanRepaid(loan)
}

export function investmentPnl(inv: Investment): { value?: number; pct?: number } {
  const cur = investmentCurrentValue(inv)
  if (cur == null) return {}
  const value = cur - inv.amountInvested
  return { value, pct: (value / inv.amountInvested) * 100 }
}
