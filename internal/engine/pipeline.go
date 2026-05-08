package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	pdfpkg "github.com/F3rreir4L19/juridico-watcher/internal/pdf"
	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
)

// PipelineResult contém o resumo do processamento após a execução do pipeline.
type PipelineResult struct {
	RuleID    int64
	RuleName  string
	Variables map[string]string
	Status    domain.ProcessingStatus
	Error     error // preenchido apenas se Status == StatusFailed
}

// ProcessPDF roda o pipeline completo em um arquivo PDF já estabilizado.
// Retorna nil mesmo que nenhuma regra gere erro; os resultados individuais
// são registrados via recorder e também retornados para análise.
//
// O parâmetro baseDir é o diretório raiz para ações com caminhos relativos
// (geralmente a pasta monitorada onde o arquivo foi encontrado).
func ProcessPDF(
	filePath string,
	rules []*domain.Rule,
	recorder ProcessedRecorder,
	baseDir string,
	logger *slog.Logger,
) ([]PipelineResult, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// 1. Calcular hash do conteúdo
	fileHash, err := hashFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("calcular hash: %w", err)
	}

	// 2. Ordenar regras por prioridade (menor = primeiro)
	sorted := make([]*domain.Rule, len(rules))
	copy(sorted, rules)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	// 3. Extrair texto do PDF (caro, faz uma única vez)
	text, err := pdfpkg.ExtractText(filePath)
	if err != nil {
		// Se não tem texto, marcamos como no_text para todas as regras aplicáveis
		if errors.Is(err, domain.ErrNoText) {
			return handleNoText(sorted, fileHash, filePath, recorder, logger)
		}
		// Erro real (ex.: arquivo corrompido) → falha para todas as regras
		return handleFatal(filePath, err, logger)
	}

	// 4. Para cada regra (ordenada) que ainda possa ser processada
	var results []PipelineResult
	originalPath := filePath
	currentPath := originalPath

	for _, rule := range sorted {
		if !rule.Active {
			continue
		}

    	// RN-11: deduplicação por hash + rule
    	already, err := hasBeenProcessedSafe(recorder, fileHash, rule.ID)
    	if err == nil && already {
        	// já processado por esta regra; pula silenciosamente
        	continue
    	}

		// Verifica se o arquivo ainda existe (pode ter sido movido por regra anterior)
		if _, statErr := os.Stat(currentPath); os.IsNotExist(statErr) {
			// Registra como skipped_moved para esta regra
			rec := &domain.ProcessedDoc{
				FileHash:     fileHash,
				OriginalPath: originalPath,
				RuleID:       rule.ID,
				Status:       domain.StatusSkippedMoved,
			}
			_ = recorder.Record(rec)
			results = append(results, PipelineResult{
				RuleID:   rule.ID,
				RuleName: rule.Name,
				Status:   domain.StatusSkippedMoved,
			})
			continue
		}

		// a) Extrai variáveis
		vars := ExtractVariables(text, rule.Extractions, logger)

		// b) Avalia condições
		if !EvaluateConditions(rule.Conditions, vars, logger) {
			rec := &domain.ProcessedDoc{
				FileHash:     fileHash,
				OriginalPath: originalPath,
				RuleID:       rule.ID,
				Status:       domain.StatusNoMatch,
			}
			_ = recorder.Record(rec)
			results = append(results, PipelineResult{
				RuleID:    rule.ID,
				RuleName:  rule.Name,
				Variables: vars,
				Status:    domain.StatusNoMatch,
			})
			continue
		}

		// c) Executa ações sequencialmente
		execErr := executeActions(currentPath, rule.Actions, vars, baseDir, logger)
		if execErr != nil {
			rec := &domain.ProcessedDoc{
				FileHash:     fileHash,
				OriginalPath: originalPath,
				RuleID:       rule.ID,
				Status:       domain.StatusFailed,
				ErrorMsg:     execErr.Error(),
			}
			_ = recorder.Record(rec)
			results = append(results, PipelineResult{
				RuleID:    rule.ID,
				RuleName:  rule.Name,
				Variables: vars,
				Status:    domain.StatusFailed,
				Error:     execErr,
			})
			// Uma ação falhou; não atualiza currentPath, regra seguinte ainda verá o original
			continue
		}

		// Sucesso
		rec := &domain.ProcessedDoc{
			FileHash:     fileHash,
			OriginalPath: originalPath,
			RuleID:       rule.ID,
			Status:       domain.StatusSuccess,
		}
		_ = recorder.Record(rec)
		results = append(results, PipelineResult{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Variables: vars,
			Status:    domain.StatusSuccess,
		})

		// Se alguma ação alterou o caminho (move/rename), atualiza currentPath
		if newPath, changed := pathAfterActions(currentPath, rule.Actions, vars, baseDir); changed {
			currentPath = newPath
		}
	}

	return results, nil
}

func executeActions(originalPath string, actions []domain.Action, vars map[string]string, baseDir string, logger *slog.Logger) (string, error) {
	current := originalPath
	for _, action := range actions {
		target := Interpolate(action.Target, vars, logger)
		newPath, err := ExecuteAction(action, target, current, baseDir, logger)
		if err != nil {
			return current, err
		}
		if newPath != "" {
			current = newPath
		}
	}
	return current, nil
	
}

// pathAfterActions retorna o caminho final após as ações, se alguma modificou.
func pathAfterActions(originalPath string, actions []domain.Action, vars map[string]string, baseDir string) (string, bool) {
	curr := originalPath
	changed := false
	for _, action := range actions {
		target := Interpolate(action.Target, vars, nil)
		switch action.Type {
		case domain.ActionMove:
			if !filepath.IsAbs(target) {
				target = filepath.Join(baseDir, target)
			}
			dest := filepath.Join(target, filepath.Base(curr))
			curr = avoidCollisionDry(dest)
			changed = true
		case domain.ActionRename:
			dir := filepath.Dir(curr)
			ext := filepath.Ext(curr)
			newName := target + ext
			curr = filepath.Join(dir, newName)
			changed = true
		}
	}
	return curr, changed
}

func avoidCollisionDry(path string) string {
	return path
}

func handleNoText(rules []*domain.Rule, hash, path string, recorder ProcessedRecorder, logger *slog.Logger) ([]PipelineResult, error) {
	var results []PipelineResult
	for _, rule := range rules {
		if !rule.Active {
			continue
		}
		rec := &domain.ProcessedDoc{
			FileHash:     hash,
			OriginalPath: path,
			RuleID:       rule.ID,
			Status:       domain.StatusNoText,
		}
		_ = recorder.Record(rec)
		results = append(results, PipelineResult{
			RuleID:   rule.ID,
			RuleName: rule.Name,
			Status:   domain.StatusNoText,
		})
	}
	return results, nil
}

func handleFatal(filePath string, err error, logger *slog.Logger) ([]PipelineResult, error) {
	return nil, fmt.Errorf("erro crítico em %s: %w", filePath, err)
}

func hashFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hasBeenProcessedSafe(recorder ProcessedRecorder, hash string, ruleID int64) (bool, error) {
	return recorder.HasBeenProcessed(hash, ruleID)
}