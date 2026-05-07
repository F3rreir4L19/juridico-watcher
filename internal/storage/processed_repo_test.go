package storage_test

import (
	"testing"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	"github.com/F3rreir4L19/juridico-watcher/test/testhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProcessedTest(t *testing.T) (*storage.ProcessedRepo, *domain.Rule) {
	t.Helper()
	db := testhelpers.TempDB(t)
	wr := storage.NewWatchRepo(db)
	rr := storage.NewRuleRepo(db)
	pr := storage.NewProcessedRepo(db)

	w := &domain.Watch{Name: "w", Path: "/x", Active: true}
	require.NoError(t, wr.Create(w))
	rule := &domain.Rule{Name: "r", Active: true, WatchIDs: []int64{w.ID}}
	require.NoError(t, rr.Create(rule))
	return pr, rule
}

func TestProcessedRepo_Record_Persiste(t *testing.T) {
	pr, rule := setupProcessedTest(t)
	p := &domain.ProcessedDoc{
		FileHash:     "abc123",
		OriginalPath: "/x/file.pdf",
		RuleID:       rule.ID,
		Status:       domain.StatusSuccess,
	}
	require.NoError(t, pr.Record(p))
	assert.NotZero(t, p.ID)
}

func TestProcessedRepo_HasBeenProcessed_True(t *testing.T) {
	pr, rule := setupProcessedTest(t)
	require.NoError(t, pr.Record(&domain.ProcessedDoc{
		FileHash: "abc", OriginalPath: "/x", RuleID: rule.ID, Status: domain.StatusSuccess,
	}))

	processed, err := pr.HasBeenProcessed("abc", rule.ID)
	require.NoError(t, err)
	assert.True(t, processed)
}

func TestProcessedRepo_HasBeenProcessed_False(t *testing.T) {
	pr, rule := setupProcessedTest(t)
	processed, err := pr.HasBeenProcessed("nunca-visto", rule.ID)
	require.NoError(t, err)
	assert.False(t, processed)
}

func TestProcessedRepo_RecordDuplicado_NaoFalha(t *testing.T) {
	pr, rule := setupProcessedTest(t)
	p := &domain.ProcessedDoc{FileHash: "h", OriginalPath: "/x", RuleID: rule.ID, Status: domain.StatusSuccess}
	require.NoError(t, pr.Record(p))
	// Segundo Record com mesmo (hash, rule) — não deve falhar (UNIQUE absorvido)
	p2 := &domain.ProcessedDoc{FileHash: "h", OriginalPath: "/x", RuleID: rule.ID, Status: domain.StatusSuccess}
	assert.NoError(t, pr.Record(p2))
}

func TestProcessedRepo_GetByHash_RetornaTodasRegras(t *testing.T) {
	pr, rule := setupProcessedTest(t)
	require.NoError(t, pr.Record(&domain.ProcessedDoc{FileHash: "h", OriginalPath: "/x", RuleID: rule.ID, Status: domain.StatusSuccess}))

	list, err := pr.GetByHash("h")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, domain.StatusSuccess, list[0].Status)
}

func TestProcessedRepo_ListRecent_OrdenadosPorData(t *testing.T) {
	pr, rule := setupProcessedTest(t)
	require.NoError(t, pr.Record(&domain.ProcessedDoc{FileHash: "h1", OriginalPath: "/a", RuleID: rule.ID, Status: domain.StatusSuccess}))
	require.NoError(t, pr.Record(&domain.ProcessedDoc{FileHash: "h2", OriginalPath: "/b", RuleID: rule.ID, Status: domain.StatusFailed}))

	list, err := pr.ListRecent(10)
	require.NoError(t, err)
	require.Len(t, list, 2)
	// O último inserido deve vir primeiro
	assert.Equal(t, "h2", list[0].FileHash)
}
