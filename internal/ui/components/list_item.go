package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TwoLineItem é um widget de lista com título em destaque e subtítulo em cinza.
// O parâmetro inactive deixa o título com cor mais clara para indicar
// item desativado.
type TwoLineItem struct {
	widget.BaseWidget
	title    *canvas.Text
	subtitle *canvas.Text
}

// NewTwoLineItem cria um item vazio. Use Update para preencher.
func NewTwoLineItem() *TwoLineItem {
	t := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.TextSize = theme.TextSize() + 1

	s := canvas.NewText("", theme.Color(theme.ColorNameDisabled))
	s.TextSize = theme.TextSize() - 1

	item := &TwoLineItem{title: t, subtitle: s}
	item.ExtendBaseWidget(item)
	return item
}

// Update preenche título, subtítulo e estado de ativo/inativo.
// Cor do título é foreground (ativo) ou disabled (inativo).
func (i *TwoLineItem) Update(title, subtitle string, active bool) {
	i.title.Text = title
	i.subtitle.Text = subtitle
	if active {
		i.title.Color = theme.Color(theme.ColorNameForeground)
	} else {
		i.title.Color = theme.Color(theme.ColorNameDisabled)
	}
	i.title.Refresh()
	i.subtitle.Refresh()
}

// UpdateWithColor é como Update mas permite definir a cor do título
// arbitrariamente. Usado pela aba Histórico para destacar status com cor
// (ex: vermelho para falha, laranja para no_text).
func (i *TwoLineItem) UpdateWithColor(title, subtitle string, titleColor color.Color) {
	i.title.Text = title
	i.subtitle.Text = subtitle
	i.title.Color = titleColor
	i.title.Refresh()
	i.subtitle.Refresh()
}

// CreateRenderer satisfaz a interface fyne.Widget.
func (i *TwoLineItem) CreateRenderer() fyne.WidgetRenderer {
	box := container.NewVBox(i.title, i.subtitle)
	return widget.NewSimpleRenderer(box)
}

// Compile-time check
var _ fyne.Widget = (*TwoLineItem)(nil)