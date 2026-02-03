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

// getRedactHeadersList retorna a lista de headers que devem ser redatados
// baseado na variável de ambiente LOGGER_REDACT_HEADERS
func getRedactHeadersList() []string {
	redactHeadersOnce.Do(func() {
		envValue := os.Getenv("LOGGER_REDACT_HEADERS")
		if envValue == "" {
			redactHeadersList = []string{}
			return
		}

		// Parsear lista separada por vírgulas
		parts := strings.Split(envValue, ",")
		redactHeadersList = make([]string, 0, len(parts))
		
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				// Normalizar para lowercase para comparação case-insensitive
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

// RedactHeaders retorna uma cópia dos headers com valores redatados
// Headers na lista LOGGER_REDACT_HEADERS terão valores substituídos por "***"
// A comparação é case-insensitive
func RedactHeaders(headers http.Header) map[string][]string {
	if headers == nil {
		return nil
	}

	result := make(map[string][]string, len(headers))
	
	for name, values := range headers {
		if shouldRedact(name) {
			// Redatar todos os valores deste header
			redactedValues := make([]string, len(values))
			for i := range values {
				redactedValues[i] = "***"
			}
			result[name] = redactedValues
		} else {
			// Manter valores originais
			result[name] = values
		}
	}
	
	return result
}

// resetRedactHeadersCache reseta o cache de headers para redação
// Esta função é usada apenas para testes
func resetRedactHeadersCache() {
	redactHeadersList = nil
	redactHeadersOnce = sync.Once{}
}
