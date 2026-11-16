package envvalidator

import (
	"strings"
	"testing"
	"time"
)

func TestEnvValidator_Valid(t *testing.T) {
	mockEnv := map[string]string{
		"PORT":         "8080",
		"APP_ENV":      "production",
		"DEBUG":        "yes",
		"TIMEOUT":      "5s",
		"DATABASE_URL": "postgres://user:pass@localhost:5432/mydb",
		"API_KEY":      "secret_api_key_1234567890abcdef",
		"DB_PASSWORD":  "super_secret_password",
	}

	schema := New()
	schema.Require("PORT").AsInt().Between(1024, 65535)
	schema.Require("APP_ENV").OneOf("development", "staging", "production")
	schema.Require("DEBUG").AsBool()
	schema.Require("TIMEOUT").AsDuration()
	schema.Require("DATABASE_URL").AsURL()
	schema.Require("API_KEY").MinLength(20)
	schema.Require("DB_PASSWORD").Masked()
	schema.Optional("WORKERS", "4").AsInt()

	env, err := schema.Validate(func(k string) string { return mockEnv[k] })
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if env.GetInt("PORT") != 8080 {
		t.Fatalf("expected PORT 8080, got %d", env.GetInt("PORT"))
	}
	if env.GetString("APP_ENV") != "production" {
		t.Fatalf("expected APP_ENV production")
	}
	if !env.GetBool("DEBUG") {
		t.Fatalf("expected DEBUG true")
	}
	if env.GetDuration("TIMEOUT") != 5*time.Second {
		t.Fatalf("expected TIMEOUT 5s")
	}
	if env.GetInt("WORKERS") != 4 {
		t.Fatalf("expected fallback WORKERS 4")
	}

	// Verify masking
	dump := env.Dump()
	if strings.Contains(dump, "super_secret_password") {
		t.Fatalf("dump leaked masked password: %s", dump)
	}
	if !strings.Contains(dump, "REDACTED") {
		t.Fatalf("dump missing REDACTED indicator")
	}
}

func TestEnvValidator_MultipleFailuresReported(t *testing.T) {
	mockEnv := map[string]string{
		"PORT":    "99999", // out of bounds
		"APP_ENV": "unknown_env", // not in allowed
		"TIMEOUT": "invalid_duration",
		// MISSING_REQUIRED omitted
	}

	schema := New()
	schema.Require("MISSING_REQUIRED")
	schema.Require("PORT").AsInt().Between(1024, 65535)
	schema.Require("APP_ENV").OneOf("dev", "prod")
	schema.Require("TIMEOUT").AsDuration()

	_, err := schema.Validate(func(k string) string { return mockEnv[k] })
	if err == nil {
		t.Fatalf("expected validation error")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "missing required variable: MISSING_REQUIRED") {
		t.Errorf("missing MISSING_REQUIRED in error: %s", errStr)
	}
	if !strings.Contains(errStr, "PORT: 99999 is greater than maximum 65535") {
		t.Errorf("missing PORT bound in error: %s", errStr)
	}
	if !strings.Contains(errStr, "APP_ENV: value 'unknown_env' is not in allowed list") {
		t.Errorf("missing APP_ENV in error: %s", errStr)
	}
	if !strings.Contains(errStr, "TIMEOUT: 'invalid_duration' is not a valid duration") {
		t.Errorf("missing TIMEOUT in error: %s", errStr)
	}
}
