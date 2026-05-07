package pdf

import (
	"strings"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/ledongthuc/pdf"
)

func ExtractText(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf strings.Builder
	totalPages := r.NumPage()

	for i := 1; i <= totalPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(text)
	}

	result := strings.TrimSpace(buf.String())
	if result == "" {
		return "", domain.ErrNoText
	}
	return result, nil
}