package widgets

import (
	"hexago/ui/rendering"
	"hexago/ui/text"
)

// Text — текст, как Text во Flutter: переносится по словам, если не помещается в ширину.
//
//	widgets.Text{Data: "Привет", Style: text.Style{FontSize: 20, FontWeight: text.Bold}}
type Text struct {
	Base
	Data      string
	Style     text.Style // размер, цвет, насыщенность
	TextAlign text.Align // выравнивание строк: text.AlignLeft, text.AlignCenter, text.AlignRight
}

func (w Text) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.ParagraphBox{Text: w.Data, Style: w.Style, Align: w.TextAlign}
}

func (w Text) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	p := b.(*rendering.ParagraphBox)
	p.Text, p.Style, p.Align = w.Data, w.Style, w.TextAlign
}
