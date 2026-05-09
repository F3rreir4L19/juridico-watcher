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

// rulesTab encapsula o estado da aba Regras.
// Segue o mesmo padrão de watchesTab (D-10): cache local + reload explícito,
// seleção via OnSelected/OnUnselected, botões habilitam/desabilitam conforme
// seleção (D-08).
type rulesTab struct {
	parent         fyne.Window
	ruleSvc        *service.RuleService
	watchSvc       *service.WatchService
	onRulesChanged func() // callback para o App reiniciar o MonitorService

	cache    []*domain.Rule
	selected int // -1 quando nada selecionado
	list     *widget.List
	emptyMsg *widget.Label

	editBtn   *widget.Button
	toggleBtn *widget.Button
	removeBtn *widget.Button

	root fyne.CanvasObject
}

// newRulesTab constrói a aba Regras.
//
// onRulesChanged é chamado depois de qualquer Create/Update/Delete bem-sucedido,
// para que o App reinicie o MonitorService (regras alteradas afetam quais
// arquivos são processados).
func newRulesTab(
	parent fyne.Window,
	ruleSvc *service.RuleService,
	watchSvc *service.WatchService,
	onRulesChanged func(),
) *rulesTab {
	t := &rulesTab{
		parent:         parent,
		ruleSvc:        ruleSvc,
		watchSvc:       watchSvc,
		onRulesChanged: onRulesChanged,
		selected:       -1,
	}
	t.build()
	t.reload()
	return t
}

// Root devolve o CanvasObject pra ser usado em container.NewTabItem.
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

	// --- Botões ---
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
		addBtn, t.editBtn, t.toggleBtn, t.removeBtn,
	)

	listArea := container.NewStack(t.list, t.emptyMsg)

	t.root = container.NewBorder(
		container.NewPadded(toolbar),
		nil, nil, nil,
		container.NewPadded(listArea),
	)
}

// reload busca regras e watches frescos do service e atualiza UI.
func (t *rulesTab) reload() {
	rules, err := t.ruleSvc.List()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Não foi possível carregar regras: %s",
			uic.HumanizeError(err)), t.parent)
		t.cache = nil
	} else {
		// Para a UI, regras com filhos completos. List() já carrega filhos.
		t.cache = rules
	}
	t.selected = -1
	t.list.UnselectAll()
	t.list.Refresh()
	t.refreshButtons()
	t.toggleEmptyState()
}

// afterChange é o callback usado pelo dialog após salvar com sucesso.
func (t *rulesTab) afterChange() {
	t.reload()
	if t.onRulesChanged != nil {
		t.onRulesChanged()
	}
}

// openCreate carrega a lista de watches e abre o dialog em modo "Adicionar".
func (t *rulesTab) openCreate() {
	watches, err := t.watchSvc.List()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Não foi possível carregar pastas: %s",
			uic.HumanizeError(err)), t.parent)
		return
	}
	showRuleDialog(t.parent, t.ruleSvc, watches, nil, t.afterChange)
}

// openEdit carrega watches e abre o dialog em modo "Editar".
// Também recarrega a regra do banco para garantir que filhos vieram completos
// (List() já carrega filhos, mas fazer GetByID é defesa explícita).
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

// currentSelection retorna a regra selecionada ou nil.
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
	// Carrega versão fresca para evitar perder filhos no Update (Update do
	// repo apaga e reinsere extractions/conditions/actions/rule_watches).
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

func (t *rulesTab) refreshButtons() {
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

func (t *rulesTab) toggleEmptyState() {
	if len(t.cache) == 0 {
		t.list.Hide()
		t.emptyMsg.Show()
	} else {
		t.list.Show()
		t.emptyMsg.Hide()
	}
}

// formatRuleSubtitle gera uma linha resumo da regra para a lista.
// Exemplo: "Prioridade 10 · ativa · 1 pasta · 2 extrações · 1 condição · 2 ações"
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

// pluralPT é um helper minúsculo pra "1 ação" / "2 ações" sem dependência externa.
func pluralPT(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}
