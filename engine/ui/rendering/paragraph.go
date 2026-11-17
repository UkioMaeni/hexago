package rendering

import (
	"math"

	"hexago/ui/geom"
	"hexago/ui/text"
)

// ParagraphBox — бокс с текстом: размер по свёрстанному тексту, перенос по ширине родителя.
type ParagraphBox struct {
	Node
	Text  string
	Style text.Style
	Align text.Align
	p     *text.Paragraph
}

func (b *ParagraphBox) PerformLayout(c Constraints) geom.Size {
	b.p = text.Layout(b.Text, b.Style, c.MaxWidth)
	return geom.Size{Width: math.Ceil(b.p.Width), Height: math.Ceil(b.p.Height)}
}

func (b *ParagraphBox) Paint(ctx *PaintContext, offset geom.Offset) {
	if b.p != nil {
		b.p.Paint(ctx.Canvas, offset, b.Align, SizeOf(b).Width)
	}
}
