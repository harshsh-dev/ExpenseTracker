// Package notion mirrors the tracker's data into Notion databases using the
// access token obtained at login (internal/auth). One-way only: the app stays
// the source of truth and Notion is a read-only reporting view.
package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	apiBase    = "https://api.notion.com/v1"
	apiVersion = "2022-06-28"
	// Notion allows ~3 requests/sec per integration; space calls out to stay under.
	minRequestGap = 340 * time.Millisecond
)

// ErrNotFound distinguishes deleted/revoked pages so the syncer can recreate them.
var ErrNotFound = errors.New("notion: not found")

// Client is a minimal Notion REST client for one access token.
type Client struct {
	token string
	hc    *http.Client

	mu      sync.Mutex
	lastReq time.Time
}

func NewClient(token string) *Client {
	return &Client{token: token, hc: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	c.throttle()

	var rd *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, apiBase+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", apiVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusTooManyRequests {
		// Honor Retry-After once, then retry the call.
		wait := 2 * time.Second
		if s, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && s > 0 {
			wait = time.Duration(s) * time.Second
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		return c.do(ctx, method, path, body, out)
	}
	if res.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		var e struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(res.Body).Decode(&e)
		return fmt.Errorf("notion: %s %s: %s (%s)", method, path, res.Status, e.Message)
	}
	if out != nil {
		return json.NewDecoder(res.Body).Decode(out)
	}
	return nil
}

func (c *Client) throttle() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if gap := time.Since(c.lastReq); gap < minRequestGap {
		time.Sleep(minRequestGap - gap)
	}
	c.lastReq = time.Now()
}

// ---- endpoints ----

type object struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// FirstSharedPage returns any page the user granted the integration during
// login — needed because the public API cannot create top-level pages.
func (c *Client) FirstSharedPage(ctx context.Context) (string, error) {
	var res struct {
		Results []object `json:"results"`
	}
	body := map[string]any{
		"filter":    map[string]string{"property": "object", "value": "page"},
		"page_size": 1,
	}
	if err := c.do(ctx, http.MethodPost, "/search", body, &res); err != nil {
		return "", err
	}
	if len(res.Results) == 0 {
		return "", errors.New("no Notion page is shared with this integration — pick at least one page on the Notion consent screen when logging in")
	}
	return res.Results[0].ID, nil
}

// FindPageByTitle returns an accessible page whose title matches exactly, so
// a fresh server re-adopts an existing "Money Tracker" page instead of
// creating a duplicate. Returns ("", "", nil) when none matches.
func (c *Client) FindPageByTitle(ctx context.Context, title string) (id, url string, err error) {
	body := map[string]any{
		"query":     title,
		"filter":    map[string]string{"property": "object", "value": "page"},
		"page_size": 20,
	}
	var res struct {
		Results []struct {
			ID         string `json:"id"`
			URL        string `json:"url"`
			Properties map[string]struct {
				Title []struct {
					PlainText string `json:"plain_text"`
				} `json:"title"`
			} `json:"properties"`
		} `json:"results"`
	}
	if err := c.do(ctx, http.MethodPost, "/search", body, &res); err != nil {
		return "", "", err
	}
	for _, r := range res.Results {
		for _, p := range r.Properties {
			if len(p.Title) > 0 && p.Title[0].PlainText == title {
				return r.ID, r.URL, nil
			}
		}
	}
	return "", "", nil
}

// CreatePage creates a child page under parentID and returns its id + URL.
func (c *Client) CreatePage(ctx context.Context, parentID, title string) (id, url string, err error) {
	body := map[string]any{
		"parent": map[string]string{"page_id": parentID},
		"properties": map[string]any{
			"title": map[string]any{"title": richText(title)},
		},
	}
	var out object
	if err := c.do(ctx, http.MethodPost, "/pages", body, &out); err != nil {
		return "", "", err
	}
	return out.ID, out.URL, nil
}

// PageExists checks a page is still reachable (not deleted / access revoked).
func (c *Client) PageExists(ctx context.Context, id string) bool {
	var out struct {
		Archived bool `json:"archived"`
	}
	if err := c.do(ctx, http.MethodGet, "/pages/"+id, nil, &out); err != nil {
		return false
	}
	return !out.Archived
}

// DatabaseExists checks a database is still reachable.
func (c *Client) DatabaseExists(ctx context.Context, id string) bool {
	return c.do(ctx, http.MethodGet, "/databases/"+id, nil, nil) == nil
}

// CreateDatabase creates an inline database under a page.
func (c *Client) CreateDatabase(ctx context.Context, parentPageID, title string, properties map[string]any) (string, error) {
	body := map[string]any{
		"parent":     map[string]string{"type": "page_id", "page_id": parentPageID},
		"title":      richText(title),
		"is_inline":  true,
		"properties": properties,
	}
	var out object
	if err := c.do(ctx, http.MethodPost, "/databases", body, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

// ChildDatabases lists child databases of a page by title, so an existing
// "Money Tracker" page can be re-adopted without creating duplicates.
func (c *Client) ChildDatabases(ctx context.Context, pageID string) (map[string]string, error) {
	found := map[string]string{} // title -> database id
	cursor := ""
	for {
		path := "/blocks/" + pageID + "/children?page_size=100"
		if cursor != "" {
			path += "&start_cursor=" + cursor
		}
		var res struct {
			Results []struct {
				ID            string `json:"id"`
				Type          string `json:"type"`
				ChildDatabase struct {
					Title string `json:"title"`
				} `json:"child_database"`
			} `json:"results"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
		}
		if err := c.do(ctx, http.MethodGet, path, nil, &res); err != nil {
			return nil, err
		}
		for _, b := range res.Results {
			if b.Type == "child_database" {
				found[b.ChildDatabase.Title] = b.ID
			}
		}
		if !res.HasMore {
			return found, nil
		}
		cursor = res.NextCursor
	}
}

// ExistingRows maps "App ID" -> Notion page id for every row in a database,
// enabling idempotent upserts.
func (c *Client) ExistingRows(ctx context.Context, databaseID string) (map[string]string, error) {
	rows := map[string]string{}
	var cursor string
	for {
		body := map[string]any{"page_size": 100}
		if cursor != "" {
			body["start_cursor"] = cursor
		}
		var res struct {
			Results []struct {
				ID         string `json:"id"`
				Properties struct {
					AppID struct {
						RichText []struct {
							PlainText string `json:"plain_text"`
						} `json:"rich_text"`
					} `json:"App ID"`
				} `json:"properties"`
			} `json:"results"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
		}
		if err := c.do(ctx, http.MethodPost, "/databases/"+databaseID+"/query", body, &res); err != nil {
			return nil, err
		}
		for _, p := range res.Results {
			if rt := p.Properties.AppID.RichText; len(rt) > 0 {
				rows[rt[0].PlainText] = p.ID
			}
		}
		if !res.HasMore {
			return rows, nil
		}
		cursor = res.NextCursor
	}
}

// CreateRow inserts a database row.
func (c *Client) CreateRow(ctx context.Context, databaseID string, properties map[string]any) error {
	body := map[string]any{
		"parent":     map[string]string{"database_id": databaseID},
		"properties": properties,
	}
	return c.do(ctx, http.MethodPost, "/pages", body, nil)
}

// UpdateRow overwrites a row's properties.
func (c *Client) UpdateRow(ctx context.Context, pageID string, properties map[string]any) error {
	return c.do(ctx, http.MethodPatch, "/pages/"+pageID, map[string]any{"properties": properties}, nil)
}

// ---- property value builders ----

func richText(s string) []map[string]any {
	return []map[string]any{{"text": map[string]any{"content": s}}}
}

func titleProp(s string) map[string]any {
	if s == "" {
		s = "Untitled"
	}
	return map[string]any{"title": richText(s)}
}

func textProp(s string) map[string]any {
	if s == "" {
		return map[string]any{"rich_text": []any{}}
	}
	return map[string]any{"rich_text": richText(s)}
}

func numberProp(n float64) map[string]any {
	return map[string]any{"number": n}
}

func selectProp(s string) map[string]any {
	if s == "" {
		return map[string]any{"select": nil}
	}
	return map[string]any{"select": map[string]string{"name": s}}
}

func dateProp(iso string) map[string]any {
	if iso == "" {
		return map[string]any{"date": nil}
	}
	return map[string]any{"date": map[string]string{"start": iso}}
}
