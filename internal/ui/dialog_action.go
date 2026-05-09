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

// actionTypeLabels mapeia o tipo de ação para um rótulo amigável e descrição.
var actionTypeLabels = []struct {
	Type        domain.ActionType
	Label       string
	Description string
}{
	{
		Type:        domain.ActionCreateFolder,
		Label:       "Criar pasta",
		Description: "Cria uma pasta dentro da pasta monitorada. Use placeholders como {nome} no caminho.",
	},
	{
		Type:        domain.ActionMove,
		Label:       "Mover arquivo",
		Description: "Move o arquivo para uma pasta dentro da pasta monitorada. Após mover, regras seguintes não são executadas.",
	},
	{
		Type:        domain.ActionRename,
		Label:       "Renomear arquivo",
		Description: "Renomeia o arquivo no mesmo lugar. A extensão (.pdf) é preservada automaticamente.",
	},
}

func actionTypeOptions() []string {
	out := make([]string, len(actionTypeLabels))
	for i, a := range actionTypeLabels {
		out[i] = a.Label
	}
	return out
}

func actionTypeByLabel(label string) (domain.ActionType, bool) {
	for _, a := range actionTypeLabels {
		if a.Label == label {
			return a.Type, true
		}
	}
	return "", false
}

func labelByActionType(at domain.ActionType) string {
	for _, e := range actionTypeLabels {
		if e.Type == at {
			return e.Label
		}
	}
	return string(at)
}

func descriptionByActionType(at domain.ActionType) string {
	for _, e := range actionTypeLabels {
		if e.Type == at {
			return e.Description
		}
	}
	return ""
}

// showActionDialog abre o sub-dialog de ação.
//
// availableVars são as variáveis disponíveis para uso como placeholders
// no campo target. Mostradas como hint para o usuário.
func showActionDialog(
	parent fyne.Window,
	existing *domain.Action,
	availableVars []string,
	onSave func(domain.Action),
) {
	isNew := existing == nil
	title := "Adicionar ação"
	if !isNew {
		title = "Editar ação"
	}

	typeSelect := widget.NewSelect(actionTypeOptions(), nil)
	typeSelect.PlaceHolder = "Escolha o tipo de ação"

	targetEntry := widget.NewEntry()

	descLabel := widget.NewLabel("")
	descLabel.Wrapping = fyne.TextWrapWord

	updateDesc := func(label string) {
		t, ok := actionTypeByLabel(label)
		if !ok {
			descLabel.SetText("")
			return
		}
		descLabel.SetText(descriptionByActionType(t))
		// Placeholder do target depende do tipo
		switch t {
		case domain.ActionCreateFolder:
			targetEntry.PlaceHolder = "Ex: {nome}  ou  documentos/{tipo}"
		case domain.ActionMove:
			targetEntry.PlaceHolder = "Ex: {nome}  ou  processados/{tipo}"
		case domain.ActionRename:
			targetEntry.PlaceHolder = "Ex: procuracao_{nome}  (sem extensão)"
		}
		targetEntry.Refresh()
	}
	typeSelect.OnChanged = updateDesc

	if !isNew {
		typeSelect.SetSelected(labelByActionType(existing.Type))
		targetEntry.SetText(existing.Target)
	} else {
		typeSelect.SetSelected(actionTypeLabels[0].Label) // dispara updateDesc
	}

	// Hint de variáveis disponíveis
	var varsHintText string
	if len(availableVars) == 0 {
		varsHintText = "Nenhuma variável extraída ainda. Você pode usar texto fixo como destino, ou adicionar extrações antes."
	} else {
		placeholders := make([]string, len(availableVars))
		for i, v := range availableVars {
			placeholders[i] = "{" + v + "}"
		}
		varsHintText = "Variáveis disponíveis: " + strings.Join(placeholders, ", ")
	}
	varsHint := widget.NewLabel(varsHintText)
	varsHint.Wrapping = fyne.TextWrapWord
	varsHint.TextStyle = fyne.TextStyle{Italic: true}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Tipo de ação", Widget: typeSelect},
			{Text: "Destino", Widget: targetEntry},
		},
	}

	content := container.NewVBox(form, descLabel, varsHint)

	dlg := dialog.NewCustomConfirm(title, "Salvar", "Cancelar", content,
		func(save bool) {
			if !save {
				return
			}
			act, err := buildAction(typeSelect, targetEntry)
			if err != nil {
				dialog.ShowError(err, parent)
				return
			}
			onSave(act)
		}, parent,
	)
	dlg.Resize(fyne.NewSize(580, 420))
	dlg.Show()
}

func buildAction(typeSelect *widget.Select, targetEntry *widget.Entry) (domain.Action, error) {
	if typeSelect.Selected == "" {
		return domain.Action{}, errors.New("Escolha o tipo de ação")
	}
	at, ok := actionTypeByLabel(typeSelect.Selected)
	if !ok {
		return domain.Action{}, fmt.Errorf("Tipo de ação desconhecido: %s", typeSelect.Selected)
	}
	target := strings.TrimSpace(targetEntry.Text)
	if target == "" {
		return domain.Action{}, errors.New("Preencha o destino da ação")
	}
	return domain.Action{
		Type:   at,
		Target: target,
	}, nil
}
