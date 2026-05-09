package domain

import "time"

// Watch representa uma pasta monitorada pelo sistema.
type Watch struct {
	ID        int64
	Name      string // nome amigável, único, usado em referências de regras
	Path      string // caminho absoluto do filesystem
	Active    bool   // se está ativamente sendo monitorada
	Recursive bool   // se monitora subpastas (default true conforme decisão D-k)
	CreatedAt time.Time
	UpdatedAt time.Time
}
