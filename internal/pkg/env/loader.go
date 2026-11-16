package goenv

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

// Load reads one or more .env files and sets environment variables if they are not already set.
func Load(filenames ...string) error {
	return loadFiles(false, filenames...)
}

// Overload reads .env files and overwrites existing environment variables.
func Overload(filenames ...string) error {
	return loadFiles(true, filenames...)
}

func loadFiles(override bool, filenames ...string) error {
	if len(filenames) == 0 {
		filenames = []string{".env"}
	}

	for _, filename := range filenames {
		f, err := os.Open(filename)
		if err != nil {
			if os.IsNotExist(err) {
				// Non-existent file is safely skipped unless required
				continue
			}
			return fmt.Errorf("env: open %s: %w", filename, err)
		}

		err = parseReader(f, override)
		f.Close()
		if err != nil {
			return fmt.Errorf("env: parse %s: %w", filename, err)
		}
	}

	return nil
}

// ParseBytes parses a raw byte slice containing .env content.
func ParseBytes(data []byte, override bool) error {
	return parseReader(bytes.NewReader(data), override)
}

func parseReader(r io.Reader, override bool) error {
	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and full line comments
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip 'export ' prefix if present
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		eqIdx := strings.Index(line, "=")
		if eqIdx <= 0 {
			continue // Invalid syntax or empty key ignored
		}

		key := strings.TrimSpace(line[:eqIdx])
		val := strings.TrimSpace(line[eqIdx+1:])

		// Check inline comments if not inside quotes
		val = parseValue(val)

		if !override {
			if _, exists := os.LookupEnv(key); exists {
				continue
			}
		}

		if err := os.Setenv(key, val); err != nil {
			return fmt.Errorf("setenv %s: %w", key, err)
		}
	}

	return scanner.Err()
}

func parseValue(val string) string {
	if len(val) == 0 {
		return ""
	}

	// Double quotes
	if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") && len(val) >= 2 {
		val = val[1 : len(val)-1]
		val = strings.ReplaceAll(val, `\n`, "\n")
		val = strings.ReplaceAll(val, `\t`, "\t")
		val = strings.ReplaceAll(val, `\"`, "\"")
		return val
	}

	// Single quotes (literal)
	if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") && len(val) >= 2 {
		return val[1 : len(val)-1]
	}

	// Unquoted: strip trailing inline comments
	if commentIdx := strings.Index(val, " #"); commentIdx != -1 {
		val = strings.TrimSpace(val[:commentIdx])
	}

	return val
}
