package engine_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteCreateFolder_Absoluto(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "novapasta")
	err := engine.ExecuteCreateFolder(dir, "")
	require.NoError(t, err)
	assert.DirExists(t, dir)
}

func TestExecuteCreateFolder_Relativo(t *testing.T) {
	base := t.TempDir()
	err := engine.ExecuteCreateFolder("sub/dir", base)
	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(base, "sub", "dir"))
}

func TestExecuteMoveFile_SemColisao(t *testing.T) {
	origDir := t.TempDir()
	destBase := t.TempDir()
	filePath := filepath.Join(origDir, "doc.pdf")
	require.NoError(t, os.WriteFile(filePath, []byte("test"), 0644))

	newPath, err := engine.ExecuteMoveFile(filePath, destBase, origDir)
	require.NoError(t, err)
	assert.FileExists(t, newPath)
	assert.True(t, filepath.Dir(newPath) == destBase) // verifica se está no destino
	assert.False(t, fileExists(filePath))             // original não existe mais
}

func TestExecuteMoveFile_Colisao(t *testing.T) {
	origDir := t.TempDir()
	destDir := t.TempDir()
	// Cria arquivo original
	origFile := filepath.Join(origDir, "doc.pdf")
	require.NoError(t, os.WriteFile(origFile, []byte("test"), 0644))
	// Cria arquivo com nome colidido no destino
	collision := filepath.Join(destDir, "doc.pdf")
	require.NoError(t, os.WriteFile(collision, []byte("existing"), 0644))

	newPath, err := engine.ExecuteMoveFile(origFile, destDir, origDir)
	require.NoError(t, err)
	// Deve ter criado "doc (2).pdf"
	expected := filepath.Join(destDir, "doc (2).pdf")
	assert.Equal(t, expected, newPath)
	assert.FileExists(t, expected)
}

func TestExecuteRenameFile(t *testing.T) {
	dir := t.TempDir()
	orig := filepath.Join(dir, "old.pdf")
	require.NoError(t, os.WriteFile(orig, []byte("x"), 0644))
	newPath, err := engine.ExecuteRenameFile(orig, "new")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "new.pdf"), newPath)
	assert.FileExists(t, newPath)
	assert.False(t, fileExists(orig))
}

func TestExecuteAction_CreateFolder(t *testing.T) {
	dir := t.TempDir()
	action := domain.Action{Type: domain.ActionCreateFolder}
	_, err := engine.ExecuteAction(action, "subpasta", "/fake", dir, nil)
	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(dir, "subpasta"))
}

func TestExecuteAction_Move(t *testing.T) {
	origDir := t.TempDir()
	dest := t.TempDir()
	filePath := filepath.Join(origDir, "f.pdf")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0644))
	action := domain.Action{Type: domain.ActionMove}
	newPath, err := engine.ExecuteAction(action, dest, filePath, origDir, nil)
	require.NoError(t, err)
	assert.FileExists(t, newPath)
}

func TestExecuteAction_Rename(t *testing.T) {
	dir := t.TempDir()
	orig := filepath.Join(dir, "orig.pdf")
	require.NoError(t, os.WriteFile(orig, []byte("x"), 0644))
	action := domain.Action{Type: domain.ActionRename}
	newPath, err := engine.ExecuteAction(action, "renamed", orig, dir, nil)
	require.NoError(t, err)
	assert.FileExists(t, newPath)
	assert.Equal(t, filepath.Join(dir, "renamed.pdf"), newPath)
}

func TestExecuteCreateFolder_TargetVazio_RetornaErro(t *testing.T) {
	err := engine.ExecuteCreateFolder("", t.TempDir())
	require.ErrorIs(t, err, engine.ErrEmptyTarget)
}

func TestExecuteCreateFolder_TargetSoEspacos_RetornaErro(t *testing.T) {
	err := engine.ExecuteCreateFolder("   ", t.TempDir())
	require.ErrorIs(t, err, engine.ErrEmptyTarget)
}

func TestExecuteMoveFile_TargetVazio_RetornaErro(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "f.pdf")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0644))
	_, err := engine.ExecuteMoveFile(filePath, "", dir)
	require.ErrorIs(t, err, engine.ErrEmptyTarget)
	assert.FileExists(t, filePath) // não foi tocado
}

func TestExecuteRenameFile_TargetVazio_RetornaErro(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "f.pdf")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0644))
	_, err := engine.ExecuteRenameFile(filePath, "")
	require.ErrorIs(t, err, engine.ErrEmptyTarget)
	assert.FileExists(t, filePath)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
