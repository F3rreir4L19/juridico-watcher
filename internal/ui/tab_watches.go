package ui

import (
	"database/sql"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/service"
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func newTabWatches(db *sql.DB) fyne.CanvasObject {
	watchService := service.NewWatchService(storage.NewWatchRepo(db), storage.NewRuleRepo(db))

	list := widget.NewList(
		func() int {
			watches, _ := watchService.List()
			return len(watches)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Modelo de item")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			watches, _ := watchService.List()
			if id < len(watches) {
				w := watches[id]
				label := obj.(*widget.Label)
				status := "✓"
				if !w.Active {
					status = "✗"
				}
				rec := ""
				if w.Recursive {
					rec = " [rec]"
				}
				label.SetText(status + " " + w.Name + rec + "\n  " + w.Path)
			}
		},
	)

	// Botão Adicionar
	addBtn := widget.NewButtonWithIcon("Adicionar", theme.ContentAddIcon(), func() {
		showWatchDialog(nil, watchService, list)
	})

	// Botão Editar
	editBtn := widget.NewButtonWithIcon("Editar", theme.DocumentCreateIcon(), func() {
		watches, err := watchService.List()
		if err != nil || len(watches) == 0 {
			dialog.ShowInformation("Aviso", "Nenhuma pasta para editar.", fyne.CurrentApp().Driver().AllWindows()[0])
			return
		}
		// Seleciona o primeiro por enquanto; ideal seria ter seleção de item
		showWatchDialog(watches[0], watchService, list)
	})

	// Botão Ativar/Desativar
	toggleBtn := widget.NewButtonWithIcon("Ativar/Desativar", theme.VisibilityIcon(), func() {
		watches, err := watchService.List()
		if err != nil || len(watches) == 0 {
			return
		}
		w := watches[0] // Simplificação; o ideal é selecionar o item da lista
		w.Active = !w.Active
		_ = watchService.Update(w)
		list.Refresh()
	})

	// Botão Remover
	removeBtn := widget.NewButtonWithIcon("Remover", theme.DeleteIcon(), func() {
		watches, err := watchService.List()
		if err != nil || len(watches) == 0 {
			return
		}
		w := watches[0] // Simplificação
		err = watchService.Delete(w.ID)
		if err == domain.ErrWatchInUse {
			dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
		} else if err != nil {
			dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
		} else {
			list.Refresh()
		}
	})

	buttons := container.NewHBox(addBtn, editBtn, toggleBtn, removeBtn)
	return container.NewBorder(nil, buttons, nil, nil, list)
}