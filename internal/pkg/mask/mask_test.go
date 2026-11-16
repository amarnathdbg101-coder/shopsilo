package gomask

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMask_Aadhaar(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"5482 9182 3491", "XXXX-XXXX-3491"},
		{"548291823491", "XXXX-XXXX-3491"},
		{"5482-9182-3491", "XXXX-XXXX-3491"},
	}

	for _, tt := range tests {
		got := Aadhaar(tt.input)
		if got != tt.expected {
			t.Errorf("Aadhaar(%q) = %q; expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestMask_PAN(t *testing.T) {
	input := "abcde1234f"
	expected := "XXXXX1234X"
	got := PAN(input)
	if got != expected {
		t.Fatalf("PAN(%q) = %q; expected %q", input, got, expected)
	}
}

func TestMask_CreditCard(t *testing.T) {
	input := "4111 2222 3333 4444"
	expected := "4111-XXXX-XXXX-4444"
	got := CreditCard(input)
	if got != expected {
		t.Fatalf("CreditCard(%q) = %q; expected %q", input, got, expected)
	}
}

func TestMask_Email(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"aditya@example.com", "a****a@example.com"},
		{"al@example.com", "a*@example.com"},
		{"k@example.com", "k*@example.com"},
	}

	for _, tt := range tests {
		got := Email(tt.input)
		if got != tt.expected {
			t.Errorf("Email(%q) = %q; expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestMask_Phone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+919876543210", "+91-98****3210"},
		{"9876543210", "98****3210"},
		{"+91 98765 43210", "+91-98****3210"},
	}

	for _, tt := range tests {
		got := Phone(tt.input)
		if got != tt.expected {
			t.Errorf("Phone(%q) = %q; expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestMask_JSONRecursively(t *testing.T) {
	raw := []byte(`{
		"user": {
			"name": "Aman",
			"password": "SuperSecretPassword123",
			"nested": {
				"token": "bearer-jwt-token-999"
			}
		},
		"card_number": "4111222233334444"
	}`)

	masked, err := JSONRecursively(raw)
	if err != nil {
		t.Fatalf("JSONRecursively failed: %v", err)
	}

	var parsed map[string]any
	_ = json.Unmarshal(masked, &parsed)

	cardVal := parsed["card_number"].(string)
	if cardVal != "********" {
		t.Fatalf("expected card_number to be masked, got %s", cardVal)
	}

	userMap := parsed["user"].(map[string]any)
	if userMap["password"].(string) != "********" {
		t.Fatal("password was not masked")
	}

	nested := userMap["nested"].(map[string]any)
	if nested["token"].(string) != "********" {
		t.Fatal("nested token was not masked")
	}

	if userMap["name"].(string) != "Aman" {
		t.Fatal("non-sensitive field name should remain untouched")
	}
}

func TestMaskString_Short(t *testing.T) {
	got := MaskString("AB", 1, 1, '*')
	if !strings.Contains(got, "*") {
		t.Fatalf("short string should be masked: %s", got)
	}
}
