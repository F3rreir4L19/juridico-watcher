package service

import (
	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
)

type RuleService struct {
	repo *storage.RuleRepo
}

func NewRuleService(repo *storage.RuleRepo) *RuleService {
	return &RuleService{repo: repo}
}

func (s *RuleService) Create(rule *domain.Rule) error {
	// Validações básicas (ex.: nome obrigatório) podem ser adicionadas aqui.
	return s.repo.Create(rule)
}

func (s *RuleService) Update(rule *domain.Rule) error {
	return s.repo.Update(rule)
}

func (s *RuleService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *RuleService) GetByID(id int64) (*domain.Rule, error) {
	return s.repo.GetByID(id)
}

func (s *RuleService) List() ([]*domain.Rule, error) {
	return s.repo.List()
}