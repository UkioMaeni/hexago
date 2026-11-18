package material

import (
	"math"

	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/rendering"
	"hexago/ui/widgets"
)

// Переключатели Material 3. Значение хранит родитель: переключатель только показывает его и сообщает
// о нажатии через OnChanged; родитель меняет своё состояние — переключатель перерисовывается.
// OnChanged nil — переключатель выключен.
//
//	material.Checkbox{Value: c.agree, OnChanged: func(v bool) { c.SetState(func() { c.agree = v }) }}

// Checkbox — флажок: квадрат 18 dp, со скруглением 2 dp; место для касания 48 dp.
type Checkbox struct {
	widgets.Base
	Value     bool
	OnChanged func(bool)
}

func (b Checkbox) Build(ctx widgets.BuildContext) widgets.Widget {
	c := Of(ctx).Colors
	enabled := b.OnChanged != nil
	box, mark, border := c.Primary, c.OnPrimary, c.OnSurfaceVariant
	if !enabled {
		box, mark, border = c.OnSurface.WithOpacity(DisabledContent), c.Surface, c.OnSurface.WithOpacity(DisabledContent)
	}
	paint := func(cv *gfx.Canvas, s geom.Size) {
		r := geom.RRect{Rect: geom.RectFromSize(geom.Zero, s), Radius: geom.RadiusAll(2)}
		if !b.Value {
			inner := geom.RRect{Rect: r.Rect.Inflate(-1), Radius: geom.RadiusAll(1)}
			cv.DrawRRect(inner, gfx.Paint{Color: border, Style: gfx.Stroke, StrokeWidth: 2})
			return
		}
		cv.DrawRRect(r, gfx.Paint{Color: box})
		var check gfx.Path
		check.MoveTo(4, 9.2)
		check.LineTo(7.4, 12.6)
		check.LineTo(14.2, 5.8)
		cv.DrawPath(&check, gfx.Paint{Color: mark, Style: gfx.Stroke, StrokeWidth: 2})
	}
	return selectable(ctx, enabled, b.Value, func() { b.OnChanged(!b.Value) }, 18, 18, paint)
}

// Radio — переключатель «один из»: круг 20 dp. Выбран, когда Value == GroupValue; при нажатии
// OnChanged получает Value.
//
//	material.Radio[string]{Value: "чай", GroupValue: c.drink, OnChanged: c.choose}
type Radio[T comparable] struct {
	widgets.Base
	Value, GroupValue T
	OnChanged         func(T)
}

func (b Radio[T]) Build(ctx widgets.BuildContext) widgets.Widget {
	c := Of(ctx).Colors
	selected, enabled := b.Value == b.GroupValue, b.OnChanged != nil
	ring := c.OnSurfaceVariant
	if selected {
		ring = c.Primary
	}
	if !enabled {
		ring = c.OnSurface.WithOpacity(DisabledContent)
	}
	paint := func(cv *gfx.Canvas, s geom.Size) {
		center := geom.Offset{X: 10, Y: 10}
		cv.DrawCircle(center, 9, gfx.Paint{Color: ring, Style: gfx.Stroke, StrokeWidth: 2})
		if selected {
			cv.DrawCircle(center, 5, gfx.Paint{Color: ring})
		}
	}
	return selectable(ctx, enabled, selected, func() { b.OnChanged(b.Value) }, 20, 20, paint)
}

// selectable — общее у флажка и переключателя «один из»: круг касания 40 dp с подложкой нажатия,
// место для касания 48 dp, сам значок рисует paint.
func selectable(ctx widgets.BuildContext, enabled, on bool, toggle func(), w, h float64, paint rendering.PaintFunc) widgets.Widget {
	c := Of(ctx).Colors
	overlay := c.OnSurface
	if on {
		overlay = c.Primary
	}
	var tap func()
	if enabled {
		tap = toggle
	}
	return widgets.Padding{Padding: geom.All(4), Child: widgets.SizedBox{Width: new(40.0), Height: new(40.0),
		Child: widgets.ClipOval{Child: InkWell{OnTap: tap, Radius: 20, OverlayColor: overlay,
			Child: widgets.Center{Child: widgets.CustomPaint{Size: geom.Size{Width: w, Height: h}, Painter: paint}}}}}}
}

// Switch — выключатель: дорожка 52×32 dp и ползунок. Включён — дорожка Primary и большой ползунок
// справа, выключен — маленький ползунок слева.
type Switch struct {
	widgets.Base
	Value     bool
	OnChanged func(bool)
}

func (b Switch) Build(ctx widgets.BuildContext) widgets.Widget {
	c := Of(ctx).Colors
	enabled := b.OnChanged != nil
	track, outline, thumb := c.SurfaceContainerHighest, c.Outline, c.Outline
	if b.Value {
		track, outline, thumb = c.Primary, c.Primary, c.OnPrimary
	}
	if !enabled {
		if b.Value {
			track, outline, thumb = c.OnSurface.WithOpacity(DisabledSurface), gfx.Transparent, c.Surface
		} else {
			track, outline, thumb = c.SurfaceContainerHighest.WithOpacity(DisabledSurface),
				c.OnSurface.WithOpacity(DisabledSurface), c.OnSurface.WithOpacity(DisabledContent)
		}
	}
	paint := func(cv *gfx.Canvas, s geom.Size) {
		r := geom.RRect{Rect: geom.RectFromSize(geom.Zero, s), Radius: geom.RadiusAll(16)}
		cv.DrawRRect(r, gfx.Paint{Color: track})
		if !b.Value && !outline.IsTransparent() {
			cv.DrawRRect(geom.RRect{Rect: r.Rect.Inflate(-1), Radius: geom.RadiusAll(15)},
				gfx.Paint{Color: outline, Style: gfx.Stroke, StrokeWidth: 2})
		}
		if b.Value {
			cv.DrawCircle(geom.Offset{X: 36, Y: 16}, 12, gfx.Paint{Color: thumb})
		} else {
			cv.DrawCircle(geom.Offset{X: 16, Y: 16}, 8, gfx.Paint{Color: thumb})
		}
	}
	var tap func()
	if enabled {
		tap = func() { b.OnChanged(!b.Value) }
	}
	return widgets.Padding{Padding: geom.Symmetric(4, 8), Child: InkWell{OnTap: tap, Radius: 16,
		Child: widgets.CustomPaint{Size: geom.Size{Width: 52, Height: 32}, Painter: paint}}}
}

// Slider — ползунок выбора числа от Min до Max (по умолчанию 0…1): нажатие или перетаскивание.
// Divisions > 0 — значение только с шагом (Max−Min)/Divisions.
type Slider struct {
	widgets.State
	Value, Min, Max float64
	Divisions       int
	OnChanged       func(float64)
	OnChangeEnd     func(float64) // палец отпущен — например, сохранить настройку

	width float64 // ширина при прошлой отрисовке: по ней точка касания переводится в значение
}

// sliderInset — отступ дорожки от краёв: чтобы ползунок у края помещался целиком.
const sliderInset = 12.0

func (s *Slider) Build(ctx widgets.BuildContext) widgets.Widget {
	c := Of(ctx).Colors
	lo, hi := s.Min, s.Max
	if hi <= lo {
		hi = lo + 1
	}
	frac := math.Max(0, math.Min(1, (s.Value-lo)/(hi-lo)))
	enabled := s.OnChanged != nil
	active, inactive, handle := c.Primary, c.SecondaryContainer, c.Primary
	if !enabled {
		active, inactive, handle = c.OnSurface.WithOpacity(DisabledContent), c.OnSurface.WithOpacity(DisabledSurface),
			c.OnSurface.WithOpacity(DisabledContent)
	}
	paint := func(cv *gfx.Canvas, size geom.Size) {
		s.width = size.Width
		y := size.Height / 2
		w := size.Width - 2*sliderInset
		x := sliderInset + w*frac
		track := func(from, to float64, col gfx.Color) {
			if to > from {
				cv.DrawRRect(geom.RRect{Rect: geom.Rect{Left: from, Top: y - 2, Right: to, Bottom: y + 2}, Radius: geom.RadiusAll(2)},
					gfx.Paint{Color: col})
			}
		}
		track(sliderInset, x, active)
		track(x, sliderInset+w, inactive)
		cv.DrawCircle(geom.Offset{X: x, Y: y}, 10, gfx.Paint{Color: handle})
	}
	value := func(localX float64) float64 {
		w := s.width - 2*sliderInset
		if w <= 0 {
			return s.Value
		}
		f := math.Max(0, math.Min(1, (localX-sliderInset)/w))
		if s.Divisions > 0 {
			f = math.Round(f*float64(s.Divisions)) / float64(s.Divisions)
		}
		return lo + f*(hi-lo)
	}
	set := func(x float64) {
		if v := value(x); v != s.Value {
			s.OnChanged(v)
		}
	}
	end := func() {
		if s.OnChangeEnd != nil {
			s.OnChangeEnd(s.Value)
		}
	}
	slider := widgets.SizedBox{Height: new(48.0), Width: new(widgets.Infinity),
		Child: widgets.CustomPaint{Painter: paint}}
	if !enabled {
		return slider
	}
	return widgets.GestureDetector{
		Behavior:               rendering.Opaque,
		OnTapUp:                func(d gestures.TapUpDetails) { set(d.Local.X); end() },
		OnHorizontalDragStart:  func(d gestures.DragStartDetails) { set(d.Local.X) },
		OnHorizontalDragUpdate: func(d gestures.DragUpdateDetails) { set(d.Local.X) },
		OnHorizontalDragEnd:    func(gestures.DragEndDetails) { end() },
		Child:                  slider,
	}
}

// LinearProgressIndicator — полоса прогресса 4 dp: Value от 0 до 1.
// Бесконечный (без значения) появится вместе с анимациями.
type LinearProgressIndicator struct {
	widgets.Base
	Value float64
}

func (p LinearProgressIndicator) Build(ctx widgets.BuildContext) widgets.Widget {
	c := Of(ctx).Colors
	v := math.Max(0, math.Min(1, p.Value))
	return widgets.SizedBox{Height: new(4.0), Width: new(widgets.Infinity), Child: widgets.CustomPaint{
		Painter: func(cv *gfx.Canvas, s geom.Size) {
			cv.DrawRRect(geom.RRect{Rect: geom.RectFromSize(geom.Zero, s), Radius: geom.RadiusAll(2)},
				gfx.Paint{Color: c.SecondaryContainer})
			if v > 0 {
				cv.DrawRRect(geom.RRect{Rect: geom.RectXYWH(0, 0, s.Width*v, s.Height), Radius: geom.RadiusAll(2)},
					gfx.Paint{Color: c.Primary})
			}
		},
	}}
}

// CircularProgressIndicator — круговой прогресс: дуга толщиной 4 dp, Value от 0 до 1.
// Size — диаметр, по умолчанию 40 dp.
type CircularProgressIndicator struct {
	widgets.Base
	Value float64
	Size  float64
}

func (p CircularProgressIndicator) Build(ctx widgets.BuildContext) widgets.Widget {
	c := Of(ctx).Colors
	d := p.Size
	if d <= 0 {
		d = 40
	}
	v := math.Max(0, math.Min(1, p.Value))
	return widgets.CustomPaint{Size: geom.Size{Width: d, Height: d}, Painter: func(cv *gfx.Canvas, s geom.Size) {
		center := geom.Offset{X: s.Width / 2, Y: s.Height / 2}
		r := s.Width/2 - 2
		cv.DrawCircle(center, r, gfx.Paint{Color: c.SecondaryContainer, Style: gfx.Stroke, StrokeWidth: 4})
		if v > 0 {
			var arc gfx.Path
			arc.AddArc(center, r, -math.Pi/2, 2*math.Pi*v)
			cv.DrawPath(&arc, gfx.Paint{Color: c.Primary, Style: gfx.Stroke, StrokeWidth: 4, StrokeCap: gfx.RoundCap})
		}
	}}
}
