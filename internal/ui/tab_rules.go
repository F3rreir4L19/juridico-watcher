package ui

import (
	"errors"
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

// rulesTab encapsula o estado da aba Regras.
type rulesTab struct {
	parent         fyne.Window
	ruleSvc        *service.RuleService
	watchSvc       *service.WatchService
	scanSvc        *service.ScanService
	onRulesChanged func()       // reinicia o monitor service após CRUD
	onScanDone     func(int)    // chamado após scan de regra (recebe count)

	cache    []*domain.Rule
	selected int // -1 quando nada selecionado
	list     *widget.List
	emptyMsg *widget.Label

	editBtn   *widget.Button
	toggleBtn *widget.Button
	scanBtn   *widget.Button
	removeBtn *widget.Button

	root fyne.CanvasObject
}

// newRulesTab constrói a aba Regras.
func newRulesTab(
	parent fyne.Window,
	ruleSvc *service.RuleService,
	watchSvc *service.WatchService,
	scanSvc *service.ScanService,
	onRulesChanged func(),
	onScanDone func(int),
) *rulesTab {
	t := &rulesTab{
		parent:         parent,
		ruleSvc:        ruleSvc,
		watchSvc:       watchSvc,
		scanSvc:        scanSvc,
		onRulesChanged: onRulesChanged,
		onScanDone:     onScanDone,
		selected:       -1,
	}
	t.build()
	t.reload()
	return t
}

func (t *rulesTab) Root() fyne.CanvasObject {
	return t.root
}

func (t *rulesTab) build() {
	t.list = widget.NewList(
		func() int { return len(t.cache) },
		func() fyne.CanvasObject { return uic.NewTwoLineItem() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(t.cache) {
				return
			}
			r := t.cache[id]
			item := obj.(*uic.TwoLineItem)
			item.Update(r.Name, formatRuleSubtitle(r), r.Active)
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

	addBtn := widget.NewButtonWithIcon("Adicionar regra", theme.ContentAddIcon(), func() {
		t.openCreate()
	})
	addBtn.Importance = widget.HighImportance

	t.editBtn = widget.NewButtonWithIcon("Editar", theme.DocumentCreateIcon(), func() {
		t.openEdit()
	})
	t.toggleBtn = widget.NewButtonWithIcon("Ativar/Desativar", theme.MediaReplayIcon(), func() {
		t.toggleSelected()
	})
	t.scanBtn = widget.NewButtonWithIcon("Atualizar regra", theme.ViewRefreshIcon(), func() {
		t.scanSelected()
	})
	t.removeBtn = widget.NewButtonWithIcon("Remover", theme.DeleteIcon(), func() {
		t.removeSelected()
	})

	t.refreshButtons()

	t.emptyMsg = widget.NewLabel(
		"Nenhuma regra cadastrada ainda.\nClique em \"Adicionar regra\" para começar.",
	)
	t.emptyMsg.Alignment = fyne.TextAlignCenter
	t.emptyMsg.Hide()

	toolbar := container.New(layout.NewHBoxLayout(),
		addBtn, t.editBtn, t.toggleBtn, t.scanBtn, t.removeBtn,
	)

	listArea := container.NewStack(t.list, t.emptyMsg)

	t.root = container.NewBorder(
		container.NewPadded(toolbar),
		nil, nil, nil,
		container.NewPadded(listArea),
	)
}

func (t *rulesTab) reload() {
	rules, err := t.ruleSvc.List()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Não foi possível carregar regras: %s",
			uic.HumanizeError(err)), t.parent)
		t.cache = nil
	} else {
		t.cache = rules
	}
	t.selected = -1
	t.list.UnselectAll()
	t.list.Refresh()
	t.refreshButtons()
	t.toggleEmptyState()
}

func (t *rulesTab) afterChange() {
	t.reload()
	if t.onRulesChanged != nil {
		t.onRulesChanged()
	}
}

func (t *rulesTab) openCreate() {
	watches, err := t.watchSvc.List()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Não foi possível carregar pastas: %s",
			uic.HumanizeError(err)), t.parent)
		return
	}
	showRuleDialog(t.parent, t.ruleSvc, watches, nil, t.afterChange)
}

func (t *rulesTab) openEdit() {
	r := t.currentSelection()
	if r == nil {
		return
	}
	watches, err := t.watchSvc.List()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Não foi possível carregar pastas: %s",
			uic.HumanizeError(err)), t.parent)
		return
	}
	fresh, err := t.ruleSvc.GetByID(r.ID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Não foi possível carregar a regra: %s",
			uic.HumanizeError(err)), t.parent)
		return
	}
	showRuleDialog(t.parent, t.ruleSvc, watches, fresh, t.afterChange)
}

func (t *rulesTab) currentSelection() *domain.Rule {
	if t.selected < 0 || t.selected >= len(t.cache) {
		return nil
	}
	return t.cache[t.selected]
}

func (t *rulesTab) toggleSelected() {
	r := t.currentSelection()
	if r == nil {
		return
	}
	fresh, err := t.ruleSvc.GetByID(r.ID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("%s", uic.HumanizeError(err)), t.parent)
		return
	}
	fresh.Active = !fresh.Active
	if err := t.ruleSvc.Update(fresh); err != nil {
		dialog.ShowError(fmt.Errorf("%s", uic.HumanizeError(err)), t.parent)
		return
	}
	t.afterChange()
}

func (t *rulesTab) removeSelected() {
	r := t.currentSelection()
	if r == nil {
		return
	}
	msg := fmt.Sprintf(
		"Tem certeza que deseja remover a regra \"%s\"?\n\nEsta ação não afeta os arquivos no seu computador. "+
			"Regras já executadas continuam no histórico.",
		r.Name,
	)
	dialog.ShowConfirm("Remover regra", msg, func(ok bool) {
		if !ok {
			return
		}
		if err := t.ruleSvc.Delete(r.ID); err != nil {
			dialog.ShowError(fmt.Errorf("%s", uic.HumanizeError(err)), t.parent)
			return
		}
		t.afterChange()
	}, t.parent)
}

// scanSelected dispara ScanRule na regra selecionada e mostra progress + resultado.
func (t *rulesTab) scanSelected() {
	r := t.currentSelection()
	if r == nil {
		return
	}
	if !r.Active {
		// O botão deveria estar desabilitado, mas defesa redundante não custa.
		dialog.ShowInformation(
			"Regra desativada",
			"Ative a regra antes de aplicá-la aos PDFs já existentes.",
			t.parent,
		)
		return
	}

	progress := dialog.NewCustomWithoutButtons(
		"Aplicando regra",
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf("Aplicando \"%s\" aos PDFs nas pastas associadas...", r.Name)),
			widget.NewProgressBarInfinite(),
		),
		t.parent,
	)
	progress.Show()

	go func() {
		count, err := t.scanSvc.ScanRule(r.ID)

		// Volta pra UI thread via fyne.Do não está disponível em 2.7.x;
		// dialogs do Fyne podem ser mostrados de qualquer goroutine pois
		// internamente o framework empurra as ações pra main thread.
		// Mas refresh de widgets é mais seguro deixar para callbacks.
		progress.Hide()

		if err != nil {
			if errors.Is(err, service.ErrInactive) {
				dialog.ShowInformation("Regra desativada",
					"A regra foi desativada antes do scan terminar.", t.parent)
				return
			}
			dialog.ShowError(fmt.Errorf("Falha ao aplicar regra: %s",
				uic.HumanizeError(err)), t.parent)
			return
		}

		dialog.ShowInformation(
			"Regra aplicada",
			fmt.Sprintf("\"%s\" foi avaliada em %d documento(s).\n\nVeja a aba Histórico para detalhes.",
				r.Name, count),
			t.parent,
		)
		if t.onScanDone != nil {
			t.onScanDone(count)
		}
	}()
}

func (t *rulesTab) refreshButtons() {
	r := t.currentSelection()
	hasSel := r != nil
	if hasSel {
		t.editBtn.Enable()
		t.toggleBtn.Enable()
		t.removeBtn.Enable()
		// Scan só faz sentido em regra ativa
		if r.Active {
			t.scanBtn.Enable()
		} else {
			t.scanBtn.Disable()
		}
	} else {
		t.editBtn.Disable()
		t.toggleBtn.Disable()
		t.scanBtn.Disable()
		t.removeBtn.Disable()
	}
}

func (t *rulesTab) toggleEmptyState() {
	if len(t.cache) == 0 {
		t.list.Hide()
		t.emptyMsg.Show()
	} else {
		t.list.Show()
		t.emptyMsg.Hide()
	}
}

// formatRuleSubtitle gera o subtítulo da regra na lista.
func formatRuleSubtitle(r *domain.Rule) string {
	parts := []string{
		fmt.Sprintf("Prioridade %d", r.Priority),
	}
	if r.Active {
		parts = append(parts, "ativa")
	} else {
		parts = append(parts, "desativada")
	}
	parts = append(parts,
		pluralPT(len(r.WatchIDs), "pasta", "pastas"),
		pluralPT(len(r.Extractions), "extração", "extrações"),
		pluralPT(len(r.Conditions), "condição", "condições"),
		pluralPT(len(r.Actions), "ação", "ações"),
	)
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "  ·  "
		}
		out += p
	}
	return out
}

func pluralPT(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}