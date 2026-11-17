package widgets

import (
	"hexago/ui/rendering"
	"hexago/ui/text"
)

// TextStyle — как выглядит текст: размер, цвет, насыщенность.
type TextStyle = text.Style

// Выравнивание строк текста и насыщенность шрифта.
const (
	TextAlignLeft   = text.AlignLeft
	TextAlignCenter = text.AlignCenter
	TextAlignRight  = text.AlignRight

	FontWeightRegular = text.Regular
	FontWeightBold    = text.Bold
)

// Text — текст, как Text во Flutter: переносится по словам, если не помещается в ширину.
type Text struct {
	Base
	Data      string
	Style     TextStyle
	TextAlign text.Align
}

func (w Text) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.ParagraphBox{Text: w.Data, Style: w.Style, Align: w.TextAlign}
}

func (w Text) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	p := b.(*rendering.ParagraphBox)
	p.Text, p.Style, p.Align = w.Data, w.Style, w.TextAlign
}
