package watcher_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
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

	err := watcher.WaitStable(path, 50*time.Millisecond, 2, 1*time.Second)
	assert.NoError(t, err)
}

func TestWaitStable_Timeout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grow.pdf")
	require.NoError(t, os.WriteFile(path, []byte("a"), 0644))

	// Garante que a goroutine de crescimento esteja viva durante toda a janela
	// do teste, sem usar sleeps frágeis.
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		size := 1
		ticker := time.NewTicker(30 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				size++
				_ = os.WriteFile(path, make([]byte, size*10), 0644)
			}
		}
	}()

	err := watcher.WaitStable(path, 50*time.Millisecond, 3, 300*time.Millisecond)
	close(stop)
	wg.Wait()

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
