-- Tabela genérica de estado da aplicação (key-value).
-- Uso atual: rastrear última visita do usuário à aba Histórico para destacar
-- falhas novas. No futuro pode armazenar outras preferências persistentes
-- (último diretório usado em pickers, configurações de UI, etc).
CREATE TABLE IF NOT EXISTS app_state (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);s