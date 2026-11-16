# go-bizhours ⏰

Zero-Dependency Real-Time Business Operating Hours, Shift Calculator, and Customer Banner Engine for Go.

## 🚀 Key Features
- **Zero External Dependencies**: Pure Go standard library (`time`, `strings`, `strconv`).
- **Overnight Shift Support**: Accurately computes operational windows that cross midnight (e.g. 22:00 to 04:00 AM).
- **Weekly Off / Holiday Recognition**: Understands English ("Sunday") and common transliterations ("Ravivar", "Itwar").
- **Real-Time Banners**: Generates ready-to-display customer messages ("Open now • Closes at 9:00 PM", "Closing soon • 15 mins left", "Opens tomorrow at 9:00 AM").

## 📦 Quickstart

```go
package main

import (
	"fmt"

	"github.com/shopsilo/go-bizhours"
)

func main() {
	// Evaluate real-time store availability
	status := bizhours.Calculate(true, "09:00", "21:30", "Sunday")

	fmt.Printf("Is Open: %v\n", status.IsOpenNow)
	fmt.Printf("Banner: %s\n", status.StatusMessage)
	fmt.Printf("Next Change: %s\n", status.NextChangeAt)
}
```
