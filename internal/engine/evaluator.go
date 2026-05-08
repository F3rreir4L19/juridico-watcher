package engine

import (
	"log/slog"
	"strings"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
)

// EvaluateConditions retorna true se todas as condições forem satisfeitas (AND).
// Comparações são case-insensitive, consistentes com a RN-06 dos delimitadores.
// Se a variável referenciada na condição não foi extraída, a condição falha.
func EvaluateConditions(conditions []domain.Condition, vars map[string]string, logger *slog.Logger) bool {
	for _, cond := range conditions {
		actual, ok := vars[cond.VariableName]
		if !ok {
			if logger != nil {
				logger.Warn("condição referencia variável não extraída", "var", cond.VariableName)
			}
			return false
		}
		if !evaluateOne(cond.Operator, actual, cond.Value) {
			return false
		}
	}
	return true
}

func evaluateOne(op domain.Operator, actual, expected string) bool {
	a := strings.ToLower(actual)
	e := strings.ToLower(expected)
	switch op {
	case domain.OpEquals:
		return a == e
	case domain.OpNotEquals:
		return a != e
	case domain.OpContains:
		return strings.Contains(a, e)
	case domain.OpNotContain:
		return !strings.Contains(a, e)
	default:
		return false
	}
}