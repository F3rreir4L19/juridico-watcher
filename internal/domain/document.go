package domain

import "time"

// ProcessingStatus representa o resultado do processamento de um documento por uma regra.
type ProcessingStatus string

const (
	StatusSuccess      ProcessingStatus = "success"
	StatusNoMatch      ProcessingStatus = "no_match"      // regra avaliada mas condição não bateu
	StatusFailed       ProcessingStatus = "failed"        // erro ao executar ação
	StatusSkippedMoved ProcessingStatus = "skipped_moved" // arquivo movido por regra anterior
	StatusNoText       ProcessingStatus = "no_text"       // PDF sem texto extraível
)

// ProcessedDoc registra que um arquivo foi avaliado por uma regra.
// Chave lógica: (FileHash, RuleID).
type ProcessedDoc struct {
	ID           int64
	FileHash     string // SHA-256 do conteúdo
	OriginalPath string // caminho onde estava quando foi processado
	RuleID       int64
	Status       ProcessingStatus
	ErrorMsg     string // preenchido quando Status = failed
	ProcessedAt  time.Time
}
