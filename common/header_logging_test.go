package common

import (
	"net/http"
	"os"
	"reflect"
	"testing"
)

func TestRedactHeaders_NoEnvVar(t *testing.T) {
	// Garantir que a variável não está setada
	os.Unsetenv("LOGGER_REDACT_HEADERS")
	
	// Resetar o cache para forçar re-parse
	resetRedactHeadersCache()
	
	headers := http.Header{
		"Authorization": []string{"Bearer token123"},
		"Content-Type":  []string{"application/json"},
	}
	
	result := RedactHeaders(headers)
	
	// Headers não devem ser redatados
	if result["Authorization"][0] != "Bearer token123" {
		t.Errorf("expected 'Bearer token123', got '%s'", result["Authorization"][0])
	}
	if result["Content-Type"][0] != "application/json" {
		t.Errorf("expected 'application/json', got '%s'", result["Content-Type"][0])
	}
}

func TestRedactHeaders_WithEnvVar(t *testing.T) {
	// Setar variável de ambiente
	os.Setenv("LOGGER_REDACT_HEADERS", "Authorization")
	defer os.Unsetenv("LOGGER_REDACT_HEADERS")
	
	// Resetar o cache para forçar re-parse
	resetRedactHeadersCache()
	
	headers := http.Header{
		"Authorization": []string{"Bearer token123"},
		"Content-Type":  []string{"application/json"},
	}
	
	result := RedactHeaders(headers)
	
	// Authorization deve ser redatado
	if result["Authorization"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result["Authorization"][0])
	}
	
	// Content-Type não deve ser redatado
	if result["Content-Type"][0] != "application/json" {
		t.Errorf("expected 'application/json', got '%s'", result["Content-Type"][0])
	}
}

func TestRedactHeaders_CaseInsensitive(t *testing.T) {
	// Setar variável com "Authorization" (maiúscula)
	os.Setenv("LOGGER_REDACT_HEADERS", "Authorization")
	defer os.Unsetenv("LOGGER_REDACT_HEADERS")
	
	// Resetar o cache
	resetRedactHeadersCache()
	
	// Testar com "authorization" (minúscula)
	headers := http.Header{
		"authorization": []string{"Bearer token123"},
	}
	
	result := RedactHeaders(headers)
	
	// Deve ser redatado mesmo com case diferente
	if result["authorization"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result["authorization"][0])
	}
	
	// Resetar novamente e testar o contrário
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
	// Setar múltiplos headers para redação
	os.Setenv("LOGGER_REDACT_HEADERS", "Authorization,X-API-Key")
	defer os.Unsetenv("LOGGER_REDACT_HEADERS")
	
	// Resetar o cache
	resetRedactHeadersCache()
	
	headers := http.Header{
		"Authorization": []string{"Bearer token"},
		"X-API-Key":     []string{"key123"},
		"Content-Type":  []string{"application/json"},
	}
	
	result := RedactHeaders(headers)
	
	// Authorization deve ser redatado
	if result["Authorization"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result["Authorization"][0])
	}
	
	// X-API-Key deve ser redatado
	if result["X-API-Key"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result["X-API-Key"][0])
	}
	
	// Content-Type não deve ser redatado
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
	
	// Header vazio também deve ser redatado
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
	
	// Todos os valores de Authorization devem ser redatados
	if len(result["Authorization"]) != 2 {
		t.Errorf("expected 2 values, got %d", len(result["Authorization"]))
	}
	if result["Authorization"][0] != "***" {
		t.Errorf("expected '***', got '%s'", result["Authorization"][0])
	}
	if result["Authorization"][1] != "***" {
		t.Errorf("expected '***', got '%s'", result["Authorization"][1])
	}
	
	// X-Custom não deve ser redatado
	if !reflect.DeepEqual(result["X-Custom"], []string{"value1", "value2"}) {
		t.Errorf("expected ['value1', 'value2'], got %v", result["X-Custom"])
	}
}

func TestRedactHeaders_WithSpaces(t *testing.T) {
	// Testar com espaços na lista
	os.Setenv("LOGGER_REDACT_HEADERS", " Authorization , X-API-Key ")
	defer os.Unsetenv("LOGGER_REDACT_HEADERS")
	
	// Resetar o cache
	resetRedactHeadersCache()
	
	headers := http.Header{
		"Authorization": []string{"Bearer token"},
		"X-API-Key":     []string{"key123"},
	}
	
	result := RedactHeaders(headers)
	
	// Ambos devem ser redatados mesmo com espaços na lista
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
