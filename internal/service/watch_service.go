package service

import (
	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
)

type WatchService struct {
	watchRepo *storage.WatchRepo
	ruleRepo  *storage.RuleRepo
}

func NewWatchService(watchRepo *storage.WatchRepo, ruleRepo *storage.RuleRepo) *WatchService {
	return &WatchService{watchRepo: watchRepo, ruleRepo: ruleRepo}
}

func (s *WatchService) Create(watch *domain.Watch) error {
	return s.watchRepo.Create(watch)
}

func (s *WatchService) Update(watch *domain.Watch) error {
	return s.watchRepo.Update(watch)
}

// Delete remove a pasta monitorada apenas se não estiver em uso por nenhuma regra.
func (s *WatchService) Delete(id int64) error {
	rules, err := s.ruleRepo.List()
	if err != nil {
		return err
	}
	for _, r := range rules {
		for _, wid := range r.WatchIDs {
			if wid == id {
				return domain.ErrWatchInUse
			}
		}
	}
	return s.watchRepo.Delete(id)
}

func (s *WatchService) GetByID(id int64) (*domain.Watch, error) {
	return s.watchRepo.GetByID(id)
}

func (s *WatchService) List() ([]*domain.Watch, error) {
	return s.watchRepo.List()
}