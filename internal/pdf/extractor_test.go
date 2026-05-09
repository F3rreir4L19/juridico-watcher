package pdf_test

import (
	"os"
	"testing"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/pdf"
	"github.com/F3rreir4L19/juridico-watcher/test/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractText_PDFComTexto(t *testing.T) {
	dir := t.TempDir()
	// Sem acentos para evitar encoding da fonte padrão
	path := testhelpers.WritePDF(t, dir, "doc.pdf", "Nome: Joao Silva\nPoder: Geral")

	text, err := pdf.ExtractText(path)
	require.NoError(t, err)
	assert.Contains(t, text, "Nome: Joao Silva")
	assert.Contains(t, text, "Poder: Geral")
}

func TestExtractText_PDFVazio(t *testing.T) {
	dir := t.TempDir()
	path := testhelpers.WriteEmptyPDF(t, dir, "vazio.pdf")

	text, err := pdf.ExtractText(path)
	require.ErrorIs(t, err, domain.ErrNoText)
	assert.Empty(t, text)
}

func TestExtractText_PDFCorrompido(t *testing.T) {
	dir := t.TempDir()
	path := testhelpers.WriteCorruptPDF(t, dir, "corrupto.pdf")

	_, err := pdf.ExtractText(path)
	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrNoText)
}

func TestExtractText_ArquivoInexistente(t *testing.T) {
	_, err := pdf.ExtractText("/caminho/inexistente.pdf")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}
