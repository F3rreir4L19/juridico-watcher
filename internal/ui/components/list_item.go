package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"image/color"
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

// CreateRenderer satisfaz a interface fyne.Widget.
func (i *TwoLineItem) CreateRenderer() fyne.WidgetRenderer {
	box := container.NewVBox(i.title, i.subtitle)
	return widget.NewSimpleRenderer(box)
}

// Compile-time check
var _ fyne.Widget = (*TwoLineItem)(nil)
var _ color.Color = color.Black // só pra evitar lint do import unused se canvas mudar
