// Package painting — то, как выглядит прямоугольник виджета: фон, градиент, рамка,
// скругления, тени, форма. Аналог painting-библиотеки Flutter (BoxDecoration и компания).
package painting

import (
	"hexago/ui/geom"
	"hexago/ui/gfx"
)

// BoxShape — форма украшения: прямоугольник (со скруглениями) или круг.
type BoxShape uint8

const (
	Rectangle BoxShape = iota
	Circle
)

// BorderSide — одна сторона рамки.
type BorderSide struct {
	Color gfx.Color
	Width float64
}

// Border — рамка из четырёх сторон.
type Border struct {
	Top, Right, Bottom, Left BorderSide
}

// BorderAll — одинаковая рамка со всех сторон, как Border.all во Flutter.
func BorderAll(color gfx.Color, width float64) *Border {
	s := BorderSide{Color: color, Width: width}
	return &Border{s, s, s, s}
}

// uniform сообщает, что все стороны одинаковые — тогда рамку можно рисовать одной обводкой.
func (b *Border) uniform() bool { return b.Top == b.Right && b.Top == b.Bottom && b.Top == b.Left }

// BoxShadow — тень, как BoxShadow во Flutter.
type BoxShadow struct {
	Color        gfx.Color
	Offset       geom.Offset
	BlurRadius   float64
	SpreadRadius float64
}

// Sigma переводит радиус размытия Flutter в σ гауссова размытия (та же формула, что во Flutter).
func (s BoxShadow) Sigma() float64 {
	if s.BlurRadius <= 0 {
		return 0
	}
	return s.BlurRadius*0.57735 + 0.5
}

// Gradient — градиент, заданный относительно прямоугольника (выравниваниями, а не точками).
type Gradient interface {
	// Shader превращает градиент в кисть для конкретного прямоугольника.
	Shader(r geom.Rect) gfx.Shader
}

// LinearGradient — линейный градиент от Begin к End, как во Flutter (по умолчанию слева направо).
type LinearGradient struct {
	Begin, End geom.Alignment
	Colors     []gfx.Color
	Stops      []float64
}

// Shader для прямоугольника r.
func (g LinearGradient) Shader(r geom.Rect) gfx.Shader {
	begin, end := g.Begin, g.End
	if begin == end { // не заданы — слева направо
		begin, end = geom.CenterLeft, geom.CenterRight
	}
	return gfx.LinearGradient{From: begin.Point(r), To: end.Point(r), Colors: g.Colors, Stops: g.Stops}
}

// RadialGradient — круговой градиент из Center; Radius — доля от меньшей стороны прямоугольника
// (0.5 — до края, как во Flutter).
type RadialGradient struct {
	Center geom.Alignment
	Radius float64
	Colors []gfx.Color
	Stops  []float64
}

// Shader для прямоугольника r.
func (g RadialGradient) Shader(r geom.Rect) gfx.Shader {
	radius := g.Radius
	if radius == 0 {
		radius = 0.5
	}
	return gfx.RadialGradient{Center: g.Center.Point(r), Radius: radius * min(r.Width(), r.Height()),
		Colors: g.Colors, Stops: g.Stops}
}

// BoxDecoration — украшение прямоугольника: цвет или градиент, рамка, скругления, тени, форма.
type BoxDecoration struct {
	Color        gfx.Color
	Gradient     Gradient
	Border       *Border
	BorderRadius geom.BorderRadius
	BoxShadow    []BoxShadow
	Shape        BoxShape
}

// IsZero сообщает, что украшение пустое и рисовать нечего.
func (d BoxDecoration) IsZero() bool {
	return d.Color.IsTransparent() && d.Gradient == nil && d.Border == nil && len(d.BoxShadow) == 0
}

// shape возвращает контур украшения для прямоугольника r.
func (d BoxDecoration) shape(r geom.Rect) geom.RRect {
	if d.Shape == Circle {
		side := min(r.Width(), r.Height())
		c := r.Center()
		sq := geom.Rect{Left: c.X - side/2, Top: c.Y - side/2, Right: c.X + side/2, Bottom: c.Y + side/2}
		return geom.RRect{Rect: sq, Radius: geom.RadiusAll(side / 2)}
	}
	return geom.RRect{Rect: r, Radius: d.BorderRadius}
}

// Clip возвращает форму, по которой нужно обрезать содержимое (для ClipRRect по украшению).
func (d BoxDecoration) Clip(r geom.Rect) geom.RRect { return d.shape(r) }

// Contains сообщает, попадает ли точка в форму украшения (для нажатий по кругу).
func (d BoxDecoration) Contains(r geom.Rect, p geom.Offset) bool {
	if d.Shape == Circle {
		c := r.Center()
		return p.Sub(c).Distance() <= min(r.Width(), r.Height())/2
	}
	return r.Contains(p)
}

// Paint рисует украшение в прямоугольнике r: тени, затем фон, затем рамку.
func (d BoxDecoration) Paint(c *gfx.Canvas, r geom.Rect) {
	shape := d.shape(r)
	for _, s := range d.BoxShadow {
		rr := shape.Inflate(s.SpreadRadius)
		rr.Rect = rr.Rect.Translate(s.Offset)
		c.DrawShadow(rr, s.Color, s.Sigma())
	}
	if d.Gradient != nil {
		c.DrawRRect(shape, gfx.Paint{Shader: d.Gradient.Shader(r)})
	} else if !d.Color.IsTransparent() {
		c.DrawRRect(shape, gfx.Paint{Color: d.Color})
	}
	if b := d.Border; b != nil {
		paintBorder(c, shape, b)
	}
}

// paintBorder рисует рамку внутри контура (как во Flutter: рамка не выходит за прямоугольник).
func paintBorder(c *gfx.Canvas, shape geom.RRect, b *Border) {
	if b.uniform() {
		if b.Top.Width <= 0 || b.Top.Color.IsTransparent() {
			return
		}
		inner := shape.Inflate(-b.Top.Width / 2) // середина линии — на половину толщины внутри
		c.DrawRRect(inner, gfx.Paint{Color: b.Top.Color, Style: gfx.Stroke, StrokeWidth: b.Top.Width})
		return
	}
	// Разные стороны: каждая — полоса вдоль своего края (углы не скругляются).
	r := shape.Rect
	side := func(s BorderSide, rect geom.Rect) {
		if s.Width > 0 && !s.Color.IsTransparent() {
			c.DrawRect(rect, gfx.Paint{Color: s.Color})
		}
	}
	side(b.Top, geom.Rect{Left: r.Left, Top: r.Top, Right: r.Right, Bottom: r.Top + b.Top.Width})
	side(b.Bottom, geom.Rect{Left: r.Left, Top: r.Bottom - b.Bottom.Width, Right: r.Right, Bottom: r.Bottom})
	side(b.Left, geom.Rect{Left: r.Left, Top: r.Top, Right: r.Left + b.Left.Width, Bottom: r.Bottom})
	side(b.Right, geom.Rect{Left: r.Right - b.Right.Width, Top: r.Top, Right: r.Right, Bottom: r.Bottom})
}

// LerpBoxDecoration — плавный переход между украшениями: цвет, скругление, рамка и тени
// интерполируются; то, что так не переходит (разная форма, градиент, разное число теней), меняется
// на середине пути. nil — «без украшения» (прозрачное).
func LerpBoxDecoration(a, b *BoxDecoration, t float64) *BoxDecoration {
	if a == nil && b == nil {
		return nil
	}
	var x, y BoxDecoration
	if a != nil {
		x = *a
	}
	if b != nil {
		y = *b
	}
	out := y
	if t < 0.5 {
		out = x
	}
	out.Color = gfx.LerpColor(x.Color, y.Color, t)
	out.BorderRadius = geom.LerpBorderRadius(x.BorderRadius, y.BorderRadius, t)
	if x.Border != nil && y.Border != nil {
		side := func(p, q BorderSide) BorderSide {
			return BorderSide{Color: gfx.LerpColor(p.Color, q.Color, t), Width: geom.Lerp(p.Width, q.Width, t)}
		}
		out.Border = &Border{side(x.Border.Top, y.Border.Top), side(x.Border.Right, y.Border.Right),
			side(x.Border.Bottom, y.Border.Bottom), side(x.Border.Left, y.Border.Left)}
	}
	if len(x.BoxShadow) == len(y.BoxShadow) && len(x.BoxShadow) > 0 {
		out.BoxShadow = make([]BoxShadow, len(x.BoxShadow))
		for i := range x.BoxShadow {
			p, q := x.BoxShadow[i], y.BoxShadow[i]
			out.BoxShadow[i] = BoxShadow{
				Color: gfx.LerpColor(p.Color, q.Color, t), Offset: geom.LerpOffset(p.Offset, q.Offset, t),
				BlurRadius: geom.Lerp(p.BlurRadius, q.BlurRadius, t), SpreadRadius: geom.Lerp(p.SpreadRadius, q.SpreadRadius, t),
			}
		}
	}
	return &out
}
