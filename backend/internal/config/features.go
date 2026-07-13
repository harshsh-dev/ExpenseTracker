// Package config resolves which product features are enabled for a given
// deployment. This lets the same binary be shipped "full-fledged" or trimmed
// to a subset of features purely via the FEATURES environment variable, e.g.:
//
//	FEATURES=all                      // everything (default)
//	FEATURES=income,expenses          // a lean expense-only deployment
//	FEATURES=investments              // a portfolio-only deployment
//
// The API only mounts routes for enabled features and advertises the resolved
// set at GET /api/config so the frontend can adapt its navigation.
package config

import (
	"log"
	"sort"
	"strings"
)

// Feature is a self-contained product capability.
type Feature string

const (
	Dashboard   Feature = "dashboard"
	Income      Feature = "income"
	Expenses    Feature = "expenses"
	Investments Feature = "investments"
	Recurring   Feature = "recurring"
	Loans       Feature = "loans"
	Categories  Feature = "categories"
	Reports     Feature = "report"
	Backup      Feature = "backup"
)

// all is the canonical, ordered list of every feature.
var all = []Feature{Dashboard, Income, Expenses, Investments, Recurring, Loans, Categories, Reports, Backup}

// deps maps a feature to other features it requires to function. Enabling a
// feature transitively enables its dependencies (e.g. expenses are categorized,
// so enabling expenses also enables categories).
var deps = map[Feature][]Feature{
	Expenses: {Categories},
	// Recurring rules materialize expenses and SIP into investments.
	Recurring: {Expenses, Investments},
}

// Features is an immutable, resolved set of enabled features.
type Features struct {
	set map[Feature]bool
}

// Parse builds the enabled set from a raw FEATURES value. Empty, "all" or "*"
// enable everything. Otherwise it's a comma/space separated list of feature
// names; unknown names are ignored (with a warning) and dependencies are added
// automatically. If the list resolves to nothing, it falls back to all so a
// misconfiguration never ships an empty app.
func Parse(raw string) Features {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" || raw == "all" || raw == "*" {
		return enableAll()
	}

	valid := map[Feature]bool{}
	for _, f := range all {
		valid[f] = true
	}

	set := map[Feature]bool{}
	for _, tok := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	}) {
		f := Feature(tok)
		if !valid[f] {
			log.Printf("config: ignoring unknown feature %q", tok)
			continue
		}
		enable(set, f)
	}

	if len(set) == 0 {
		log.Printf("config: FEATURES=%q resolved to nothing, defaulting to all", raw)
		return enableAll()
	}
	return Features{set: set}
}

// enable adds a feature and all of its (transitive) dependencies.
func enable(set map[Feature]bool, f Feature) {
	if set[f] {
		return
	}
	set[f] = true
	for _, d := range deps[f] {
		enable(set, d)
	}
}

func enableAll() Features {
	set := map[Feature]bool{}
	for _, f := range all {
		set[f] = true
	}
	return Features{set: set}
}

// Enabled reports whether a feature is active in this deployment.
func (f Features) Enabled(x Feature) bool { return f.set[x] }

// List returns the enabled features in canonical order.
func (f Features) List() []string {
	out := make([]string, 0, len(f.set))
	for _, x := range all {
		if f.set[x] {
			out = append(out, string(x))
		}
	}
	return out
}

// IsAll reports whether every feature is enabled.
func (f Features) IsAll() bool { return len(f.set) == len(all) }

// String is a stable, comma-joined view for logging.
func (f Features) String() string {
	l := f.List()
	sort.Strings(l)
	return strings.Join(l, ",")
}
