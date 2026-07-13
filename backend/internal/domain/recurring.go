package domain

import "time"

// Recurring kinds and cadences.
const (
	RecurringExpense = "expense"
	RecurringSIP     = "sip"

	CadenceMonthly = "monthly"
	CadenceWeekly  = "weekly"
	CadenceYearly  = "yearly"
)

const isoDate = "2006-01-02"

// NextOccurrence returns the occurrence after current. The start date anchors
// the schedule: its day-of-month for monthly (clamped to short months, so a
// 31st SIP runs on Feb 28), its month+day for yearly, its weekday for weekly.
// Returns "" for unparseable input, which stops materialization.
func NextOccurrence(cadence, start, current string) string {
	cur, err := time.Parse(isoDate, current)
	if err != nil {
		return ""
	}
	st, err := time.Parse(isoDate, start)
	if err != nil {
		st = cur
	}
	switch cadence {
	case CadenceWeekly:
		return cur.AddDate(0, 0, 7).Format(isoDate)
	case CadenceYearly:
		return clampedDate(cur.Year()+1, st.Month(), st.Day()).Format(isoDate)
	default: // monthly
		return clampedDate(cur.Year(), cur.Month()+1, st.Day()).Format(isoDate)
	}
}

// clampedDate builds a date, clamping the day to the month's length.
// The month may be out of range (e.g. 13) — time.Date normalizes it first.
func clampedDate(year int, month time.Month, day int) time.Time {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	if last := first.AddDate(0, 1, -1).Day(); day > last {
		day = last
	}
	return time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, time.UTC)
}
