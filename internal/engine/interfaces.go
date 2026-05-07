package engine

import "github.com/F3rreir4L19/juridico-watcher/internal/domain"

// ProcessedRecorder define como registrar o resultado do processamento de um documento.
type ProcessedRecorder interface {
	Record(doc *domain.ProcessedDoc) error
}