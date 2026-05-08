package watcher_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/F3rreir4L19/juridico-watcher/internal/watcher"
	"github.com/F3rreir4L19/juridico-watcher/test/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestWatcher_DetectaPDFNovo(t *testing.T) {
	dir := t.TempDir()
	fw, err := watcher.NewFileWatcher([]string{dir}, false, nil)
	require.NoError(t, err)
	require.NoError(t, fw.Start())
	defer fw.Stop()

	go func() {
		time.Sleep(100 * time.Millisecond)
		testhelpers.WritePDF(t, dir, "novo.pdf", "x")
	}()

	select {
	case ev := <-fw.Events():
		require.Equal(t, watcher.EventCreate, ev.Type)
		require.Equal(t, "novo.pdf", filepath.Base(ev.Path))
	case <-time.After(3 * time.Second):
		t.Fatal("evento de criação não recebido")
	}
}

func TestWatcher_IgnoraArquivosNaoPDF(t *testing.T) {
	dir := t.TempDir()
	fw, err := watcher.NewFileWatcher([]string{dir}, false, nil)
	require.NoError(t, err)
	require.NoError(t, fw.Start())
	defer fw.Stop()

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = os.WriteFile(filepath.Join(dir, "ignorar.txt"), []byte("x"), 0644)
	}()

	select {
	case ev := <-fw.Events():
		t.Fatalf("não deveria ter recebido evento, recebeu: %+v", ev)
	case <-time.After(500 * time.Millisecond):
		// ok, não chegou nenhum evento como esperado
	}
}

func TestWatcher_SubpastaNovaEhMonitoradaQuandoRecursivo(t *testing.T) {
	dir := t.TempDir()
	fw, err := watcher.NewFileWatcher([]string{dir}, true, nil)
	require.NoError(t, err)
	require.NoError(t, fw.Start())
	defer fw.Stop()

	go func() {
		time.Sleep(100 * time.Millisecond)
		sub := filepath.Join(dir, "subnova")
		require.NoError(t, os.Mkdir(sub, 0755))
		// Tempo para o watcher detectar e adicionar a subpasta
		time.Sleep(300 * time.Millisecond)
		testhelpers.WritePDF(t, sub, "interno.pdf", "x")
	}()

	select {
	case ev := <-fw.Events():
		require.Equal(t, "interno.pdf", filepath.Base(ev.Path))
	case <-time.After(3 * time.Second):
		t.Fatal("PDF em subpasta nova não foi detectado")
	}
}