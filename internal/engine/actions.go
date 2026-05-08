package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
)

// ErrEmptyTarget indica que o target interpolado é vazio. Geralmente significa
// que uma variável usada no template não foi extraída (RN-10).
var ErrEmptyTarget = errors.New("alvo da ação é vazio após interpolação")

// ExecuteCreateFolder cria uma pasta. Se target for relativo, é relativo a baseDir.
// Se a pasta já existir, não é erro.
func ExecuteCreateFolder(target string, baseDir string) error {
	if strings.TrimSpace(target) == "" {
		return ErrEmptyTarget
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(baseDir, target)
	}
	return os.MkdirAll(target, 0755)
}

// ExecuteMoveFile move o arquivo para o diretório targetDir (que pode ser absoluto ou relativo a baseDir).
// Mantém o nome do arquivo. Em caso de colisão, adiciona sufixo numérico (ex: "arquivo (2).pdf").
// Retorna o novo caminho absoluto do arquivo movido.
func ExecuteMoveFile(originalPath, targetDir, baseDir string) (string, error) {
	if strings.TrimSpace(targetDir) == "" {
		return "", ErrEmptyTarget
	}
	if !filepath.IsAbs(targetDir) {
		targetDir = filepath.Join(baseDir, targetDir)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("criar diretório de destino: %w", err)
	}

	baseName := filepath.Base(originalPath)
	destPath := filepath.Join(targetDir, baseName)
	destPath = avoidCollision(destPath)

	if err := os.Rename(originalPath, destPath); err != nil {
		return "", fmt.Errorf("mover arquivo: %w", err)
	}
	return destPath, nil
}

// ExecuteRenameFile renomeia o arquivo no mesmo diretório, usando newBaseName (sem extensão).
// A extensão original é preservada. newBaseName pode conter placeholder já interpolado.
// Colisões são tratadas com sufixo numérico.
// Retorna o novo caminho absoluto.
func ExecuteRenameFile(originalPath, newBaseName string) (string, error) {
	if strings.TrimSpace(newBaseName) == "" {
		return "", ErrEmptyTarget
	}
	dir := filepath.Dir(originalPath)
	ext := filepath.Ext(originalPath)
	newName := newBaseName + ext
	destPath := filepath.Join(dir, newName)
	destPath = avoidCollision(destPath)

	if err := os.Rename(originalPath, destPath); err != nil {
		return "", fmt.Errorf("renomear arquivo: %w", err)
	}
	return destPath, nil
}

// ExecuteAction executa uma única ação, conforme seu tipo, usando o target interpolado.
// Retorna o novo caminho se a ação for move ou rename; para create_folder retorna "".
func ExecuteAction(action domain.Action, target string, originalPath string, baseDir string, logger *slog.Logger) (string, error) {
	switch action.Type {
	case domain.ActionCreateFolder:
		return "", ExecuteCreateFolder(target, baseDir)
	case domain.ActionMove:
		newPath, err := ExecuteMoveFile(originalPath, target, baseDir)
		if err != nil {
			return "", err
		}
		if logger != nil {
			logger.Info("arquivo movido", "de", originalPath, "para", newPath)
		}
		return newPath, nil
	case domain.ActionRename:
		newPath, err := ExecuteRenameFile(originalPath, target)
		if err != nil {
			return "", err
		}
		if logger != nil {
			logger.Info("arquivo renomeado", "de", originalPath, "para", newPath)
		}
		return newPath, nil
	default:
		return "", errors.New("tipo de ação desconhecido")
	}
}

// avoidCollision adiciona sufixo numérico ao caminho se o arquivo já existir.
func avoidCollision(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		newPath := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
	}
}