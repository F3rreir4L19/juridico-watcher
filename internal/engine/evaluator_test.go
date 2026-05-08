package engine_test

import (
	"testing"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/engine"
	"github.com/stretchr/testify/assert"
)

func TestEvaluateConditions_AllTrue(t *testing.T) {
	vars := map[string]string{"a": "1", "b": "hello"}
	conds := []domain.Condition{
		{VariableName: "a", Operator: domain.OpEquals, Value: "1"},
		{VariableName: "b", Operator: domain.OpContains, Value: "ell"},
	}
	assert.True(t, engine.EvaluateConditions(conds, vars, nil))
}

func TestEvaluateConditions_OneFalse(t *testing.T) {
	vars := map[string]string{"a": "1"}
	conds := []domain.Condition{
		{VariableName: "a", Operator: domain.OpEquals, Value: "1"},
		{VariableName: "a", Operator: domain.OpEquals, Value: "2"},
	}
	assert.False(t, engine.EvaluateConditions(conds, vars, nil))
}

func TestEvaluateConditions_VarNotFound(t *testing.T) {
	vars := map[string]string{"a": "1"}
	conds := []domain.Condition{
		{VariableName: "missing", Operator: domain.OpEquals, Value: "x"},
	}
	assert.False(t, engine.EvaluateConditions(conds, vars, setupLogger()))
}

func TestEvaluateConditions_Empty(t *testing.T) {
	assert.True(t, engine.EvaluateConditions([]domain.Condition{}, map[string]string{}, nil))
}

func TestEvaluateConditions_Operators(t *testing.T) {
	vars := map[string]string{"x": "abc"}
	assert.True(t, engine.EvaluateConditions([]domain.Condition{
		{VariableName: "x", Operator: domain.OpEquals, Value: "abc"},
	}, vars, nil))
	assert.False(t, engine.EvaluateConditions([]domain.Condition{
		{VariableName: "x", Operator: domain.OpNotEquals, Value: "abc"},
	}, vars, nil))
	assert.True(t, engine.EvaluateConditions([]domain.Condition{
		{VariableName: "x", Operator: domain.OpContains, Value: "a"},
	}, vars, nil))
	assert.False(t, engine.EvaluateConditions([]domain.Condition{
		{VariableName: "x", Operator: domain.OpNotContain, Value: "a"},
	}, vars, nil))
}

func TestEvaluateConditions_CaseInsensitive(t *testing.T) {
	vars := map[string]string{"tipo": "Procuração"}
	// eq deve bater independente do case
	assert.True(t, engine.EvaluateConditions([]domain.Condition{
		{VariableName: "tipo", Operator: domain.OpEquals, Value: "procuração"},
	}, vars, nil))
	// neq deve ser false (porque eq é true)
	assert.False(t, engine.EvaluateConditions([]domain.Condition{
		{VariableName: "tipo", Operator: domain.OpNotEquals, Value: "procuração"},
	}, vars, nil))
	// contains deve achar
	assert.True(t, engine.EvaluateConditions([]domain.Condition{
		{VariableName: "tipo", Operator: domain.OpContains, Value: "PROCURA"},
	}, vars, nil))
}