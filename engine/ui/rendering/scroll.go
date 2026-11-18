package rendering

import (
	"math"

	"hexago/ui/geom"
)

// ScrollMetrics — размеры прокрутки после раскладки (в dp, по оси прокрутки).
type ScrollMetrics struct {
	Viewport float64 // длина видимой области
	Content  float64 // длина содержимого
}

// MaxOffset — насколько содержимое можно прокрутить.
func (m ScrollMetrics) MaxOffset() float64 { return math.Max(0, m.Content-m.Viewport) }

// ScrollViewportBox — видимая область прокрутки: ребёнок раскладывается без ограничения по оси
// прокрутки (по другой оси — как сам бокс), сдвигается на Offset и обрезается по краям бокса.
// Размеры записываются в Metrics — по ним виджет прокрутки знает, докуда можно прокрутить.
type ScrollViewportBox struct {
	Proxy
	Axis    Axis
	Offset  float64
	Metrics *ScrollMetrics
}

func (b *ScrollViewportBox) PerformLayout(c Constraints) geom.Size {
	ch := Child(b)
	if ch == nil {
		return c.Smallest()
	}
	cc := Constraints{MinWidth: c.MinWidth, MaxWidth: c.MaxWidth, MinHeight: 0, MaxHeight: Inf}
	if b.Axis == Horizontal {
		cc = Constraints{MinWidth: 0, MaxWidth: Inf, MinHeight: c.MinHeight, MaxHeight: c.MaxHeight}
	}
	content := Layout(ch, cc)
	size := c.Constrain(content)
	m := ScrollMetrics{Viewport: size.Height, Content: content.Height}
	if b.Axis == Horizontal {
		m = ScrollMetrics{Viewport: size.Width, Content: content.Width}
	}
	if b.Metrics != nil {
		*b.Metrics = m
	}
	off := math.Max(0, math.Min(b.Offset, m.MaxOffset()))
	if b.Axis == Horizontal {
		SetOffset(ch, geom.Offset{X: -off})
	} else {
		SetOffset(ch, geom.Offset{Y: -off})
	}
	return size
}

func (b *ScrollViewportBox) Paint(ctx *PaintContext, offset geom.Offset) {
	c := ctx.Canvas
	c.Save()
	c.ClipRect(geom.RectFromSize(offset, SizeOf(b)))
	PaintChildren(ctx, b, offset)
	c.Restore()
}

// PageViewportBox — страницы во всю свою ширину (или высоту), одна за другой; Page — какая страница
// у левого края, дробная во время перелистывания. Видны и рисуются только страницы в кадре.
type PageViewportBox struct {
	Node
	Axis    Axis
	Page    float64
	Metrics *ScrollMetrics // Viewport — размер страницы, Content — всех страниц
}

func (b *PageViewportBox) PerformLayout(c Constraints) geom.Size {
	size := c.Biggest()
	if b.Metrics != nil {
		n := float64(len(Children(b)))
		if b.Axis == Vertical {
			*b.Metrics = ScrollMetrics{Viewport: size.Height, Content: n * size.Height}
		} else {
			*b.Metrics = ScrollMetrics{Viewport: size.Width, Content: n * size.Width}
		}
	}
	for i, ch := range Children(b) {
		Layout(ch, Tight(size))
		pos := (float64(i) - b.Page)
		if b.Axis == Vertical {
			SetOffset(ch, geom.Offset{Y: pos * size.Height})
		} else {
			SetOffset(ch, geom.Offset{X: pos * size.Width})
		}
	}
	return size
}

func (b *PageViewportBox) Paint(ctx *PaintContext, offset geom.Offset) {
	c := ctx.Canvas
	size := SizeOf(b)
	c.Save()
	c.ClipRect(geom.RectFromSize(offset, size))
	for _, ch := range Children(b) {
		o := OffsetOf(ch)
		if math.Abs(o.X) < size.Width && math.Abs(o.Y) < size.Height {
			ctx.PaintChild(ch, offset)
		}
	}
	c.Restore()
}
