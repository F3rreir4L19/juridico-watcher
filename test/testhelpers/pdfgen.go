package testhelpers

import (
	"path/filepath"
	"testing"

	"github.com/jung-kurt/gofpdf"
	"github.com/stretchr/testify/require"
)

// WritePDF cria um PDF com o texto fornecido em uma única página.
// Retorna o caminho absoluto do arquivo criado.
//
// O texto é escrito em fonte padrão Arial 12. Quebras de linha (\n) no
// input geram quebras visuais no PDF, MAS NÃO são preservadas como '\n'
// na extração via ledongthuc/pdf — viram espaços. Veja CLAUDE.md §7.4.1.
func WritePDF(t *testing.T, dir, filename, textContent string) string {
	t.Helper()
	path := filepath.Join(dir, filename)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 12)
	// MultiCell escreve com quebra de linha automática e respeita \n
	pdf.MultiCell(0, 6, textContent, "", "L", false)

	err := pdf.OutputFileAndClose(path)
	require.NoError(t, err, "falha ao gerar PDF de teste")
	return path
}

// WriteEmptyPDF cria um PDF vazio (sem texto) — útil para testar
// o caso de PDF sem camada de texto extraível.
func WriteEmptyPDF(t *testing.T, dir, filename string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	// nenhum texto adicionado
	err := pdf.OutputFileAndClose(path)
	require.NoError(t, err)
	return path
}

// WriteCorruptPDF cria um arquivo .pdf com bytes inválidos —
// útil para testar tratamento de erro de extração.
func WriteCorruptPDF(t *testing.T, dir, filename string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	err := writeBytes(path, []byte("isso não é um PDF válido"))
	require.NoError(t, err)
	return path
}
