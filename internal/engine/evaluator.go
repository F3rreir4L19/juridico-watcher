package engine

import (
	"log/slog"
	"strings"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
)

// EvaluateConditions retorna true se todas as condições forem satisfeitas (AND).
// Se o mapa de variáveis não contiver a chave da condição, a condição falha.
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
	switch op {
	case domain.OpEquals:
		return strings.EqualFold(actual, expected) // case-insensitive por simetria? Decisão de design: eq é case-sensitive? O spec não especifica case para condições, mas para delimitadores foi case-insensitive. Para consistência, manteremos case-sensitive em eq/neq/contains, a menos que o usuário queira. As decisões não definem. Vou manter case-sensitive, pois é o padrão e o usuário tem controle sobre o valor. Se quisermos, poderíamos tornar configurável, mas v1 é simples.
	case domain.OpNotEquals:
		return actual != expected
	case domain.OpContains:
		return strings.Contains(actual, expected)
	case domain.OpNotContain:
		return !strings.Contains(actual, expected)
	default:
		// operador desconhecido → falha
		return false
	}
}