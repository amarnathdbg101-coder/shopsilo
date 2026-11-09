// Package reuse provides reusable application helpers and utilities.
package reuse

import (
	"fmt"
	"math"
	"time"

	gohumanize "github.com/amarnathdbg101-coder/go/go-humanize"
)

// HumanizeIndianRupees formats a decimal rupee amount into standard Indian grouped format (e.g. "₹12,50,000.50").
func HumanizeIndianRupees(rupees float64) string {
	sign := ""
	if rupees < 0 {
		sign = "-"
		rupees = -rupees
	}

	intPart := int64(math.Floor(rupees))
	paise := int64(math.Round((rupees - float64(intPart)) * 100))
	if paise == 100 {
		intPart++
		paise = 0
	}

	formattedInt := gohumanize.FormatIndian(intPart)
	if paise > 0 {
		return fmt.Sprintf("%s₹%s.%02d", sign, formattedInt, paise)
	}
	return fmt.Sprintf("%s₹%s", sign, formattedInt)
}

// HumanizeIndianCompact formats large rupee figures into readable Indian financial milestones (e.g. "₹1.5 Lakh", "₹2.4 Cr").
func HumanizeIndianCompact(rupees float64) string {
	if rupees < 0 {
		return "-" + HumanizeIndianCompact(-rupees)
	}
	return "₹" + gohumanize.CompactIndian(int64(rupees))
}

// HumanizeTimeAgo returns a human-friendly relative duration (e.g. "just now", "15 minutes ago", "yesterday").
func HumanizeTimeAgo(t time.Time) string {
	return gohumanize.TimeAgo(t)
}

// HumanizeItemCount returns a count-aware singular or plural phrase (e.g. "1 item", "5 items").
func HumanizeItemCount(count int64, singular, plural string) string {
	return gohumanize.Plural(int(count), singular, plural)
}
