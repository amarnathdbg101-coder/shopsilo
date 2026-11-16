package deadlock

import (
	"fmt"
	"os"
	"time"
)

// Config controls deadlock and lock-contention detection behavior.
type Config struct {
	// Timeout specifies the maximum duration to wait for a lock before reporting a potential deadlock/starvation.
	// Default is 2 seconds. Set to 0 to disable timeout monitoring.
	Timeout time.Duration

	// OnPotentialDeadlock is invoked when an AB-BA lock acquisition cycle is detected.
	// If nil, it writes a detailed error stack trace to os.Stderr.
	OnPotentialDeadlock func(msg string)

	// OnTimeout is invoked when lock acquisition exceeds Timeout.
	// If nil, it writes a warning stack trace to os.Stderr.
	OnTimeout func(msg string)

	// Disabled disables lock order graph tracking for maximum raw performance.
	Disabled bool

	// MaxStackDepth limits the captured caller frame depth. Default is 15.
	MaxStackDepth int
}

var globalConfig = Config{
	Timeout:       2 * time.Second,
	MaxStackDepth: 15,
	OnPotentialDeadlock: func(msg string) {
		fmt.Fprintf(os.Stderr, "\n[DEADLOCK ALERT] %s\n", msg)
	},
	OnTimeout: func(msg string) {
		fmt.Fprintf(os.Stderr, "\n[LOCK TIMEOUT ALERT] %s\n", msg)
	},
}

// SetConfig updates the global deadlock detection configuration.
func SetConfig(c Config) {
	if c.MaxStackDepth <= 0 {
		c.MaxStackDepth = 15
	}
	globalConfig = c
}

// GetConfig returns a copy of the current global configuration.
func GetConfig() Config {
	return globalConfig
}
