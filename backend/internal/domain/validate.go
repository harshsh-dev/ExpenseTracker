package domain

import "errors"

func (in *Income) Validate() error {
	if in.Source == "" {
		return errors.New("source is required")
	}
	if in.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	if in.Month < 1 || in.Month > 12 {
		return errors.New("month must be 1-12")
	}
	if in.Year < 1970 {
		return errors.New("year is invalid")
	}
	if in.Currency == "" {
		in.Currency = "INR"
	}
	return nil
}

func (in *Expense) Validate() error {
	if in.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	if in.CategoryID == "" {
		return errors.New("categoryId is required")
	}
	if in.Date == "" {
		return errors.New("date is required")
	}
	if in.PaymentMethod == "" {
		in.PaymentMethod = "other"
	}
	if in.Currency == "" {
		in.Currency = "INR"
	}
	return nil
}

func (in *Investment) Validate() error {
	if in.Name == "" {
		return errors.New("name is required")
	}
	if in.Type == "" {
		return errors.New("type is required")
	}
	if in.AmountInvested <= 0 {
		return errors.New("amountInvested must be positive")
	}
	if in.InvestedOn == "" {
		return errors.New("investedOn is required")
	}
	if in.Provider == "" {
		in.Provider = "manual"
	}
	if in.Currency == "" {
		in.Currency = "INR"
	}
	return nil
}

func (in *Recurring) Validate() error {
	if in.Kind != RecurringExpense && in.Kind != RecurringSIP {
		return errors.New("kind must be expense or sip")
	}
	if in.Name == "" {
		return errors.New("name is required")
	}
	if in.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	switch in.Cadence {
	case CadenceMonthly, CadenceWeekly, CadenceYearly:
	case "":
		in.Cadence = CadenceMonthly
	default:
		return errors.New("cadence must be monthly, weekly, or yearly")
	}
	if in.StartDate == "" {
		return errors.New("startDate is required")
	}
	if in.Kind == RecurringExpense && in.CategoryID == "" {
		return errors.New("categoryId is required for a recurring expense")
	}
	if in.Kind == RecurringSIP && in.InvestmentID == "" {
		return errors.New("investmentId is required for a SIP")
	}
	if in.PaymentMethod == "" {
		in.PaymentMethod = "other"
	}
	if in.Currency == "" {
		in.Currency = "INR"
	}
	return nil
}

func (in *Loan) Validate() error {
	if in.Borrower == "" {
		return errors.New("borrower is required")
	}
	if in.Principal <= 0 {
		return errors.New("principal must be positive")
	}
	if in.LentOn == "" {
		return errors.New("lentOn is required")
	}
	for _, r := range in.Repayments {
		if r.Amount <= 0 {
			return errors.New("repayment amounts must be positive")
		}
		if r.Date == "" {
			return errors.New("repayment dates are required")
		}
	}
	if in.Currency == "" {
		in.Currency = "INR"
	}
	return nil
}

func (in *Category) Validate() error {
	if in.Name == "" {
		return errors.New("name is required")
	}
	if in.Color == "" {
		in.Color = "#64748b"
	}
	if in.Subcategories == nil {
		in.Subcategories = []string{}
	}
	return nil
}
