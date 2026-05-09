package watcher

import "time"

// EventType classifica o tipo de evento do sistema de arquivos.
type EventType int

const (
	EventCreate EventType = iota
	EventRemove
	EventRename
	EventWrite
)

// Event representa um evento de arquivo detectado pelo watcher.
type Event struct {
	Type EventType
	Path string
	Time time.Time
}

// FileWatcher observa diretórios e notifica eventos de arquivos PDF.
type FileWatcher interface {
	// Start inicia a observação. Deve ser chamado antes de obter canais.
	Start() error
	// Stop interrompe a observação e fecha os canais.
	Stop() error
	// Events retorna o canal de eventos.
	Events() <-chan Event
	// Errors retorna o canal de erros internos.
	Errors() <-chan error
}
