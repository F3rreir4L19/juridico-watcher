package engine_test

import (
	"path/filepath"
	"testing"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/engine"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	"github.com/F3rreir4L19/juridico-watcher/test/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recorderAdapter adapta o repositório real para a interface esperada pelo pipeline.
type recorderAdapter struct {
	repo *storage.ProcessedRepo
}

func (a *recorderAdapter) HasBeenProcessed(hash string, ruleID int64) (bool, error) {
	return a.repo.HasBeenProcessed(hash, ruleID)
}

func (a *recorderAdapter) Record(doc *domain.ProcessedDoc) error {
	return a.repo.Record(doc)
}

func TestPipeline_SucessoSimples(t *testing.T) {
	dir := t.TempDir()
	pdfPath := testhelpers.WritePDF(t, dir, "doc.pdf",
		"Nome: Joao\nTipo: Procuração")

	db := testhelpers.TempDB(t)
	wr := storage.NewWatchRepo(db)
	rr := storage.NewRuleRepo(db)
	pr := storage.NewProcessedRepo(db)

	// Cria watch e regra
	w := &domain.Watch{Name: "w", Path: dir, Active: true}
	require.NoError(t, wr.Create(w))
	rule := &domain.Rule{
		Name:      "Procuração",
		Priority:  1,
		Active:    true,
		WatchIDs:  []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Nome: ", EndDelim: "\n"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "{nome}", Order: 1},
			{Type: domain.ActionMove, Target: "{nome}", Order: 2},
		},
	}
	require.NoError(t, rr.Create(rule))

	recorder := &recorderAdapter{repo: pr}
	results, err := engine.ProcessPDF(pdfPath, []*domain.Rule{rule}, recorder, dir, nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)

	// Verifica que a pasta foi criada e o arquivo movido
	assert.DirExists(t, filepath.Join(dir, "Joao"))
	assert.NoFileExists(t, pdfPath)
	assert.FileExists(t, filepath.Join(dir, "Joao", "doc.pdf"))
}

func TestPipeline_NoMatch(t *testing.T) {
	dir := t.TempDir()
	pdfPath := testhelpers.WritePDF(t, dir, "doc.pdf", "Conteudo irrelevante")

	db := testhelpers.TempDB(t)
	wr := storage.NewWatchRepo(db)
	rr := storage.NewRuleRepo(db)
	pr := storage.NewProcessedRepo(db)

	w := &domain.Watch{Name: "w", Path: dir, Active: true}
	require.NoError(t, wr.Create(w))
	rule := &domain.Rule{
		Name:     "Exige nome",
		Priority: 1,
		Active:   true,
		WatchIDs: []int64{w.ID},
		Conditions: []domain.Condition{
			{VariableName: "nome", Operator: domain.OpEquals, Value: "Joao"},
		},
	}
	require.NoError(t, rr.Create(rule))

	results, err := engine.ProcessPDF(pdfPath, []*domain.Rule{rule}, &recorderAdapter{pr}, dir, nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNoMatch, results[0].Status)
}

func TestPipeline_PDFSEMTexto(t *testing.T) {
	dir := t.TempDir()
	pdfPath := testhelpers.WriteEmptyPDF(t, dir, "empty.pdf")

	db := testhelpers.TempDB(t)
	wr := storage.NewWatchRepo(db)
	rr := storage.NewRuleRepo(db)
	pr := storage.NewProcessedRepo(db)

	w := &domain.Watch{Name: "w", Path: dir, Active: true}
	require.NoError(t, wr.Create(w))
	rule := &domain.Rule{
		Name:     "R1", Priority: 1, Active: true, WatchIDs: []int64{w.ID},
	}
	require.NoError(t, rr.Create(rule))

	results, err := engine.ProcessPDF(pdfPath, []*domain.Rule{rule}, &recorderAdapter{pr}, dir, nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNoText, results[0].Status)
}

func TestPipeline_MovePulaRegrasSeguintes(t *testing.T) {
	dir := t.TempDir()
	pdfPath := testhelpers.WritePDF(t, dir, "doc.pdf", "Nome: Maria\n")

	db := testhelpers.TempDB(t)
	wr := storage.NewWatchRepo(db)
	rr := storage.NewRuleRepo(db)
	pr := storage.NewProcessedRepo(db)

	w := &domain.Watch{Name: "w", Path: dir, Active: true}
	require.NoError(t, wr.Create(w))

	rule1 := &domain.Rule{
		Name:     "Move",
		Priority: 1,
		Active:   true,
		WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Nome: ", EndDelim: "\n"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionMove, Target: "{nome}", Order: 1},
		},
	}
	rule2 := &domain.Rule{
		Name:     "DeveriaPular",
		Priority: 2,
		Active:   true,
		WatchIDs: []int64{w.ID},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "inutil"},
		},
	}
	require.NoError(t, rr.Create(rule1))
	require.NoError(t, rr.Create(rule2))

	results, err := engine.ProcessPDF(pdfPath, []*domain.Rule{rule1, rule2}, &recorderAdapter{pr}, dir, nil)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Equal(t, domain.StatusSkippedMoved, results[1].Status)
	assert.NoDirExists(t, filepath.Join(dir, "inutil"))
}

func TestPipeline_DeduplicacaoPorHash(t *testing.T) {
	dir := t.TempDir()
	pdfPath := testhelpers.WritePDF(t, dir, "doc.pdf", "Nome: Joao\n")

	db := testhelpers.TempDB(t)
	wr := storage.NewWatchRepo(db)
	rr := storage.NewRuleRepo(db)
	pr := storage.NewProcessedRepo(db)

	w := &domain.Watch{Name: "w", Path: dir, Active: true}
	require.NoError(t, wr.Create(w))
	rule := &domain.Rule{
		Name: "r", Priority: 1, Active: true, WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Nome: ", EndDelim: "\n"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "{nome}"},
		},
	}
	require.NoError(t, rr.Create(rule))

	rec := &recorderAdapter{repo: pr}

	// Primeira execução: cria pasta e registra
	_, err := engine.ProcessPDF(pdfPath, []*domain.Rule{rule}, rec, dir, nil)
	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(dir, "Joao"))

	// Limpa a pasta criada para verificar que segunda execução NÃO recria
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "Joao")))

	// Segunda execução: deduplicação deve impedir nova execução
	results, err := engine.ProcessPDF(pdfPath, []*domain.Rule{rule}, rec, dir, nil)
	require.NoError(t, err)
	assert.Empty(t, results) // pulada por dedup
	assert.NoDirExists(t, filepath.Join(dir, "Joao"))
}