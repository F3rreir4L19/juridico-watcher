package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
)

type RuleRepo struct {
	db *sql.DB
}

func NewRuleRepo(db *sql.DB) *RuleRepo {
	return &RuleRepo{db: db}
}

func (r *RuleRepo) Create(rule *domain.Rule) error {
	now := time.Now().UTC()
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO rules (name, priority, active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		rule.Name, rule.Priority, boolToInt(rule.Active), now, now,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return fmt.Errorf("%w: %s", domain.ErrDuplicateName, rule.Name)
		}
		return fmt.Errorf("insert rule: %w", err)
	}
	rule.ID, _ = res.LastInsertId()

	if err := insertRuleChildren(tx, rule); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	rule.CreatedAt = now
	rule.UpdatedAt = now
	return nil
}

func (r *RuleRepo) GetByID(id int64) (*domain.Rule, error) {
	row := r.db.QueryRow(
		`SELECT id, name, priority, active, created_at, updated_at
		 FROM rules WHERE id = ?`, id,
	)
	rule, err := scanRule(row)
	if err != nil {
		return nil, err
	}
	if err := r.loadChildren(rule); err != nil {
		return nil, err
	}
	return rule, nil
}

func (r *RuleRepo) List() ([]*domain.Rule, error) {
	rows, err := r.db.Query(
		`SELECT id, name, priority, active, created_at, updated_at
		 FROM rules ORDER BY priority ASC, name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	var out []*domain.Rule
	for rows.Next() {
		rule, err := doScanRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, rule := range out {
		if err := r.loadChildren(rule); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ListByWatchID retorna todas as regras ativas que se aplicam a um watch específico,
// ordenadas por prioridade. Usado pelo motor de regras (Sprint 4).
func (r *RuleRepo) ListByWatchID(watchID int64) ([]*domain.Rule, error) {
	rows, err := r.db.Query(`
		SELECT r.id, r.name, r.priority, r.active, r.created_at, r.updated_at
		FROM rules r
		INNER JOIN rule_watches rw ON rw.rule_id = r.id
		WHERE rw.watch_id = ? AND r.active = 1
		ORDER BY r.priority ASC, r.name ASC
	`, watchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Rule
	for rows.Next() {
		rule, err := doScanRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, rule := range out {
		if err := r.loadChildren(rule); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *RuleRepo) Update(rule *domain.Rule) error {
	now := time.Now().UTC()
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`UPDATE rules SET name = ?, priority = ?, active = ?, updated_at = ?
		 WHERE id = ?`,
		rule.Name, rule.Priority, boolToInt(rule.Active), now, rule.ID,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return fmt.Errorf("%w: %s", domain.ErrDuplicateName, rule.Name)
		}
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}

	// Estratégia simples: deleta filhos antigos e reinsere.
	// Como Rule não é gigante, isso é OK e mantém o código simples.
	for _, table := range []string{"extractions", "conditions", "actions", "rule_watches"} {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE rule_id = ?", table), rule.ID); err != nil {
			return fmt.Errorf("limpar %s: %w", table, err)
		}
	}

	if err := insertRuleChildren(tx, rule); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	rule.UpdatedAt = now
	return nil
}

func (r *RuleRepo) Delete(id int64) error {
	res, err := r.db.Exec("DELETE FROM rules WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// --- internos ---

func insertRuleChildren(tx *sql.Tx, rule *domain.Rule) error {
	for _, watchID := range rule.WatchIDs {
		if _, err := tx.Exec(
			`INSERT INTO rule_watches (rule_id, watch_id) VALUES (?, ?)`,
			rule.ID, watchID,
		); err != nil {
			return fmt.Errorf("inserir rule_watches: %w", err)
		}
	}

	for i := range rule.Extractions {
		ex := &rule.Extractions[i]
		ex.RuleID = rule.ID
		ex.Order = i
		res, err := tx.Exec(
			`INSERT INTO extractions (rule_id, variable_name, start_delim, end_delim, sort_order)
			 VALUES (?, ?, ?, ?, ?)`,
			ex.RuleID, ex.VariableName, ex.StartDelim, ex.EndDelim, ex.Order,
		)
		if err != nil {
			return fmt.Errorf("inserir extraction: %w", err)
		}
		ex.ID, _ = res.LastInsertId()
	}

	for i := range rule.Conditions {
		c := &rule.Conditions[i]
		c.RuleID = rule.ID
		c.Order = i
		res, err := tx.Exec(
			`INSERT INTO conditions (rule_id, variable_name, operator, value, sort_order)
			 VALUES (?, ?, ?, ?, ?)`,
			c.RuleID, c.VariableName, string(c.Operator), c.Value, c.Order,
		)
		if err != nil {
			return fmt.Errorf("inserir condition: %w", err)
		}
		c.ID, _ = res.LastInsertId()
	}

	for i := range rule.Actions {
		a := &rule.Actions[i]
		a.RuleID = rule.ID
		a.Order = i
		res, err := tx.Exec(
			`INSERT INTO actions (rule_id, type, target, sort_order)
			 VALUES (?, ?, ?, ?)`,
			a.RuleID, string(a.Type), a.Target, a.Order,
		)
		if err != nil {
			return fmt.Errorf("inserir action: %w", err)
		}
		a.ID, _ = res.LastInsertId()
	}

	return nil
}

func (r *RuleRepo) loadChildren(rule *domain.Rule) error {
	// Extractions
	rows, err := r.db.Query(
		`SELECT id, rule_id, variable_name, start_delim, end_delim, sort_order
		 FROM extractions WHERE rule_id = ? ORDER BY sort_order ASC`, rule.ID,
	)
	if err != nil {
		return err
	}
	for rows.Next() {
		var ex domain.Extraction
		if err := rows.Scan(&ex.ID, &ex.RuleID, &ex.VariableName, &ex.StartDelim, &ex.EndDelim, &ex.Order); err != nil {
			rows.Close()
			return err
		}
		rule.Extractions = append(rule.Extractions, ex)
	}
	rows.Close()

	// Conditions
	rows, err = r.db.Query(
		`SELECT id, rule_id, variable_name, operator, value, sort_order
		 FROM conditions WHERE rule_id = ? ORDER BY sort_order ASC`, rule.ID,
	)
	if err != nil {
		return err
	}
	for rows.Next() {
		var c domain.Condition
		var op string
		if err := rows.Scan(&c.ID, &c.RuleID, &c.VariableName, &op, &c.Value, &c.Order); err != nil {
			rows.Close()
			return err
		}
		c.Operator = domain.Operator(op)
		rule.Conditions = append(rule.Conditions, c)
	}
	rows.Close()

	// Actions
	rows, err = r.db.Query(
		`SELECT id, rule_id, type, target, sort_order
		 FROM actions WHERE rule_id = ? ORDER BY sort_order ASC`, rule.ID,
	)
	if err != nil {
		return err
	}
	for rows.Next() {
		var a domain.Action
		var typ string
		if err := rows.Scan(&a.ID, &a.RuleID, &typ, &a.Target, &a.Order); err != nil {
			rows.Close()
			return err
		}
		a.Type = domain.ActionType(typ)
		rule.Actions = append(rule.Actions, a)
	}
	rows.Close()

	// WatchIDs
	rows, err = r.db.Query(
		`SELECT watch_id FROM rule_watches WHERE rule_id = ?`, rule.ID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var wid int64
		if err := rows.Scan(&wid); err != nil {
			return err
		}
		rule.WatchIDs = append(rule.WatchIDs, wid)
	}
	return rows.Err()
}

func scanRule(row *sql.Row) (*domain.Rule, error) {
	rule, err := doScanRule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return rule, err
}

func doScanRule(row rowScanner) (*domain.Rule, error) {
	var rule domain.Rule
	var active int
	err := row.Scan(&rule.ID, &rule.Name, &rule.Priority, &active, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return nil, err
	}
	rule.Active = active != 0
	return &rule, nil
}
