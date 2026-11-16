package goledger

import "time"

// AgingItem represents an unpaid invoice, credit ticket, or khata advance.
type AgingItem struct {
	Reference      string    `json:"reference"`
	Date           time.Time `json:"date"`
	AmountSubunits int64     `json:"amount_subunits"` // In minor subunits (paisa/cents)
}

// AgingReport breaks down outstanding receivables/payables into risk delinquency brackets.
type AgingReport struct {
	CurrentSubunits      int64 `json:"current_subunits"`       // 0 - 30 days
	Days31To60Subunits   int64 `json:"days_31_to_60_subunits"` // 31 - 60 days
	Days61To90Subunits   int64 `json:"days_61_to_90_subunits"` // 61 - 90 days
	Over90DaysSubunits   int64 `json:"over_90_days_subunits"`  // 91+ days
	TotalSubunits        int64 `json:"total_subunits"`
	TotalOutstandingFlag bool  `json:"has_overdue"`
}

// CurrentFloat returns current (0-30 days) balance in major currency units.
func (r AgingReport) CurrentFloat() float64 { return float64(r.CurrentSubunits) / 100.0 }

// Days31To60Float returns 31-60 days balance in major currency units.
func (r AgingReport) Days31To60Float() float64 { return float64(r.Days31To60Subunits) / 100.0 }

// Days61To90Float returns 61-90 days balance in major currency units.
func (r AgingReport) Days61To90Float() float64 { return float64(r.Days61To90Subunits) / 100.0 }

// Over90DaysFloat returns 90+ days balance in major currency units.
func (r AgingReport) Over90DaysFloat() float64 { return float64(r.Over90DaysSubunits) / 100.0 }

// TotalFloat returns total balance in major currency units.
func (r AgingReport) TotalFloat() float64 { return float64(r.TotalSubunits) / 100.0 }

// CalculateAging classifies outstanding items into standard accounting aging buckets relative to asOf date.
func CalculateAging(items []AgingItem, asOf time.Time) AgingReport {
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}

	var report AgingReport

	for _, item := range items {
		if item.AmountSubunits <= 0 {
			continue
		}

		daysOld := int(asOf.Sub(item.Date).Hours() / 24)
		if daysOld < 0 {
			daysOld = 0
		}

		report.TotalSubunits += item.AmountSubunits

		switch {
		case daysOld <= 30:
			report.CurrentSubunits += item.AmountSubunits
		case daysOld <= 60:
			report.Days31To60Subunits += item.AmountSubunits
		case daysOld <= 90:
			report.Days61To90Subunits += item.AmountSubunits
		default:
			report.Over90DaysSubunits += item.AmountSubunits
		}
	}

	report.TotalOutstandingFlag = (report.Days31To60Subunits + report.Days61To90Subunits + report.Over90DaysSubunits) > 0
	return report
}
