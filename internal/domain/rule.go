package domain

import "time"

// Operator representa o tipo de comparação numa Condition.
type Operator string

const (
	OpEquals     Operator = "eq"
	OpNotEquals  Operator = "neq"
	OpContains   Operator = "contains"
	OpNotContain Operator = "not_contains"
)

// ActionType representa o tipo de ação a ser executada.
type ActionType string

const (
	ActionCreateFolder ActionType = "create_folder"
	ActionMove         ActionType = "move"
	ActionRename       ActionType = "rename"
)

// Extraction define como capturar uma variável a partir do texto do documento.
type Extraction struct {
	ID            int64
	RuleID        int64
	VariableName  string // nome da variável, ex: "nome", "tipo"
	StartDelim    string // delimitador de início (vazio = começo do texto)
	EndDelim      string // delimitador de fim (vazio = fim do texto)
	Order         int    // ordem de aplicação (irrelevante para correção mas bom para UI)
}

// Condition define uma comparação que precisa ser verdadeira para a regra disparar.
type Condition struct {
	ID           int64
	RuleID       int64
	VariableName string
	Operator     Operator
	Value        string
	Order        int
}

// Action define uma operação a ser executada quando a regra dá match.
type Action struct {
	ID     int64
	RuleID int64
	Type   ActionType
	// Target é o argumento da ação:
	// - CreateFolder: nome (ou caminho relativo) da pasta a criar
	// - Move: pasta de destino
	// - Rename: novo nome do arquivo (sem extensão; extensão é preservada)
	Target string
	Order  int
}

// Rule é a entidade principal: combina pastas-alvo, extrações, condições e ações.
type Rule struct {
	ID         int64
	Name       string
	Priority   int           // menor = roda primeiro (decisão D-b)
	Active     bool
	WatchIDs   []int64       // IDs de Watches onde esta regra se aplica
	Extractions []Extraction
	Conditions  []Condition
	Actions     []Action
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
