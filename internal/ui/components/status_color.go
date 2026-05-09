package components

import (
	"image/color"

	"fyne.io/fyne/v2/theme"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
)

// ColorForStatus retorna a cor a ser usada no título do TwoLineItem para
// destacar visualmente o status do processamento. Retorna a cor padrão
// (foreground) para status normais e cores destacadas para status de atenção.
func ColorForStatus(s domain.ProcessingStatus) color.Color {
	switch s {
	case domain.StatusFailed:
		return color.NRGBA{R: 200, G: 40, B: 40, A: 255}
	case domain.StatusNoText:
		return color.NRGBA{R: 200, G: 130, B: 0, A: 255}
	case domain.StatusSkippedMoved:
		return theme.Color(theme.ColorNameDisabled)
	case domain.StatusNoMatch:
		return theme.Color(theme.ColorNameDisabled)
	default: // StatusSuccess e fallback
		return theme.Color(theme.ColorNameForeground)
	}
}

// LabelForStatus retorna um rótulo amigável em português para o status.
func LabelForStatus(s domain.ProcessingStatus) string {
	switch s {
	case domain.StatusSuccess:
		return "sucesso"
	case domain.StatusNoMatch:
		return "não correspondeu"
	case domain.StatusFailed:
		return "falha"
	case domain.StatusSkippedMoved:
		return "pulado (já movido)"
	case domain.StatusNoText:
		return "sem texto"
	default:
		return string(s)
	}
}