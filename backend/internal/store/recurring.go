package store

import (
	"time"

	"moneytracker/backend/internal/domain"
)

// ---------- Recurring ----------

func (s *Store) ListRecurring() []domain.Recurring {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedRecurring(s.recurring)
}

func (s *Store) CreateRecurring(in domain.Recurring) (domain.Recurring, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	in.Base = newBase()
	if in.NextRunOn == "" {
		in.NextRunOn = in.StartDate
	}
	s.recurring[in.ID] = in
	return in, s.persist()
}

func (s *Store) UpdateRecurring(id string, in domain.Recurring) (domain.Recurring, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.recurring[id]
	if !ok {
		return domain.Recurring{}, ErrNotFound
	}
	in.ID = id
	in.CreatedAt = existing.CreatedAt
	in.UpdatedAt = time.Now().UTC()
	if in.NextRunOn == "" {
		in.NextRunOn = in.StartDate
	}
	s.recurring[id] = in
	return in, s.persist()
}

func (s *Store) DeleteRecurring(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.recurring[id]; !ok {
		return ErrNotFound
	}
	delete(s.recurring, id)
	return s.persist()
}

// ---------- Loan ----------

func (s *Store) ListLoans() []domain.Loan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedLoans(s.loans)
}

func (s *Store) CreateLoan(in domain.Loan) (domain.Loan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	in.Base = newBase()
	fillRepaymentIDs(&in)
	s.loans[in.ID] = in
	return in, s.persist()
}

func (s *Store) UpdateLoan(id string, in domain.Loan) (domain.Loan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.loans[id]
	if !ok {
		return domain.Loan{}, ErrNotFound
	}
	in.ID = id
	in.CreatedAt = existing.CreatedAt
	in.UpdatedAt = time.Now().UTC()
	fillRepaymentIDs(&in)
	s.loans[id] = in
	return in, s.persist()
}

func (s *Store) DeleteLoan(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.loans[id]; !ok {
		return ErrNotFound
	}
	delete(s.loans, id)
	return s.persist()
}

func fillRepaymentIDs(l *domain.Loan) {
	if l.Repayments == nil {
		l.Repayments = []domain.Repayment{}
	}
	for i := range l.Repayments {
		if l.Repayments[i].ID == "" {
			l.Repayments[i].ID = newID()
		}
	}
}

// ---------- materializer ----------

// maxOccurrencesPerRun bounds backfill per rule per run (a monthly rule
// backdated 10 years still terminates promptly; the rest catches up on the
// next run an hour later).
const maxOccurrencesPerRun = 120

// MaterializeRecurring generates every entry due up to now: expenses for
// recurring-expense rules, invested-amount (and units, when price-tracked)
// bumps for SIPs. Called at boot and on an interval; on hosts that sleep
// (Render free) the boot run backfills whatever came due while asleep.
// A SIP whose investment was deleted pauses itself instead of failing forever.
func (s *Store) MaterializeRecurring(now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	today := now.UTC().Format("2006-01-02")
	created := 0
	changed := false

	for id, r := range s.recurring {
		if r.Paused || r.NextRunOn == "" {
			continue
		}
		next := r.NextRunOn
		runs := 0
		for next != "" && next <= today && (r.EndDate == "" || next <= r.EndDate) && runs < maxOccurrencesPerRun {
			if !s.materializeOneLocked(&r, next, now) {
				break // rule paused itself (missing SIP investment)
			}
			created++
			runs++
			next = domain.NextOccurrence(r.Cadence, r.StartDate, next)
		}
		if next != r.NextRunOn || r.Paused {
			r.NextRunOn = next
			r.UpdatedAt = now.UTC()
			s.recurring[id] = r
			changed = true
		}
	}

	if !changed && created == 0 {
		return 0, nil
	}
	return created, s.persist()
}

// materializeOneLocked applies a single occurrence. Returns false when the
// rule can no longer run (it flips itself to paused).
func (s *Store) materializeOneLocked(r *domain.Recurring, date string, now time.Time) bool {
	switch r.Kind {
	case domain.RecurringSIP:
		inv, ok := s.investments[r.InvestmentID]
		if !ok {
			r.Paused = true
			return false
		}
		inv.AmountInvested += r.Amount
		if inv.Quantity != nil && inv.LastPrice != nil && *inv.LastPrice > 0 {
			q := *inv.Quantity + r.Amount / *inv.LastPrice
			inv.Quantity = &q
		}
		inv.UpdatedAt = now.UTC()
		s.investments[inv.ID] = inv
	default: // expense
		e := domain.Expense{
			Base:          newBase(),
			Amount:        r.Amount,
			Currency:      r.Currency,
			CategoryID:    r.CategoryID,
			Subcategory:   r.Subcategory,
			Date:          date,
			PaymentMethod: r.PaymentMethod,
			Note:          r.Name,
			RecurringID:   r.ID,
		}
		s.expenses[e.ID] = e
	}
	return true
}
