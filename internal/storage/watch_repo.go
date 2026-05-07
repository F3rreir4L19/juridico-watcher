package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
)

type WatchRepo struct {
	db *sql.DB
}

func NewWatchRepo(db *sql.DB) *WatchRepo {
	return &WatchRepo{db: db}
}

func (r *WatchRepo) Create(w *domain.Watch) error {
	now := time.Now().UTC()
	res, err := r.db.Exec(
		`INSERT INTO watches (name, path, active, recursive, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		w.Name, w.Path, boolToInt(w.Active), boolToInt(w.Recursive), now, now,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return fmt.Errorf("%w: %s", domain.ErrDuplicateName, w.Name)
		}
		return fmt.Errorf("create watch: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}
	w.ID = id
	w.CreatedAt = now
	w.UpdatedAt = now
	return nil
}

func (r *WatchRepo) GetByID(id int64) (*domain.Watch, error) {
	row := r.db.QueryRow(
		`SELECT id, name, path, active, recursive, created_at, updated_at
		 FROM watches WHERE id = ?`, id,
	)
	return scanWatch(row)
}

func (r *WatchRepo) GetByName(name string) (*domain.Watch, error) {
	row := r.db.QueryRow(
		`SELECT id, name, path, active, recursive, created_at, updated_at
		 FROM watches WHERE name = ?`, name,
	)
	return scanWatch(row)
}

func (r *WatchRepo) List() ([]*domain.Watch, error) {
	rows, err := r.db.Query(
		`SELECT id, name, path, active, recursive, created_at, updated_at
		 FROM watches ORDER BY name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list watches: %w", err)
	}
	defer rows.Close()

	var out []*domain.Watch
	for rows.Next() {
		w, err := scanWatchRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *WatchRepo) Update(w *domain.Watch) error {
	now := time.Now().UTC()
	res, err := r.db.Exec(
		`UPDATE watches
		 SET name = ?, path = ?, active = ?, recursive = ?, updated_at = ?
		 WHERE id = ?`,
		w.Name, w.Path, boolToInt(w.Active), boolToInt(w.Recursive), now, w.ID,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return fmt.Errorf("%w: %s", domain.ErrDuplicateName, w.Name)
		}
		return fmt.Errorf("update watch: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	w.UpdatedAt = now
	return nil
}

func (r *WatchRepo) Delete(id int64) error {
	// Verifica se há regras referenciando este watch (decisão D-j)
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM rule_watches WHERE watch_id = ?", id,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("checar regras dependentes: %w", err)
	}
	if count > 0 {
		return domain.ErrWatchInUse
	}

	res, err := r.db.Exec("DELETE FROM watches WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete watch: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// --- helpers internos ---

type rowScanner interface {
	Scan(dest ...any) error
}

func scanWatch(row *sql.Row) (*domain.Watch, error) {
	w, err := doScanWatch(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return w, err
}

func scanWatchRows(row rowScanner) (*domain.Watch, error) {
	return doScanWatch(row)
}

func doScanWatch(row rowScanner) (*domain.Watch, error) {
	var w domain.Watch
	var active, recursive int
	err := row.Scan(&w.ID, &w.Name, &w.Path, &active, &recursive, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	w.Active = active != 0
	w.Recursive = recursive != 0
	return &w, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	// modernc.org/sqlite retorna mensagens com "UNIQUE constraint failed"
	return strings.Contains(err.Error(), "UNIQUE constraint failed") ||
		strings.Contains(err.Error(), "constraint failed: UNIQUE")
}
