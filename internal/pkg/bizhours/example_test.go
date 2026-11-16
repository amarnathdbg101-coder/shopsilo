package bizhours_test

import (
	"fmt"
	"time"

	"shopMe/internal/pkg/bizhours"
)

func ExampleCalculateAt() {
	// A shop open from 09:00 to 21:00 with Sunday off
	// Reference time: Thursday at 14:00 (2:00 PM)
	now := time.Date(2026, 11, 5, 14, 0, 0, 0, time.UTC)

	status := bizhours.CalculateAt(true, "09:00", "21:00", "Sunday", now)
	fmt.Println("Is Open:", status.IsOpenNow)
	fmt.Println("Status:", status.Status)
	fmt.Println("Minutes to close:", status.MinutesToClose)

	// Output:
	// Is Open: true
	// Status: OPEN
	// Minutes to close: 420
}

func ExampleFormatTime() {
	formatted1 := bizhours.FormatTime("09:00")
	formatted2 := bizhours.FormatTime("21:30")
	fmt.Println(formatted1)
	fmt.Println(formatted2)

	// Output:
	// 9:00 AM
	// 9:30 PM
}
