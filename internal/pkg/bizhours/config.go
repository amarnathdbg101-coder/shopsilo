package bizhours

import "time"

// Option configures operating hours evaluation parameters.
type Option func(*Config)

// Config specifies schedule calculation rules and localization settings.
type Config struct {
	// ClosingSoonThreshold defines remaining minutes before switching to StatusClosingSoon (Default: 30 mins).
	ClosingSoonThreshold int

	// Location defines the timezone to interpret relative times (Default: local timezone).
	Location *time.Location
}

// DefaultConfig provides sensible business hour defaults.
func DefaultConfig() *Config {
	return &Config{
		ClosingSoonThreshold: 30,
		Location:             time.Local,
	}
}

// WithClosingSoonThreshold customizes when StatusClosingSoon triggers.
func WithClosingSoonThreshold(minutes int) Option {
	return func(c *Config) {
		if minutes > 0 {
			c.ClosingSoonThreshold = minutes
		}
	}
}

// WithLocation sets the timezone for time calculations.
func WithLocation(loc *time.Location) Option {
	return func(c *Config) {
		if loc != nil {
			c.Location = loc
		}
	}
}
