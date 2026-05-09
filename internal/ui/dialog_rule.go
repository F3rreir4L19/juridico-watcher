package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

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

// showRuleDialog abre o dialog de criar/editar regra.
//
// availableWatches deve conter todos os watches existentes no banco (ativos
// e inativos), pois mesmo um watch inativo pode estar associado a uma regra.
//
// existing == nil → modo "Adicionar". Senão, "Editar" com campos preenchidos.
//
// Pré-condição importante: se availableWatches estiver vazio, este dialog
// não deveria ser chamado. A chamada vinda de tab_rules.go faz essa checagem
// e mostra mensagem orientando o usuário a criar pasta primeiro.
func showRuleDialog(
	parent fyne.Window,
	ruleService *service.RuleService,
	availableWatches []*domain.Watch,
	existing *domain.Rule,
	onSaved func(),
) {
	if len(availableWatches) == 0 {
		dialog.ShowInformation(
			"Cadastre uma pasta primeiro",
			"Antes de criar regras, você precisa cadastrar pelo menos uma pasta na aba \"Monitorar\".\n\n"+
				"As regras se aplicam às pastas monitoradas; sem nenhuma, não há onde a regra atuar.",
			parent,
		)
		return
	}

	st := newRuleDialogState(parent, availableWatches, existing)

	dlg := dialog.NewCustomConfirm(
		st.title(),
		"Salvar",
		"Cancelar",
		st.buildContent(),
		func(save bool) {
			if !save {
				return
			}
			if err := st.commit(ruleService); err != nil {
				dialog.ShowError(fmt.Errorf("%s", uic.HumanizeError(err)), parent)
				return
			}
			if onSaved != nil {
				onSaved()
			}
		},
		parent,
	)
	dlg.Resize(fyne.NewSize(720, 640))
	dlg.Show()
}

// ruleDialogState mantém o estado em memória do dialog, conforme D-10.
// Tudo é editado em cópias locais; só vai ao banco no commit().
type ruleDialogState struct {
	parent           fyne.Window
	availableWatches []*domain.Watch
	existing         *domain.Rule // nil em modo adicionar

	// Cópias locais editáveis dos dados da regra
	name        string
	priority    int
	active      bool
	watchIDs    map[int64]bool // watchID -> selected
	extractions []domain.Extraction
	conditions  []domain.Condition
	actions     []domain.Action

	// Widgets de metadados — mantidos como campos para serem lidos no commit
	nameEntry     *widget.Entry
	priorityEntry *widget.Entry
	activeCheck   *widget.Check
	watchChecks   map[int64]*widget.Check

	// Sublistas — guardamos referências para re-renderização cruzada.
	// Quando uma extração é adicionada/removida, a hint da sublista de ações
	// e o select da sublista de condições precisam refletir isso.
	extSection  *subListSection
	condSection *subListSection
	actSection  *subListSection
}

func newRuleDialogState(parent fyne.Window, availableWatches []*domain.Watch, existing *domain.Rule) *ruleDialogState {
	st := &ruleDialogState{
		parent:           parent,
		availableWatches: availableWatches,
		existing:         existing,
		watchIDs:         make(map[int64]bool),
		watchChecks:      make(map[int64]*widget.Check),
		priority:         100, // padrão razoável (mesmo do schema SQL)
		active:           true,
	}
	if existing != nil {
		st.name = existing.Name
		st.priority = existing.Priority
		st.active = existing.Active
		for _, wid := range existing.WatchIDs {
			st.watchIDs[wid] = true
		}
		st.extractions = append(st.extractions, existing.Extractions...)
		st.conditions = append(st.conditions, existing.Conditions...)
		st.actions = append(st.actions, existing.Actions...)
	}
	return st
}

func (st *ruleDialogState) title() string {
	if st.existing == nil {
		return "Adicionar regra"
	}
	return "Editar regra"
}

// availableVarNames retorna os nomes das extrações já adicionadas — usados
// pelos sub-dialogs de condição e ação como hint/select.
func (st *ruleDialogState) availableVarNames() []string {
	out := make([]string, len(st.extractions))
	for i, ex := range st.extractions {
		out[i] = ex.VariableName
	}
	return out
}

// buildContent monta o CanvasObject do conteúdo do dialog.
func (st *ruleDialogState) buildContent() fyne.CanvasObject {
	st.nameEntry = widget.NewEntry()
	st.nameEntry.PlaceHolder = "Ex: Procuração"
	st.nameEntry.SetText(st.name)

	st.priorityEntry = widget.NewEntry()
	st.priorityEntry.PlaceHolder = "100"
	st.priorityEntry.SetText(strconv.Itoa(st.priority))

	st.activeCheck = widget.NewCheck("Regra ativa", nil)
	st.activeCheck.Checked = st.active

	metaForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Nome", Widget: st.nameEntry, HintText: "Identifica a regra na lista"},
			{Text: "Prioridade", Widget: st.priorityEntry, HintText: "Menor número executa primeiro"},
			{Text: "", Widget: st.activeCheck},
		},
	}

	watchesSection := st.buildWatchesSection()
	st.extSection = st.buildExtractionsSection()
	st.condSection = st.buildConditionsSection()
	st.actSection = st.buildActionsSection()

	accordion := widget.NewAccordion(
		widget.NewAccordionItem("Extrações ("+strconv.Itoa(len(st.extractions))+")", st.extSection.root),
		widget.NewAccordionItem("Condições ("+strconv.Itoa(len(st.conditions))+")", st.condSection.root),
		widget.NewAccordionItem("Ações ("+strconv.Itoa(len(st.actions))+")", st.actSection.root),
	)
	// Começa com a primeira seção aberta (Extrações é a primeira coisa que o usuário precisa preencher)
	accordion.Open(0)

	// Re-renderiza títulos do acordeon ao mudar contagem
	st.extSection.onListChanged = func() {
		accordion.Items[0].Title = "Extrações (" + strconv.Itoa(len(st.extractions)) + ")"
		accordion.Refresh()
		// Variáveis mudaram → redesenha sub-dialog data nos outros
		st.condSection.refreshList()
		st.actSection.refreshList()
	}
	st.condSection.onListChanged = func() {
		accordion.Items[1].Title = "Condições (" + strconv.Itoa(len(st.conditions)) + ")"
		accordion.Refresh()
	}
	st.actSection.onListChanged = func() {
		accordion.Items[2].Title = "Ações (" + strconv.Itoa(len(st.actions)) + ")"
		accordion.Refresh()
	}

	scroll := container.NewVScroll(container.NewVBox(
		metaForm,
		widget.NewSeparator(),
		watchesSection,
		widget.NewSeparator(),
		accordion,
	))
	// Tamanho mínimo legível dentro do dialog. O dialog em si tem 720x640;
	// o scroll garante que conteúdo maior continua acessível.
	scroll.SetMinSize(fyne.NewSize(680, 540))
	return scroll
}

// buildWatchesSection monta a área de seleção de pastas em colunas (D-Sprint 9).
func (st *ruleDialogState) buildWatchesSection() fyne.CanvasObject {
	header := widget.NewLabel("Pastas monitoradas (selecione pelo menos uma)")
	header.TextStyle = fyne.TextStyle{Bold: true}

	// Layout em grade de 2 colunas para listas compactas.
	// Para listas muito grandes, scroll vertical absorve.
	grid := container.New(layout.NewGridLayout(2))

	for _, w := range st.availableWatches {
		watch := w // captura para closure
		check := widget.NewCheck(watch.Name, func(checked bool) {
			st.watchIDs[watch.ID] = checked
		})
		check.Checked = st.watchIDs[watch.ID]
		// Mostrar caminho como tooltip seria ideal, mas Fyne não tem tooltip nativo.
		// Subtítulo em cinza ao lado seria poluente em grade. Decisão: nome só.
		st.watchChecks[watch.ID] = check
		grid.Add(check)
	}

	scroll := container.NewVScroll(grid)
	scroll.SetMinSize(fyne.NewSize(680, 90))

	return container.NewVBox(header, scroll)
}

// --- Sub-list sections (extrações, condições, ações) ---

// subListSection é um helper genérico que encapsula uma sublista com
// botões Adicionar/Editar/Remover. Cada uma das três seções do acordeon
// é uma instância dessa estrutura.
type subListSection struct {
	root          fyne.CanvasObject
	list          *widget.List
	selected      int
	addBtn        *widget.Button
	editBtn       *widget.Button
	removeBtn     *widget.Button
	emptyLabel    *widget.Label
	getCount      func() int
	getItem       func(int) (title, subtitle string)
	onAdd         func()
	onEdit        func(int)
	onRemove      func(int)
	emptyMessage  string
	onListChanged func() // chamada após qualquer mudança na slice (add/edit/remove)
}

// refreshList re-renderiza a sublista e o estado dos botões.
// Útil quando a slice muda externamente (por exemplo, quando uma extração é
// removida e isso invalida uma condição que dependia daquela variável — neste
// caso o sub-dialog vai mostrar uma lista vazia, mas o item órfão ainda
// existe na slice; a UX aceitável é deixar o usuário ver e corrigir.)
func (s *subListSection) refreshList() {
	if s == nil || s.list == nil {
		return
	}
	if s.getCount() == 0 {
		s.list.Hide()
		s.emptyLabel.Show()
	} else {
		s.list.Show()
		s.emptyLabel.Hide()
	}
	s.list.Refresh()
	s.refreshButtons()
}

func (s *subListSection) refreshButtons() {
	hasSel := s.selected >= 0 && s.selected < s.getCount()
	if hasSel {
		s.editBtn.Enable()
		s.removeBtn.Enable()
	} else {
		s.editBtn.Disable()
		s.removeBtn.Disable()
	}
}

// newSubListSection constrói a estrutura padrão de uma sublista com toolbar.
func newSubListSection(
	getCount func() int,
	getItem func(int) (string, string),
	onAdd func(),
	onEdit func(int),
	onRemove func(int),
	emptyMessage string,
) *subListSection {
	s := &subListSection{
		selected:     -1,
		getCount:     getCount,
		getItem:      getItem,
		onAdd:        onAdd,
		onEdit:       onEdit,
		onRemove:     onRemove,
		emptyMessage: emptyMessage,
	}

	s.list = widget.NewList(
		getCount,
		func() fyne.CanvasObject { return uic.NewTwoLineItem() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= getCount() {
				return
			}
			title, subtitle := getItem(id)
			obj.(*uic.TwoLineItem).Update(title, subtitle, true)
		},
	)
	s.list.OnSelected = func(id widget.ListItemID) {
		s.selected = id
		s.refreshButtons()
	}
	s.list.OnUnselected = func(id widget.ListItemID) {
		if s.selected == id {
			s.selected = -1
		}
		s.refreshButtons()
	}

	s.addBtn = widget.NewButtonWithIcon("Adicionar", theme.ContentAddIcon(), func() { onAdd() })
	s.editBtn = widget.NewButtonWithIcon("Editar", theme.DocumentCreateIcon(), func() {
		if s.selected >= 0 && s.selected < s.getCount() {
			onEdit(s.selected)
		}
	})
	s.removeBtn = widget.NewButtonWithIcon("Remover", theme.DeleteIcon(), func() {
		if s.selected >= 0 && s.selected < s.getCount() {
			onRemove(s.selected)
		}
	})
	s.refreshButtons()

	s.emptyLabel = widget.NewLabel(emptyMessage)
	s.emptyLabel.Alignment = fyne.TextAlignCenter
	s.emptyLabel.Wrapping = fyne.TextWrapWord

	toolbar := container.New(layout.NewHBoxLayout(),
		s.addBtn, s.editBtn, s.removeBtn,
	)

	listArea := container.NewStack(s.list, s.emptyLabel)

	// Altura fixa razoável: caso típico tem 3-5 itens; rolagem pra mais
	wrapped := container.NewBorder(toolbar, nil, nil, nil, listArea)
	scroll := container.NewVScroll(wrapped)
	scroll.SetMinSize(fyne.NewSize(680, 200))

	s.root = scroll

	// Estado inicial: vazio?
	if getCount() == 0 {
		s.list.Hide()
	} else {
		s.emptyLabel.Hide()
	}

	return s
}

// --- Construção de cada seção ---

func (st *ruleDialogState) buildExtractionsSection() *subListSection {
	var s *subListSection

	getCount := func() int { return len(st.extractions) }
	getItem := func(i int) (string, string) {
		ex := st.extractions[i]
		title := "{" + ex.VariableName + "}"
		subtitle := formatDelimDescription(ex.StartDelim, ex.EndDelim)
		return title, subtitle
	}

	onAdd := func() {
		showExtractionDialog(st.parent, nil, st.availableVarNames(), "", func(ex domain.Extraction) {
			st.extractions = append(st.extractions, ex)
			s.list.UnselectAll()
			s.selected = -1
			s.refreshList()
			if s.onListChanged != nil {
				s.onListChanged()
			}
		})
	}
	onEdit := func(idx int) {
		current := st.extractions[idx]
		showExtractionDialog(st.parent, &current, st.availableVarNames(), current.VariableName, func(ex domain.Extraction) {
			st.extractions[idx] = ex
			s.refreshList()
			if s.onListChanged != nil {
				s.onListChanged()
			}
		})
	}
	onRemove := func(idx int) {
		removed := st.extractions[idx]
		// Aviso se a variável removida está em uso por condição/ação
		used := st.varInUse(removed.VariableName)
		if used != "" {
			dialog.ShowConfirm(
				"Remover extração?",
				fmt.Sprintf(
					"A variável {%s} é usada %s.\n\nSe você remover, esses itens podem deixar de funcionar como esperado. Continuar?",
					removed.VariableName, used,
				),
				func(ok bool) {
					if !ok {
						return
					}
					st.extractions = append(st.extractions[:idx], st.extractions[idx+1:]...)
					s.list.UnselectAll()
					s.selected = -1
					s.refreshList()
					if s.onListChanged != nil {
						s.onListChanged()
					}
				}, st.parent,
			)
			return
		}
		st.extractions = append(st.extractions[:idx], st.extractions[idx+1:]...)
		s.list.UnselectAll()
		s.selected = -1
		s.refreshList()
		if s.onListChanged != nil {
			s.onListChanged()
		}
	}

	s = newSubListSection(getCount, getItem, onAdd, onEdit, onRemove,
		"Nenhuma extração ainda. Clique em \"Adicionar\" para definir a primeira.")
	return s
}

func (st *ruleDialogState) buildConditionsSection() *subListSection {
	var s *subListSection

	getCount := func() int { return len(st.conditions) }
	getItem := func(i int) (string, string) {
		c := st.conditions[i]
		title := fmt.Sprintf("{%s} %s %q", c.VariableName, labelByOperator(c.Operator), c.Value)
		subtitle := "" // condições são sucintas; nada a acrescentar
		return title, subtitle
	}

	onAdd := func() {
		showConditionDialog(st.parent, nil, st.availableVarNames(), func(c domain.Condition) {
			st.conditions = append(st.conditions, c)
			s.list.UnselectAll()
			s.selected = -1
			s.refreshList()
			if s.onListChanged != nil {
				s.onListChanged()
			}
		})
	}
	onEdit := func(idx int) {
		current := st.conditions[idx]
		showConditionDialog(st.parent, &current, st.availableVarNames(), func(c domain.Condition) {
			st.conditions[idx] = c
			s.refreshList()
			if s.onListChanged != nil {
				s.onListChanged()
			}
		})
	}
	onRemove := func(idx int) {
		st.conditions = append(st.conditions[:idx], st.conditions[idx+1:]...)
		s.list.UnselectAll()
		s.selected = -1
		s.refreshList()
		if s.onListChanged != nil {
			s.onListChanged()
		}
	}

	s = newSubListSection(getCount, getItem, onAdd, onEdit, onRemove,
		"Nenhuma condição. Sem condições, a regra executa as ações em todos os documentos.")
	return s
}

func (st *ruleDialogState) buildActionsSection() *subListSection {
	var s *subListSection

	getCount := func() int { return len(st.actions) }
	getItem := func(i int) (string, string) {
		a := st.actions[i]
		title := labelByActionType(a.Type)
		subtitle := "destino: " + a.Target
		return title, subtitle
	}

	onAdd := func() {
		showActionDialog(st.parent, nil, st.availableVarNames(), func(a domain.Action) {
			st.actions = append(st.actions, a)
			s.list.UnselectAll()
			s.selected = -1
			s.refreshList()
			if s.onListChanged != nil {
				s.onListChanged()
			}
		})
	}
	onEdit := func(idx int) {
		current := st.actions[idx]
		showActionDialog(st.parent, &current, st.availableVarNames(), func(a domain.Action) {
			st.actions[idx] = a
			s.refreshList()
			if s.onListChanged != nil {
				s.onListChanged()
			}
		})
	}
	onRemove := func(idx int) {
		st.actions = append(st.actions[:idx], st.actions[idx+1:]...)
		s.list.UnselectAll()
		s.selected = -1
		s.refreshList()
		if s.onListChanged != nil {
			s.onListChanged()
		}
	}

	s = newSubListSection(getCount, getItem, onAdd, onEdit, onRemove,
		"Nenhuma ação. Adicione pelo menos uma — sem ações, a regra não faz nada.")
	return s
}

// commit valida o estado completo, monta o domain.Rule e chama Create/Update.
func (st *ruleDialogState) commit(svc *service.RuleService) error {
	// Lê valores correntes dos widgets de metadados
	name := strings.TrimSpace(st.nameEntry.Text)
	if name == "" {
		return errors.New("Por favor, preencha o nome da regra")
	}

	priorityStr := strings.TrimSpace(st.priorityEntry.Text)
	if priorityStr == "" {
		priorityStr = "100"
	}
	priority, err := strconv.Atoi(priorityStr)
	if err != nil {
		return fmt.Errorf("A prioridade precisa ser um número inteiro (você digitou %q)", priorityStr)
	}

	// Coleta watchIDs marcados
	var watchIDs []int64
	for _, w := range st.availableWatches {
		if st.watchIDs[w.ID] {
			watchIDs = append(watchIDs, w.ID)
		}
	}
	if len(watchIDs) == 0 {
		return errors.New("Selecione pelo menos uma pasta monitorada")
	}

	// Validação UX exigida por D-Sprint9: ao menos 1 ação
	if len(st.actions) == 0 {
		return errors.New("Adicione pelo menos uma ação. Sem ações, a regra não faz nada com os documentos.")
	}

	rule := &domain.Rule{
		Name:        name,
		Priority:    priority,
		Active:      st.activeCheck.Checked,
		WatchIDs:    watchIDs,
		Extractions: st.extractions,
		Conditions:  st.conditions,
		Actions:     st.actions,
	}

	if st.existing == nil {
		return svc.Create(rule)
	}
	rule.ID = st.existing.ID
	return svc.Update(rule)
}

// varInUse retorna uma descrição não vazia se a variável é usada em alguma
// condição ou ação. Usado pra avisar o usuário antes de remover uma extração.
func (st *ruleDialogState) varInUse(varName string) string {
	placeholder := "{" + varName + "}"
	var usages []string
	for _, c := range st.conditions {
		if c.VariableName == varName {
			usages = append(usages, "em uma condição")
			break
		}
	}
	for _, a := range st.actions {
		if strings.Contains(a.Target, placeholder) {
			usages = append(usages, "em uma ação")
			break
		}
	}
	if len(usages) == 0 {
		return ""
	}
	return strings.Join(usages, " e ")
}

// formatDelimDescription gera o subtítulo amigável de uma extração na lista.
func formatDelimDescription(start, end string) string {
	startDesc := "início do texto"
	if start != "" {
		startDesc = fmt.Sprintf("após %q", start)
	}
	endDesc := "fim do texto"
	if end != "" {
		endDesc = fmt.Sprintf("antes de %q", end)
	}
	return startDesc + " · " + endDesc
}
