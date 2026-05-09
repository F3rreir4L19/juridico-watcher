package engine

import (
	"log/slog"
	"strings"
)

// Interpolate substitui placeholders {nomeVar} pelos valores em vars (RN-10).
// Variável inexistente vira string vazia + warning. Faz uma única passagem
// pela string para que o resultado não dependa da ordem de iteração do map.
func Interpolate(s string, vars map[string]string, logger *slog.Logger) string {
	var b strings.Builder
	b.Grow(len(s))

	i := 0
	for i < len(s) {
		if s[i] != '{' {
			b.WriteByte(s[i])
			i++
			continue
		}
		// Procura o '}' correspondente a partir de i+1
		rest := s[i+1:]
		closeIdx := strings.IndexByte(rest, '}')
		if closeIdx == -1 {
			// '{' sem '}' até o fim — preserva como literal
			b.WriteString(s[i:])
			break
		}
		varName := rest[:closeIdx]
		if value, ok := vars[varName]; ok {
			b.WriteString(value)
		} else if logger != nil {
			logger.Warn("placeholder não resolvido na interpolação", "var", varName)
		}
		// Avança para depois do '}'
		i += 1 + closeIdx + 1
	}
	return b.String()
}
