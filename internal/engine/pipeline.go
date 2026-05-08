package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	pdfpkg "github.com/F3rreir4L19/juridico-watcher/internal/pdf"
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
// Os resultados individuais são registrados via recorder e também retornados
// para análise. Regras já processadas para o mesmo hash são puladas (RN-11).
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
		// Se não tem texto, marca como no_text para todas as regras aplicáveis
		if errors.Is(err, domain.ErrNoText) {
			return handleNoText(sorted, fileHash, filePath, recorder, logger)
		}
		// Erro real (ex.: arquivo corrompido) → falha global
		return nil, fmt.Errorf("erro crítico em %s: %w", filePath, err)
	}

	// 4. Para cada regra (ordenada)
	var results []PipelineResult
	originalPath := filePath
	currentPath := originalPath

	for _, rule := range sorted {
		if !rule.Active {
			continue
		}

		// RN-11: deduplicação por hash + rule
		already, derr := recorder.HasBeenProcessed(fileHash, rule.ID)
		if derr == nil && already {
			// já processado por esta regra; pula silenciosamente
			continue
		}

		// Verifica se o arquivo ainda existe (pode ter sido movido por regra anterior)
		if _, statErr := os.Stat(currentPath); os.IsNotExist(statErr) {
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

		// c) Executa ações sequencialmente; o caminho final retornado já reflete
		// movimentações e renomeações reais (com colisão tratada).
		finalPath, execErr := executeActions(currentPath, rule.Actions, vars, baseDir, logger)
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
			// Ação falhou; mantém currentPath pois execução parou no meio
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

		// Atualiza currentPath para refletir o estado real após as ações.
		currentPath = finalPath
	}

	return results, nil
}

// executeActions executa cada ação da regra na ordem declarada, interpolando os
// targets e propagando o caminho atual entre elas. Retorna o caminho final real
// (após move/rename com colisão tratada) e o erro, se houver.
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

func handleNoText(rules []*domain.Rule, hash, path string, recorder ProcessedRecorder, logger *slog.Logger) ([]PipelineResult, error) {
	var results []PipelineResult
	for _, rule := range rules {
		if !rule.Active {
			continue
		}
		// Mesmo no_text passa por dedup: não registra duas vezes
		already, derr := recorder.HasBeenProcessed(hash, rule.ID)
		if derr == nil && already {
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