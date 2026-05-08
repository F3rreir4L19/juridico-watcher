package ui

import (
	"database/sql"
	"fmt"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/service"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func newTabRules(db *sql.DB) fyne.CanvasObject {
	ruleService := service.NewRuleService(storage.NewRuleRepo(db))
	watchService := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))

	var rules []*domain.Rule
	list := widget.NewList(
		func() int {
			var err error
			rules, err = ruleService.List()
			if err != nil {
				return 0
			}
			return len(rules)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Modelo de item")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(rules) {
				r := rules[id]
				label := obj.(*widget.Label)
				status := "✓"
				if !r.Active {
					status = "✗"
				}
				label.SetText(fmt.Sprintf("%s [%d] %s — %d ext / %d cond / %d ações",
					status, r.Priority, r.Name, len(r.Extractions), len(r.Conditions), len(r.Actions)))
			}
		},
	)

	addBtn := widget.NewButtonWithIcon("Adicionar", theme.ContentAddIcon(), func() {
		showRuleDialog(nil, ruleService, watchService, list)
	})

	editBtn := widget.NewButtonWithIcon("Editar", theme.DocumentCreateIcon(), func() {
		if len(rules) == 0 {
			dialog.ShowInformation("Aviso", "Nenhuma regra para editar.", fyne.CurrentApp().Driver().AllWindows()[0])
			return
		}
		// Simplificado: edita a primeira; ideal seria selecionar da lista
		showRuleDialog(rules[0], ruleService, watchService, list)
	})

	toggleBtn := widget.NewButtonWithIcon("Ativar/Desativar", theme.VisibilityIcon(), func() {
		if len(rules) == 0 {
			return
		}
		r := rules[0]
		r.Active = !r.Active
		_ = ruleService.Update(r)
		list.Refresh()
	})

	removeBtn := widget.NewButtonWithIcon("Remover", theme.DeleteIcon(), func() {
		if len(rules) == 0 {
			return
		}
		r := rules[0]
		dialog.ShowConfirm("Remover regra", "Tem certeza?", func(ok bool) {
			if ok {
				if err := ruleService.Delete(r.ID); err != nil {
					dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
				} else {
					list.Refresh()
				}
			}
		}, fyne.CurrentApp().Driver().AllWindows()[0])
	})

	buttons := container.NewHBox(addBtn, editBtn, toggleBtn, removeBtn)
	return container.NewBorder(nil, buttons, nil, nil, list)
}