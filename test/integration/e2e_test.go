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

// TestE2E_FluxoCompletoProcuracao replica o cenário do briefing original:
//
//   1. Usuário define pasta "digitalizadoras" como monitorada
//   2. Cria regra "procuracao" com extrações (nome, tipo) e ações
//      (create_folder, move, rename)
//   3. Documentos preexistentes são processados via ScanService
//   4. Novos documentos chegando ativam o pipeline via MonitorService
//   5. Documentos com tipo != "procuracao" são ignorados (no_match)
//
// Este é o teste-âncora: se ele passar, o sistema funciona end-to-end
// para o uso real.
func TestE2E_FluxoCompletoProcuracao(t *testing.T) {
	db := testhelpers.TempDB(t)
	monDir := t.TempDir() // simula a pasta "digitalizadoras"

	wsvc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))
	rsvc := service.NewRuleService(storage.NewRuleRepo(db))
	pr := storage.NewProcessedRepo(db)

	// === Fase 1: usuário cadastra a pasta digitalizadoras ===
	w := &domain.Watch{
		Name:      "digitalizadoras",
		Path:      monDir,
		Active:    true,
		Recursive: false,
	}
	require.NoError(t, wsvc.Create(w))

	// === Fase 2: cria a regra "procuracao" ===
	// - extrai "nome" entre "Outorgante: " e " RG"
	// - extrai "tipo" do início do texto até " Outorgante:"
	// - condicional: tipo CONTÉM "PROCURACAO"
	// - ações: cria pasta {nome}, move pra ela, renomeia para "procuracao_{nome}"
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

	// === Fase 3: documentos preexistentes ===
	// Antes do programa começar a monitorar, já tinha um PDF na pasta.
	preExisting := testhelpers.WritePDF(t, monDir, "scan001.pdf",
		"PROCURACAO BASTANTE FORMA Outorgante: Maria Silva RG numero 123 Outorgado: ACME Para fins de venda")

	// E um documento que NÃO é procuração
	nonProcuracao := testhelpers.WritePDF(t, monDir, "scan002.pdf",
		"CERTIDAO DE NASCIMENTO Outorgante: Joao Souza RG numero 456")

	// Usuário clica em "Atualizar" → ScanService roda
	scanSvc := service.NewScanService(db, nil)
	require.NoError(t, scanSvc.ScanWatch(w.ID))

	// Maria Silva (procuração) → organizada
	mariaPath := filepath.Join(monDir, "Maria Silva", "procuracao_Maria Silva.pdf")
	assert.FileExists(t, mariaPath, "procuração deveria estar organizada")
	assert.NoFileExists(t, preExisting, "scan001.pdf não deveria mais estar na raiz")

	// Joao Souza (certidão) → não foi tocado
	assert.FileExists(t, nonProcuracao, "certidão deveria continuar na raiz")

	// Histórico tem 2 entradas: Maria=success, Joao=no_match
	all, err := pr.ListRecent(10)
	require.NoError(t, err)
	require.Len(t, all, 2)

	// === Fase 4: monitor service em runtime ===
	// Agora um novo PDF chega via "scanner" no diretório
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

	// Histórico agora tem 3 entradas
	all, err = pr.ListRecent(10)
	require.NoError(t, err)
	require.Len(t, all, 3)

	// O mais recente é Carlos com status success
	assert.Equal(t, domain.StatusSuccess, all[0].Status)
}

// TestE2E_DeduplicacaoEntreScans valida RN-11 no fluxo completo:
// rodar scan duas vezes seguidas não reprocessa nem duplica histórico.
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

	// Primeira execução
	require.NoError(t, scan.ScanWatch(w.ID))
	first, _ := pr.ListRecent(10)
	assert.Len(t, first, 1)

	// Segunda execução — não deveria adicionar nada
	require.NoError(t, scan.ScanWatch(w.ID))
	second, _ := pr.ListRecent(10)
	assert.Len(t, second, 1, "scan repetido não deveria duplicar registro")
}