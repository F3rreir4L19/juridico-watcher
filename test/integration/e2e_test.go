package integration_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/service"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	"github.com/F3rreir4L19/juridico-watcher/test/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_FluxoCompletoProcuracao(t *testing.T) {
	db := testhelpers.TempDB(t)
	monDir := t.TempDir()

	wsvc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))
	rsvc := service.NewRuleService(storage.NewRuleRepo(db))
	pr := storage.NewProcessedRepo(db)

	w := &domain.Watch{
		Name:      "digitalizadoras",
		Path:      monDir,
		Active:    true,
		Recursive: false,
	}
	require.NoError(t, wsvc.Create(w))

	rule := &domain.Rule{
		Name:     "procuracao",
		Priority: 10,
		Active:   true,
		WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "tipo", StartDelim: "", EndDelim: " Outorgante:", Order: 0},
			{VariableName: "nome", StartDelim: "Outorgante: ", EndDelim: " RG", Order: 1},
		},
		Conditions: []domain.Condition{
			{VariableName: "tipo", Operator: domain.OpContains, Value: "PROCURACAO"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "{nome}", Order: 0},
			{Type: domain.ActionMove, Target: "{nome}", Order: 1},
			{Type: domain.ActionRename, Target: "procuracao_{nome}", Order: 2},
		},
	}
	require.NoError(t, rsvc.Create(rule))

	preExisting := testhelpers.WritePDF(t, monDir, "scan001.pdf",
		"PROCURACAO BASTANTE FORMA Outorgante: Maria Silva RG numero 123 Outorgado: ACME Para fins de venda")

	nonProcuracao := testhelpers.WritePDF(t, monDir, "scan002.pdf",
		"CERTIDAO DE NASCIMENTO Outorgante: Joao Souza RG numero 456")

	scanSvc := service.NewScanService(db, nil)
	_, err := scanSvc.ScanWatch(w.ID)
	require.NoError(t, err)

	mariaPath := filepath.Join(monDir, "Maria Silva", "procuracao_Maria Silva.pdf")
	assert.FileExists(t, mariaPath, "procuração deveria estar organizada")
	assert.NoFileExists(t, preExisting, "scan001.pdf não deveria mais estar na raiz")

	assert.FileExists(t, nonProcuracao, "certidão deveria continuar na raiz")

	all, err := pr.ListRecent(10)
	require.NoError(t, err)
	require.Len(t, all, 2)

	mon := service.NewMonitorService(db, nil)
	require.NoError(t, mon.StartMonitoring())
	defer mon.StopAll()

	novoPDF := testhelpers.WritePDF(t, monDir, "scan003.pdf",
		"PROCURACAO BASTANTE Outorgante: Carlos Pereira RG numero 789 Outorgado: ACME")

	expectedNovo := filepath.Join(monDir, "Carlos Pereira", "procuracao_Carlos Pereira.pdf")
	require.Eventually(t, func() bool {
		_, err := os.Stat(expectedNovo)
		return err == nil
	}, 8*time.Second, 200*time.Millisecond,
		"PDF novo deveria ter sido organizado automaticamente")

	assert.NoFileExists(t, novoPDF)

	all, err = pr.ListRecent(10)
	require.NoError(t, err)
	require.Len(t, all, 3)

	assert.Equal(t, domain.StatusSuccess, all[0].Status)
}

func TestE2E_DeduplicacaoEntreScans(t *testing.T) {
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
		},
	}))

	testhelpers.WritePDF(t, monDir, "doc.pdf",
		"PROCURACAO Outorgante: Ana RG numero 1")

	scan := service.NewScanService(db, nil)

	_, err := scan.ScanWatch(w.ID)
	require.NoError(t, err)
	first, _ := pr.ListRecent(10)
	assert.Len(t, first, 1)

	_, err = scan.ScanWatch(w.ID)
	require.NoError(t, err)
	second, _ := pr.ListRecent(10)
	assert.Len(t, second, 1, "scan repetido não deveria duplicar registro")
}