package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	"github.com/F3rreir4L19/juridico-watcher/internal/ui"
)

func main() {
	// Diretório de dados: %APPDATA%/juridico-watcher no Windows, ~/.juridico-watcher no Linux
	dataDir, err := dataDirPath()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal(err)
	}

	dbPath := filepath.Join(dataDir, "data.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := ui.NewApp(db)
	app.Run()
}

func dataDirPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".juridico-watcher"), nil
}