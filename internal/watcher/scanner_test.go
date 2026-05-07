package watcher_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/F3rreir4L19/juridico-watcher/internal/watcher"
	"github.com/F3rreir4L19/juridico-watcher/test/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanPDFs_Recursivo(t *testing.T) {
	dir := t.TempDir()
	testhelpers.WritePDF(t, dir, "a.pdf", "a")
	sub := filepath.Join(dir, "sub")
	require.NoError(t, os.Mkdir(sub, 0755))
	testhelpers.WritePDF(t, sub, "b.pdf", "b")
	// arquivo não PDF
	require.NoError(t, os.WriteFile(filepath.Join(sub, "c.txt"), []byte("c"), 0644))

	files, err := watcher.ScanPDFs(dir, true)
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestScanPDFs_NaoRecursivo(t *testing.T) {
	dir := t.TempDir()
	testhelpers.WritePDF(t, dir, "root.pdf", "r")
	sub := filepath.Join(dir, "sub")
	require.NoError(t, os.Mkdir(sub, 0755))
	testhelpers.WritePDF(t, sub, "sub.pdf", "s")

	files, err := watcher.ScanPDFs(dir, false)
	require.NoError(t, err)
	assert.Len(t, files, 1)
}