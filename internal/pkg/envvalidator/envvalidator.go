package envvalidator

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type varType int

const (
	typeString varType = iota
	typeInt
	typeBool
	typeDuration
	typeURL
)

// Rule defines validation constraints on an environment variable.
type Rule struct {
	key          string
	required     bool
	defaultVal   string
	expectedType varType
	minInt       *int
	maxInt       *int
	minLength    int
	oneOf        []string
	regexPattern *regexp.Regexp
	masked       bool
}

// AsInt specifies that the variable must parse as an integer.
func (r *Rule) AsInt() *Rule {
	r.expectedType = typeInt
	return r
}

// Between constrains integer value within [min, max] inclusive.
func (r *Rule) Between(min, max int) *Rule {
	r.minInt = &min
	r.maxInt = &max
	return r
}

// AsBool specifies that the variable must parse as a boolean (true/false/1/0/yes/no).
func (r *Rule) AsBool() *Rule {
	r.expectedType = typeBool
	return r
}

// AsDuration specifies that the variable must parse as time.Duration (e.g. "5s", "10m").
func (r *Rule) AsDuration() *Rule {
	r.expectedType = typeDuration
	return r
}

// AsURL specifies that the variable must parse as a valid URL with scheme and host.
func (r *Rule) AsURL() *Rule {
	r.expectedType = typeURL
	return r
}

// OneOf specifies that the string value must match one of the allowed options.
func (r *Rule) OneOf(allowed ...string) *Rule {
	r.oneOf = allowed
	return r
}

// Regex requires the value to match the given regular expression pattern.
func (r *Rule) Regex(pattern string) *Rule {
	r.regexPattern = regexp.MustCompile(pattern)
	return r
}

// MinLength enforces minimum character count.
func (r *Rule) MinLength(length int) *Rule {
	r.minLength = length
	return r
}

// Masked marks the value as sensitive (will be redacted in dumps).
func (r *Rule) Masked() *Rule {
	r.masked = true
	return r
}

// Schema aggregates validation rules.
type Schema struct {
	rules []*Rule
}

// New creates an empty Schema.
func New() *Schema {
	return &Schema{}
}

// Require declares a mandatory environment variable.
func (s *Schema) Require(key string) *Rule {
	r := &Rule{key: key, required: true}
	s.rules = append(s.rules, r)
	return r
}

// Optional declares an optional variable with a fallback default.
func (s *Schema) Optional(key, defaultVal string) *Rule {
	r := &Rule{key: key, required: false, defaultVal: defaultVal}
	s.rules = append(s.rules, r)
	return r
}

// Environment holds validated parsed variables.
type Environment struct {
	strings   map[string]string
	ints      map[string]int
	bools     map[string]bool
	durations map[string]time.Duration
	masked    map[string]bool
}

// Validate processes environment variables against the schema.
// If lookup is nil, os.Getenv is used.
func (s *Schema) Validate(lookup func(string) string) (*Environment, error) {
	if lookup == nil {
		lookup = os.Getenv
	}

	env := &Environment{
		strings:   make(map[string]string),
		ints:      make(map[string]int),
		bools:     make(map[string]bool),
		durations: make(map[string]time.Duration),
		masked:    make(map[string]bool),
	}

	var errMsgs []string

	for _, r := range s.rules {
		val := strings.TrimSpace(lookup(r.key))
		if val == "" {
			if r.required {
				errMsgs = append(errMsgs, fmt.Sprintf("missing required variable: %s", r.key))
				continue
			}
			val = r.defaultVal
		}

		env.masked[r.key] = r.masked

		// Length check
		if r.minLength > 0 && len(val) < r.minLength {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: length %d is less than required minimum %d", r.key, len(val), r.minLength))
			continue
		}

		// Regex check
		if r.regexPattern != nil && !r.regexPattern.MatchString(val) {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: value '%s' does not match pattern '%s'", r.key, val, r.regexPattern.String()))
			continue
		}

		// OneOf check
		if len(r.oneOf) > 0 {
			matched := false
			for _, allowed := range r.oneOf {
				if val == allowed {
					matched = true
					break
				}
			}
			if !matched {
				errMsgs = append(errMsgs, fmt.Sprintf("%s: value '%s' is not in allowed list [%s]", r.key, val, strings.Join(r.oneOf, ", ")))
				continue
			}
		}

		// Type check & cast
		switch r.expectedType {
		case typeString:
			env.strings[r.key] = val

		case typeInt:
			n, err := strconv.Atoi(val)
			if err != nil {
				errMsgs = append(errMsgs, fmt.Sprintf("%s: '%s' is not a valid integer", r.key, val))
				continue
			}
			if r.minInt != nil && n < *r.minInt {
				errMsgs = append(errMsgs, fmt.Sprintf("%s: %d is less than minimum %d", r.key, n, *r.minInt))
				continue
			}
			if r.maxInt != nil && n > *r.maxInt {
				errMsgs = append(errMsgs, fmt.Sprintf("%s: %d is greater than maximum %d", r.key, n, *r.maxInt))
				continue
			}
			env.ints[r.key] = n
			env.strings[r.key] = val

		case typeBool:
			b, err := parseBool(val)
			if err != nil {
				errMsgs = append(errMsgs, fmt.Sprintf("%s: '%s' is not a valid boolean (true/false/1/0/yes/no)", r.key, val))
				continue
			}
			env.bools[r.key] = b
			env.strings[r.key] = val

		case typeDuration:
			d, err := time.ParseDuration(val)
			if err != nil {
				errMsgs = append(errMsgs, fmt.Sprintf("%s: '%s' is not a valid duration (e.g., '10s', '1h')", r.key, val))
				continue
			}
			env.durations[r.key] = d
			env.strings[r.key] = val

		case typeURL:
			u, err := url.ParseRequestURI(val)
			if err != nil || u.Scheme == "" || u.Host == "" {
				errMsgs = append(errMsgs, fmt.Sprintf("%s: '%s' is not a valid URL (must include scheme and host)", r.key, val))
				continue
			}
			env.strings[r.key] = val
		}
	}

	if len(errMsgs) > 0 {
		return nil, fmt.Errorf("%w:\n  - %s", ErrValidationFailed, strings.Join(errMsgs, "\n  - "))
	}

	return env, nil
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "t", "yes", "y", "on":
		return true, nil
	case "false", "0", "f", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool: %s", s)
	}
}

// GetString returns the string value for key.
func (e *Environment) GetString(key string) string {
	return e.strings[key]
}

// GetInt returns the parsed integer value for key.
func (e *Environment) GetInt(key string) int {
	return e.ints[key]
}

// GetBool returns the parsed boolean value for key.
func (e *Environment) GetBool(key string) bool {
	return e.bools[key]
}

// GetDuration returns the parsed time.Duration value for key.
func (e *Environment) GetDuration(key string) time.Duration {
	return e.durations[key]
}

// Dump returns a formatted string table of all variables with secrets redacted.
func (e *Environment) Dump() string {
	var sb strings.Builder
	sb.WriteString("Loaded Environment Configuration:\n")
	for k, v := range e.strings {
		display := v
		if e.masked[k] {
			display = "****** [REDACTED]"
		}
		sb.WriteString(fmt.Sprintf("  %s = %s\n", k, display))
	}
	return sb.String()
}
