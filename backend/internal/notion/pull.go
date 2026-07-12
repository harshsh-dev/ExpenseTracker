package notion

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"moneytracker/backend/internal/auth"
	"moneytracker/backend/internal/domain"
)

// Pull imports rows from the Notion databases back into the app:
//   - rows without an App ID were added in Notion → validate, create in the
//     app, and stamp the new id back onto the row
//   - rows with a known App ID update the app entity when the Notion edit is
//     newer AND the values actually differ (round-trip pushes bump Notion's
//     last_edited_time, so a bare timestamp check would ping-pong)
//   - rows whose App ID no longer exists in the app are tombstones of entries
//     deleted in the app — never resurrected (deletions don't propagate)
//   - invalid rows are skipped and reported, never guessed at

const maxSkipReasons = 10

// PullResult summarizes one pull run.
type PullResult struct {
	StartedAt   time.Time `json:"startedAt"`
	FinishedAt  time.Time `json:"finishedAt"`
	Error       string    `json:"error,omitempty"`
	Created     int       `json:"created"`
	Updated     int       `json:"updated"`
	Unchanged   int       `json:"unchanged"`
	Skipped     int       `json:"skipped"`
	SkipReasons []string  `json:"skipReasons,omitempty"`
}

func (r *PullResult) skip(reason string) {
	r.Skipped++
	if len(r.SkipReasons) < maxSkipReasons {
		r.SkipReasons = append(r.SkipReasons, reason)
	}
}

// StartPull kicks off a background Notion → app import. Shares the running
// gate with push syncs so the two never interleave.
func (y *Syncer) StartPull(userID string) error {
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
	go y.runPull(acc, token)
	return nil
}

func (y *Syncer) runPull(acc auth.Account, token string) {
	res := &PullResult{StartedAt: time.Now().UTC()}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	err := y.pull(ctx, acc, token, res)
	res.FinishedAt = time.Now().UTC()
	if err != nil {
		res.Error = err.Error()
		log.Printf("notion pull: %v", err)
	} else {
		log.Printf("notion pull: %d created, %d updated, %d skipped", res.Created, res.Updated, res.Skipped)
	}

	y.mu.Lock()
	y.running = false
	y.lastPull = res
	y.mu.Unlock()
}

func (y *Syncer) pull(ctx context.Context, acc auth.Account, token string, res *PullResult) error {
	c := NewClient(token)
	state, err := y.ensureTargets(ctx, c, acc)
	if err != nil {
		return err
	}

	// Category names in Notion map back to ids; unknown names are created so
	// a phone-entered expense never gets dropped over a new category.
	catByName := map[string]string{}
	for _, cat := range y.store.ListCategories() {
		catByName[strings.ToLower(cat.Name)] = cat.ID
	}
	resolveCategory := func(name string) (string, error) {
		if name == "" {
			return "", errors.New("category is empty")
		}
		if id, ok := catByName[strings.ToLower(name)]; ok {
			return id, nil
		}
		cat := domain.Category{Name: name, Subcategories: []string{}}
		_ = cat.Validate() // fills default color
		created, err := y.store.CreateCategory(cat)
		if err != nil {
			return "", err
		}
		catByName[strings.ToLower(name)] = created.ID
		return created.ID, nil
	}

	if err := y.pullExpenses(ctx, c, state.ExpensesDBID, resolveCategory, res); err != nil {
		return fmt.Errorf("expenses: %w", err)
	}
	if err := y.pullIncomes(ctx, c, state.IncomesDBID, res); err != nil {
		return fmt.Errorf("incomes: %w", err)
	}
	if err := y.pullInvestments(ctx, c, state.InvestmentsDBID, res); err != nil {
		return fmt.Errorf("investments: %w", err)
	}
	return nil
}

// stamp writes the app id onto a Notion row so later runs treat it as synced.
func stamp(ctx context.Context, c *Client, pageID, appID string) {
	if err := c.UpdateRow(ctx, pageID, map[string]any{"App ID": textProp(appID)}); err != nil {
		log.Printf("notion pull: stamp %s: %v", pageID, err)
	}
}

func (y *Syncer) pullExpenses(ctx context.Context, c *Client, dbID string, resolveCategory func(string) (string, error), res *PullResult) error {
	rows, err := c.QueryRows(ctx, dbID)
	if err != nil {
		return err
	}
	existing := map[string]domain.Expense{}
	for _, e := range y.store.ListExpenses() {
		existing[e.ID] = e
	}

	for _, row := range rows {
		label := fmt.Sprintf("expense %q (%s)", row.Props["Name"].Text(), row.Props["Date"].DateStr())
		e := domain.Expense{
			Amount:        row.Props["Amount"].Num(),
			Currency:      row.Props["Currency"].Sel(),
			Subcategory:   row.Props["Subcategory"].Text(),
			Date:          row.Props["Date"].DateStr(),
			PaymentMethod: row.Props["Payment Method"].Sel(),
			Note:          row.Props["Note"].Text(),
		}
		if name := row.Props["Category"].Sel(); name != "" {
			id, err := resolveCategory(name)
			if err != nil {
				res.skip(label + ": " + err.Error())
				continue
			}
			e.CategoryID = id
		}
		if err := e.Validate(); err != nil {
			res.skip(label + ": " + err.Error())
			continue
		}

		appID := row.Props["App ID"].Text()
		switch cur, known := existing[appID]; {
		case appID == "":
			created, err := y.store.CreateExpense(e)
			if err != nil {
				return err
			}
			stamp(ctx, c, row.PageID, created.ID)
			res.Created++
		case !known:
			res.skip(label + ": deleted in app (not resurrected)")
		case !row.LastEdited.After(cur.UpdatedAt):
			res.Unchanged++
		default:
			e.Base = cur.Base
			if e == cur {
				res.Unchanged++
				continue
			}
			if _, err := y.store.UpdateExpense(appID, e); err != nil {
				return err
			}
			res.Updated++
		}
	}
	return nil
}

func (y *Syncer) pullIncomes(ctx context.Context, c *Client, dbID string, res *PullResult) error {
	rows, err := c.QueryRows(ctx, dbID)
	if err != nil {
		return err
	}
	existing := map[string]domain.Income{}
	for _, in := range y.store.ListIncomes() {
		existing[in.ID] = in
	}

	for _, row := range rows {
		label := fmt.Sprintf("income %q", row.Props["Source"].Text())
		in := domain.Income{
			Source:     row.Props["Source"].Text(),
			Amount:     row.Props["Amount"].Num(),
			Currency:   row.Props["Currency"].Sel(),
			ReceivedOn: row.Props["Received On"].DateStr(),
			Month:      int(row.Props["Month"].Num()),
			Year:       int(row.Props["Year"].Num()),
			Note:       row.Props["Note"].Text(),
		}
		// Month/year are easy to forget in Notion; derive them from the date.
		if (in.Month == 0 || in.Year == 0) && in.ReceivedOn != "" {
			if t, err := time.Parse("2006-01-02", in.ReceivedOn); err == nil {
				if in.Month == 0 {
					in.Month = int(t.Month())
				}
				if in.Year == 0 {
					in.Year = t.Year()
				}
			}
		}
		if err := in.Validate(); err != nil {
			res.skip(label + ": " + err.Error())
			continue
		}

		appID := row.Props["App ID"].Text()
		switch cur, known := existing[appID]; {
		case appID == "":
			created, err := y.store.CreateIncome(in)
			if err != nil {
				return err
			}
			stamp(ctx, c, row.PageID, created.ID)
			res.Created++
		case !known:
			res.skip(label + ": deleted in app (not resurrected)")
		case !row.LastEdited.After(cur.UpdatedAt):
			res.Unchanged++
		default:
			in.Base = cur.Base
			if in == cur {
				res.Unchanged++
				continue
			}
			if _, err := y.store.UpdateIncome(appID, in); err != nil {
				return err
			}
			res.Updated++
		}
	}
	return nil
}

func (y *Syncer) pullInvestments(ctx context.Context, c *Client, dbID string, res *PullResult) error {
	rows, err := c.QueryRows(ctx, dbID)
	if err != nil {
		return err
	}
	existing := map[string]domain.Investment{}
	for _, inv := range y.store.ListInvestments() {
		existing[inv.ID] = inv
	}

	for _, row := range rows {
		label := fmt.Sprintf("investment %q", row.Props["Name"].Text())
		appID := row.Props["App ID"].Text()
		cur, known := existing[appID]

		// Overlay only the Notion-visible fields; provider, quantity, and the
		// price cache are app-owned and must survive a pull untouched.
		inv := cur
		inv.Name = row.Props["Name"].Text()
		inv.Type = row.Props["Type"].Sel()
		inv.Platform = row.Props["Platform"].Text()
		inv.Symbol = row.Props["Symbol"].Text()
		inv.AmountInvested = row.Props["Amount Invested"].Num()
		inv.Currency = row.Props["Currency"].Sel()
		inv.InvestedOn = row.Props["Invested On"].DateStr()
		inv.Note = row.Props["Note"].Text()
		// "Current Value" in Notion is derived when the app tracks
		// quantity+price — only pull it back as the manual value otherwise.
		if cur.Quantity == nil || cur.LastPrice == nil {
			inv.CurrentValue = row.Props["Current Value"].Number
		}
		if err := inv.Validate(); err != nil {
			res.skip(label + ": " + err.Error())
			continue
		}

		switch {
		case appID == "":
			created, err := y.store.CreateInvestment(inv)
			if err != nil {
				return err
			}
			stamp(ctx, c, row.PageID, created.ID)
			res.Created++
		case !known:
			res.skip(label + ": deleted in app (not resurrected)")
		case !row.LastEdited.After(cur.UpdatedAt):
			res.Unchanged++
		default:
			inv.Base = cur.Base
			if reflect.DeepEqual(inv, cur) {
				res.Unchanged++
				continue
			}
			if _, err := y.store.UpdateInvestment(appID, inv); err != nil {
				return err
			}
			res.Updated++
		}
	}
	return nil
}
