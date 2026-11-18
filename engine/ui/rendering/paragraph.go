package rendering

import (
	"math"

	"hexago/ui/geom"
	"hexago/ui/gfx"
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

// IconBox рисует векторный значок (путь в квадрате 24×24) квадратом Size×Size dp.
type IconBox struct {
	Node
	Path  *gfx.Path
	Size  float64
	Color gfx.Color
}

func (b *IconBox) PerformLayout(c Constraints) geom.Size {
	return c.Constrain(geom.Size{Width: b.Size, Height: b.Size})
}

func (b *IconBox) Paint(ctx *PaintContext, offset geom.Offset) {
	if b.Path == nil || b.Color.IsTransparent() {
		return
	}
	s := SizeOf(b)
	c := ctx.Canvas
	c.Save()
	// Значок по центру бокса (если ограничения сделали бокс больше значка).
	c.Translate(offset.X+(s.Width-b.Size)/2, offset.Y+(s.Height-b.Size)/2)
	c.Scale(b.Size/24, b.Size/24)
	c.DrawPath(b.Path, gfx.Paint{Color: b.Color})
	c.Restore()
}
