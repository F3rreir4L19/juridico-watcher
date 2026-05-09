package integration_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/service"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	"github.com/F3rreir4L19/juridico-watcher/test/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanService_ProcessaPDFsExistentes(t *testing.T) {
	db := testhelpers.TempDB(t)
	monDir := t.TempDir()

	wsvc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))
	rsvc := service.NewRuleService(storage.NewRuleRepo(db))
	pr := storage.NewProcessedRepo(db)

	w := &domain.Watch{Name: "w", Path: monDir, Active: true}
	require.NoError(t, wsvc.Create(w))
	require.NoError(t, rsvc.Create(&domain.Rule{
		Name: "r", Priority: 1, Active: true, WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Outorgante: ", EndDelim: " RG"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "{nome}"},
			{Type: domain.ActionMove, Target: "{nome}"},
		},
	}))

	testhelpers.WritePDF(t, monDir, "a.pdf", "PROCURACAO Outorgante: Ana RG numero 1")
	testhelpers.WritePDF(t, monDir, "b.pdf", "PROCURACAO Outorgante: Bruno RG numero 2")
	testhelpers.WritePDF(t, monDir, "c.pdf", "PROCURACAO Outorgante: Carla RG numero 3")

	scan := service.NewScanService(db, nil)
	count, err := scan.ScanWatch(w.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	assert.FileExists(t, filepath.Join(monDir, "Ana", "a.pdf"))
	assert.FileExists(t, filepath.Join(monDir, "Bruno", "b.pdf"))
	assert.FileExists(t, filepath.Join(monDir, "Carla", "c.pdf"))

	all, err := pr.ListRecent(10)
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestScanService_WatchInativo_RetornaErrInactive(t *testing.T) {
	db := testhelpers.TempDB(t)
	monDir := t.TempDir()

	wsvc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))
	rsvc := service.NewRuleService(storage.NewRuleRepo(db))
	pr := storage.NewProcessedRepo(db)

	w := &domain.Watch{Name: "w", Path: monDir, Active: false}
	require.NoError(t, wsvc.Create(w))
	require.NoError(t, rsvc.Create(&domain.Rule{
		Name: "r", Active: true, WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Outorgante: ", EndDelim: " RG"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "{nome}"},
		},
	}))

	testhelpers.WritePDF(t, monDir, "a.pdf",
		"PROCURACAO Outorgante: Ana RG numero 1")

	scan := service.NewScanService(db, nil)
	count, err := scan.ScanWatch(w.ID)
	assert.Equal(t, 0, count)
	assert.True(t, errors.Is(err, service.ErrInactive))

	all, _ := pr.ListRecent(10)
	assert.Empty(t, all)
	assert.NoDirExists(t, filepath.Join(monDir, "Ana"))
}

func TestScanService_Recursivo_VarreSubpastas(t *testing.T) {
	db := testhelpers.TempDB(t)
	monDir := t.TempDir()

	wsvc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))
	rsvc := service.NewRuleService(storage.NewRuleRepo(db))
	pr := storage.NewProcessedRepo(db)

	w := &domain.Watch{Name: "w", Path: monDir, Active: true, Recursive: true}
	require.NoError(t, wsvc.Create(w))
	require.NoError(t, rsvc.Create(&domain.Rule{
		Name: "r", Active: true, WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Outorgante: ", EndDelim: " RG"},
		},
		Conditions: []domain.Condition{
			{VariableName: "nome", Operator: domain.OpNotEquals, Value: ""},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "processados"},
		},
	}))

	require.NoError(t, os.MkdirAll(filepath.Join(monDir, "sub1"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(monDir, "sub2", "deep"), 0755))
	testhelpers.WritePDF(t, filepath.Join(monDir, "sub1"), "x.pdf",
		"PROCURACAO Outorgante: X RG numero 1")
	testhelpers.WritePDF(t, filepath.Join(monDir, "sub2", "deep"), "y.pdf",
		"PROCURACAO Outorgante: Y RG numero 2")

	scan := service.NewScanService(db, nil)
	count, err := scan.ScanWatch(w.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	all, _ := pr.ListRecent(10)
	assert.Len(t, all, 2)
}

// TestScanService_ScanRule_AplicaApenasARegraSelecionada valida que
// ScanRule processa só a regra alvo, ignorando outras associadas à pasta.
func TestScanService_ScanRule_AplicaApenasARegraSelecionada(t *testing.T) {
	db := testhelpers.TempDB(t)
	monDir := t.TempDir()

	wsvc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))
	rsvc := service.NewRuleService(storage.NewRuleRepo(db))
	pr := storage.NewProcessedRepo(db)

	w := &domain.Watch{Name: "w", Path: monDir, Active: true}
	require.NoError(t, wsvc.Create(w))

	// Duas regras na mesma pasta
	regraA := &domain.Rule{
		Name: "RegraA", Priority: 1, Active: true, WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Outorgante: ", EndDelim: " RG"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "A_{nome}"},
		},
	}
	regraB := &domain.Rule{
		Name: "RegraB", Priority: 2, Active: true, WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Outorgante: ", EndDelim: " RG"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "B_{nome}"},
		},
	}
	require.NoError(t, rsvc.Create(regraA))
	require.NoError(t, rsvc.Create(regraB))

	testhelpers.WritePDF(t, monDir, "doc.pdf",
		"PROCURACAO Outorgante: Maria RG numero 1")

	scan := service.NewScanService(db, nil)
	count, err := scan.ScanRule(regraA.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Só a pasta A_Maria foi criada — RegraB não rodou
	assert.DirExists(t, filepath.Join(monDir, "A_Maria"))
	assert.NoDirExists(t, filepath.Join(monDir, "B_Maria"))

	// Histórico tem 1 entrada (só RegraA processou)
	all, _ := pr.ListRecent(10)
	assert.Len(t, all, 1)
	assert.Equal(t, regraA.ID, all[0].RuleID)
}

// TestScanService_ScanRule_RegraInativa_RetornaErrInactive
func TestScanService_ScanRule_RegraInativa_RetornaErrInactive(t *testing.T) {
	db := testhelpers.TempDB(t)
	monDir := t.TempDir()

	wsvc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))
	rsvc := service.NewRuleService(storage.NewRuleRepo(db))

	w := &domain.Watch{Name: "w", Path: monDir, Active: true}
	require.NoError(t, wsvc.Create(w))

	regra := &domain.Rule{
		Name: "Inativa", Active: false, WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Outorgante: ", EndDelim: " RG"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "{nome}"},
		},
	}
	require.NoError(t, rsvc.Create(regra))

	scan := service.NewScanService(db, nil)
	count, err := scan.ScanRule(regra.ID)
	assert.Equal(t, 0, count)
	assert.True(t, errors.Is(err, service.ErrInactive))
}

// TestScanService_ScanAll_ProcessaTodasPastasAtivas
func TestScanService_ScanAll_ProcessaTodasPastasAtivas(t *testing.T) {
	db := testhelpers.TempDB(t)
	dirA := t.TempDir()
	dirB := t.TempDir()
	dirInativo := t.TempDir()

	wsvc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))
	rsvc := service.NewRuleService(storage.NewRuleRepo(db))
	pr := storage.NewProcessedRepo(db)

	wA := &domain.Watch{Name: "A", Path: dirA, Active: true}
	wB := &domain.Watch{Name: "B", Path: dirB, Active: true}
	wInactive := &domain.Watch{Name: "Inativa", Path: dirInativo, Active: false}
	require.NoError(t, wsvc.Create(wA))
	require.NoError(t, wsvc.Create(wB))
	require.NoError(t, wsvc.Create(wInactive))

	require.NoError(t, rsvc.Create(&domain.Rule{
		Name: "r", Active: true, WatchIDs: []int64{wA.ID, wB.ID, wInactive.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Outorgante: ", EndDelim: " RG"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "{nome}"},
		},
	}))

	testhelpers.WritePDF(t, dirA, "a.pdf", "PROCURACAO Outorgante: Ana RG 1")
	testhelpers.WritePDF(t, dirB, "b.pdf", "PROCURACAO Outorgante: Bruno RG 2")
	// PDF na pasta inativa NÃO deve ser processado
	testhelpers.WritePDF(t, dirInativo, "x.pdf", "PROCURACAO Outorgante: X RG 3")

	scan := service.NewScanService(db, nil)
	count, err := scan.ScanAll()
	require.NoError(t, err)
	assert.Equal(t, 2, count, "esperado 2 PDFs processados (pasta inativa pulada)")

	all, _ := pr.ListRecent(10)
	assert.Len(t, all, 2)

	assert.DirExists(t, filepath.Join(dirA, "Ana"))
	assert.DirExists(t, filepath.Join(dirB, "Bruno"))
	assert.NoDirExists(t, filepath.Join(dirInativo, "X"))
}