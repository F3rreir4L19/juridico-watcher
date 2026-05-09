package service_test

import (
	"errors"
	"testing"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/service"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	"github.com/F3rreir4L19/juridico-watcher/test/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper: cria DB com um watch e devolve o RuleService + ID do watch.
func setupRuleServiceTest(t *testing.T) (*service.RuleService, int64) {
	t.Helper()
	db := testhelpers.TempDB(t)
	wr := storage.NewWatchRepo(db)
	rr := storage.NewRuleRepo(db)

	w := &domain.Watch{Name: "w", Path: "/x", Active: true}
	require.NoError(t, wr.Create(w))
	return service.NewRuleService(rr), w.ID
}

func TestRuleService_Create_NomeVazio_RetornaErrInvalidInput(t *testing.T) {
	svc, watchID := setupRuleServiceTest(t)
	rule := &domain.Rule{Name: "", Active: true, WatchIDs: []int64{watchID}}

	err := svc.Create(rule)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
}

func TestRuleService_Create_NomeSoEspacos_RetornaErrInvalidInput(t *testing.T) {
	svc, watchID := setupRuleServiceTest(t)
	rule := &domain.Rule{Name: "   ", Active: true, WatchIDs: []int64{watchID}}

	err := svc.Create(rule)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
}

func TestRuleService_Create_SemWatches_RetornaErrInvalidInput(t *testing.T) {
	svc, _ := setupRuleServiceTest(t)
	rule := &domain.Rule{Name: "ok", Active: true, WatchIDs: nil}

	err := svc.Create(rule)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
}

func TestRuleService_Create_Valida_PersistirNormal(t *testing.T) {
	svc, watchID := setupRuleServiceTest(t)
	rule := &domain.Rule{Name: "regra ok", Active: true, WatchIDs: []int64{watchID}}

	require.NoError(t, svc.Create(rule))
	assert.NotZero(t, rule.ID)
}

func TestRuleService_Update_RegraInvalida_RetornaErrInvalidInput(t *testing.T) {
	svc, watchID := setupRuleServiceTest(t)
	rule := &domain.Rule{Name: "ok", Active: true, WatchIDs: []int64{watchID}}
	require.NoError(t, svc.Create(rule))

	// Tenta esvaziar os watches no update
	rule.WatchIDs = nil
	err := svc.Update(rule)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
}

func TestRuleService_Update_NomeVazio_RetornaErrInvalidInput(t *testing.T) {
	svc, watchID := setupRuleServiceTest(t)
	rule := &domain.Rule{Name: "ok", Active: true, WatchIDs: []int64{watchID}}
	require.NoError(t, svc.Create(rule))

	rule.Name = ""
	err := svc.Update(rule)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
}

func TestRuleService_Create_RegraNula_RetornaErrInvalidInput(t *testing.T) {
	svc, _ := setupRuleServiceTest(t)
	err := svc.Create(nil)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
}
