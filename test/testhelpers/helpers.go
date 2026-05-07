package testhelpers

import (
	"database/sql"
	"path/filepath"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
)

// TempDB cria um SQLite em arquivo temporário com migrations aplicadas.
// O banco é destruído quando o teste termina.
func TempDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// AssertFileExists falha se o arquivo não existe.
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	require.NoError(t, err, "arquivo deveria existir: %s", path)
}

// AssertFileNotExists falha se o arquivo existe.
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err), "arquivo NÃO deveria existir: %s", path)
}

// AssertDirExists falha se o diretório não existe.
func AssertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err, "diretório deveria existir: %s", path)
	require.True(t, info.IsDir(), "%s deveria ser um diretório", path)
}

func writeBytes(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}
