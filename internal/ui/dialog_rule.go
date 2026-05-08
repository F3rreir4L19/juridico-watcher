package ui

import (
	"fmt"
	"strconv"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/service"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func showRuleDialog(existing *domain.Rule, ruleService *service.RuleService, watchService *service.WatchService, list *widget.List) {
	win := fyne.CurrentApp().Driver().AllWindows()[0]
	isNew := existing == nil
	title := "Adicionar Regra"
	if !isNew {
		title = "Editar Regra"
	}

	nameEntry := widget.NewEntry()
	nameEntry.PlaceHolder = "Nome da regra"
	priorityEntry := widget.NewEntry()
	priorityEntry.Text = "0"
	activeCheck := widget.NewCheck("Ativo", nil)
	activeCheck.Checked = true

	// Carrega watches disponíveis para seleção
	allWatches, err := watchService.List()
	if err != nil {
		allWatches = nil
	}
	var watchChecks []*widget.Check
	watchIDs := make(map[int64]bool)
	if !isNew {
		for _, wid := range existing.WatchIDs {
			watchIDs[wid] = true
		}
		nameEntry.Text = existing.Name
		priorityEntry.Text = strconv.Itoa(existing.Priority)
		activeCheck.Checked = existing.Active
	}

	watchContainer := container.NewVBox()
	for _, w := range allWatches {
		ch := widget.NewCheck(w.Name, nil)
		ch.Checked = watchIDs[w.ID]
		watchChecks = append(watchChecks, ch)
		watchContainer.Add(ch)
	}

	// Sub-forms para Extrações, Condições, Ações
	extractions := []domain.Extraction{}
	conditions := []domain.Condition{}
	actions := []domain.Action{}
	if !isNew {
		extractions = existing.Extractions
		conditions = existing.Conditions
		actions = existing.Actions
	}

	extractForm := buildExtractionsForm(&extractions)
	condForm := buildConditionsForm(&conditions)
	actionForm := buildActionsForm(&actions)

	accordion := widget.NewAccordion(
		widget.NewAccordionItem("Extrações", extractForm),
		widget.NewAccordionItem("Condições", condForm),
		widget.NewAccordionItem("Ações", actionForm),
	)
	accordion.MultiOpen = true

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Nome", Widget: nameEntry},
			{Text: "Prioridade", Widget: priorityEntry},
			{Text: "", Widget: activeCheck},
			{Text: "Pastas monitoradas", Widget: watchContainer},
		},
	}

	content := container.NewVBox(form, widget.NewSeparator(), accordion)
	scroll := container.NewScroll(content)

	dlg := dialog.NewCustomConfirm(title, "Salvar", "Cancelar", scroll, func(save bool) {
		if !save {
			return
		}
		name := nameEntry.Text
		priority, _ := strconv.Atoi(priorityEntry.Text)

		var selectedWatches []int64
		for i, ch := range watchChecks {
			if ch.Checked && i < len(allWatches) {
				selectedWatches = append(selectedWatches, allWatches[i].ID)
			}
		}

		rule := &domain.Rule{
			Name:        name,
			Priority:    priority,
			Active:      activeCheck.Checked,
			WatchIDs:    selectedWatches,
			Extractions: extractions,
			Conditions:  conditions,
			Actions:     actions,
		}
		var err error
		if isNew {
			err = ruleService.Create(rule)
		} else {
			rule.ID = existing.ID
			err = ruleService.Update(rule)
		}
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		list.Refresh()
	}, win)

	dlg.Resize(fyne.NewSize(600, 700))
	dlg.Show()
}

// Helpers para construir os sub-forms de extractions, conditions, actions

func buildExtractionsForm(items *[]domain.Extraction) fyne.CanvasObject {
	label := widget.NewLabel(fmt.Sprintf("%d extrações definidas", len(*items)))
	return container.NewVBox(label)
}

func buildConditionsForm(items *[]domain.Condition) fyne.CanvasObject {
	label := widget.NewLabel(fmt.Sprintf("%d condições definidas", len(*items)))
	return container.NewVBox(label)
}

func buildActionsForm(items *[]domain.Action) fyne.CanvasObject {
	label := widget.NewLabel(fmt.Sprintf("%d ações definidas", len(*items)))
	return container.NewVBox(label)
}