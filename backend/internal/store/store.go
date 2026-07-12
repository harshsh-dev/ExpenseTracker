package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"

	"moneytracker/backend/internal/domain"
	"moneytracker/backend/internal/storage"
)

// ErrNotFound is returned when an entity id does not exist.
var ErrNotFound = errors.New("not found")

// Store is the in-memory database. Every mutation is persisted to a JSON
// snapshot blob (a local file by default, Firestore on diskless hosts) so a
// restart/redeploy can rehydrate — true to the in-memory-first design, with
// the snapshot as the durability + backup layer.
type Store struct {
	mu   sync.RWMutex
	blob storage.Blob

	incomes     map[string]domain.Income
	expenses    map[string]domain.Expense
	investments map[string]domain.Investment
	categories  map[string]domain.Category
}

// New creates a store backed by the given snapshot blob. If the blob exists it
// is loaded; otherwise an empty store is seeded with the default categories.
func New(blob storage.Blob) (*Store, error) {
	s := &Store{
		blob:        blob,
		incomes:     map[string]domain.Income{},
		expenses:    map[string]domain.Expense{},
		investments: map[string]domain.Investment{},
		categories:  map[string]domain.Category{},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	if len(s.categories) == 0 {
		for _, c := range domain.DefaultCategories() {
			c.Base = newBase()
			s.categories[c.ID] = c
		}
		if err := s.persist(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// ---- persistence ----

func (s *Store) load() error {
	b, err := s.blob.Load()
	if err != nil {
		return err
	}
	if b == nil {
		return nil
	}
	var snap Snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return err
	}
	s.replaceLocked(snap.Data)
	return nil
}

// persist writes the snapshot atomically (the blob backend guarantees it).
func (s *Store) persist() error {
	snap := s.snapshotLocked()
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return s.blob.Save(b)
}

func (s *Store) snapshotLocked() Snapshot {
	return Snapshot{
		SchemaVersion: SchemaVersion,
		ExportedAt:    time.Now().UTC(),
		App:           AppName,
		Data: SnapshotData{
			Incomes:     sortedIncomes(s.incomes),
			Expenses:    sortedExpenses(s.expenses),
			Investments: sortedInvestments(s.investments),
			Categories:  sortedCategories(s.categories),
		},
	}
}

func (s *Store) replaceLocked(d SnapshotData) {
	s.incomes = map[string]domain.Income{}
	s.expenses = map[string]domain.Expense{}
	s.investments = map[string]domain.Investment{}
	s.categories = map[string]domain.Category{}
	for _, x := range d.Incomes {
		s.incomes[x.ID] = x
	}
	for _, x := range d.Expenses {
		s.expenses[x.ID] = x
	}
	for _, x := range d.Investments {
		s.investments[x.ID] = x
	}
	for _, x := range d.Categories {
		s.categories[x.ID] = x
	}
}

// Export returns the full snapshot (for the export endpoint).
func (s *Store) Export() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshotLocked()
}

// Import replaces all data with the snapshot's data and persists it.
func (s *Store) Import(snap Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replaceLocked(snap.Data)
	return s.persist()
}

// ---- helpers ----

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func newBase() domain.Base {
	now := time.Now().UTC()
	return domain.Base{ID: newID(), CreatedAt: now, UpdatedAt: now}
}

func sortedIncomes(m map[string]domain.Income) []domain.Income {
	out := make([]domain.Income, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func sortedExpenses(m map[string]domain.Expense) []domain.Expense {
	out := make([]domain.Expense, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	return out
}

func sortedInvestments(m map[string]domain.Investment) []domain.Investment {
	out := make([]domain.Investment, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].InvestedOn > out[j].InvestedOn })
	return out
}

func sortedCategories(m map[string]domain.Category) []domain.Category {
	out := make([]domain.Category, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
