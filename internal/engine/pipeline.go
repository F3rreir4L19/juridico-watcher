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

	// movedByPrevious: a partir do momento em que uma regra anterior executou
	// uma ação `move` com sucesso, todas as regras seguintes são puladas com
	// status skipped_moved (RN-02). Não depende de checar o filesystem porque
	// o arquivo real existe no novo lugar — o que importa é a semântica.
	movedByPrevious := false

	for _, rule := range sorted {
		if !rule.Active {
			continue
		}

		// RN-11: deduplicação por hash + rule
		already, derr := recorder.HasBeenProcessed(fileHash, rule.ID)
		if derr == nil && already {
			continue
		}

		// RN-02: regra anterior moveu o arquivo → pula
		if movedByPrevious {
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

		// Sanidade: arquivo ainda existe? (caso seja deletado externamente
		// entre regras, registramos skipped_moved também por consistência)
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
		finalPath, ruleMoved, execErr := executeActions(currentPath, rule.Actions, vars, baseDir, logger)
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
			// Se a falha aconteceu depois de já ter movido, sinaliza para
			// regras seguintes não atuarem em path inválido
			if ruleMoved {
				movedByPrevious = true
				currentPath = finalPath
			}
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

		currentPath = finalPath
		if ruleMoved {
			movedByPrevious = true
		}
	}

	return results, nil
}

// executeActions executa cada ação da regra na ordem declarada, interpolando os
// targets e propagando o caminho atual entre elas. Retorna:
//   - finalPath: caminho real após todas as ações (com colisão tratada)
//   - moved: true se alguma das ações foi um Move executado com sucesso
//   - err: erro da primeira ação que falhou, se houver
func executeActions(originalPath string, actions []domain.Action, vars map[string]string, baseDir string, logger *slog.Logger) (string, bool, error) {
	current := originalPath
	moved := false
	for _, action := range actions {
		target := Interpolate(action.Target, vars, logger)
		newPath, err := ExecuteAction(action, target, current, baseDir, logger)
		if err != nil {
			return current, moved, err
		}
		if newPath != "" {
			current = newPath
		}
		if action.Type == domain.ActionMove {
			moved = true
		}
	}
	return current, moved, nil
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