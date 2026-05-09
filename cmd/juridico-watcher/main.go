package main

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	"github.com/F3rreir4L19/juridico-watcher/internal/ui"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	dataDir, err := dataDirPath()
	if err != nil {
		log.Fatalf("não foi possível determinar diretório de dados: %v", err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("não foi possível criar diretório de dados: %v", err)
	}

	dbPath := filepath.Join(dataDir, "data.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		log.Fatalf("não foi possível abrir banco de dados: %v", err)
	}
	defer db.Close()

	logger.Info("juridico-watcher iniciado", "data_dir", dataDir, "db", dbPath)

	app := ui.NewApp(db, logger)
	app.Run()
}

// dataDirPath retorna ~/.juridico-watcher (Linux/Mac) ou
// %USERPROFILE%/.juridico-watcher (Windows). Decisão: usar mesmo diretório
// nos dois SOs simplifica suporte e fica óbvio onde achar.
func dataDirPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".juridico-watcher"), nil
}
