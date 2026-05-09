package watcher

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// ScanPDFs retorna os caminhos absolutos de todos os arquivos .pdf
// dentro de root, opcionalmente de forma recursiva.
func ScanPDFs(root string, recursive bool) ([]string, error) {
	var files []string
	walker := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // ignora erros de permissão, etc.
		}
		if d.IsDir() {
			if !recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".pdf") {
			abs, err := filepath.Abs(path)
			if err == nil {
				files = append(files, abs)
			}
		}
		return nil
	}
	err := filepath.WalkDir(root, walker)
	return files, err
}
