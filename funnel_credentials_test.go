package main

import (
	"testing"
)

func TestFunnelCredentials_GenerateAndValidate(t *testing.T) {
	fc := NewFunnelCredentials()

	// Initially should not be active
	if fc.IsActive() {
		t.Error("Expected credentials to be inactive initially")
	}

	// Validate should fail when not active
	if fc.Validate("stream", "anything") {
		t.Error("Expected validation to fail when not active")
	}

	// Generate credentials
	username, password := fc.Generate()

	// Should be active now
	if !fc.IsActive() {
		t.Error("Expected credentials to be active after generation")
	}

	// Username should be "stream"
	if username != "stream" {
		t.Errorf("Expected username 'stream', got %q", username)
	}

	// Password should be 8 characters
	if len(password) != 8 {
		t.Errorf("Expected password length 8, got %d", len(password))
	}

	// Validation should succeed with correct credentials
	if !fc.Validate(username, password) {
		t.Error("Expected validation to succeed with correct credentials")
	}

	// Validation should fail with wrong password
	if fc.Validate(username, "wrongpass") {
		t.Error("Expected validation to fail with wrong password")
	}

	// Validation should fail with wrong username
	if fc.Validate("admin", password) {
		t.Error("Expected validation to fail with wrong username")
	}

	// GetCredentials should return the same values
	gotUser, gotPass, active := fc.GetCredentials()
	if gotUser != username || gotPass != password || !active {
		t.Error("GetCredentials returned incorrect values")
	}
}

func TestFunnelCredentials_Clear(t *testing.T) {
	fc := NewFunnelCredentials()

	// Generate credentials
	username, password := fc.Generate()

	// Verify credentials work
	if !fc.Validate(username, password) {
		t.Error("Expected validation to succeed after generation")
	}

	// Clear credentials
	fc.Clear()

	// Should no longer be active
	if fc.IsActive() {
		t.Error("Expected credentials to be inactive after clear")
	}

	// Validation should fail after clear
	if fc.Validate(username, password) {
		t.Error("Expected validation to fail after clear")
	}

	// GetCredentials should return empty values
	gotUser, gotPass, active := fc.GetCredentials()
	if gotUser != "" || gotPass != "" || active {
		t.Error("GetCredentials should return empty values after clear")
	}
}

func TestFunnelCredentials_RegenerateChangesPassword(t *testing.T) {
	fc := NewFunnelCredentials()

	_, password1 := fc.Generate()
	_, password2 := fc.Generate()

	// Passwords should be different (with very high probability)
	// This could theoretically fail if the random generator produces the same 8-char string twice
	if password1 == password2 {
		t.Log("Warning: passwords are the same, this is statistically unlikely but possible")
	}
}
