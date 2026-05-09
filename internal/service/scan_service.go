package service

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/engine"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	"github.com/F3rreir4L19/juridico-watcher/internal/watcher"
)

// ErrInactive sinaliza que o alvo do scan está desativado e a operação
// foi pulada (não é erro fatal — a UI mostra mensagem amigável).
var ErrInactive = errors.New("alvo do scan está desativado")

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

// ScanWatch escaneia todos os PDFs em uma pasta monitorada e processa cada um
// contra todas as regras ativas associadas àquela pasta.
//
// Retorna o número de PDFs encontrados e processados (independente do status
// final — sucesso, no_match, no_text, etc. todos contam como "processado").
func (s *ScanService) ScanWatch(watchID int64) (int, error) {
	wr := storage.NewWatchRepo(s.db)
	rr := storage.NewRuleRepo(s.db)
	pr := storage.NewProcessedRepo(s.db)

	watch, err := wr.GetByID(watchID)
	if err != nil {
		return 0, err
	}
	if !watch.Active {
		return 0, ErrInactive
	}

	allRules, err := rr.List()
	if err != nil {
		return 0, err
	}
	applicable := filterRulesForWatch(allRules, watch.ID)

	pdfs, err := watcher.ScanPDFs(watch.Path, watch.Recursive)
	if err != nil {
		return 0, err
	}

	recorder := &processedRecorder{repo: pr}
	count := 0
	for _, pdfPath := range pdfs {
		if _, err := engine.ProcessPDF(pdfPath, applicable, recorder, watch.Path, s.logger); err != nil {
			s.logger.Error("erro no scan", "path", pdfPath, "err", err)
			continue
		}
		count++
	}
	return count, nil
}

// ScanRule escaneia todas as pastas associadas a uma regra específica e
// processa os PDFs encontrados usando APENAS aquela regra (ignora outras
// regras associadas às mesmas pastas).
//
// Útil para "aplicar uma regra recém-criada/editada aos PDFs já existentes"
// sem reprocessar tudo.
//
// Se a regra estiver inativa, retorna ErrInactive sem processar.
func (s *ScanService) ScanRule(ruleID int64) (int, error) {
	rr := storage.NewRuleRepo(s.db)
	wr := storage.NewWatchRepo(s.db)
	pr := storage.NewProcessedRepo(s.db)

	rule, err := rr.GetByID(ruleID)
	if err != nil {
		return 0, err
	}
	if !rule.Active {
		return 0, ErrInactive
	}

	recorder := &processedRecorder{repo: pr}
	count := 0
	for _, watchID := range rule.WatchIDs {
		watch, err := wr.GetByID(watchID)
		if err != nil {
			s.logger.Warn("watch não encontrado durante ScanRule", "watch_id", watchID, "err", err)
			continue
		}
		if !watch.Active {
			continue // pulamos watches inativos silenciosamente
		}
		pdfs, err := watcher.ScanPDFs(watch.Path, watch.Recursive)
		if err != nil {
			s.logger.Error("falha ao listar PDFs", "path", watch.Path, "err", err)
			continue
		}
		for _, pdfPath := range pdfs {
			// Passa lista contendo só essa regra — pipeline avalia somente ela.
			if _, err := engine.ProcessPDF(pdfPath, []*domain.Rule{rule}, recorder, watch.Path, s.logger); err != nil {
				s.logger.Error("erro processando PDF em ScanRule", "path", pdfPath, "err", err)
				continue
			}
			count++
		}
	}
	return count, nil
}

// ScanAll escaneia todas as pastas ativas. Retorna o número total de PDFs
// processados em todas as pastas. Erros em pastas individuais são logados
// mas não interrompem o scan global — o usuário recebe o melhor esforço.
func (s *ScanService) ScanAll() (int, error) {
	wr := storage.NewWatchRepo(s.db)
	watches, err := wr.List()
	if err != nil {
		return 0, fmt.Errorf("listar pastas: %w", err)
	}
	total := 0
	for _, w := range watches {
		if !w.Active {
			continue
		}
		count, err := s.ScanWatch(w.ID)
		if err != nil {
			s.logger.Error("scan parcial falhou", "watch", w.Name, "err", err)
			continue
		}
		total += count
	}
	return total, nil
}

// filterRulesForWatch retorna as regras ativas associadas a um watch.
func filterRulesForWatch(allRules []*domain.Rule, watchID int64) []*domain.Rule {
	var out []*domain.Rule
	for _, r := range allRules {
		if !r.Active {
			continue
		}
		for _, wid := range r.WatchIDs {
			if wid == watchID {
				out = append(out, r)
				break
			}
		}
	}
	return out
}