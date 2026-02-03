package common

import (
	"net/http"
	"os"
	"reflect"
	"testing"
)

func TestRedactHeaders_NoEnvVar(t *testing.T) {
	// Ensure the variable is not set
	os.Unsetenv("LOGGER_REDACT_HEADERS")
	
	// Reset cache to force re-parse
	resetRedactHeadersCache()
	
	headers := http.Header{
		"Authorization": []string{"Bearer token123"},
		"Content-Type":  []string{"application/json"},
	}
	
	result := RedactHeaders(headers)
	
	// Headers should not be redacted
	if result["Authorization"][0] != "Bearer token123" {
		t.Errorf("expected 'Bearer token123', got '%s'", result["Authorization"][0])
	}
	if result["Content-Type"][0] != "application/json" {
		t.Errorf("expected 'application/json', got '%s'", result["Content-Type"][0])
	}
}

func TestRedactHeaders_WithEnvVar(t *testing.T) {
	// Set environment variable
	os.Setenv("LOGGER_REDACT_HEADERS", "Authorization")
	defer os.Unsetenv("LOGGER_REDACT_HEADERS")
	
	// Reset cache to force re-parse
	resetRedactHeadersCache()
	
	headers := http.Header{
		"Authorization": []string{"Bearer token123"},
		"Content-Type":  []string{"application/json"},
	}
	
	result := RedactHeaders(headers)
	
	// Authorization should be redacted
	if result["Authorization"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result["Authorization"][0])
	}
	
	// Content-Type should not be redacted
	if result["Content-Type"][0] != "application/json" {
		t.Errorf("expected 'application/json', got '%s'", result["Content-Type"][0])
	}
}

func TestRedactHeaders_CaseInsensitive(t *testing.T) {
	// Set variable with "Authorization" (uppercase)
	os.Setenv("LOGGER_REDACT_HEADERS", "Authorization")
	defer os.Unsetenv("LOGGER_REDACT_HEADERS")
	
	// Reset cache
	resetRedactHeadersCache()
	
	// Test with "authorization" (lowercase)
	headers := http.Header{
		"authorization": []string{"Bearer token123"},
	}
	
	result := RedactHeaders(headers)
	
	// Should be redacted even with different case
	if result["authorization"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result["authorization"][0])
	}
	
	// Reset again and test the opposite
	resetRedactHeadersCache()
	os.Setenv("LOGGER_REDACT_HEADERS", "authorization")
	
	headers2 := http.Header{
		"Authorization": []string{"Bearer token123"},
	}
	
	result2 := RedactHeaders(headers2)
	
	if result2["Authorization"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result2["Authorization"][0])
	}
}

func TestRedactHeaders_MultipleHeaders(t *testing.T) {
	// Set multiple headers for redaction
	os.Setenv("LOGGER_REDACT_HEADERS", "Authorization,X-API-Key")
	defer os.Unsetenv("LOGGER_REDACT_HEADERS")
	
	// Reset cache
	resetRedactHeadersCache()
	
	headers := http.Header{
		"Authorization": []string{"Bearer token"},
		"X-API-Key":     []string{"key123"},
		"Content-Type":  []string{"application/json"},
	}
	
	result := RedactHeaders(headers)
	
	// Authorization should be redacted
	if result["Authorization"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result["Authorization"][0])
	}
	
	// X-API-Key should be redacted
	if result["X-API-Key"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result["X-API-Key"][0])
	}
	
	// Content-Type should not be redacted
	if result["Content-Type"][0] != "application/json" {
		t.Errorf("expected 'application/json', got '%s'", result["Content-Type"][0])
	}
}

func TestRedactHeaders_EmptyValue(t *testing.T) {
	os.Setenv("LOGGER_REDACT_HEADERS", "Authorization")
	defer os.Unsetenv("LOGGER_REDACT_HEADERS")
	
	// Resetar o cache
	resetRedactHeadersCache()
	
	headers := http.Header{
		"Authorization": []string{""},
	}
	
	result := RedactHeaders(headers)
	
	// Empty header should also be redacted
	if result["Authorization"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result["Authorization"][0])
	}
}

func TestRedactHeaders_MultipleValues(t *testing.T) {
	os.Setenv("LOGGER_REDACT_HEADERS", "Authorization")
	defer os.Unsetenv("LOGGER_REDACT_HEADERS")
	
	// Resetar o cache
	resetRedactHeadersCache()
	
	headers := http.Header{
		"Authorization": []string{"Bearer token1", "Bearer token2"},
		"X-Custom":      []string{"value1", "value2"},
	}
	
	result := RedactHeaders(headers)
	
	// All Authorization values should be redacted
	if len(result["Authorization"]) != 2 {
		t.Errorf("expected 2 values, got %d", len(result["Authorization"]))
	}
	if result["Authorization"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result["Authorization"][0])
	}
	if result["Authorization"][1] != "***" {
		t.Errorf("expected '***', got '%s'", result["Authorization"][1])
	}
	
	// X-Custom should not be redacted
	if !reflect.DeepEqual(result["X-Custom"], []string{"value1", "value2"}) {
		t.Errorf("expected ['value1', 'value2'], got %v", result["X-Custom"])
	}
}

func TestRedactHeaders_WithSpaces(t *testing.T) {
	// Test with spaces in the list
	os.Setenv("LOGGER_REDACT_HEADERS", " Authorization , X-API-Key ")
	defer os.Unsetenv("LOGGER_REDACT_HEADERS")
	
	// Reset cache
	resetRedactHeadersCache()
	
	headers := http.Header{
		"Authorization": []string{"Bearer token"},
		"X-API-Key":     []string{"key123"},
	}
	
	result := RedactHeaders(headers)
	
	// Both should be redacted even with spaces in the list
	if result["Authorization"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result["Authorization"][0])
	}
	if result["X-API-Key"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result["X-API-Key"][0])
	}
}

func TestRedactHeaders_NilHeaders(t *testing.T) {
	result := RedactHeaders(nil)
	
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestRedactHeaders_EmptyHeaders(t *testing.T) {
	headers := http.Header{}
	result := RedactHeaders(headers)
	
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}
