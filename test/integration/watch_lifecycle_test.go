package integration_test

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

// TestWatchLifecycle_CRUDCompleto valida criar → ler → atualizar → deletar
// no fluxo do WatchService (camada que a UI vai usar).
func TestWatchLifecycle_CRUDCompleto(t *testing.T) {
	db := testhelpers.TempDB(t)
	svc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))

	// CREATE
	w := &domain.Watch{
		Name:      "digitalizadoras",
		Path:      t.TempDir(),
		Active:    true,
		Recursive: true,
	}
	require.NoError(t, svc.Create(w))
	assert.NotZero(t, w.ID, "ID deve ser preenchido após Create")

	// READ por ID
	loaded, err := svc.GetByID(w.ID)
	require.NoError(t, err)
	assert.Equal(t, "digitalizadoras", loaded.Name)
	assert.True(t, loaded.Active)
	assert.True(t, loaded.Recursive)

	// LIST
	all, err := svc.List()
	require.NoError(t, err)
	assert.Len(t, all, 1)

	// UPDATE
	loaded.Active = false
	loaded.Recursive = false
	require.NoError(t, svc.Update(loaded))

	updated, err := svc.GetByID(w.ID)
	require.NoError(t, err)
	assert.False(t, updated.Active)
	assert.False(t, updated.Recursive)

	// DELETE
	require.NoError(t, svc.Delete(w.ID))
	_, err = svc.GetByID(w.ID)
	assert.True(t, errors.Is(err, domain.ErrNotFound), "watch deletado não deve mais existir")
}

// TestWatchLifecycle_NomeDuplicado_RetornaErro garante a UNIQUE constraint
// chega na camada de service como erro tipado.
func TestWatchLifecycle_NomeDuplicado_RetornaErro(t *testing.T) {
	db := testhelpers.TempDB(t)
	svc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))

	require.NoError(t, svc.Create(&domain.Watch{Name: "x", Path: t.TempDir(), Active: true}))

	err := svc.Create(&domain.Watch{Name: "x", Path: t.TempDir(), Active: true})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrDuplicateName))
}

// TestWatchLifecycle_DeleteEmUso_RetornaErrWatchInUse valida a RN-08.
// Esta é a regra crítica: o usuário não pode deletar uma pasta monitorada
// que está em uso por uma regra; precisa primeiro editar/deletar as regras.
func TestWatchLifecycle_DeleteEmUso_RetornaErrWatchInUse(t *testing.T) {
	db := testhelpers.TempDB(t)
	wsvc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))
	rsvc := service.NewRuleService(storage.NewRuleRepo(db))

	// Cria watch e regra que aponta para ele
	w := &domain.Watch{Name: "w", Path: t.TempDir(), Active: true}
	require.NoError(t, wsvc.Create(w))

	rule := &domain.Rule{
		Name:     "regra que usa w",
		Active:   true,
		WatchIDs: []int64{w.ID},
	}
	require.NoError(t, rsvc.Create(rule))

	// Tentar deletar watch deve falhar
	err := wsvc.Delete(w.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrWatchInUse),
		"deletar watch em uso deve retornar ErrWatchInUse, recebeu: %v", err)

	// Watch ainda existe
	_, err = wsvc.GetByID(w.ID)
	assert.NoError(t, err)

	// Após deletar a regra, deletar o watch funciona
	require.NoError(t, rsvc.Delete(rule.ID))
	require.NoError(t, wsvc.Delete(w.ID))
}

// TestWatchLifecycle_DesativarReativar_NaoAfetaPersistencia
// Valida que o flag Active é manipulável independente do resto.
func TestWatchLifecycle_DesativarReativar_NaoAfetaPersistencia(t *testing.T) {
	db := testhelpers.TempDB(t)
	svc := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))

	w := &domain.Watch{Name: "w", Path: t.TempDir(), Active: true, Recursive: true}
	require.NoError(t, svc.Create(w))

	// Desativa
	w.Active = false
	require.NoError(t, svc.Update(w))
	loaded, _ := svc.GetByID(w.ID)
	assert.False(t, loaded.Active)
	assert.True(t, loaded.Recursive, "outros campos não devem ser alterados")

	// Reativa
	w.Active = true
	require.NoError(t, svc.Update(w))
	loaded, _ = svc.GetByID(w.ID)
	assert.True(t, loaded.Active)
}
