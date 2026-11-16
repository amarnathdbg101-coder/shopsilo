package goenv

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestLoader_ParseBytes(t *testing.T) {
	raw := []byte(`
# Server Configuration
PORT=8080
APP_NAME="ShopSilo API"
DEBUG=true # inline comment
export REDIS_HOST=localhost:6379

# Multi-line / Escaped
GREETING="Hello\nWorld"
SINGLE_QUOTED='literal_value'
`)

	err := ParseBytes(raw, true)
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	if os.Getenv("PORT") != "8080" {
		t.Fatalf("expected PORT=8080, got %s", os.Getenv("PORT"))
	}
	if os.Getenv("APP_NAME") != "ShopSilo API" {
		t.Fatalf("expected APP_NAME='ShopSilo API', got %s", os.Getenv("APP_NAME"))
	}
	if os.Getenv("DEBUG") != "true" {
		t.Fatalf("expected DEBUG='true', got %s", os.Getenv("DEBUG"))
	}
	if os.Getenv("REDIS_HOST") != "localhost:6379" {
		t.Fatalf("expected REDIS_HOST='localhost:6379', got %s", os.Getenv("REDIS_HOST"))
	}
	if os.Getenv("GREETING") != "Hello\nWorld" {
		t.Fatalf("expected unescaped newline in GREETING, got %q", os.Getenv("GREETING"))
	}
	if os.Getenv("SINGLE_QUOTED") != "literal_value" {
		t.Fatalf("expected SINGLE_QUOTED='literal_value', got %s", os.Getenv("SINGLE_QUOTED"))
	}
}

func TestEnv_TypeGetters(t *testing.T) {
	os.Setenv("TEST_STR", "hello")
	os.Setenv("TEST_INT", "42")
	os.Setenv("TEST_FLOAT", "99.95")
	os.Setenv("TEST_BOOL_1", "1")
	os.Setenv("TEST_BOOL_ON", "on")
	os.Setenv("TEST_BOOL_FALSE", "false")
	os.Setenv("TEST_DUR", "15m")
	os.Setenv("TEST_SLICE", "apple, banana , cherry")

	// Strings
	if GetString("TEST_STR", "def") != "hello" {
		t.Fatal("GetString mismatch")
	}
	if GetString("TEST_NON_EXISTENT", "def") != "def" {
		t.Fatal("GetString fallback failed")
	}

	// Ints
	if GetInt("TEST_INT", 0) != 42 {
		t.Fatal("GetInt mismatch")
	}
	if GetInt("TEST_NON_EXISTENT", 100) != 100 {
		t.Fatal("GetInt fallback failed")
	}

	// Float
	if GetFloat64("TEST_FLOAT", 0.0) != 99.95 {
		t.Fatal("GetFloat64 mismatch")
	}

	// Bools
	if !GetBool("TEST_BOOL_1", false) {
		t.Fatal("GetBool '1' should be true")
	}
	if !GetBool("TEST_BOOL_ON", false) {
		t.Fatal("GetBool 'on' should be true")
	}
	if GetBool("TEST_BOOL_FALSE", true) {
		t.Fatal("GetBool 'false' should be false")
	}

	// Duration
	if GetDuration("TEST_DUR", time.Minute) != 15*time.Minute {
		t.Fatal("GetDuration mismatch")
	}

	// Slice
	slice := GetSlice("TEST_SLICE", nil, ",")
	if len(slice) != 3 || slice[0] != "apple" || slice[1] != "banana" || slice[2] != "cherry" {
		t.Fatalf("GetSlice parsed incorrectly: %+v", slice)
	}
}

func TestEnv_MustGetters(t *testing.T) {
	os.Setenv("VALID_INT", "100")
	os.Setenv("INVALID_INT", "not_a_number")

	val, err := MustGetInt("VALID_INT")
	if err != nil || val != 100 {
		t.Fatalf("MustGetInt failed: %v", err)
	}

	_, err = MustGetInt("INVALID_INT")
	if err == nil || !errors.Is(err, ErrInvalidType) {
		t.Fatalf("expected ErrInvalidType, got %v", err)
	}

	_, err = MustGetString("MISSING_KEY_XYZ")
	if err == nil || !errors.Is(err, ErrMissingEnv) {
		t.Fatalf("expected ErrMissingEnv, got %v", err)
	}
}

func TestEnv_Validate(t *testing.T) {
	os.Setenv("REQ_A", "present_a")
	os.Setenv("REQ_B", "present_b")
	os.Unsetenv("REQ_C")
	os.Unsetenv("REQ_D")

	err := Validate("REQ_A", "REQ_B")
	if err != nil {
		t.Fatalf("Validate failed on present keys: %v", err)
	}

	err = Validate("REQ_A", "REQ_C", "REQ_D")
	if err == nil {
		t.Fatal("expected error on missing required keys")
	}

	if !errors.Is(err, ErrMissingEnv) {
		t.Fatalf("expected ErrMissingEnv unwrap, got %v", err)
	}
}

func TestEnv_DumpMasked(t *testing.T) {
	os.Setenv("DATABASE_DSN", "postgres://user:password@localhost:5432/db")
	os.Setenv("APP_PORT", "8080")
	os.Setenv("JWT_SECRET", "super-secret-key-123")

	dump := DumpMasked()
	if dump["DATABASE_DSN"] != "********" {
		t.Fatalf("DATABASE_DSN was not masked: %s", dump["DATABASE_DSN"])
	}
	if dump["JWT_SECRET"] != "********" {
		t.Fatalf("JWT_SECRET was not masked: %s", dump["JWT_SECRET"])
	}
	if dump["APP_PORT"] != "8080" {
		t.Fatalf("APP_PORT should not be masked: %s", dump["APP_PORT"])
	}
}
