package service

import (
	"database/sql"
	"log/slog"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/engine"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	"github.com/F3rreir4L19/juridico-watcher/internal/watcher"
)

type ScanService struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewScanService(db *sql.DB, logger *slog.Logger) *ScanService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ScanService{db: db, logger: logger}
}

func (s *ScanService) ScanWatch(watchID int64) error {
	wr := storage.NewWatchRepo(s.db)
	rr := storage.NewRuleRepo(s.db)
	pr := storage.NewProcessedRepo(s.db)

	watch, err := wr.GetByID(watchID)
	if err != nil {
		return err
	}
	if !watch.Active {
		return nil
	}

	allRules, err := rr.List()
	if err != nil {
		return err
	}
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

	pdfs, err := watcher.ScanPDFs(watch.Path, watch.Recursive)
	if err != nil {
		return err
	}

	recorder := &processedRecorder{repo: pr}
	baseDir := watch.Path

	for _, pdfPath := range pdfs {
		_, err := engine.ProcessPDF(pdfPath, applicable, recorder, baseDir, s.logger)
		if err != nil {
			s.logger.Error("erro no scan", "path", pdfPath, "err", err)
		}
	}
	return nil
}