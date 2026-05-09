package service

import (
	"fmt"
	"strings"

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
	if err := validateRule(rule); err != nil {
		return err
	}
	return s.repo.Create(rule)
}

func (s *RuleService) Update(rule *domain.Rule) error {
	if err := validateRule(rule); err != nil {
		return err
	}
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

// validateRule garante invariantes mínimas de domínio que não dependem de UI:
//   - nome não pode ser vazio
//   - regra precisa estar associada a pelo menos uma pasta monitorada
//
// Validações de UX (como "pelo menos uma ação") ficam na camada de UI.
// Aqui é a defesa real contra dados inválidos chegarem ao banco.
func validateRule(rule *domain.Rule) error {
	if rule == nil {
		return fmt.Errorf("%w: regra nula", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(rule.Name) == "" {
		return fmt.Errorf("%w: nome é obrigatório", domain.ErrInvalidInput)
	}
	if len(rule.WatchIDs) == 0 {
		return fmt.Errorf("%w: regra deve estar associada a pelo menos uma pasta", domain.ErrInvalidInput)
	}
	return nil
}
