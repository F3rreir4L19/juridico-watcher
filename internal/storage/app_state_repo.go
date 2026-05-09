package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
)

// AppStateRepo armazena pares chave-valor de estado da aplicação.
// Valores são strings; helpers tipados (GetTime, SetTime) cobrem casos comuns.
type AppStateRepo struct {
	db *sql.DB
}

func NewAppStateRepo(db *sql.DB) *AppStateRepo {
	return &AppStateRepo{db: db}
}

// Get retorna o valor da chave. Se não existir, retorna ("", domain.ErrNotFound).
func (r *AppStateRepo) Get(key string) (string, error) {
	var value string
	err := r.db.QueryRow(`SELECT value FROM app_state WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get app_state %q: %w", key, err)
	}
	return value, nil
}

// Set grava o par. Sobrescreve se já existir.
func (r *AppStateRepo) Set(key, value string) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(`
		INSERT INTO app_state (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, now)
	if err != nil {
		return fmt.Errorf("set app_state %q: %w", key, err)
	}
	return nil
}

// GetTime lê uma chave como time.Time (nanos epoch). Se inexistente, retorna zero
// time + nil — cenário normal pra "primeira vez" sem precisar tratar erro.
func (r *AppStateRepo) GetTime(key string) (time.Time, error) {
	value, err := r.Get(key)
	if errors.Is(err, domain.ErrNotFound) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	nanos, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q (key=%s): %w", value, key, err)
	}
	return time.Unix(0, nanos).UTC(), nil
}

// SetTime grava uma chave como time.Time (nanos epoch).
func (r *AppStateRepo) SetTime(key string, t time.Time) error {
	return r.Set(key, strconv.FormatInt(t.UnixNano(), 10))
}