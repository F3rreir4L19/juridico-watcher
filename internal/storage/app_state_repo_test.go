package storage_test

import (
	"errors"
	"testing"
	"time"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	"github.com/F3rreir4L19/juridico-watcher/test/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppStateRepo_GetSet_RoundTrip(t *testing.T) {
	repo := storage.NewAppStateRepo(testhelpers.TempDB(t))

	require.NoError(t, repo.Set("foo", "bar"))
	v, err := repo.Get("foo")
	require.NoError(t, err)
	assert.Equal(t, "bar", v)
}

func TestAppStateRepo_Get_NaoExiste_RetornaErrNotFound(t *testing.T) {
	repo := storage.NewAppStateRepo(testhelpers.TempDB(t))
	_, err := repo.Get("inexistente")
	assert.True(t, errors.Is(err, domain.ErrNotFound))
}

func TestAppStateRepo_Set_Sobrescreve(t *testing.T) {
	repo := storage.NewAppStateRepo(testhelpers.TempDB(t))
	require.NoError(t, repo.Set("k", "v1"))
	require.NoError(t, repo.Set("k", "v2"))
	v, _ := repo.Get("k")
	assert.Equal(t, "v2", v)
}

func TestAppStateRepo_GetTime_Inexistente_RetornaZeroSemErro(t *testing.T) {
	repo := storage.NewAppStateRepo(testhelpers.TempDB(t))
	v, err := repo.GetTime("history_last_visit")
	require.NoError(t, err)
	assert.True(t, v.IsZero())
}

func TestAppStateRepo_SetGetTime_RoundTrip(t *testing.T) {
	repo := storage.NewAppStateRepo(testhelpers.TempDB(t))
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)

	require.NoError(t, repo.SetTime("k", now))
	got, err := repo.GetTime("k")
	require.NoError(t, err)
	assert.True(t, now.Equal(got), "esperado %v, recebeu %v", now, got)
}s