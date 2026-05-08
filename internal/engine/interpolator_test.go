package engine_test

import (
	"testing"

	"github.com/F3rreir4L19/juridico-watcher/internal/engine"
	"github.com/stretchr/testify/assert"
)

func TestInterpolate_SubstituicaoSimples(t *testing.T) {
	vars := map[string]string{"nome": "Joao", "tipo": "procuracao"}
	result := engine.Interpolate("docs/{nome}/{tipo}.pdf", vars, nil)
	assert.Equal(t, "docs/Joao/procuracao.pdf", result)
}

func TestInterpolate_VariavelInexistente_FicaVazia(t *testing.T) {
	vars := map[string]string{"a": "1"}
	result := engine.Interpolate("{a}{b}", vars, setupLogger())
	assert.Equal(t, "1", result)
}

func TestInterpolate_SemPlaceholders(t *testing.T) {
	result := engine.Interpolate("texto puro", map[string]string{}, nil)
	assert.Equal(t, "texto puro", result)
}

func TestInterpolate_VariavelVazia_NaoConfundeComInexistente(t *testing.T) {
	vars := map[string]string{"a": ""}
	result := engine.Interpolate("x{a}y", vars, nil)
	assert.Equal(t, "xy", result)
}

func TestInterpolate_ChaveAberturaSemFechamento_TrataComoLiteral(t *testing.T) {
	result := engine.Interpolate("texto {sem fim", map[string]string{}, nil)
	assert.Equal(t, "texto {sem fim", result)
}

func TestInterpolate_OrdemDeterministica(t *testing.T) {
	// Variáveis com prefixo comum não devem causar problema independente da ordem
	vars := map[string]string{"a": "X", "ab": "Y"}
	for i := 0; i < 100; i++ {
		assert.Equal(t, "Yc", engine.Interpolate("{ab}c", vars, nil))
		assert.Equal(t, "Xc", engine.Interpolate("{a}c", vars, nil))
	}
}