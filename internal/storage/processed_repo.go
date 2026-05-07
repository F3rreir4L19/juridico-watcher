package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
)

type ProcessedRepo struct {
	db *sql.DB
}

func NewProcessedRepo(db *sql.DB) *ProcessedRepo {
	return &ProcessedRepo{db: db}
}

// Record cria um registro de processamento. Se já existir (mesmo hash + rule),
// retorna nil silenciosamente — o motor de regras consulta antes de chamar Record,
// mas a UNIQUE constraint serve como segurança extra.
func (r *ProcessedRepo) Record(p *domain.ProcessedDoc) error {
	now := time.Now().UTC()
	res, err := r.db.Exec(
		`INSERT INTO processed_documents (file_hash, original_path, rule_id, status, error_msg, processed_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.FileHash, p.OriginalPath, p.RuleID, string(p.Status), p.ErrorMsg, now,
	)
	if err != nil {
		// Race condition: outro processo registrou primeiro. OK.
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil
		}
		return fmt.Errorf("record processed: %w", err)
	}
	p.ID, _ = res.LastInsertId()
	p.ProcessedAt = now
	return nil
}

// HasBeenProcessed retorna true se este hash já foi processado por esta regra.
func (r *ProcessedRepo) HasBeenProcessed(fileHash string, ruleID int64) (bool, error) {
	var exists int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM processed_documents WHERE file_hash = ? AND rule_id = ?`,
		fileHash, ruleID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// GetByHash retorna todos os processamentos de um arquivo (todas as regras).
func (r *ProcessedRepo) GetByHash(fileHash string) ([]*domain.ProcessedDoc, error) {
	rows, err := r.db.Query(
		`SELECT id, file_hash, original_path, rule_id, status, error_msg, processed_at
		 FROM processed_documents WHERE file_hash = ? ORDER BY processed_at DESC`,
		fileHash,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.ProcessedDoc
	for rows.Next() {
		var p domain.ProcessedDoc
		var status string
		if err := rows.Scan(&p.ID, &p.FileHash, &p.OriginalPath, &p.RuleID, &status, &p.ErrorMsg, &p.ProcessedAt); err != nil {
			return nil, err
		}
		p.Status = domain.ProcessingStatus(status)
		out = append(out, &p)
	}
	return out, rows.Err()
}

// ListRecent retorna os N processamentos mais recentes (para histórico na UI).
func (r *ProcessedRepo) ListRecent(limit int) ([]*domain.ProcessedDoc, error) {
	rows, err := r.db.Query(
		`SELECT id, file_hash, original_path, rule_id, status, error_msg, processed_at
		 FROM processed_documents ORDER BY processed_at DESC, id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.ProcessedDoc
	for rows.Next() {
		var p domain.ProcessedDoc
		var status string
		if err := rows.Scan(&p.ID, &p.FileHash, &p.OriginalPath, &p.RuleID, &status, &p.ErrorMsg, &p.ProcessedAt); err != nil {
			return nil, err
		}
		p.Status = domain.ProcessingStatus(status)
		out = append(out, &p)
	}
	return out, rows.Err()
}

// suppress unused warning
var _ = errors.New
var _ = sql.ErrNoRows
