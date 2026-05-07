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

func TestInterpolate_VariavelInexistente(t *testing.T) {
	vars := map[string]string{"a": "1"}
	logger := setupLogger()
	result := engine.Interpolate("{a}{b}", vars, logger)
	assert.Equal(t, "1", result)
	// O logger emitiria um warning, mas não afeta o resultado.
	// Para verificar se o logger foi chamado, poderíamos usar um buffer,
	// mas para teste unitário confiamos no comportamento de remover o placeholder.
}

func TestInterpolate_SemPlaceholders(t *testing.T) {
	result := engine.Interpolate("texto puro", map[string]string{}, nil)
	assert.Equal(t, "texto puro", result)
}