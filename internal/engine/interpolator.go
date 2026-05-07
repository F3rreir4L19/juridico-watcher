package engine

import (
	"log/slog"
	"strings"
)

// Interpolate substitui placeholders {var} pelos respectivos valores no mapa.
// Se uma variável não existir, substitui por string vazia e gera um warning (se logger != nil).
func Interpolate(s string, vars map[string]string, logger *slog.Logger) string {
	result := s
	for name, value := range vars {
		placeholder := "{" + name + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	// Verificar se restaram placeholders não resolvidos
	// Ex: "{inexistente}"
	for {
		start := strings.Index(result, "{")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		varName := result[start+1 : start+end]
		if logger != nil {
			logger.Warn("placeholder não resolvido na interpolação", "var", varName)
		}
		// Remove o placeholder do resultado?
		// Decisão D-l: placeholder não resolvido retorna string vazia (não mantém literal).
		// Como o placeholder completo "{var}" não deve permanecer, substituímos por vazio.
		result = result[:start] + result[start+end+1:]
	}
	return result
}