package watcher_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/F3rreir4L19/juridico-watcher/internal/watcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitStable_EstabilizaRapidamente(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.pdf")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0644))

	// O arquivo foi escrito e não está sendo alterado
	err := watcher.WaitStable(path, 50*time.Millisecond, 2, 1*time.Second)
	assert.NoError(t, err)
}

func TestWaitStable_Timeout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grow.pdf")
	require.NoError(t, os.WriteFile(path, []byte("a"), 0644))

	// Simula arquivo crescendo em background
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(100 * time.Millisecond)
			_ = os.WriteFile(path, []byte(string(make([]byte, i*10))), 0644)
		}
	}()

	err := watcher.WaitStable(path, 50*time.Millisecond, 3, 300*time.Millisecond)
	assert.ErrorIs(t, err, watcher.ErrStabilizationTimeout)
}

func TestWaitStable_ArquivoRemovido(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "remove.pdf")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))
	require.NoError(t, os.Remove(path))

	err := watcher.WaitStable(path, 10*time.Millisecond, 2, 100*time.Millisecond)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}