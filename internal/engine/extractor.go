package engine

import (
	"log/slog"
	"strings"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
)

// ExtractVariables aplica as extrações definidas na regra ao texto completo do documento.
// Retorna um mapa com todas as variáveis (nome da variável -> valor).
// Variáveis cujos delimitadores não forem encontrados recebem string vazia e geram um
// warning no logger (se logger != nil).
func ExtractVariables(text string, extractions []domain.Extraction, logger *slog.Logger) map[string]string {
	result := make(map[string]string, len(extractions))
	textLower := strings.ToLower(text)

	for _, ext := range extractions {
		value := ""
		startDelim := strings.ToLower(ext.StartDelim)
		endDelim := strings.ToLower(ext.EndDelim)

		// Encontrar posição inicial
		startIdx := 0
		if startDelim != "" {
			pos := strings.Index(textLower, startDelim)
			if pos == -1 {
				if logger != nil {
					logger.Warn("delimitador de início não encontrado",
						"var", ext.VariableName,
						"start_delim", ext.StartDelim)
				}
				// variável não encontrada, mantém vazia
				result[ext.VariableName] = ""
				continue
			}
			startIdx = pos + len(startDelim)
		}

		// Subtexto a partir do delimitador de início
		remaining := text[startIdx:]
		remainingLower := textLower[startIdx:]

		if endDelim == "" {
			// Delimitador de fim vazio → até o final do texto
			value = remaining
		} else {
			endPos := strings.Index(remainingLower, endDelim)
			if endPos == -1 {
				if logger != nil {
					logger.Warn("delimitador de fim não encontrado",
						"var", ext.VariableName,
						"end_delim", ext.EndDelim)
				}
				value = "" // não encontrado, vazio
			} else {
				value = remaining[:endPos]
			}
		}

		// Remove espaços extras que possam ter sobrado
		value = strings.TrimSpace(value)
		result[ext.VariableName] = value
	}

	return result
}