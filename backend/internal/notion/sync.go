package notion

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"moneytracker/backend/internal/auth"
	"moneytracker/backend/internal/domain"
	"moneytracker/backend/internal/store"
)

const (
	pageTitle        = "Money Tracker"
	expensesTitle    = "Expenses"
	incomesTitle     = "Incomes"
	investmentsTitle = "Investments"
)

// Result summarizes one sync run (exposed at /api/notion/status).
type Result struct {
	StartedAt   time.Time `json:"startedAt"`
	FinishedAt  time.Time `json:"finishedAt"`
	Error       string    `json:"error,omitempty"`
	Created     int       `json:"created"`
	Updated     int       `json:"updated"`
	Archived    int       `json:"archived"`
	Expenses    int       `json:"expenses"`
	Incomes     int       `json:"incomes"`
	Investments int       `json:"investments"`
}

// Syncer runs one-way exports of the store into a Notion workspace. Runs are
// asynchronous (Notion's rate limit makes large syncs slow), so the API
// starts a run and the frontend polls status.
//
// The Notion token comes from the logged-in user's OAuth account, or — when
// NOTION_TOKEN is set — from that fixed internal-integration token, whose
// sync state lives under a synthetic "internal" account.
type Syncer struct {
	store    *store.Store
	accounts *auth.Accounts
	envToken string

	mu       sync.Mutex
	running  bool
	last     *Result
	lastPull *PullResult
}

const internalUserID = "internal"

func NewSyncer(s *store.Store, accounts *auth.Accounts, envToken string) *Syncer {
	return &Syncer{store: s, accounts: accounts, envToken: envToken}
}

// resolve picks the account that owns the sync state and the token to use.
func (y *Syncer) resolve(userID string) (auth.Account, string, error) {
	if y.envToken != "" {
		acc, ok := y.accounts.Get(internalUserID)
		if !ok {
			acc = auth.Account{UserID: internalUserID, Name: "Notion (internal token)", ConnectedAt: time.Now().UTC()}
			if err := y.accounts.Put(acc); err != nil {
				return auth.Account{}, "", err
			}
		}
		return acc, y.envToken, nil
	}
	acc, ok := y.accounts.Get(userID)
	if !ok || acc.AccessToken == "" {
		return auth.Account{}, "", errors.New("no Notion account connected")
	}
	return acc, acc.AccessToken, nil
}

// Status is the JSON shape the frontend polls.
type Status struct {
	Connected     bool        `json:"connected"`
	WorkspaceName string      `json:"workspaceName,omitempty"`
	PageURL       string      `json:"pageUrl,omitempty"`
	Running       bool        `json:"running"`
	LastSyncedAt  *time.Time  `json:"lastSyncedAt,omitempty"`
	Last          *Result     `json:"last,omitempty"`
	LastPull      *PullResult `json:"lastPull,omitempty"`
}

func (y *Syncer) Status(userID string) Status {
	y.mu.Lock()
	st := Status{Running: y.running, Last: y.last, LastPull: y.lastPull}
	y.mu.Unlock()
	lookup := userID
	if y.envToken != "" {
		st.Connected = true
		lookup = internalUserID
	}
	if acc, ok := y.accounts.Get(lookup); ok {
		st.Connected = st.Connected || acc.AccessToken != ""
		st.WorkspaceName = acc.WorkspaceName
		st.PageURL = acc.Sync.PageURL
		st.LastSyncedAt = acc.Sync.LastSyncedAt
	}
	return st
}

// Start kicks off a background sync for the given account. Only one run at a
// time; a second Start while running returns an error.
func (y *Syncer) Start(userID string) error {
	acc, token, err := y.resolve(userID)
	if err != nil {
		return err
	}
	y.mu.Lock()
	defer y.mu.Unlock()
	if y.running {
		return errors.New("a sync is already running")
	}
	y.running = true
	go y.run(acc, token)
	return nil
}

func (y *Syncer) run(acc auth.Account, token string) {
	res := &Result{StartedAt: time.Now().UTC()}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	err := y.sync(ctx, acc, token, res)
	res.FinishedAt = time.Now().UTC()
	if err != nil {
		res.Error = err.Error()
		log.Printf("notion sync: %v", err)
	} else {
		now := res.FinishedAt
		_ = y.accounts.UpdateSync(acc.UserID, func(s *auth.SyncState) { s.LastSyncedAt = &now })
		log.Printf("notion sync: %d created, %d updated", res.Created, res.Updated)
	}

	y.mu.Lock()
	y.running = false
	y.last = res
	y.mu.Unlock()
}

func (y *Syncer) sync(ctx context.Context, acc auth.Account, token string, res *Result) error {
	c := NewClient(token)
	state, err := y.ensureTargets(ctx, c, acc)
	if err != nil {
		return err
	}

	snap := y.store.Export().Data
	categories := map[string]string{}
	for _, cat := range snap.Categories {
		categories[cat.ID] = cat.Name
	}

	if err := y.upsertAll(ctx, c, state.ExpensesDBID, len(snap.Expenses), res, func(i int) (string, map[string]any) {
		e := snap.Expenses[i]
		res.Expenses++
		return e.ID, expenseProps(e, categories[e.CategoryID])
	}); err != nil {
		return fmt.Errorf("expenses: %w", err)
	}

	if err := y.upsertAll(ctx, c, state.IncomesDBID, len(snap.Incomes), res, func(i int) (string, map[string]any) {
		in := snap.Incomes[i]
		res.Incomes++
		return in.ID, incomeProps(in)
	}); err != nil {
		return fmt.Errorf("incomes: %w", err)
	}

	if err := y.upsertAll(ctx, c, state.InvestmentsDBID, len(snap.Investments), res, func(i int) (string, map[string]any) {
		inv := snap.Investments[i]
		res.Investments++
		return inv.ID, investmentProps(inv)
	}); err != nil {
		return fmt.Errorf("investments: %w", err)
	}
	return nil
}

// upsertAll loads the existing App ID -> row map once, creates or updates
// each entity, then archives rows whose entity was deleted in the app (the
// app is the source of truth; Notion's trash keeps them recoverable). Rows
// without an App ID are untouched — they're Notion-side additions awaiting a
// pull.
func (y *Syncer) upsertAll(ctx context.Context, c *Client, dbID string, n int, res *Result, row func(i int) (appID string, props map[string]any)) error {
	existing, err := c.ExistingRows(ctx, dbID)
	if err != nil {
		return err
	}
	pushed := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		appID, props := row(i)
		pushed[appID] = true
		if pageID, ok := existing[appID]; ok {
			if err := c.UpdateRow(ctx, pageID, props); err != nil {
				return err
			}
			res.Updated++
		} else {
			if err := c.CreateRow(ctx, dbID, props); err != nil {
				return err
			}
			res.Created++
		}
	}
	for appID, pageID := range existing {
		if !pushed[appID] {
			if err := c.ArchiveRow(ctx, pageID); err != nil {
				return err
			}
			res.Archived++
		}
	}
	return nil
}

// ensureTargets finds or creates the "Money Tracker" page and its three
// databases, persisting their ids so later syncs (and re-logins) reuse them.
func (y *Syncer) ensureTargets(ctx context.Context, c *Client, acc auth.Account) (auth.SyncState, error) {
	state := acc.Sync

	if state.PageID == "" || !c.PageExists(ctx, state.PageID) {
		// Prefer an existing "Money Tracker" page (e.g. created by a previous
		// deployment) over making a duplicate.
		id, url, err := c.FindPageByTitle(ctx, pageTitle)
		if err != nil {
			return state, err
		}
		if id == "" {
			parent, err := c.FirstSharedPage(ctx)
			if err != nil {
				return state, err
			}
			id, url, err = c.CreatePage(ctx, parent, pageTitle)
			if err != nil {
				return state, err
			}
		}
		state = auth.SyncState{PageID: id, PageURL: url}
	}

	// Re-adopt databases already on the page (e.g. auth.json was lost).
	existing, err := c.ChildDatabases(ctx, state.PageID)
	if err != nil {
		return state, err
	}
	ensure := func(current *string, title string, props map[string]any) error {
		if *current != "" && c.DatabaseExists(ctx, *current) {
			return nil
		}
		if id, ok := existing[title]; ok {
			*current = id
			return nil
		}
		id, err := c.CreateDatabase(ctx, state.PageID, title, props)
		if err != nil {
			return err
		}
		*current = id
		return nil
	}
	if err := ensure(&state.ExpensesDBID, expensesTitle, expensesSchema()); err != nil {
		return state, err
	}
	if err := ensure(&state.IncomesDBID, incomesTitle, incomesSchema()); err != nil {
		return state, err
	}
	if err := ensure(&state.InvestmentsDBID, investmentsTitle, investmentsSchema()); err != nil {
		return state, err
	}

	if state != acc.Sync {
		final := state
		if err := y.accounts.UpdateSync(acc.UserID, func(s *auth.SyncState) {
			last := s.LastSyncedAt
			*s = final
			s.LastSyncedAt = last
		}); err != nil {
			return state, err
		}
	}
	return state, nil
}

// ---- schemas & row mappers (Notion property shapes for each entity) ----

func expensesSchema() map[string]any {
	return map[string]any{
		"Name":           map[string]any{"title": map[string]any{}},
		"Amount":         map[string]any{"number": map[string]any{}},
		"Currency":       map[string]any{"select": map[string]any{}},
		"Category":       map[string]any{"select": map[string]any{}},
		"Subcategory":    map[string]any{"rich_text": map[string]any{}},
		"Date":           map[string]any{"date": map[string]any{}},
		"Payment Method": map[string]any{"select": map[string]any{}},
		"Note":           map[string]any{"rich_text": map[string]any{}},
		"App ID":         map[string]any{"rich_text": map[string]any{}},
	}
}

func expenseProps(e domain.Expense, categoryName string) map[string]any {
	title := categoryName
	if title == "" {
		title = "Expense"
	}
	if e.Subcategory != "" {
		title += " · " + e.Subcategory
	}
	return map[string]any{
		"Name":           titleProp(title),
		"Amount":         numberProp(e.Amount),
		"Currency":       selectProp(e.Currency),
		"Category":       selectProp(categoryName),
		"Subcategory":    textProp(e.Subcategory),
		"Date":           dateProp(e.Date),
		"Payment Method": selectProp(e.PaymentMethod),
		"Note":           textProp(e.Note),
		"App ID":         textProp(e.ID),
	}
}

func incomesSchema() map[string]any {
	return map[string]any{
		"Source":      map[string]any{"title": map[string]any{}},
		"Amount":      map[string]any{"number": map[string]any{}},
		"Currency":    map[string]any{"select": map[string]any{}},
		"Received On": map[string]any{"date": map[string]any{}},
		"Month":       map[string]any{"number": map[string]any{}},
		"Year":        map[string]any{"number": map[string]any{}},
		"Note":        map[string]any{"rich_text": map[string]any{}},
		"App ID":      map[string]any{"rich_text": map[string]any{}},
	}
}

func incomeProps(in domain.Income) map[string]any {
	return map[string]any{
		"Source":      titleProp(in.Source),
		"Amount":      numberProp(in.Amount),
		"Currency":    selectProp(in.Currency),
		"Received On": dateProp(in.ReceivedOn),
		"Month":       numberProp(float64(in.Month)),
		"Year":        numberProp(float64(in.Year)),
		"Note":        textProp(in.Note),
		"App ID":      textProp(in.ID),
	}
}

func investmentsSchema() map[string]any {
	return map[string]any{
		"Name":            map[string]any{"title": map[string]any{}},
		"Type":            map[string]any{"select": map[string]any{}},
		"Platform":        map[string]any{"rich_text": map[string]any{}},
		"Symbol":          map[string]any{"rich_text": map[string]any{}},
		"Amount Invested": map[string]any{"number": map[string]any{}},
		"Current Value":   map[string]any{"number": map[string]any{}},
		"Currency":        map[string]any{"select": map[string]any{}},
		"Invested On":     map[string]any{"date": map[string]any{}},
		"Note":            map[string]any{"rich_text": map[string]any{}},
		"App ID":          map[string]any{"rich_text": map[string]any{}},
	}
}

func investmentProps(inv domain.Investment) map[string]any {
	props := map[string]any{
		"Name":            titleProp(inv.Name),
		"Type":            selectProp(inv.Type),
		"Platform":        textProp(inv.Platform),
		"Symbol":          textProp(inv.Symbol),
		"Amount Invested": numberProp(inv.AmountInvested),
		"Currency":        selectProp(inv.Currency),
		"Invested On":     dateProp(inv.InvestedOn),
		"Note":            textProp(inv.Note),
		"App ID":          textProp(inv.ID),
	}
	// Same derivation the UI uses: live quantity*price wins over the manual value.
	if inv.Quantity != nil && inv.LastPrice != nil {
		props["Current Value"] = numberProp(*inv.Quantity * *inv.LastPrice)
	} else if inv.CurrentValue != nil {
		props["Current Value"] = numberProp(*inv.CurrentValue)
	} else {
		props["Current Value"] = map[string]any{"number": nil}
	}
	return props
}
