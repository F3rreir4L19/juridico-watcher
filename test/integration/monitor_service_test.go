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

func TestMonitorService_EndToEnd(t *testing.T) {
	db := testhelpers.TempDB(t)
	monDir := t.TempDir()

	wr := storage.NewWatchRepo(db)
	rr := storage.NewRuleRepo(db)
	pr := storage.NewProcessedRepo(db)

	w := &domain.Watch{Name: "pasta", Path: monDir, Active: true, Recursive: false}
	require.NoError(t, wr.Create(w))

	rule := &domain.Rule{
		Name:     "Organiza",
		Priority: 1,
		Active:   true,
		WatchIDs: []int64{w.ID},
		Extractions: []domain.Extraction{
			{VariableName: "nome", StartDelim: "Nome: ", EndDelim: " Tipo:"},
		},
		Actions: []domain.Action{
			{Type: domain.ActionCreateFolder, Target: "{nome}", Order: 1},
			{Type: domain.ActionMove, Target: "{nome}", Order: 2},
		},
	}
	require.NoError(t, rr.Create(rule))

	mon := service.NewMonitorService(db, nil)
	err := mon.StartMonitoring()
	require.NoError(t, err)
	defer mon.StopAll()

	// Cria o PDF depois que o monitoramento já iniciou
	pdfPath := testhelpers.WritePDF(t, monDir, "doc.pdf", "Nome: Joao Tipo: Procuracao Fim.")

	// Espera ativa: arquivo deve aparecer na subpasta Joao
	expectedPath := filepath.Join(monDir, "Joao", "doc.pdf")
	require.Eventually(t, func() bool {
		_, err := os.Stat(expectedPath)
		return err == nil
	}, 5*time.Second, 200*time.Millisecond, "arquivo não foi movido para %s", expectedPath)

	// Arquivo original não existe mais
	assert.NoFileExists(t, pdfPath)

	// Registro no banco
	list, err := pr.ListRecent(1)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, domain.StatusSuccess, list[0].Status)
}
