package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open abre (ou cria) o banco SQLite no caminho fornecido,
// aplica todas as migrations pendentes e retorna a conexão.
//
// Use ":memory:" como path para um banco em memória (útil para testes).
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("abrir banco: %w", err)
	}

	// Verifica conectividade imediatamente
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	if err := applyMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("aplicar migrations: %w", err)
	}

	return db, nil
}
