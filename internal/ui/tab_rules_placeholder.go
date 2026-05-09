package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// newRulesPlaceholder devolve um conteúdo informativo para a aba Regras.
// Será substituído pela aba real na Sprint 9.
func newRulesPlaceholder() fyne.CanvasObject {
	title := widget.NewLabel("Aba Regras — em construção")
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	body := widget.NewLabel(
		"Esta aba permitirá criar regras de organização automática de documentos.\n\n" +
			"Funcionalidade prevista para a próxima atualização.",
	)
	body.Alignment = fyne.TextAlignCenter
	body.Wrapping = fyne.TextWrapWord

	return container.NewCenter(container.NewVBox(title, body))
}
