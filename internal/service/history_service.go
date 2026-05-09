package service

import (
	"database/sql"
	"time"

	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
)

// historyLastVisitKey é a chave usada em app_state para guardar o timestamp
// da última visita do usuário à aba Histórico.
const historyLastVisitKey = "history_last_visit"

// HistoryService encapsula as consultas necessárias para a aba Histórico.
// Mantemos como service (e não acesso direto a repos da UI) para preservar
// a regra de dependência: ui → service, nunca ui → storage.
type HistoryService struct {
	processedRepo *storage.ProcessedRepo
	stateRepo     *storage.AppStateRepo
	defaultLimit  int
}

func NewHistoryService(db *sql.DB) *HistoryService {
	return &HistoryService{
		processedRepo: storage.NewProcessedRepo(db),
		stateRepo:     storage.NewAppStateRepo(db),
		defaultLimit:  100, // limite razoável para a UI; ajustável se preciso no futuro
	}
}

// List retorna os processamentos mais recentes (até defaultLimit).
func (s *HistoryService) List() ([]*storage.HistoryItem, error) {
	return s.processedRepo.ListRecentWithRuleNames(s.defaultLimit)
}

// CountNewFailures retorna o número de falhas registradas após a última
// visita do usuário à aba Histórico. Se nunca visitou, conta todas as falhas.
func (s *HistoryService) CountNewFailures() (int, error) {
	since, err := s.stateRepo.GetTime(historyLastVisitKey)
	if err != nil {
		return 0, err
	}
	// since.IsZero() significa "nunca visitou" — passamos epoch zero, conta tudo.
	return s.processedRepo.CountFailuresAfter(since)
}

// MarkVisited grava o timestamp atual como última visita à aba Histórico.
// Chamado quando o usuário SAI da aba (não quando entra), para dar tempo de
// olhar as falhas com calma antes de marcar como vistas.
func (s *HistoryService) MarkVisited() error {
	return s.stateRepo.SetTime(historyLastVisitKey, time.Now().UTC())
}