-- Habilita foreign keys (SQLite não habilita por padrão)
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS watches (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL UNIQUE,
    path        TEXT    NOT NULL,
    active      INTEGER NOT NULL DEFAULT 1,
    recursive   INTEGER NOT NULL DEFAULT 1,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_watches_active ON watches(active);

CREATE TABLE IF NOT EXISTS rules (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL UNIQUE,
    priority   INTEGER NOT NULL DEFAULT 100,
    active     INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rules_active_priority ON rules(active, priority);

-- Relação N:N entre rules e watches
CREATE TABLE IF NOT EXISTS rule_watches (
    rule_id  INTEGER NOT NULL,
    watch_id INTEGER NOT NULL,
    PRIMARY KEY (rule_id, watch_id),
    FOREIGN KEY (rule_id)  REFERENCES rules(id)   ON DELETE CASCADE,
    FOREIGN KEY (watch_id) REFERENCES watches(id) ON DELETE RESTRICT
    -- ON DELETE RESTRICT garante decisão D-j: não dá pra deletar watch se está em regra
);

CREATE INDEX IF NOT EXISTS idx_rule_watches_watch ON rule_watches(watch_id);

CREATE TABLE IF NOT EXISTS extractions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id       INTEGER NOT NULL,
    variable_name TEXT    NOT NULL,
    start_delim   TEXT    NOT NULL DEFAULT '',
    end_delim     TEXT    NOT NULL DEFAULT '',
    sort_order    INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (rule_id) REFERENCES rules(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_extractions_rule ON extractions(rule_id);

CREATE TABLE IF NOT EXISTS conditions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id       INTEGER NOT NULL,
    variable_name TEXT    NOT NULL,
    operator      TEXT    NOT NULL,
    value         TEXT    NOT NULL,
    sort_order    INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (rule_id) REFERENCES rules(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_conditions_rule ON conditions(rule_id);

CREATE TABLE IF NOT EXISTS actions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id    INTEGER NOT NULL,
    type       TEXT    NOT NULL,
    target     TEXT    NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (rule_id) REFERENCES rules(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_actions_rule ON actions(rule_id);

CREATE TABLE IF NOT EXISTS processed_documents (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    file_hash     TEXT    NOT NULL,
    original_path TEXT    NOT NULL,
    rule_id       INTEGER NOT NULL,
    status        TEXT    NOT NULL,
    error_msg     TEXT    NOT NULL DEFAULT '',
    processed_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (rule_id) REFERENCES rules(id) ON DELETE CASCADE,
    UNIQUE (file_hash, rule_id)
);

CREATE INDEX IF NOT EXISTS idx_processed_hash ON processed_documents(file_hash);
CREATE INDEX IF NOT EXISTS idx_processed_rule ON processed_documents(rule_id);