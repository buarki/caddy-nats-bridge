package common

import (
	"net/http"
	"os"
	"strings"
	"sync"
)

var (
	redactHeadersList []string
	redactHeadersOnce sync.Once
)

// getRedactHeadersList returns the list of headers that should be redacted
// based on the LOGGER_REDACT_HEADERS environment variable
func getRedactHeadersList() []string {
	redactHeadersOnce.Do(func() {
		envValue := os.Getenv("LOGGER_REDACT_HEADERS")
		if envValue == "" {
			redactHeadersList = []string{}
			return
		}

		// Parse comma-separated list
		parts := strings.Split(envValue, ",")
		redactHeadersList = make([]string, 0, len(parts))
		
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				// Normalize to lowercase for case-insensitive comparison
				redactHeadersList = append(redactHeadersList, strings.ToLower(trimmed))
			}
		}
	})
	
	return redactHeadersList
}

// shouldRedact verifica se um header deve ser redatado
func shouldRedact(headerName string) bool {
	headerLower := strings.ToLower(headerName)
	redactList := getRedactHeadersList()
	
	for _, redactHeader := range redactList {
		if headerLower == redactHeader {
			return true
		}
	}
	
	return false
}

// RedactHeaders returns a copy of headers with redacted values
// Headers in the LOGGER_REDACT_HEADERS list will have their values replaced with "***"
// The comparison is case-insensitive
func RedactHeaders(headers http.Header) map[string][]string {
	if headers == nil {
		return nil
	}

	result := make(map[string][]string, len(headers))
	
	for name, values := range headers {
		if shouldRedact(name) {
			// Redact all values for this header
			redactedValues := make([]string, len(values))
			for i := range values {
				redactedValues[i] = "***"
			}
			result[name] = redactedValues
		} else {
			// Keep original values
			result[name] = values
		}
	}
	
	return result
}

// resetRedactHeadersCache resets the header redaction cache
// This function is used only for testing
func resetRedactHeadersCache() {
	redactHeadersList = nil
	redactHeadersOnce = sync.Once{}
}
