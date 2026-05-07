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

func newWatchRepo(t *testing.T) *storage.WatchRepo {
	t.Helper()
	db := testhelpers.TempDB(t)
	return storage.NewWatchRepo(db)
}

func TestWatchRepo_Create_PersisteEDevolveID(t *testing.T) {
	repo := newWatchRepo(t)
	w := &domain.Watch{Name: "digitalizadoras", Path: "/tmp/x", Active: true, Recursive: true}

	err := repo.Create(w)
	require.NoError(t, err)
	assert.NotZero(t, w.ID)
	assert.False(t, w.CreatedAt.IsZero())
}

func TestWatchRepo_GetByID_RetornaWatchSalvo(t *testing.T) {
	repo := newWatchRepo(t)
	original := &domain.Watch{Name: "a", Path: "/x", Active: true, Recursive: false}
	require.NoError(t, repo.Create(original))

	loaded, err := repo.GetByID(original.ID)
	require.NoError(t, err)
	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.Path, loaded.Path)
	assert.Equal(t, original.Active, loaded.Active)
	assert.Equal(t, original.Recursive, loaded.Recursive)
}

func TestWatchRepo_GetByName_RetornaWatchSalvo(t *testing.T) {
	repo := newWatchRepo(t)
	require.NoError(t, repo.Create(&domain.Watch{Name: "digitalizadoras", Path: "/x", Active: true}))

	loaded, err := repo.GetByName("digitalizadoras")
	require.NoError(t, err)
	assert.Equal(t, "digitalizadoras", loaded.Name)
}

func TestWatchRepo_GetByName_NaoExiste_RetornaErrNotFound(t *testing.T) {
	repo := newWatchRepo(t)
	_, err := repo.GetByName("inexistente")
	assert.True(t, errors.Is(err, domain.ErrNotFound))
}

func TestWatchRepo_GetByID_NaoExiste_RetornaErrNotFound(t *testing.T) {
	repo := newWatchRepo(t)
	_, err := repo.GetByID(9999)
	assert.True(t, errors.Is(err, domain.ErrNotFound))
}

func TestWatchRepo_List_RetornaTodosOrdenadosPorNome(t *testing.T) {
	repo := newWatchRepo(t)
	require.NoError(t, repo.Create(&domain.Watch{Name: "zeta", Path: "/z"}))
	require.NoError(t, repo.Create(&domain.Watch{Name: "alfa", Path: "/a"}))
	require.NoError(t, repo.Create(&domain.Watch{Name: "beta", Path: "/b"}))

	list, err := repo.List()
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, "alfa", list[0].Name)
	assert.Equal(t, "beta", list[1].Name)
	assert.Equal(t, "zeta", list[2].Name)
}

func TestWatchRepo_Update_PersisteAlteracoes(t *testing.T) {
	repo := newWatchRepo(t)
	w := &domain.Watch{Name: "a", Path: "/x", Active: true, Recursive: true}
	require.NoError(t, repo.Create(w))

	w.Name = "b"
	w.Path = "/y"
	w.Active = false
	require.NoError(t, repo.Update(w))

	loaded, err := repo.GetByID(w.ID)
	require.NoError(t, err)
	assert.Equal(t, "b", loaded.Name)
	assert.Equal(t, "/y", loaded.Path)
	assert.False(t, loaded.Active)
}

func TestWatchRepo_Update_NaoExiste_RetornaErrNotFound(t *testing.T) {
	repo := newWatchRepo(t)
	w := &domain.Watch{ID: 9999, Name: "x", Path: "/x"}
	err := repo.Update(w)
	assert.True(t, errors.Is(err, domain.ErrNotFound))
}

func TestWatchRepo_Delete_Remove(t *testing.T) {
	repo := newWatchRepo(t)
	w := &domain.Watch{Name: "a", Path: "/x"}
	require.NoError(t, repo.Create(w))

	require.NoError(t, repo.Delete(w.ID))
	_, err := repo.GetByID(w.ID)
	assert.True(t, errors.Is(err, domain.ErrNotFound))
}

func TestWatchRepo_NomeUnico_CreateDuplicado_Falha(t *testing.T) {
	repo := newWatchRepo(t)
	require.NoError(t, repo.Create(&domain.Watch{Name: "a", Path: "/x"}))

	err := repo.Create(&domain.Watch{Name: "a", Path: "/y"})
	assert.True(t, errors.Is(err, domain.ErrDuplicateName))
}

// O teste TestWatchRepo_Delete_ComRegrasReferenciando_RetornaErro
// será adicionado depois que o RuleRepo existir, na próxima sub-tarefa.
