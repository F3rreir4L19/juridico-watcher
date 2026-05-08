package engine

import "github.com/F3rreir4L19/juridico-watcher/internal/domain"

// ProcessedRecorder define como o engine consulta e registra processamentos.
// O engine não conhece o storage real — recebe esta interface.
type ProcessedRecorder interface {
	Record(doc *domain.ProcessedDoc) error
	HasBeenProcessed(fileHash string, ruleID int64) (bool, error)
}