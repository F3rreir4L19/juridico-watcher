package watcher

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type watcherImpl struct {
	mu        sync.Mutex
	watcher   *fsnotify.Watcher
	dirs      map[string]struct{} // diretórios já adicionados (evitar duplicados)
	events    chan Event
	errors    chan error
	done      chan struct{}
	recursive bool
	logger    *slog.Logger
}

// NewFileWatcher cria um observador que monitora as pastas fornecidas.
// O parâmetro recursive indica se subpastas também serão observadas.
func NewFileWatcher(paths []string, recursive bool, logger *slog.Logger) (FileWatcher, error) {
	if logger == nil {
		logger = slog.Default()
	}
	fsWatch, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("criar fsnotify watcher: %w", err)
	}
	w := &watcherImpl{
		watcher:   fsWatch,
		dirs:      make(map[string]struct{}),
		events:    make(chan Event, 100),
		errors:    make(chan error, 10),
		done:      make(chan struct{}),
		recursive: recursive,
		logger:    logger,
	}
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			fsWatch.Close()
			return nil, fmt.Errorf("resolver caminho %q: %w", p, err)
		}
		if err := w.addDir(abs); err != nil {
			fsWatch.Close()
			return nil, err
		}
	}
	return w, nil
}

func (w *watcherImpl) Start() error {
	go w.loop()
	return nil
}

func (w *watcherImpl) Stop() error {
	close(w.done)
	return w.watcher.Close()
}

func (w *watcherImpl) Events() <-chan Event {
	return w.events
}

func (w *watcherImpl) Errors() <-chan error {
	return w.errors
}

// addDir adiciona um diretório ao observador e, se recursivo, suas subpastas.
func (w *watcherImpl) addDir(dir string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, ok := w.dirs[dir]; ok {
		return nil
	}
	if err := w.watcher.Add(dir); err != nil {
		return fmt.Errorf("adicionar diretório %q: %w", dir, err)
	}
	w.dirs[dir] = struct{}{}

	if w.recursive {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("ler diretório %q: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				sub := filepath.Join(dir, entry.Name())
				_ = w.addDirLocked(sub)
			}
		}
	}
	return nil
}

// addDirLocked é a versão de addDir que assume que o lock já está adquirido.
// Usada na recursão interna de addDir para não deadlocar.
func (w *watcherImpl) addDirLocked(dir string) error {
	if _, ok := w.dirs[dir]; ok {
		return nil
	}
	if err := w.watcher.Add(dir); err != nil {
		return fmt.Errorf("adicionar diretório %q: %w", dir, err)
	}
	w.dirs[dir] = struct{}{}

	if w.recursive {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("ler diretório %q: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				sub := filepath.Join(dir, entry.Name())
				_ = w.addDirLocked(sub)
			}
		}
	}
	return nil
}

func (w *watcherImpl) loop() {
	for {
		select {
		case <-w.done:
			return
		case rawEvent, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(rawEvent)
		case rawErr, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.errors <- rawErr
		}
	}
}

func (w *watcherImpl) handleEvent(raw fsnotify.Event) {
	// Apenas criação ou rename interessam
	if raw.Op&fsnotify.Create == 0 && raw.Op&fsnotify.Rename == 0 {
		return
	}
	path := filepath.Clean(raw.Name)

	info, err := os.Stat(path)
	if err != nil {
		// Arquivo foi removido entre o evento e o stat — ignora silenciosamente
		return
	}

	// Diretório novo: se estamos em modo recursivo, adiciona ao observador
	if info.IsDir() {
		if w.recursive && raw.Op&fsnotify.Create != 0 {
			if err := w.addDir(path); err != nil {
				w.logger.Warn("falha ao adicionar subpasta nova", "path", path, "err", err)
			}
		}
		return
	}

	// Arquivo: só interessam .pdf
	if !strings.EqualFold(filepath.Ext(path), ".pdf") {
		return
	}

	w.events <- Event{
		Type: EventCreate,
		Path: path,
		Time: time.Now(),
	}
}
