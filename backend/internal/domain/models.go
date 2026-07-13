package domain

import "time"

// Base fields shared by every entity.
type Base struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Income recorded once per month, possibly from multiple sources.
type Income struct {
	Base
	Source     string  `json:"source"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	Month      int     `json:"month"` // 1-12
	Year       int     `json:"year"`
	ReceivedOn string  `json:"receivedOn"` // ISO date (YYYY-MM-DD)
	Note       string  `json:"note,omitempty"`
}

// Expense recorded daily against a category.
type Expense struct {
	Base
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	CategoryID    string  `json:"categoryId"`
	Subcategory   string  `json:"subcategory,omitempty"`
	Date          string  `json:"date"` // ISO date
	PaymentMethod string  `json:"paymentMethod"`
	Note          string  `json:"note,omitempty"`
	RecurringID   string  `json:"recurringId,omitempty"` // set when auto-created by a recurring rule
}

// Investment with optional auto price-fetch identity and a price cache.
type Investment struct {
	Base
	Name           string   `json:"name"`
	Type           string   `json:"type"` // stocks, mutual_fund, fd, rd, gold, crypto, bonds, real_estate, other
	Platform       string   `json:"platform,omitempty"`
	Symbol         string   `json:"symbol,omitempty"`
	Provider       string   `json:"provider"` // coingecko, mfapi, stock (NSE), bse, manual
	Quantity       *float64 `json:"quantity,omitempty"`
	AmountInvested float64  `json:"amountInvested"`
	CurrentValue   *float64 `json:"currentValue,omitempty"` // manual fallback when no auto price
	Currency       string   `json:"currency"`
	InvestedOn     string   `json:"investedOn"` // ISO date
	Note           string   `json:"note,omitempty"`

	// Price cache (refreshed from a quotes provider; not business source of truth).
	LastPrice   *float64   `json:"lastPrice,omitempty"`
	LastPriceAt *time.Time `json:"lastPriceAt,omitempty"`
}

// Recurring is a schedule that materializes entries automatically: a
// recurring expense (subscription, EMI) or a SIP contribution into an
// existing investment. NextRunOn is the materializer's cursor.
type Recurring struct {
	Base
	Kind      string  `json:"kind"` // expense | sip
	Name      string  `json:"name"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Cadence   string  `json:"cadence"`   // monthly | weekly | yearly
	StartDate string  `json:"startDate"` // ISO date; anchors day-of-month / weekday
	EndDate   string  `json:"endDate,omitempty"`
	Paused    bool    `json:"paused"`
	Note      string  `json:"note,omitempty"`

	// kind == expense
	CategoryID    string `json:"categoryId,omitempty"`
	Subcategory   string `json:"subcategory,omitempty"`
	PaymentMethod string `json:"paymentMethod,omitempty"`

	// kind == sip
	InvestmentID string `json:"investmentId,omitempty"`

	NextRunOn string `json:"nextRunOn,omitempty"` // next due occurrence (managed by the server)
}

// Loan given to someone; repaid in EMIs, lump sums, or any order.
// Outstanding balance is computed (principal − Σ repayments), never stored.
type Loan struct {
	Base
	Borrower   string      `json:"borrower"`
	Principal  float64     `json:"principal"`
	Currency   string      `json:"currency"`
	LentOn     string      `json:"lentOn"` // ISO date
	DueOn      string      `json:"dueOn,omitempty"`
	Note       string      `json:"note,omitempty"`
	Repayments []Repayment `json:"repayments"`
}

// Repayment is one return payment against a loan.
type Repayment struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Date   string  `json:"date"` // ISO date
	Note   string  `json:"note,omitempty"`
}

// Category for expenses; config-driven and user-editable.
type Category struct {
	Base
	Name          string   `json:"name"`
	Color         string   `json:"color"`
	Icon          string   `json:"icon,omitempty"`
	Subcategories []string `json:"subcategories"`
	Archived      bool     `json:"archived"`
}
