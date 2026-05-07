package watcher

import (
	"errors"
	"fmt"
	"os"
	"time"
)

var (
	ErrStabilizationTimeout = errors.New("timeout aguardando estabilização do arquivo")
)

// WaitStable aguarda o tamanho do arquivo não mudar por `cycles` verificações
// espaçadas por `interval`. Retorna nil se estabilizar; ErrStabilizationTimeout
// se o tempo total exceder `timeout`.
func WaitStable(filePath string, interval time.Duration, cycles int, timeout time.Duration) error {
	deadline := time.After(timeout)
	var lastSize int64 = -1
	stableCount := 0

	for {
		select {
		case <-deadline:
			return fmt.Errorf("%w: %s", ErrStabilizationTimeout, filePath)
		default:
		}

		info, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				// arquivo foi removido antes de estabilizar → aborta
				return fmt.Errorf("arquivo removido: %w", err)
			}
			return fmt.Errorf("stat do arquivo: %w", err)
		}
		size := info.Size()

		if size == lastSize {
			stableCount++
			if stableCount >= cycles {
				return nil
			}
		} else {
			stableCount = 0
			lastSize = size
		}

		time.Sleep(interval)
	}
}