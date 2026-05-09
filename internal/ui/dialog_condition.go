package ui

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/F3rreir4L19/juridico-watcher/internal/domain"
)

// operatorLabel mapeia o operador de domínio para o rótulo amigável.
var operatorLabels = []struct {
	Op    domain.Operator
	Label string
}{
	{domain.OpEquals, "é igual a"},
	{domain.OpNotEquals, "é diferente de"},
	{domain.OpContains, "contém"},
	{domain.OpNotContain, "não contém"},
}

func operatorOptions() []string {
	out := make([]string, len(operatorLabels))
	for i, op := range operatorLabels {
		out[i] = op.Label
	}
	return out
}

func operatorByLabel(label string) (domain.Operator, bool) {
	for _, op := range operatorLabels {
		if op.Label == label {
			return op.Op, true
		}
	}
	return "", false
}

func labelByOperator(op domain.Operator) string {
	for _, e := range operatorLabels {
		if e.Op == op {
			return e.Label
		}
	}
	return string(op)
}

// showConditionDialog abre o sub-dialog de condição.
//
// availableVars é a lista de nomes de variáveis disponíveis (vinda das extrações
// já adicionadas na regra). Se vazia, o dialog mostra mensagem orientando o
// usuário a criar extrações primeiro e não permite salvar.
func showConditionDialog(
	parent fyne.Window,
	existing *domain.Condition,
	availableVars []string,
	onSave func(domain.Condition),
) {
	isNew := existing == nil
	title := "Adicionar condição"
	if !isNew {
		title = "Editar condição"
	}

	if len(availableVars) == 0 {
		dialog.ShowInformation(
			"Adicione extrações primeiro",
			"Condições comparam o valor de uma variável extraída.\n\n"+
				"Crie pelo menos uma extração antes de adicionar condições.",
			parent,
		)
		return
	}

	varSelect := widget.NewSelect(availableVars, nil)
	varSelect.PlaceHolder = "Escolha a variável"

	opSelect := widget.NewSelect(operatorOptions(), nil)
	opSelect.PlaceHolder = "Escolha o operador"
	opSelect.SetSelected(operatorLabels[0].Label)

	valueEntry := widget.NewEntry()
	valueEntry.PlaceHolder = "Valor a comparar"

	if !isNew {
		varSelect.SetSelected(existing.VariableName)
		opSelect.SetSelected(labelByOperator(existing.Operator))
		valueEntry.SetText(existing.Value)
	} else {
		varSelect.SetSelected(availableVars[0])
	}

	hint := widget.NewLabel(
		"A regra só executa as ações se TODAS as condições forem verdadeiras. " +
			"Comparações ignoram diferença entre maiúsculas e minúsculas.",
	)
	hint.Wrapping = fyne.TextWrapWord

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Variável", Widget: varSelect},
			{Text: "Operador", Widget: opSelect},
			{Text: "Valor", Widget: valueEntry},
		},
	}

	content := container.NewVBox(form, hint)

	dlg := dialog.NewCustomConfirm(title, "Salvar", "Cancelar", content,
		func(save bool) {
			if !save {
				return
			}
			cond, err := buildCondition(varSelect, opSelect, valueEntry)
			if err != nil {
				dialog.ShowError(err, parent)
				return
			}
			onSave(cond)
		}, parent,
	)
	dlg.Resize(fyne.NewSize(560, 380))
	dlg.Show()
}

func buildCondition(
	varSelect, opSelect *widget.Select,
	valueEntry *widget.Entry,
) (domain.Condition, error) {
	if varSelect.Selected == "" {
		return domain.Condition{}, errors.New("Escolha qual variável a condição compara")
	}
	if opSelect.Selected == "" {
		return domain.Condition{}, errors.New("Escolha o operador da comparação")
	}
	op, ok := operatorByLabel(opSelect.Selected)
	if !ok {
		return domain.Condition{}, fmt.Errorf("Operador desconhecido: %s", opSelect.Selected)
	}
	// Valor pode ser vazio (ex.: "nome é diferente de (vazio)").
	return domain.Condition{
		VariableName: varSelect.Selected,
		Operator:     op,
		Value:        valueEntry.Text,
	}, nil
}
