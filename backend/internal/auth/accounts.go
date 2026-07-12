package auth

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"moneytracker/backend/internal/storage"
)

// Account is a Notion user who has logged in, together with the OAuth access
// token the Notion sync uses and the sync bookkeeping (created page/database
// ids). Stored in auth.json — intentionally outside the backup snapshot so
// exported backups never contain secrets.
type Account struct {
	UserID        string    `json:"userId"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	AvatarURL     string    `json:"avatarUrl,omitempty"`
	AccessToken   string    `json:"accessToken"`
	BotID         string    `json:"botId"`
	WorkspaceID   string    `json:"workspaceId"`
	WorkspaceName string    `json:"workspaceName"`
	ConnectedAt   time.Time `json:"connectedAt"`
	Sync          SyncState `json:"sync"`
}

// SyncState remembers where in the user's workspace the data is mirrored.
type SyncState struct {
	PageID          string     `json:"pageId,omitempty"`
	PageURL         string     `json:"pageUrl,omitempty"`
	ExpensesDBID    string     `json:"expensesDbId,omitempty"`
	IncomesDBID     string     `json:"incomesDbId,omitempty"`
	InvestmentsDBID string     `json:"investmentsDbId,omitempty"`
	LastSyncedAt    *time.Time `json:"lastSyncedAt,omitempty"`
}

// Accounts is the persisted account set (its own blob, like the store
// snapshot but separate and non-exportable).
type Accounts struct {
	mu   sync.RWMutex
	blob storage.Blob
	byID map[string]Account
}

func loadAccounts(blob storage.Blob) (*Accounts, error) {
	a := &Accounts{blob: blob, byID: map[string]Account{}}
	b, err := blob.Load()
	if err != nil {
		return nil, err
	}
	if b == nil {
		return a, nil
	}
	var list []Account
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, err
	}
	for _, acc := range list {
		a.byID[acc.UserID] = acc
	}
	return a, nil
}

func (a *Accounts) Get(userID string) (Account, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	acc, ok := a.byID[userID]
	return acc, ok
}

func (a *Accounts) Put(acc Account) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	// A re-login replaces the token but keeps the existing sync targets.
	if prev, ok := a.byID[acc.UserID]; ok && acc.Sync == (SyncState{}) {
		acc.Sync = prev.Sync
	}
	a.byID[acc.UserID] = acc
	return a.persist()
}

// UpdateSync mutates one account's sync state and persists.
func (a *Accounts) UpdateSync(userID string, fn func(*SyncState)) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	acc, ok := a.byID[userID]
	if !ok {
		return errors.New("account not found")
	}
	fn(&acc.Sync)
	a.byID[userID] = acc
	return a.persist()
}

// persist writes through the blob backend (atomic; contains tokens).
func (a *Accounts) persist() error {
	list := make([]Account, 0, len(a.byID))
	for _, acc := range a.byID {
		list = append(list, acc)
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return a.blob.Save(b)
}
