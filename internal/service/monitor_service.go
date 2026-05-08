package service

import (
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/engine"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	"github.com/F3rreir4L19/juridico-watcher/internal/watcher"
)

type MonitorService struct {
	db          *sql.DB
	logger      *slog.Logger
	mu          sync.Mutex
	watchers    map[int64]watcher.FileWatcher
	cancelFuncs map[int64]context.CancelFunc
}

func NewMonitorService(db *sql.DB, logger *slog.Logger) *MonitorService {
	if logger == nil {
		logger = slog.Default()
	}
	return &MonitorService{
		db:          db,
		logger:      logger,
		watchers:    make(map[int64]watcher.FileWatcher),
		cancelFuncs: make(map[int64]context.CancelFunc),
	}
}

func (m *MonitorService) StartMonitoring() error {
	wr := storage.NewWatchRepo(m.db)
	rr := storage.NewRuleRepo(m.db)

	watches, err := wr.List()
	if err != nil {
		return err
	}
	rules, err := rr.List()
	if err != nil {
		return err
	}

	for _, w := range watches {
		if !w.Active {
			continue
		}
		if err := m.startWatch(w, rules); err != nil {
			m.logger.Error("falha ao iniciar monitoramento", "watch", w.Name, "err", err)
		}
	}
	return nil
}

func (m *MonitorService) startWatch(watch *domain.Watch, allRules []*domain.Rule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.watchers[watch.ID]; exists {
		return nil
	}

	fw, err := watcher.NewFileWatcher([]string{watch.Path}, watch.Recursive, m.logger)
	if err != nil {
		return err
	}
	if err := fw.Start(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.watchers[watch.ID] = fw
	m.cancelFuncs[watch.ID] = cancel

	go m.handleEvents(ctx, watch, fw, allRules)
	go m.handleErrors(ctx, fw)
	return nil
}

func (m *MonitorService) StopMonitoring(watchID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fw, ok := m.watchers[watchID]
	if !ok {
		return nil
	}
	cancel := m.cancelFuncs[watchID]
	cancel()
	fw.Stop()
	delete(m.watchers, watchID)
	delete(m.cancelFuncs, watchID)
	return nil
}

func (m *MonitorService) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id := range m.watchers {
		m.cancelFuncs[id]()
		m.watchers[id].Stop()
	}
	m.watchers = make(map[int64]watcher.FileWatcher)
	m.cancelFuncs = make(map[int64]context.CancelFunc)
}

func (m *MonitorService) handleEvents(ctx context.Context, watch *domain.Watch, fw watcher.FileWatcher, allRules []*domain.Rule) {
	var applicable []*domain.Rule
	for _, r := range allRules {
		if !r.Active {
			continue
		}
		for _, wid := range r.WatchIDs {
			if wid == watch.ID {
				applicable = append(applicable, r)
				break
			}
		}
	}
	if len(applicable) == 0 {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-fw.Events():
			if !ok {
				return
			}
			if event.Type != watcher.EventCreate {
				continue
			}
			go m.processFile(ctx, event.Path, watch, applicable)
		}
	}
}

func (m *MonitorService) handleErrors(ctx context.Context, fw watcher.FileWatcher) {
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-fw.Errors():
			if !ok {
				return
			}
			m.logger.Error("erro do watcher", "err", err)
		}
	}
}

func (m *MonitorService) processFile(ctx context.Context, filePath string, watch *domain.Watch, rules []*domain.Rule) {
	if err := watcher.WaitStable(filePath, 500*time.Millisecond, 2, 30*time.Second); err != nil {
		m.logger.Warn("arquivo não estabilizou", "path", filePath, "err", err)
		return
	}

	pr := storage.NewProcessedRepo(m.db)
	recorder := &processedRecorder{repo: pr}

	baseDir := watch.Path
	if abs, err := filepath.Abs(baseDir); err == nil {
		baseDir = abs
	}

	results, err := engine.ProcessPDF(filePath, rules, recorder, baseDir, m.logger)
	if err != nil {
		m.logger.Error("pipeline falhou", "path", filePath, "err", err)
		return
	}
	for _, res := range results {
		if res.Status == domain.StatusFailed {
			m.logger.Error("ação falhou", "rule", res.RuleName, "err", res.Error)
		}
	}
}

type processedRecorder struct {
	repo *storage.ProcessedRepo
}

func (r *processedRecorder) Record(doc *domain.ProcessedDoc) error {
	return r.repo.Record(doc)
}

func (r *processedRecorder) HasBeenProcessed(hash string, ruleID int64) (bool, error) {
	return r.repo.HasBeenProcessed(hash, ruleID)
}