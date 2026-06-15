package store

import (
	"time"

	"moneytracker/backend/internal/domain"
)

// SchemaVersion is bumped when the snapshot shape changes (add a migration too).
const SchemaVersion = 1

// AppName identifies snapshots produced by this app on import.
const AppName = "money-tracker"

// Snapshot is the full, versioned data export/import + on-disk format.
type Snapshot struct {
	SchemaVersion int          `json:"schemaVersion"`
	ExportedAt    time.Time    `json:"exportedAt"`
	App           string       `json:"app"`
	Data          SnapshotData `json:"data"`
}

type SnapshotData struct {
	Incomes     []domain.Income     `json:"incomes"`
	Expenses    []domain.Expense    `json:"expenses"`
	Investments []domain.Investment `json:"investments"`
	Categories  []domain.Category   `json:"categories"`
}
