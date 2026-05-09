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
	"github.com/F3rreir4L19/juridico-watcher/internal/storage"
	uic "github.com/F3rreir4L19/juridico-watcher/internal/ui/components"
)

const (
	allRulesLabel    = "Todas as regras"
	allStatusesLabel = "Todos os status"
)

// historyStatusOptions retorna a lista de status para o filtro, com rótulos
// amigáveis em português + opção "todos" no topo.
var historyStatusOptions = []struct {
	Status domain.ProcessingStatus
	Label  string
}{
	{"", allStatusesLabel}, // primeiro = "todos"
	{domain.StatusSuccess, uic.LabelForStatus(domain.StatusSuccess)},
	{domain.StatusNoMatch, uic.LabelForStatus(domain.StatusNoMatch)},
	{domain.StatusFailed, uic.LabelForStatus(domain.StatusFailed)},
	{domain.StatusSkippedMoved, uic.LabelForStatus(domain.StatusSkippedMoved)},
	{domain.StatusNoText, uic.LabelForStatus(domain.StatusNoText)},
}

func historyStatusLabels() []string {
	out := make([]string, len(historyStatusOptions))
	for i, opt := range historyStatusOptions {
		out[i] = opt.Label
	}
	return out
}

func statusByHistoryLabel(label string) (domain.ProcessingStatus, bool) {
	for _, opt := range historyStatusOptions {
		if opt.Label == label {
			return opt.Status, true
		}
	}
	return "", false
}

// historyTab encapsula o estado da aba Histórico.
//
// Cache local + reload explícito (D-10), igual às outras abas. A diferença
// é que aqui temos filtros — o cache contém TUDO e a renderização aplica
// os filtros sobre o cache.
type historyTab struct {
	parent     fyne.Window
	historySvc *service.HistoryService
	ruleSvc    *service.RuleService

	// cache contém todos os itens recentes; filtered é a view filtrada
	// que alimenta a lista visível. Filtramos em memória porque a lista
	// cabe (limite de 100 itens).
	cache    []*storage.HistoryItem
	filtered []*storage.HistoryItem

	list     *widget.List
	emptyMsg *widget.Label

	ruleFilter   *widget.Select
	statusFilter *widget.Select

	root fyne.CanvasObject
}

// newHistoryTab constrói a aba Histórico.
func newHistoryTab(
	parent fyne.Window,
	historySvc *service.HistoryService,
	ruleSvc *service.RuleService,
) *historyTab {
	t := &historyTab{
		parent:     parent,
		historySvc: historySvc,
		ruleSvc:    ruleSvc,
	}
	t.build()
	t.reload()
	return t
}

// Root devolve o CanvasObject pra ser usado em container.NewTabItem.
func (t *historyTab) Root() fyne.CanvasObject {
	return t.root
}

// Reload recarrega cache do banco e re-aplica filtros.
// Exposto pra ser chamado pelo App após scan global.
func (t *historyTab) Reload() {
	t.reload()
}

// MarkVisited grava o timestamp da última visita.
// Chamado pelo App quando o usuário sai da aba.
func (t *historyTab) MarkVisited() {
	if err := t.historySvc.MarkVisited(); err != nil {
		// não é fatal; só logamos via dialog se o user mexer e falhar
		// silenciosamente é OK aqui — pior caso o "(N falhas)" não atualiza
		fmt.Printf("falha ao marcar visita: %v\n", err)
	}
}

func (t *historyTab) build() {
	t.list = widget.NewList(
		func() int { return len(t.filtered) },
		func() fyne.CanvasObject { return uic.NewTwoLineItem() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(t.filtered) {
				return
			}
			item := t.filtered[id]
			cell := obj.(*uic.TwoLineItem)

			title := titleForHistoryItem(item)
			subtitle := subtitleForHistoryItem(item)
			color := uic.ColorForStatus(item.Status)
			cell.UpdateWithColor(title, subtitle, color)
		},
	)

	// --- Filtros ---
	// IMPORTANTE: criamos os Selects sem OnChanged primeiro, definimos
	// Selected via campo (não SetSelected, que dispararia o callback nil),
	// e só DEPOIS atribuímos OnChanged. Isso evita NPE durante o build.
	t.ruleFilter = widget.NewSelect([]string{allRulesLabel}, nil)
	t.ruleFilter.Selected = allRulesLabel

	t.statusFilter = widget.NewSelect(historyStatusLabels(), nil)
	t.statusFilter.Selected = allStatusesLabel

	// Agora ambos existem — seguro atribuir os callbacks.
	t.ruleFilter.OnChanged = func(_ string) { t.applyFilters() }
	t.statusFilter.OnChanged = func(_ string) { t.applyFilters() }

	refreshBtn := widget.NewButtonWithIcon("Atualizar lista", theme.ViewRefreshIcon(), func() {
		t.reload()
	})

	filterRow := container.New(
		layout.NewHBoxLayout(),
		widget.NewLabel("Regra:"), t.ruleFilter,
		widget.NewLabel("Status:"), t.statusFilter,
		layout.NewSpacer(),
		refreshBtn,
	)

	// --- Empty state ---
	t.emptyMsg = widget.NewLabel("Nenhum processamento registrado ainda.")
	t.emptyMsg.Alignment = fyne.TextAlignCenter
	t.emptyMsg.Hide()

	listArea := container.NewStack(t.list, t.emptyMsg)

	t.root = container.NewBorder(
		container.NewPadded(filterRow),
		nil, nil, nil,
		container.NewPadded(listArea),
	)
}

// reload busca o histórico fresco do banco, atualiza o filtro de regras,
// e re-aplica filtros para popular a lista visível.
func (t *historyTab) reload() {
	items, err := t.historySvc.List()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Não foi possível carregar histórico: %s",
			uic.HumanizeError(err)), t.parent)
		t.cache = nil
	} else {
		t.cache = items
	}
	t.refreshRuleFilterOptions()
	t.applyFilters()
}

// refreshRuleFilterOptions sincroniza a lista de regras do filtro com as
// regras existentes no banco. Mantém a seleção atual se a regra ainda existe.
func (t *historyTab) refreshRuleFilterOptions() {
	rules, err := t.ruleSvc.List()
	if err != nil {
		// se não der pra carregar, deixamos só "Todas"
		t.ruleFilter.Options = []string{allRulesLabel}
		t.ruleFilter.SetSelected(allRulesLabel)
		t.ruleFilter.Refresh()
		return
	}
	options := []string{allRulesLabel}
	for _, r := range rules {
		options = append(options, r.Name)
	}
	current := t.ruleFilter.Selected
	t.ruleFilter.Options = options
	// Se a seleção atual ainda existe, preserva; senão volta pra "Todas".
	found := false
	for _, opt := range options {
		if opt == current {
			found = true
			break
		}
	}
	if !found {
		t.ruleFilter.SetSelected(allRulesLabel)
	}
	t.ruleFilter.Refresh()
}

// applyFilters re-popula t.filtered a partir de t.cache aplicando os filtros
// atuais e atualiza a UI da lista.
func (t *historyTab) applyFilters() {
	ruleFilter := t.ruleFilter.Selected
	statusFilter, _ := statusByHistoryLabel(t.statusFilter.Selected)

	t.filtered = t.filtered[:0]
	for _, item := range t.cache {
		if ruleFilter != allRulesLabel && item.RuleName != ruleFilter {
			continue
		}
		if statusFilter != "" && item.Status != statusFilter {
			continue
		}
		t.filtered = append(t.filtered, item)
	}

	if len(t.filtered) == 0 {
		t.list.Hide()
		if len(t.cache) == 0 {
			t.emptyMsg.SetText("Nenhum processamento registrado ainda.")
		} else {
			t.emptyMsg.SetText("Nenhum item corresponde aos filtros selecionados.")
		}
		t.emptyMsg.Show()
	} else {
		t.list.Show()
		t.emptyMsg.Hide()
	}
	t.list.UnselectAll()
	t.list.Refresh()
}

// --- Helpers de formatação ---

func titleForHistoryItem(item *storage.HistoryItem) string {
	// Mostra o nome do arquivo (basename) — caminho completo fica no subtítulo
	return basenameOrPath(item.OriginalPath)
}

func subtitleForHistoryItem(item *storage.HistoryItem) string {
	rule := item.RuleName
	if rule == "" {
		rule = "(regra removida)"
	}
	timestamp := item.ProcessedAt.Local().Format("02/01/2006 15:04:05")
	parts := []string{
		"regra: " + rule,
		"status: " + uic.LabelForStatus(item.Status),
		timestamp,
	}
	subtitle := joinWithDot(parts)
	// Se houver mensagem de erro, mostra
	if item.ErrorMsg != "" {
		subtitle += "  ·  " + item.ErrorMsg
	}
	return subtitle
}

// basenameOrPath retorna a última parte de path (após / ou \).
// Reimplementado em vez de usar filepath.Base para não depender do separador
// do SO atual — o path foi gravado no banco com o separador do SO em que
// rodou e pode ser diferente do atual (cenário raro, mas possível).
func basenameOrPath(path string) string {
	if path == "" {
		return "(arquivo desconhecido)"
	}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

func joinWithDot(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "  ·  "
		}
		out += p
	}
	return out
}