package integration_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/engine"
	"github.com/F3rreir4L19/juridico-watcher/internal/service"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	"github.com/F3rreir4L19/juridico-watcher/test/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRuleLifecycle_CRUDCompletoComFilhos valida que Extractions, Conditions,
// Actions e WatchIDs sobrevivem a Create → GetByID → Update → GetByID → Delete.
func TestRuleLifecycle_CRUDCompletoComFilhos(t *testing.T) {
	db := testhelpers.TempDB(t)
	wsvc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))
	rsvc := service.NewRuleService(storage.NewRuleRepo(db))

	// Pré-condição: watch existe
	w := &domain.Watch{Name: "w", Path: t.TempDir(), Active: true}
	require.NoError(t, wsvc.Create(w))

	// CREATE com filhos
	rule := &domain.Rule{
		Name:     "Procuracao Outorgante",
		Priority: 10,
		Active:   true,
		WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Outorgante: ", EndDelim: " RG"},
			{VariableName: "rg", StartDelim: "RG numero ", EndDelim: " Outorgado:"},
		},
		Conditions: []domain.Condition{
			{VariableName: "nome", Operator: domain.OpNotEquals, Value: ""},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "{nome}"},
			{Type: domain.ActionMove, Target: "{nome}"},
		},
	}
	require.NoError(t, rsvc.Create(rule))
	assert.NotZero(t, rule.ID)

	// READ — verifica que os filhos voltam corretamente
	loaded, err := rsvc.GetByID(rule.ID)
	require.NoError(t, err)
	require.Len(t, loaded.Extractions, 2)
	require.Len(t, loaded.Conditions, 1)
	require.Len(t, loaded.Actions, 2)
	require.Len(t, loaded.WatchIDs, 1)
	assert.Equal(t, "nome", loaded.Extractions[0].VariableName)
	assert.Equal(t, domain.OpNotEquals, loaded.Conditions[0].Operator)

	// UPDATE — substitui filhos
	loaded.Extractions = []domain.Extraction{
		{VariableName: "tipo", StartDelim: "PROCURACAO ", EndDelim: " Outorgante:"},
	}
	loaded.Actions = []domain.Action{
		{Type: domain.ActionRename, Target: "{tipo}"},
	}
	require.NoError(t, rsvc.Update(loaded))

	reloaded, err := rsvc.GetByID(rule.ID)
	require.NoError(t, err)
	require.Len(t, reloaded.Extractions, 1)
	assert.Equal(t, "tipo", reloaded.Extractions[0].VariableName)
	require.Len(t, reloaded.Actions, 1)
	assert.Equal(t, domain.ActionRename, reloaded.Actions[0].Type)

	// DELETE
	require.NoError(t, rsvc.Delete(rule.ID))
	_, err = rsvc.GetByID(rule.ID)
	assert.True(t, errors.Is(err, domain.ErrNotFound))
}

// TestRuleLifecycle_DeleteRegra_CascataLimpaFilhos valida que ON DELETE CASCADE
// no schema realmente apaga extractions/conditions/actions/rule_watches órfãos.
// Esse teste protege contra regressão futura (alguém remover o CASCADE no SQL).
func TestRuleLifecycle_DeleteRegra_CascataLimpaFilhos(t *testing.T) {
	db := testhelpers.TempDB(t)
	wsvc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))
	rsvc := service.NewRuleService(storage.NewRuleRepo(db))

	w := &domain.Watch{Name: "w", Path: t.TempDir(), Active: true}
	require.NoError(t, wsvc.Create(w))

	rule := &domain.Rule{
		Name: "r", Active: true, WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "v", StartDelim: "a", EndDelim: "b"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "x"},
		},
	}
	require.NoError(t, rsvc.Create(rule))
	require.NoError(t, rsvc.Delete(rule.ID))

	// Após deletar a regra, deletar o watch deve funcionar (não há mais
	// entradas em rule_watches apontando pra ele).
	require.NoError(t, wsvc.Delete(w.ID))
}

// TestRuleLifecycle_ProcessamentoRealCompleto valida o ciclo:
// criar regra → colocar PDF → processar via engine → arquivo organizado +
// histórico registrado.
//
// Esse teste é o mais próximo do que o usuário fará na prática.
func TestRuleLifecycle_ProcessamentoRealCompleto(t *testing.T) {
	db := testhelpers.TempDB(t)
	monDir := t.TempDir()

	wsvc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))
	rsvc := service.NewRuleService(storage.NewRuleRepo(db))
	pr := storage.NewProcessedRepo(db)

	w := &domain.Watch{Name: "pasta", Path: monDir, Active: true, Recursive: false}
	require.NoError(t, wsvc.Create(w))

	rule := &domain.Rule{
		Name:     "Organiza Procuracao",
		Priority: 1,
		Active:   true,
		WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Outorgante: ", EndDelim: " RG"},
		},
		Conditions: []domain.Condition{
			{VariableName: "nome", Operator: domain.OpNotEquals, Value: ""},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "{nome}", Order: 1},
			{Type: domain.ActionMove, Target: "{nome}", Order: 2},
		},
	}
	require.NoError(t, rsvc.Create(rule))

	// Coloca um PDF compatível na pasta
	pdfText := "PROCURACAO BASTANTE FORMA Outorgante: Maria Silva RG numero 123 Outorgado: ACME"
	pdfPath := testhelpers.WritePDF(t, monDir, "doc.pdf", pdfText)

	// Recarrega a regra para garantir que filhos vieram do banco
	loadedRule, err := rsvc.GetByID(rule.ID)
	require.NoError(t, err)

	// Processa via engine direto (simula o ScanService)
	rec := &integrationRecorder{repo: pr}
	results, err := engine.ProcessPDF(pdfPath, []*domain.Rule{loadedRule}, rec, monDir, nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)

	// Verifica filesystem
	expected := filepath.Join(monDir, "Maria Silva", "doc.pdf")
	assert.FileExists(t, expected)
	assert.NoFileExists(t, pdfPath)

	// Verifica histórico
	processed, err := pr.ListRecent(1)
	require.NoError(t, err)
	require.Len(t, processed, 1)
	assert.Equal(t, domain.StatusSuccess, processed[0].Status)
	assert.Equal(t, rule.ID, processed[0].RuleID)
}

// TestRuleLifecycle_RegraDesativada_NaoExecuta valida o flag Active.
func TestRuleLifecycle_RegraDesativada_NaoExecuta(t *testing.T) {
	db := testhelpers.TempDB(t)
	monDir := t.TempDir()

	wsvc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))
	rsvc := service.NewRuleService(storage.NewRuleRepo(db))
	pr := storage.NewProcessedRepo(db)

	w := &domain.Watch{Name: "pasta", Path: monDir, Active: true}
	require.NoError(t, wsvc.Create(w))

	rule := &domain.Rule{
		Name:     "Inativa",
		Active:   false, // <-- desativada
		WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Outorgante: ", EndDelim: " RG"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "{nome}"},
		},
	}
	require.NoError(t, rsvc.Create(rule))

	pdfPath := testhelpers.WritePDF(t, monDir, "doc.pdf",
		"PROCURACAO Outorgante: Joao RG numero 1")

	loadedRule, err := rsvc.GetByID(rule.ID)
	require.NoError(t, err)

	rec := &integrationRecorder{repo: pr}
	results, err := engine.ProcessPDF(pdfPath, []*domain.Rule{loadedRule}, rec, monDir, nil)
	require.NoError(t, err)
	assert.Empty(t, results, "regra inativa não deve produzir resultados")
	assert.NoDirExists(t, filepath.Join(monDir, "Joao"))
}

// integrationRecorder é o adaptador comum aos testes de integração.
// Definido aqui em vez de em monitor_service_test.go pra ficar reutilizável
// neste arquivo e nos outros stubs.
type integrationRecorder struct {
	repo *storage.ProcessedRepo
}

func (r *integrationRecorder) Record(doc *domain.ProcessedDoc) error {
	return r.repo.Record(doc)
}

func (r *integrationRecorder) HasBeenProcessed(hash string, ruleID int64) (bool, error) {
	return r.repo.HasBeenProcessed(hash, ruleID)
}