package widgets

import (
	"reflect"
	"time"

	"hexago/ui/animation"
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/painting"
)

// Неявные анимации: виджет сам плавно переходит к новым значениям параметров, когда родитель их
// меняет, — как AnimatedOpacity, AnimatedContainer во Flutter. Duration — длительность перехода
// (0 — 300 мс), Curve — кривая (nil — animation.Standard, плавная кривая Material 3).
//
//	widgets.AnimatedOpacity{Opacity: op, Duration: 200 * time.Millisecond, Child: ...}

const defaultDuration = 300 * time.Millisecond

// tween — значение, которое плавно переходит к новому при каждом set.
type tween[T any] struct {
	from, to T
	ctrl     *animation.Controller
	lerp     func(a, b T, t float64) T
	equal    func(a, b T) bool
}

func newTween[T comparable](ctx BuildContext, value T, lerp func(a, b T, t float64) T) *tween[T] {
	return &tween[T]{from: value, to: value, ctrl: NewAnimation(ctx, defaultDuration), lerp: lerp,
		equal: func(a, b T) bool { return a == b }}
}

// set начинает переход к value от текущего значения (если value новое).
func (a *tween[T]) set(value T, d time.Duration, curve animation.Curve) {
	if a.equal(value, a.to) {
		return
	}
	a.from, a.to = a.value(), value
	if d <= 0 {
		d = defaultDuration
	}
	if curve == nil {
		curve = animation.Standard
	}
	a.ctrl.SetValue(0)
	a.ctrl.AnimateTo(1, d, curve)
}

// value — текущее значение перехода.
func (a *tween[T]) value() T {
	if !a.ctrl.IsAnimating() && a.ctrl.Value() >= 1 {
		return a.to
	}
	return a.lerp(a.from, a.to, a.ctrl.Value())
}

// AnimatedValue плавно ведёт число к Value и строит виджет по текущему значению — как
// TweenAnimationBuilder во Flutter. Основа остальных неявных анимаций; своя — одна функция:
//
//	widgets.AnimatedValue{Value: c.progress, Builder: func(ctx widgets.BuildContext, v float64) widgets.Widget {
//		return material.LinearProgressIndicator{Value: v}
//	}}
type AnimatedValue struct {
	State
	Value    float64
	Duration time.Duration
	Curve    animation.Curve
	Builder  func(ctx BuildContext, value float64) Widget

	t *tween[float64]
}

func (a *AnimatedValue) Init(ctx BuildContext) { a.t = newTween(ctx, a.Value, geom.Lerp) }

func (a *AnimatedValue) Update(_ BuildContext, _ AnimatedValue) {
	a.t.set(a.Value, a.Duration, a.Curve)
}

func (a *AnimatedValue) Build(ctx BuildContext) Widget { return a.Builder(ctx, a.t.value()) }

// AnimatedOpacity — плавное изменение прозрачности.
type AnimatedOpacity struct {
	Base
	Opacity  float64
	Duration time.Duration
	Curve    animation.Curve
	Child    Widget
}

func (w AnimatedOpacity) Build(BuildContext) Widget {
	return AnimatedValue{Value: w.Opacity, Duration: w.Duration, Curve: w.Curve,
		Builder: func(_ BuildContext, v float64) Widget { return Opacity{Opacity: v, Child: w.Child} }}
}

// AnimatedScale — плавное изменение масштаба (вокруг центра).
type AnimatedScale struct {
	Base
	Scale    float64
	Duration time.Duration
	Curve    animation.Curve
	Child    Widget
}

func (w AnimatedScale) Build(BuildContext) Widget {
	return AnimatedValue{Value: w.Scale, Duration: w.Duration, Curve: w.Curve,
		Builder: func(_ BuildContext, v float64) Widget { return Scale(v, w.Child) }}
}

// AnimatedRotation — плавный поворот: Angle в радианах, вокруг центра.
type AnimatedRotation struct {
	Base
	Angle    float64
	Duration time.Duration
	Curve    animation.Curve
	Child    Widget
}

func (w AnimatedRotation) Build(BuildContext) Widget {
	return AnimatedValue{Value: w.Angle, Duration: w.Duration, Curve: w.Curve,
		Builder: func(_ BuildContext, v float64) Widget { return Rotate(v, w.Child) }}
}

// AnimatedAlign — плавное изменение выравнивания ребёнка.
type AnimatedAlign struct {
	State
	Alignment geom.Alignment
	Duration  time.Duration
	Curve     animation.Curve
	Child     Widget

	t *tween[geom.Alignment]
}

func (w *AnimatedAlign) Init(ctx BuildContext) { w.t = newTween(ctx, w.Alignment, geom.LerpAlignment) }

func (w *AnimatedAlign) Update(BuildContext, AnimatedAlign) {
	w.t.set(w.Alignment, w.Duration, w.Curve)
}

func (w *AnimatedAlign) Build(BuildContext) Widget {
	return Align{Alignment: w.t.value(), Child: w.Child}
}

// AnimatedPadding — плавное изменение отступов.
type AnimatedPadding struct {
	State
	Padding  geom.EdgeInsets
	Duration time.Duration
	Curve    animation.Curve
	Child    Widget

	t *tween[geom.EdgeInsets]
}

func (w *AnimatedPadding) Init(ctx BuildContext) {
	w.t = newTween(ctx, w.Padding, geom.LerpEdgeInsets)
}

func (w *AnimatedPadding) Update(BuildContext, AnimatedPadding) {
	w.t.set(w.Padding, w.Duration, w.Curve)
}

func (w *AnimatedPadding) Build(BuildContext) Widget {
	return Padding{Padding: w.t.value(), Child: w.Child}
}

// AnimatedContainer — Container, который плавно переходит к новым размерам, цвету, отступам,
// выравниванию и украшению (цвет, скругление, рамка, тени). Размер переходит плавно, только если
// задан и до, и после.
type AnimatedContainer struct {
	State
	Width, Height *float64
	Padding       geom.EdgeInsets
	Margin        geom.EdgeInsets
	Alignment     *geom.Alignment
	Color         gfx.Color
	Decoration    *painting.BoxDecoration
	Duration      time.Duration
	Curve         animation.Curve
	Child         Widget

	t *tween[containerLook]
}

// containerLook — то, что у AnimatedContainer переходит плавно.
type containerLook struct {
	width, height   float64
	hasW, hasH      bool
	padding, margin geom.EdgeInsets
	align           geom.Alignment
	hasAlign        bool
	color           gfx.Color
	decoration      painting.BoxDecoration
	hasDecoration   bool
}

func (w *AnimatedContainer) look() containerLook {
	l := containerLook{padding: w.Padding, margin: w.Margin, color: w.Color}
	if w.Width != nil {
		l.width, l.hasW = *w.Width, true
	}
	if w.Height != nil {
		l.height, l.hasH = *w.Height, true
	}
	if w.Alignment != nil {
		l.align, l.hasAlign = *w.Alignment, true
	}
	if w.Decoration != nil {
		l.decoration, l.hasDecoration = *w.Decoration, true
	}
	return l
}

func lerpLook(a, b containerLook, t float64) containerLook {
	out := b
	if a.hasW && b.hasW {
		out.width = geom.Lerp(a.width, b.width, t)
	}
	if a.hasH && b.hasH {
		out.height = geom.Lerp(a.height, b.height, t)
	}
	out.padding = geom.LerpEdgeInsets(a.padding, b.padding, t)
	out.margin = geom.LerpEdgeInsets(a.margin, b.margin, t)
	if a.hasAlign && b.hasAlign {
		out.align = geom.LerpAlignment(a.align, b.align, t)
	}
	out.color = gfx.LerpColor(a.color, b.color, t)
	if a.hasDecoration || b.hasDecoration {
		var x, y *painting.BoxDecoration
		if a.hasDecoration {
			x = &a.decoration
		}
		if b.hasDecoration {
			y = &b.decoration
		}
		out.decoration, out.hasDecoration = *painting.LerpBoxDecoration(x, y, t), true
	}
	return out
}

func (w *AnimatedContainer) Init(ctx BuildContext) {
	w.t = &tween[containerLook]{from: w.look(), to: w.look(), ctrl: NewAnimation(ctx, defaultDuration), lerp: lerpLook,
		equal: func(a, b containerLook) bool { return reflect.DeepEqual(a, b) }}
}

func (w *AnimatedContainer) Update(BuildContext, AnimatedContainer) {
	w.t.set(w.look(), w.Duration, w.Curve)
}

func (w *AnimatedContainer) Build(BuildContext) Widget {
	l := w.t.value()
	c := Container{Padding: l.padding, Margin: l.margin, Color: l.color, Child: w.Child}
	if l.hasW {
		c.Width = new(l.width)
	}
	if l.hasH {
		c.Height = new(l.height)
	}
	if l.hasAlign {
		c.Alignment = new(l.align)
	}
	if l.hasDecoration {
		d := l.decoration
		c.Decoration = &d
		c.Color = gfx.Color{} // Container не принимает и цвет, и украшение
	}
	return c
}
