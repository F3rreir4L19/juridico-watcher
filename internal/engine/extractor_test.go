package engine_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/engine"
	"github.com/stretchr/testify/assert"
)

func setupLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestExtractVariables_Simples(t *testing.T) {
	text := "Nome: Joao Silva\nPoder: Geral"
	extractions := []domain.Extraction{
		{VariableName: "nome", StartDelim: "Nome: ", EndDelim: "\n"},
		{VariableName: "poder", StartDelim: "Poder: ", EndDelim: ""},
	}
	vars := engine.ExtractVariables(text, extractions, nil)
	assert.Equal(t, "Joao Silva", vars["nome"])
	assert.Equal(t, "Geral", vars["poder"])
}

func TestExtractVariables_CaseInsensitive(t *testing.T) {
	text := "NOME: maria"
	extractions := []domain.Extraction{
		{VariableName: "nome", StartDelim: "nome: ", EndDelim: ""},
	}
	vars := engine.ExtractVariables(text, extractions, nil)
	assert.Equal(t, "maria", vars["nome"])
}

func TestExtractVariables_DelimitadorVazio(t *testing.T) {
	text := "procuração bastante texto final"
	extractions := []domain.Extraction{
		{VariableName: "tudo", StartDelim: "", EndDelim: ""},
		{VariableName: "inicio", StartDelim: "procuração ", EndDelim: ""},
		{VariableName: "fim", StartDelim: "texto ", EndDelim: ""},
	}
	vars := engine.ExtractVariables(text, extractions, nil)
	assert.Equal(t, text, vars["tudo"])
	assert.Equal(t, "bastante texto final", vars["inicio"])
	assert.Equal(t, "final", vars["fim"])
}

func TestExtractVariables_NaoEncontrado(t *testing.T) {
	text := "apenas um texto"
	extractions := []domain.Extraction{
		{VariableName: "var1", StartDelim: "inicio", EndDelim: "fim"},
	}
	vars := engine.ExtractVariables(text, extractions, setupLogger())
	assert.Equal(t, "", vars["var1"])
}

func TestExtractVariables_MultiplasOcorrenciasPegaPrimeira(t *testing.T) {
	text := "Nome: A\nNome: B"
	extractions := []domain.Extraction{
		{VariableName: "nome", StartDelim: "Nome: ", EndDelim: "\n"},
	}
	vars := engine.ExtractVariables(text, extractions, nil)
	assert.Equal(t, "A", vars["nome"])
}
