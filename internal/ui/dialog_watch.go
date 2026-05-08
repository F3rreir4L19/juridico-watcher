package ui

import (
	"path/filepath"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/service"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func showWatchDialog(existing *domain.Watch, watchService *service.WatchService, list *widget.List) {
	win := fyne.CurrentApp().Driver().AllWindows()[0]

	isNew := existing == nil
	title := "Adicionar Pasta"
	if !isNew {
		title = "Editar Pasta"
	}

	nameEntry := widget.NewEntry()
	nameEntry.PlaceHolder = "Nome da pasta (ex: Digitalizadora)"
	pathEntry := widget.NewEntry()
	pathEntry.PlaceHolder = "Caminho absoluto (ex: C:\\Scans)"
	recursiveCheck := widget.NewCheck("Monitorar subpastas", nil)
	activeCheck := widget.NewCheck("Ativo", nil)

	// Valores padrão
	recursiveCheck.Checked = true // RN-09: recursivo por padrão
	activeCheck.Checked = true
	if !isNew {
		nameEntry.Text = existing.Name
		pathEntry.Text = existing.Path
		recursiveCheck.Checked = existing.Recursive
		activeCheck.Checked = existing.Active
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Nome", Widget: nameEntry},
			{Text: "Caminho", Widget: pathEntry},
			{Text: "", Widget: recursiveCheck},
			{Text: "", Widget: activeCheck},
		},
	}

	content := container.NewVBox(form)

	dlg := dialog.NewCustomConfirm(title, "Salvar", "Cancelar", content, func(save bool) {
		if !save {
			return
		}
		name := nameEntry.Text
		path := pathEntry.Text
		if name == "" || path == "" {
			dialog.ShowError(domain.ErrInvalidInput, win)
			return
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			dialog.ShowError(err, win)
			return
		}

		w := &domain.Watch{
			Name:      name,
			Path:      absPath,
			Recursive: recursiveCheck.Checked,
			Active:    activeCheck.Checked,
		}
		if isNew {
			err = watchService.Create(w)
		} else {
			w.ID = existing.ID
			err = watchService.Update(w)
		}
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		list.Refresh()
	}, win)

	dlg.Resize(fyne.NewSize(500, 300))
	dlg.Show()
}