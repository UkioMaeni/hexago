package rendering

import (
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/painting"
)

// ColoredBox заливает свой прямоугольник цветом и рисует ребёнка поверх.
type ColoredBox struct {
	Proxy
	Color gfx.Color
}

func (b *ColoredBox) Paint(ctx *PaintContext, offset geom.Offset) {
	if s := SizeOf(b); !s.IsEmpty() && !b.Color.IsTransparent() {
		ctx.Canvas.DrawRect(geom.RectFromSize(offset, s), gfx.Paint{Color: b.Color})
	}
	PaintChildren(ctx, b, offset)
}

func (b *ColoredBox) HitTestSelf(geom.Offset) bool { return true }

// PaddingBox добавляет отступы вокруг ребёнка.
type PaddingBox struct {
	Proxy
	Padding geom.EdgeInsets
}

func (b *PaddingBox) PerformLayout(c Constraints) geom.Size {
	p := b.Padding
	ch := Child(b)
	if ch == nil {
		return geom.Size{Width: p.Horizontal(), Height: p.Vertical()}
	}
	s := Layout(ch, c.Deflate(p))
	SetOffset(ch, p.TopLeft())
	return geom.Size{Width: s.Width + p.Horizontal(), Height: s.Height + p.Vertical()}
}

// AlignBox ставит ребёнка в точку Alignment внутри себя (Align и Center).
// Если задан WidthFactor, ширина бокса — ширина ребёнка × фактор; иначе — всё доступное место.
type AlignBox struct {
	Proxy
	Alignment    geom.Alignment
	WidthFactor  *float64
	HeightFactor *float64
}

func (b *AlignBox) PerformLayout(c Constraints) geom.Size {
	shrinkW := b.WidthFactor != nil || !c.HasBoundedWidth()
	shrinkH := b.HeightFactor != nil || !c.HasBoundedHeight()
	ch := Child(b)
	if ch == nil {
		w, h := Inf, Inf
		if shrinkW {
			w = 0
		}
		if shrinkH {
			h = 0
		}
		return c.Constrain(geom.Size{Width: w, Height: h})
	}
	s := Layout(ch, c.Loosen())
	size := geom.Size{Width: Inf, Height: Inf}
	if shrinkW {
		size.Width = s.Width * factor(b.WidthFactor)
	}
	if shrinkH {
		size.Height = s.Height * factor(b.HeightFactor)
	}
	size = c.Constrain(size)
	SetOffset(ch, b.Alignment.Along(geom.Size{Width: size.Width - s.Width, Height: size.Height - s.Height}))
	return size
}

func factor(f *float64) float64 {
	if f == nil {
		return 1
	}
	return *f
}

// ConstrainedBox добавляет ребёнку свои ограничения (SizedBox, ConstrainedBox).
type ConstrainedBox struct {
	Proxy
	Additional Constraints
}

func (b *ConstrainedBox) PerformLayout(c Constraints) geom.Size {
	inner := b.Additional.Enforce(c)
	if ch := Child(b); ch != nil {
		return Layout(ch, inner)
	}
	return inner.Constrain(geom.Size{})
}

// LimitedBox ограничивает размер, только если родитель не ограничил его сам
// (например, внутри ListView по оси прокрутки).
type LimitedBox struct {
	Proxy
	MaxWidth, MaxHeight float64
}

func (b *LimitedBox) PerformLayout(c Constraints) geom.Size {
	limited := c
	if !c.HasBoundedWidth() {
		limited.MaxWidth = max(c.MinWidth, b.MaxWidth)
	}
	if !c.HasBoundedHeight() {
		limited.MaxHeight = max(c.MinHeight, b.MaxHeight)
	}
	if ch := Child(b); ch != nil {
		return c.Constrain(Layout(ch, limited))
	}
	return limited.Constrain(geom.Size{})
}

// AspectRatioBox держит соотношение сторон ширина/высота = Ratio.
type AspectRatioBox struct {
	Proxy
	Ratio float64
}

func (b *AspectRatioBox) PerformLayout(c Constraints) geom.Size {
	size := b.size(c)
	if ch := Child(b); ch != nil {
		Layout(ch, Tight(size))
	}
	return size
}

func (b *AspectRatioBox) size(c Constraints) geom.Size {
	if c.IsTight() || b.Ratio <= 0 {
		return c.Smallest()
	}
	w := c.MaxWidth
	var h float64
	if c.HasBoundedWidth() {
		h = w / b.Ratio
	} else {
		h = c.MaxHeight
		w = h * b.Ratio
	}
	if w > c.MaxWidth {
		w, h = c.MaxWidth, c.MaxWidth/b.Ratio
	}
	if h > c.MaxHeight {
		h, w = c.MaxHeight, c.MaxHeight*b.Ratio
	}
	if w < c.MinWidth {
		w, h = c.MinWidth, c.MinWidth/b.Ratio
	}
	if h < c.MinHeight {
		h, w = c.MinHeight, c.MinHeight*b.Ratio
	}
	return c.Constrain(geom.Size{Width: w, Height: h})
}

// FractionalBox даёт ребёнку долю доступного места (FractionallySizedBox).
type FractionalBox struct {
	Proxy
	WidthFactor  *float64
	HeightFactor *float64
	Alignment    geom.Alignment
}

func (b *FractionalBox) PerformLayout(c Constraints) geom.Size {
	inner := c
	if b.WidthFactor != nil && c.HasBoundedWidth() {
		w := c.MaxWidth * *b.WidthFactor
		inner.MinWidth, inner.MaxWidth = w, w
	}
	if b.HeightFactor != nil && c.HasBoundedHeight() {
		h := c.MaxHeight * *b.HeightFactor
		inner.MinHeight, inner.MaxHeight = h, h
	}
	ch := Child(b)
	if ch == nil {
		return c.Constrain(inner.Constrain(geom.Size{}))
	}
	s := Layout(ch, inner)
	size := c.Constrain(s)
	SetOffset(ch, b.Alignment.Along(geom.Size{Width: size.Width - s.Width, Height: size.Height - s.Height}))
	return size
}

// DecoratedBox рисует украшение (фон, рамку, тени) под ребёнком или поверх него.
type DecoratedBox struct {
	Proxy
	Decoration painting.BoxDecoration
	Foreground bool // рисовать поверх ребёнка
}

func (b *DecoratedBox) Paint(ctx *PaintContext, offset geom.Offset) {
	r := geom.RectFromSize(offset, SizeOf(b))
	if !b.Foreground {
		b.Decoration.Paint(ctx.Canvas, r)
	}
	PaintChildren(ctx, b, offset)
	if b.Foreground {
		b.Decoration.Paint(ctx.Canvas, r)
	}
}

func (b *DecoratedBox) HitTestSelf(pos geom.Offset) bool {
	return b.Decoration.Contains(geom.RectFromSize(geom.Zero, SizeOf(b)), pos)
}

// OpacityBox рисует ребёнка полупрозрачным: 0 — невидим, 1 — как есть.
type OpacityBox struct {
	Proxy
	Opacity float64
}

func (b *OpacityBox) Paint(ctx *PaintContext, offset geom.Offset) {
	if b.Opacity <= 0 {
		return
	}
	ctx.Canvas.Save()
	ctx.Canvas.MultiplyOpacity(b.Opacity)
	PaintChildren(ctx, b, offset)
	ctx.Canvas.Restore()
}

// ClipKind — форма обрезки.
type ClipKind uint8

const (
	ClipToRect  ClipKind = iota // прямоугольник
	ClipToRRect                 // прямоугольник со скруглениями Radius
	ClipToOval                  // эллипс, вписанный в прямоугольник
)

// ClipBox обрезает ребёнка по своей форме: всё, что выходит за неё, не рисуется и не ловит касания.
type ClipBox struct {
	Proxy
	Kind   ClipKind
	Radius geom.BorderRadius
}

func (b *ClipBox) Paint(ctx *PaintContext, offset geom.Offset) {
	r := geom.RectFromSize(offset, SizeOf(b))
	c := ctx.Canvas
	c.Save()
	switch b.Kind {
	case ClipToRect:
		c.ClipRect(r)
	case ClipToRRect:
		c.ClipRRect(geom.RRect{Rect: r, Radius: b.Radius})
	case ClipToOval:
		var p gfx.Path
		p.AddOval(r, false)
		c.ClipPath(&p)
	}
	PaintChildren(ctx, b, offset)
	c.Restore()
}

func (b *ClipBox) HitTest(result *HitTestResult, pos geom.Offset) bool {
	s := SizeOf(b)
	if b.Kind == ClipToOval {
		dx, dy := (pos.X-s.Width/2)/(s.Width/2), (pos.Y-s.Height/2)/(s.Height/2)
		if dx*dx+dy*dy > 1 {
			return false
		}
	}
	return DefaultHitTest(b, result, pos)
}

// TransformBox поворачивает, масштабирует или сдвигает ребёнка при отрисовке.
// Лэйаут ребёнка не меняется — как Transform во Flutter. Преобразование применяется
// относительно точки Alignment внутри бокса (Center — поворот вокруг середины).
type TransformBox struct {
	Proxy
	Transform geom.Affine
	Alignment geom.Alignment
}

// effective — преобразование из координат ребёнка в координаты родителя бокса.
func (b *TransformBox) effective() geom.Affine {
	o := b.Alignment.Point(geom.RectFromSize(geom.Zero, SizeOf(b)))
	return geom.Translation(o.X, o.Y).Mul(b.Transform).Mul(geom.Translation(-o.X, -o.Y))
}

func (b *TransformBox) Paint(ctx *PaintContext, offset geom.Offset) {
	c := ctx.Canvas
	c.Save()
	c.Translate(offset.X, offset.Y)
	c.Transform(b.effective())
	PaintChildren(ctx, b, geom.Zero)
	c.Restore()
}

func (b *TransformBox) HitTest(result *HitTestResult, pos geom.Offset) bool {
	inv, ok := b.effective().Invert()
	if !ok {
		return false
	}
	local := inv.Apply(pos)
	result.PushTransform(inv)
	hit := HitTestChildren(b, result, local)
	result.PopTransform()
	if !hit {
		return false
	}
	result.Add(b, pos)
	return true
}

// OffstageBox раскладывает ребёнка, но не рисует его и не занимает места.
type OffstageBox struct {
	Proxy
	Offstage bool
}

func (b *OffstageBox) PerformLayout(c Constraints) geom.Size {
	if !b.Offstage {
		return b.Proxy.PerformLayout(c)
	}
	if ch := Child(b); ch != nil {
		Layout(ch, c)
	}
	return c.Smallest()
}

func (b *OffstageBox) Paint(ctx *PaintContext, offset geom.Offset) {
	if !b.Offstage {
		PaintChildren(ctx, b, offset)
	}
}

func (b *OffstageBox) HitTest(result *HitTestResult, pos geom.Offset) bool {
	return !b.Offstage && DefaultHitTest(b, result, pos)
}

// PaintFunc — своя отрисовка для CustomPaint: рисует на холсте в прямоугольнике (0, 0, size).
type PaintFunc func(c *gfx.Canvas, size geom.Size)

// CustomPaintBox вызывает свою функцию отрисовки под ребёнком (Painter) и поверх него (Foreground).
type CustomPaintBox struct {
	Proxy
	Painter    PaintFunc
	Foreground PaintFunc
	Preferred  geom.Size // размер без ребёнка
}

func (b *CustomPaintBox) PerformLayout(c Constraints) geom.Size {
	if Child(b) != nil {
		return b.Proxy.PerformLayout(c)
	}
	return c.Constrain(b.Preferred)
}

func (b *CustomPaintBox) Paint(ctx *PaintContext, offset geom.Offset) {
	call := func(f PaintFunc) {
		if f == nil {
			return
		}
		ctx.Canvas.Save()
		ctx.Canvas.Translate(offset.X, offset.Y)
		f(ctx.Canvas, SizeOf(b))
		ctx.Canvas.Restore()
	}
	call(b.Painter)
	PaintChildren(ctx, b, offset)
	call(b.Foreground)
}
