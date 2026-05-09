package integration_test

import (
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
	require.NoError(t, scan.ScanWatch(w.ID))

	assert.FileExists(t, filepath.Join(monDir, "Ana", "a.pdf"))
	assert.FileExists(t, filepath.Join(monDir, "Bruno", "b.pdf"))
	assert.FileExists(t, filepath.Join(monDir, "Carla", "c.pdf"))

	all, err := pr.ListRecent(10)
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestScanService_WatchInativo_NaoProcessa(t *testing.T) {
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
	require.NoError(t, scan.ScanWatch(w.ID))

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
	require.NoError(t, scan.ScanWatch(w.ID))

	all, _ := pr.ListRecent(10)
	assert.Len(t, all, 2)
}
