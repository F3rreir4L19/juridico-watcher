package storage_test

import (
	"errors"
	"testing"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	"github.com/F3rreir4L19/juridico-watcher/test/testhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper: cria DB com um watch e retorna repos + watch
func setupRuleTest(t *testing.T) (*storage.RuleRepo, *storage.WatchRepo, *domain.Watch) {
	t.Helper()
	db := testhelpers.TempDB(t)
	wr := storage.NewWatchRepo(db)
	rr := storage.NewRuleRepo(db)

	w := &domain.Watch{Name: "digitalizadoras", Path: "/tmp/x", Active: true, Recursive: true}
	require.NoError(t, wr.Create(w))
	return rr, wr, w
}

func TestRuleRepo_Create_PersisteEDevolveID(t *testing.T) {
	rr, _, w := setupRuleTest(t)
	rule := &domain.Rule{
		Name:     "procuração",
		Priority: 100,
		Active:   true,
		WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "outorgante:", EndDelim: ","},
			{VariableName: "tipo", StartDelim: "", EndDelim: "pelo"},
		},
		Conditions: []domain.Condition{
			{VariableName: "tipo", Operator: domain.OpEquals, Value: "procuração"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "{nome}"},
			{Type: domain.ActionMove, Target: "{nome}"},
			{Type: domain.ActionRename, Target: "procuracao_{nome}"},
		},
	}

	err := rr.Create(rule)
	require.NoError(t, err)
	assert.NotZero(t, rule.ID)

	// IDs dos filhos foram preenchidos
	for _, ex := range rule.Extractions {
		assert.NotZero(t, ex.ID)
		assert.Equal(t, rule.ID, ex.RuleID)
	}
}

func TestRuleRepo_GetByID_RetornaRegraComTodosFilhos(t *testing.T) {
	rr, _, w := setupRuleTest(t)
	original := &domain.Rule{
		Name:     "r",
		Active:   true,
		WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "v1", StartDelim: "a", EndDelim: "b"},
		},
		Conditions: []domain.Condition{
			{VariableName: "v1", Operator: domain.OpContains, Value: "x"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "{v1}"},
		},
	}
	require.NoError(t, rr.Create(original))

	loaded, err := rr.GetByID(original.ID)
	require.NoError(t, err)
	assert.Equal(t, "r", loaded.Name)
	require.Len(t, loaded.Extractions, 1)
	assert.Equal(t, "v1", loaded.Extractions[0].VariableName)
	require.Len(t, loaded.Conditions, 1)
	assert.Equal(t, domain.OpContains, loaded.Conditions[0].Operator)
	require.Len(t, loaded.Actions, 1)
	assert.Equal(t, domain.ActionCreateFolder, loaded.Actions[0].Type)
	require.Len(t, loaded.WatchIDs, 1)
	assert.Equal(t, w.ID, loaded.WatchIDs[0])
}

func TestRuleRepo_List_RetornaTodasOrdenadasPorPrioridade(t *testing.T) {
	rr, _, w := setupRuleTest(t)
	require.NoError(t, rr.Create(&domain.Rule{Name: "c", Priority: 30, WatchIDs: []int64{w.ID}}))
	require.NoError(t, rr.Create(&domain.Rule{Name: "a", Priority: 10, WatchIDs: []int64{w.ID}}))
	require.NoError(t, rr.Create(&domain.Rule{Name: "b", Priority: 20, WatchIDs: []int64{w.ID}}))

	list, err := rr.List()
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, "a", list[0].Name)
	assert.Equal(t, "b", list[1].Name)
	assert.Equal(t, "c", list[2].Name)
}

func TestRuleRepo_ListByWatchID_FiltraCorretamente(t *testing.T) {
	rr, wr, w1 := setupRuleTest(t)
	w2 := &domain.Watch{Name: "outra", Path: "/tmp/y", Active: true}
	require.NoError(t, wr.Create(w2))

	require.NoError(t, rr.Create(&domain.Rule{Name: "r1", Active: true, WatchIDs: []int64{w1.ID}}))
	require.NoError(t, rr.Create(&domain.Rule{Name: "r2", Active: true, WatchIDs: []int64{w2.ID}}))
	require.NoError(t, rr.Create(&domain.Rule{Name: "r3", Active: true, WatchIDs: []int64{w1.ID, w2.ID}}))
	require.NoError(t, rr.Create(&domain.Rule{Name: "r4_inactive", Active: false, WatchIDs: []int64{w1.ID}}))

	list, err := rr.ListByWatchID(w1.ID)
	require.NoError(t, err)
	// r1 e r3 (não inclui r2 nem r4 inativa)
	require.Len(t, list, 2)
	names := []string{list[0].Name, list[1].Name}
	assert.Contains(t, names, "r1")
	assert.Contains(t, names, "r3")
}

func TestRuleRepo_Update_SubstituiFilhos(t *testing.T) {
	rr, _, w := setupRuleTest(t)
	rule := &domain.Rule{
		Name:     "r",
		Active:   true,
		WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "v1", StartDelim: "a", EndDelim: "b"},
		},
	}
	require.NoError(t, rr.Create(rule))

	rule.Extractions = []domain.Extraction{
		{VariableName: "v2", StartDelim: "x", EndDelim: "y"},
		{VariableName: "v3", StartDelim: "p", EndDelim: "q"},
	}
	require.NoError(t, rr.Update(rule))

	loaded, err := rr.GetByID(rule.ID)
	require.NoError(t, err)
	require.Len(t, loaded.Extractions, 2)
	assert.Equal(t, "v2", loaded.Extractions[0].VariableName)
	assert.Equal(t, "v3", loaded.Extractions[1].VariableName)
}

func TestRuleRepo_Delete_Remove(t *testing.T) {
	rr, _, w := setupRuleTest(t)
	rule := &domain.Rule{Name: "r", Active: true, WatchIDs: []int64{w.ID}}
	require.NoError(t, rr.Create(rule))

	require.NoError(t, rr.Delete(rule.ID))
	_, err := rr.GetByID(rule.ID)
	assert.True(t, errors.Is(err, domain.ErrNotFound))
}

func TestRuleRepo_NomeUnico_CreateDuplicado_Falha(t *testing.T) {
	rr, _, w := setupRuleTest(t)
	require.NoError(t, rr.Create(&domain.Rule{Name: "r", WatchIDs: []int64{w.ID}}))

	err := rr.Create(&domain.Rule{Name: "r", WatchIDs: []int64{w.ID}})
	assert.True(t, errors.Is(err, domain.ErrDuplicateName))
}

// Agora o teste pendente da Sprint 1.7:
func TestWatchRepo_Delete_ComRegrasReferenciando_RetornaErro(t *testing.T) {
	rr, wr, w := setupRuleTest(t)
	require.NoError(t, rr.Create(&domain.Rule{Name: "r", WatchIDs: []int64{w.ID}}))

	err := wr.Delete(w.ID)
	assert.True(t, errors.Is(err, domain.ErrWatchInUse))
}
