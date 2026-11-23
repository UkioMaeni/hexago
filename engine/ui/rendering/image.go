package rendering

import (
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/painting"
)

// ImageBox рисует картинку в своём прямоугольнике, как RenderImage во Flutter. Один пиксель
// картинки — один логический пиксель (dp), если Width и Height не задают другой размер.
type ImageBox struct {
	Node
	Image         *gfx.Bitmap // nil — картинки ещё нет: бокс занимает Width×Height (или наименьший размер)
	Width, Height *float64
	Fit           painting.BoxFit
	Alignment     geom.Alignment
}

func (b *ImageBox) PerformLayout(c Constraints) geom.Size {
	c = TightFor(b.Width, b.Height).Enforce(c)
	if b.Image == nil {
		return c.Smallest()
	}
	return c.ConstrainAspect(geom.Size{Width: float64(b.Image.Width()), Height: float64(b.Image.Height())})
}

func (b *ImageBox) Paint(ctx *PaintContext, offset geom.Offset) {
	size := SizeOf(b)
	if b.Image == nil || size.IsEmpty() {
		return
	}
	input := geom.Size{Width: float64(b.Image.Width()), Height: float64(b.Image.Height())}
	src, dst := painting.FitSizes(b.Fit, input, size)
	ctx.Canvas.DrawBitmap(b.Image,
		painting.Inscribe(b.Alignment, src, geom.Rect{Right: input.Width, Bottom: input.Height}),
		painting.Inscribe(b.Alignment, dst, geom.RectFromSize(offset, size)), 1)
}

func (b *ImageBox) HitTestSelf(geom.Offset) bool { return true }
