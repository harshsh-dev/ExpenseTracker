package store

import (
	"time"

	"moneytracker/backend/internal/domain"
)

// ---------- Income ----------

func (s *Store) ListIncomes() []domain.Income {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedIncomes(s.incomes)
}

func (s *Store) CreateIncome(in domain.Income) (domain.Income, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	in.Base = newBase()
	s.incomes[in.ID] = in
	return in, s.persist()
}

func (s *Store) UpdateIncome(id string, in domain.Income) (domain.Income, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.incomes[id]
	if !ok {
		return domain.Income{}, ErrNotFound
	}
	in.ID = id
	in.CreatedAt = existing.CreatedAt
	in.UpdatedAt = time.Now().UTC()
	s.incomes[id] = in
	return in, s.persist()
}

func (s *Store) DeleteIncome(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.incomes[id]; !ok {
		return ErrNotFound
	}
	delete(s.incomes, id)
	return s.persist()
}

// ---------- Expense ----------

func (s *Store) ListExpenses() []domain.Expense {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedExpenses(s.expenses)
}

func (s *Store) CreateExpense(in domain.Expense) (domain.Expense, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	in.Base = newBase()
	s.expenses[in.ID] = in
	return in, s.persist()
}

func (s *Store) UpdateExpense(id string, in domain.Expense) (domain.Expense, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.expenses[id]
	if !ok {
		return domain.Expense{}, ErrNotFound
	}
	in.ID = id
	in.CreatedAt = existing.CreatedAt
	in.UpdatedAt = time.Now().UTC()
	s.expenses[id] = in
	return in, s.persist()
}

func (s *Store) DeleteExpense(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.expenses[id]; !ok {
		return ErrNotFound
	}
	delete(s.expenses, id)
	return s.persist()
}

// ---------- Investment ----------

func (s *Store) ListInvestments() []domain.Investment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedInvestments(s.investments)
}

func (s *Store) CreateInvestment(in domain.Investment) (domain.Investment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	in.Base = newBase()
	s.investments[in.ID] = in
	return in, s.persist()
}

func (s *Store) UpdateInvestment(id string, in domain.Investment) (domain.Investment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.investments[id]
	if !ok {
		return domain.Investment{}, ErrNotFound
	}
	in.ID = id
	in.CreatedAt = existing.CreatedAt
	in.UpdatedAt = time.Now().UTC()
	s.investments[id] = in
	return in, s.persist()
}

func (s *Store) DeleteInvestment(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.investments[id]; !ok {
		return ErrNotFound
	}
	delete(s.investments, id)
	return s.persist()
}

// SetInvestmentPrice updates the cached price for an investment (used by the
// quotes refresh). It does not touch user-edited fields.
func (s *Store) SetInvestmentPrice(id string, price float64, at time.Time) (domain.Investment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.investments[id]
	if !ok {
		return domain.Investment{}, ErrNotFound
	}
	inv.LastPrice = &price
	inv.LastPriceAt = &at
	inv.UpdatedAt = time.Now().UTC()
	s.investments[id] = inv
	return inv, s.persist()
}

// ---------- Category ----------

func (s *Store) ListCategories() []domain.Category {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedCategories(s.categories)
}

func (s *Store) CreateCategory(in domain.Category) (domain.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	in.Base = newBase()
	if in.Subcategories == nil {
		in.Subcategories = []string{}
	}
	s.categories[in.ID] = in
	return in, s.persist()
}

func (s *Store) UpdateCategory(id string, in domain.Category) (domain.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.categories[id]
	if !ok {
		return domain.Category{}, ErrNotFound
	}
	in.ID = id
	in.CreatedAt = existing.CreatedAt
	in.UpdatedAt = time.Now().UTC()
	if in.Subcategories == nil {
		in.Subcategories = []string{}
	}
	s.categories[id] = in
	return in, s.persist()
}

func (s *Store) DeleteCategory(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.categories[id]; !ok {
		return ErrNotFound
	}
	delete(s.categories, id)
	return s.persist()
}
