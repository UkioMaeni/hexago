package widgets

import (
	"hexago/ui/rendering"
	"hexago/ui/text"
)

// Text — текст, как Text во Flutter: переносится по словам, если не помещается в ширину.
// Незаданные поля Style берутся у ближайшего DefaultTextStyle (например, из темы Material).
//
//	widgets.Text{Data: "Привет", Style: text.Style{FontSize: 20, FontWeight: text.Bold}}
type Text struct {
	Base
	Data      string
	Style     text.Style // размер, цвет, насыщенность, высота строки, разрядка
	TextAlign text.Align // выравнивание строк: text.AlignLeft, text.AlignCenter, text.AlignRight
}

func (w Text) CreateRenderObject(ctx BuildContext) rendering.Box {
	return &rendering.ParagraphBox{Text: w.Data, Style: w.Style.Merge(DefaultTextStyleOf(ctx)), Align: w.TextAlign}
}

func (w Text) UpdateRenderObject(ctx BuildContext, b rendering.Box) {
	p := b.(*rendering.ParagraphBox)
	p.Text, p.Style, p.Align = w.Data, w.Style.Merge(DefaultTextStyleOf(ctx)), w.TextAlign
}

// DefaultTextStyle задаёт стиль текста поддереву: у Text незаданные поля стиля берутся отсюда.
// Его ставит, например, material.App — цвет и размер основного текста темы.
type DefaultTextStyle struct {
	Base
	Style text.Style
	Child Widget
}

func (w DefaultTextStyle) ChildWidget() Widget { return w.Child }

func (w DefaultTextStyle) UpdateShouldNotify(old Widget) bool {
	return old.(DefaultTextStyle).Style != w.Style
}

// DefaultTextStyleOf — стиль текста по умолчанию в этом месте дерева (нулевой, если не задан).
func DefaultTextStyleOf(ctx BuildContext) text.Style {
	if d, ok := DependOn[DefaultTextStyle](ctx); ok {
		return d.Style
	}
	return text.Style{}
}
