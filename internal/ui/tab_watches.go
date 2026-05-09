package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
	"github.com/F3rreir4L19/juridico-watcher/internal/service"
	uic "github.com/F3rreir4L19/juridico-watcher/internal/ui/components"
)

// watchesTab encapsula o estado da aba Monitorar.
// Manter como struct facilita evitar closures gigantes e variáveis globais.
type watchesTab struct {
	parent           fyne.Window
	watchSvc         *service.WatchService
	onWatchesChanged func() // callback para o App reiniciar o MonitorService

	cache    []*domain.Watch
	selected int // -1 quando nada selecionado
	list     *widget.List
	emptyMsg *widget.Label

	editBtn   *widget.Button
	toggleBtn *widget.Button
	removeBtn *widget.Button

	root fyne.CanvasObject
}

// newWatchesTab constrói a aba Monitorar.
//
// onWatchesChanged é chamado depois de qualquer Create/Update/Delete bem-sucedido,
// para que o App possa reiniciar o MonitorService.
func newWatchesTab(
	parent fyne.Window,
	watchSvc *service.WatchService,
	onWatchesChanged func(),
) *watchesTab {
	t := &watchesTab{
		parent:           parent,
		watchSvc:         watchSvc,
		onWatchesChanged: onWatchesChanged,
		selected:         -1,
	}
	t.build()
	t.reload()
	return t
}

// Root devolve o CanvasObject pra ser usado em container.NewTabItem.
func (t *watchesTab) Root() fyne.CanvasObject {
	return t.root
}

func (t *watchesTab) build() {
	t.list = widget.NewList(
		// length
		func() int { return len(t.cache) },
		// create template
		func() fyne.CanvasObject { return uic.NewTwoLineItem() },
		// update
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(t.cache) {
				return
			}
			w := t.cache[id]
			item := obj.(*uic.TwoLineItem)
			subtitle := w.Path
			if !w.Recursive {
				subtitle += "  ·  apenas esta pasta"
			} else {
				subtitle += "  ·  inclui subpastas"
			}
			if !w.Active {
				subtitle += "  ·  desativada"
			}
			item.Update(w.Name, subtitle, w.Active)
		},
	)
	t.list.OnSelected = func(id widget.ListItemID) {
		t.selected = id
		t.refreshButtons()
	}
	t.list.OnUnselected = func(id widget.ListItemID) {
		if t.selected == id {
			t.selected = -1
		}
		t.refreshButtons()
	}

	// --- Botões ---
	addBtn := widget.NewButtonWithIcon("Adicionar pasta", theme.ContentAddIcon(), func() {
		showWatchDialog(t.parent, t.watchSvc, nil, t.afterChange)
	})
	addBtn.Importance = widget.HighImportance

	t.editBtn = widget.NewButtonWithIcon("Editar", theme.DocumentCreateIcon(), func() {
		w := t.currentSelection()
		if w == nil {
			return
		}
		showWatchDialog(t.parent, t.watchSvc, w, t.afterChange)
	})
	t.toggleBtn = widget.NewButtonWithIcon("Ativar/Desativar", theme.MediaReplayIcon(), func() {
		t.toggleSelected()
	})
	t.removeBtn = widget.NewButtonWithIcon("Remover", theme.DeleteIcon(), func() {
		t.removeSelected()
	})

	t.refreshButtons()

	// --- Empty state label ---
	t.emptyMsg = widget.NewLabel(
		"Nenhuma pasta cadastrada ainda.\nClique em \"Adicionar pasta\" para começar.",
	)
	t.emptyMsg.Alignment = fyne.TextAlignCenter
	t.emptyMsg.Hide()

	// --- Layout ---
	// Topo: barra de botões. Centro: lista (ou empty msg).
	toolbar := container.New(layout.NewHBoxLayout(),
		addBtn, t.editBtn, t.toggleBtn, t.removeBtn,
	)

	// Wrap na lista para conseguir alternar empty state ↔ lista
	listArea := container.NewStack(t.list, t.emptyMsg)

	t.root = container.NewBorder(
		container.NewPadded(toolbar), // top
		nil, nil, nil,
		container.NewPadded(listArea),
	)
}

// reload busca dados frescos do service e atualiza UI.
func (t *watchesTab) reload() {
	list, err := t.watchSvc.List()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Não foi possível carregar pastas: %s",
			uic.HumanizeError(err)), t.parent)
		t.cache = nil
	} else {
		t.cache = list
	}
	t.selected = -1
	t.list.UnselectAll()
	t.list.Refresh()
	t.refreshButtons()
	t.toggleEmptyState()
}

// afterChange é o callback usado pelo dialog após salvar com sucesso.
func (t *watchesTab) afterChange() {
	t.reload()
	if t.onWatchesChanged != nil {
		t.onWatchesChanged()
	}
}

// currentSelection retorna o watch selecionado ou nil.
func (t *watchesTab) currentSelection() *domain.Watch {
	if t.selected < 0 || t.selected >= len(t.cache) {
		return nil
	}
	return t.cache[t.selected]
}

func (t *watchesTab) toggleSelected() {
	w := t.currentSelection()
	if w == nil {
		return
	}
	w.Active = !w.Active
	if err := t.watchSvc.Update(w); err != nil {
		dialog.ShowError(fmt.Errorf("%s", uic.HumanizeError(err)), t.parent)
		return
	}
	t.afterChange()
}

func (t *watchesTab) removeSelected() {
	w := t.currentSelection()
	if w == nil {
		return
	}
	msg := fmt.Sprintf(
		"Tem certeza que deseja remover a pasta \"%s\"?\n\nEsta ação não afeta os arquivos no seu computador, "+
			"apenas faz o programa parar de monitorá-la.",
		w.Name,
	)
	dialog.ShowConfirm("Remover pasta", msg, func(ok bool) {
		if !ok {
			return
		}
		if err := t.watchSvc.Delete(w.ID); err != nil {
			dialog.ShowError(fmt.Errorf("%s", uic.HumanizeError(err)), t.parent)
			return
		}
		t.afterChange()
	}, t.parent)
}

func (t *watchesTab) refreshButtons() {
	hasSel := t.selected >= 0
	if hasSel {
		t.editBtn.Enable()
		t.toggleBtn.Enable()
		t.removeBtn.Enable()
	} else {
		t.editBtn.Disable()
		t.toggleBtn.Disable()
		t.removeBtn.Disable()
	}
}

func (t *watchesTab) toggleEmptyState() {
	if len(t.cache) == 0 {
		t.list.Hide()
		t.emptyMsg.Show()
	} else {
		t.list.Show()
		t.emptyMsg.Hide()
	}
}
