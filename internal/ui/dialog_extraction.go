package ui

import (
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
)

// showExtractionDialog abre um sub-dialog de extração.
//
// Se existing == nil, é modo "Adicionar".
// Senão, é modo "Editar" e o sub-dialog vem preenchido.
//
// Ao salvar com sucesso, chama onSave com a Extraction (campos preenchidos,
// sem ID/RuleID — esses são gerenciados pelo dialog principal/repo).
//
// existingNames são as variáveis já em uso no escopo da regra atual; usadas
// para alertar sobre duplicação. excludeName permite ignorar o próprio nome
// quando estamos editando (para não acusar a si mesmo).
func showExtractionDialog(
	parent fyne.Window,
	existing *domain.Extraction,
	existingNames []string,
	excludeName string,
	onSave func(domain.Extraction),
) {
	isNew := existing == nil
	title := "Adicionar extração"
	if !isNew {
		title = "Editar extração"
	}

	nameEntry := widget.NewEntry()
	nameEntry.PlaceHolder = "Ex: nome, tipo, rg"

	startEntry := widget.NewEntry()
	startEntry.PlaceHolder = "Texto que aparece antes (vazio = início do documento)"

	endEntry := widget.NewEntry()
	endEntry.PlaceHolder = "Texto que aparece depois (vazio = fim do documento)"

	if !isNew {
		nameEntry.SetText(existing.VariableName)
		startEntry.SetText(existing.StartDelim)
		endEntry.SetText(existing.EndDelim)
	}

	hint := widget.NewLabel(
		"A variável recebe o texto que aparece entre os delimitadores. " +
			"Exemplo: para extrair o nome de \"Outorgante: João RG\", use \"Outorgante: \" como início e \" RG\" como fim.",
	)
	hint.Wrapping = fyne.TextWrapWord

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Nome da variável", Widget: nameEntry, HintText: "Sem espaços; será usada como {nome}"},
			{Text: "Delimitador inicial", Widget: startEntry},
			{Text: "Delimitador final", Widget: endEntry},
		},
	}

	content := container.NewVBox(form, hint)

	dlg := dialog.NewCustomConfirm(title, "Salvar", "Cancelar", content,
		func(save bool) {
			if !save {
				return
			}
			ex, err := buildExtraction(nameEntry, startEntry, endEntry, existingNames, excludeName)
			if err != nil {
				dialog.ShowError(err, parent)
				return
			}
			onSave(ex)
		}, parent,
	)
	dlg.Resize(fyne.NewSize(560, 380))
	dlg.Show()
}

func buildExtraction(
	nameEntry, startEntry, endEntry *widget.Entry,
	existingNames []string,
	excludeName string,
) (domain.Extraction, error) {
	name := strings.TrimSpace(nameEntry.Text)
	if name == "" {
		return domain.Extraction{}, errors.New("Por favor, preencha o nome da variável")
	}
	if strings.ContainsAny(name, " \t\n\r{}") {
		return domain.Extraction{}, errors.New("O nome da variável não pode conter espaços ou os caracteres { }")
	}
	for _, existing := range existingNames {
		if existing == excludeName {
			continue
		}
		if strings.EqualFold(existing, name) {
			return domain.Extraction{}, fmt.Errorf("Já existe uma extração com o nome %q", name)
		}
	}
	return domain.Extraction{
		VariableName: name,
		// Importante: NÃO usar TrimSpace nos delimitadores — espaços
		// são frequentemente parte do delimitador (ex.: "Outorgante: ").
		StartDelim: startEntry.Text,
		EndDelim:   endEntry.Text,
	}, nil
}
