package widgets

import (
	"math"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/painting"
	"hexago/ui/rendering"
	"hexago/ui/text"
)

// Infinity — «сколько есть места»: SizedBox{Width: new(widgets.Infinity)} растягивается на всю ширину.
var Infinity = math.Inf(1)

// ColoredBox заливает свой прямоугольник цветом и рисует ребёнка поверх.
type ColoredBox struct {
	Base
	Color gfx.Color
	Child Widget
}

func (w ColoredBox) ChildWidget() Widget { return w.Child }
func (w ColoredBox) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.ColoredBox{Color: w.Color}
}
func (w ColoredBox) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	b.(*rendering.ColoredBox).Color = w.Color
}

// Padding добавляет отступы вокруг ребёнка.
type Padding struct {
	Base
	Padding geom.EdgeInsets
	Child   Widget
}

func (w Padding) ChildWidget() Widget { return w.Child }
func (w Padding) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.PaddingBox{Padding: w.Padding}
}
func (w Padding) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	b.(*rendering.PaddingBox).Padding = w.Padding
}

// Align ставит ребёнка в точку Alignment внутри доступного места.
// WidthFactor/HeightFactor — размер как у ребёнка, умноженный на фактор, вместо всего места.
type Align struct {
	Base
	Alignment    geom.Alignment
	WidthFactor  *float64
	HeightFactor *float64
	Child        Widget
}

func (w Align) ChildWidget() Widget { return w.Child }
func (w Align) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.AlignBox{Alignment: w.Alignment, WidthFactor: w.WidthFactor, HeightFactor: w.HeightFactor}
}
func (w Align) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	a := b.(*rendering.AlignBox)
	a.Alignment, a.WidthFactor, a.HeightFactor = w.Alignment, w.WidthFactor, w.HeightFactor
}

// Center ставит ребёнка по центру.
type Center struct {
	Base
	WidthFactor  *float64
	HeightFactor *float64
	Child        Widget
}

func (w Center) ChildWidget() Widget { return w.Child }
func (w Center) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.AlignBox{Alignment: geom.Center, WidthFactor: w.WidthFactor, HeightFactor: w.HeightFactor}
}
func (w Center) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	a := b.(*rendering.AlignBox)
	a.WidthFactor, a.HeightFactor = w.WidthFactor, w.HeightFactor
}

// SizedBox — бокс заданного размера. Без размера — по ребёнку; new(widgets.Infinity) — на всё место.
type SizedBox struct {
	Base
	Width  *float64
	Height *float64
	Child  Widget
}

func (w SizedBox) ChildWidget() Widget { return w.Child }
func (w SizedBox) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.ConstrainedBox{Additional: rendering.TightFor(w.Width, w.Height)}
}
func (w SizedBox) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	b.(*rendering.ConstrainedBox).Additional = rendering.TightFor(w.Width, w.Height)
}

// SizedBoxExpand — бокс на всё доступное место, как SizedBox.expand во Flutter.
func SizedBoxExpand(child Widget) Widget {
	return SizedBox{Width: &Infinity, Height: &Infinity, Child: child}
}

// SizedBoxShrink — бокс нулевого размера, как SizedBox.shrink.
func SizedBoxShrink() Widget { return SizedBox{Width: new(0.0), Height: new(0.0)} }

// ConstrainedBox добавляет ребёнку свои ограничения размера.
type ConstrainedBox struct {
	Base
	Constraints rendering.Constraints
	Child       Widget
}

func (w ConstrainedBox) ChildWidget() Widget { return w.Child }
func (w ConstrainedBox) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.ConstrainedBox{Additional: w.Constraints}
}
func (w ConstrainedBox) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	b.(*rendering.ConstrainedBox).Additional = w.Constraints
}

// LimitedBox ограничивает размер, только если родитель его не ограничил
// (например, внутри списка по оси прокрутки). nil — без ограничения.
type LimitedBox struct {
	Base
	MaxWidth  *float64
	MaxHeight *float64
	Child     Widget
}

func (w LimitedBox) ChildWidget() Widget { return w.Child }
func (w LimitedBox) box() *rendering.LimitedBox {
	b := &rendering.LimitedBox{MaxWidth: Infinity, MaxHeight: Infinity}
	if w.MaxWidth != nil {
		b.MaxWidth = *w.MaxWidth
	}
	if w.MaxHeight != nil {
		b.MaxHeight = *w.MaxHeight
	}
	return b
}
func (w LimitedBox) CreateRenderObject(BuildContext) rendering.Box { return w.box() }
func (w LimitedBox) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	n := w.box()
	l := b.(*rendering.LimitedBox)
	l.MaxWidth, l.MaxHeight = n.MaxWidth, n.MaxHeight
}

// AspectRatio держит соотношение сторон ширина/высота: 16.0/9.0.
type AspectRatio struct {
	Base
	Ratio float64
	Child Widget
}

func (w AspectRatio) ChildWidget() Widget { return w.Child }
func (w AspectRatio) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.AspectRatioBox{Ratio: w.Ratio}
}
func (w AspectRatio) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	b.(*rendering.AspectRatioBox).Ratio = w.Ratio
}

// FractionallySizedBox даёт ребёнку долю доступного места: WidthFactor 0.5 — половину ширины.
type FractionallySizedBox struct {
	Base
	WidthFactor  *float64
	HeightFactor *float64
	Alignment    geom.Alignment
	Child        Widget
}

func (w FractionallySizedBox) ChildWidget() Widget { return w.Child }
func (w FractionallySizedBox) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.FractionalBox{WidthFactor: w.WidthFactor, HeightFactor: w.HeightFactor, Alignment: w.Alignment}
}
func (w FractionallySizedBox) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	f := b.(*rendering.FractionalBox)
	f.WidthFactor, f.HeightFactor, f.Alignment = w.WidthFactor, w.HeightFactor, w.Alignment
}

// DecorationPosition — рисовать украшение под ребёнком или поверх.
type DecorationPosition uint8

const (
	DecorationBackground DecorationPosition = iota
	DecorationForeground
)

// DecoratedBox рисует украшение: фон, градиент, рамку, скругления, тени.
type DecoratedBox struct {
	Base
	Decoration painting.BoxDecoration
	Position   DecorationPosition
	Child      Widget
}

func (w DecoratedBox) ChildWidget() Widget { return w.Child }
func (w DecoratedBox) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.DecoratedBox{Decoration: w.Decoration, Foreground: w.Position == DecorationForeground}
}
func (w DecoratedBox) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	d := b.(*rendering.DecoratedBox)
	d.Decoration, d.Foreground = w.Decoration, w.Position == DecorationForeground
}

// Opacity делает ребёнка полупрозрачным: 0 — невидим, 1 — как есть.
type Opacity struct {
	Base
	Opacity float64
	Child   Widget
}

func (w Opacity) ChildWidget() Widget { return w.Child }
func (w Opacity) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.OpacityBox{Opacity: w.Opacity}
}
func (w Opacity) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	b.(*rendering.OpacityBox).Opacity = w.Opacity
}

// ClipRect обрезает ребёнка по своему прямоугольнику.
type ClipRect struct {
	Base
	Child Widget
}

func (w ClipRect) ChildWidget() Widget { return w.Child }
func (w ClipRect) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.ClipBox{Kind: rendering.ClipToRect}
}
func (w ClipRect) UpdateRenderObject(BuildContext, rendering.Box) {}

// ClipRRect обрезает ребёнка по прямоугольнику со скруглёнными углами.
type ClipRRect struct {
	Base
	BorderRadius geom.BorderRadius
	Child        Widget
}

func (w ClipRRect) ChildWidget() Widget { return w.Child }
func (w ClipRRect) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.ClipBox{Kind: rendering.ClipToRRect, Radius: w.BorderRadius}
}
func (w ClipRRect) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	b.(*rendering.ClipBox).Radius = w.BorderRadius
}

// ClipOval обрезает ребёнка по эллипсу (по кругу, если бокс квадратный).
type ClipOval struct {
	Base
	Child Widget
}

func (w ClipOval) ChildWidget() Widget { return w.Child }
func (w ClipOval) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.ClipBox{Kind: rendering.ClipToOval}
}
func (w ClipOval) UpdateRenderObject(BuildContext, rendering.Box) {}

// Transform поворачивает, масштабирует или сдвигает ребёнка при отрисовке (лэйаут не меняется).
// Alignment — точка, относительно которой применяется преобразование; nil — левый верхний угол.
type Transform struct {
	Base
	Transform geom.Affine
	Alignment *geom.Alignment
	Child     Widget
}

func (w Transform) ChildWidget() Widget { return w.Child }
func (w Transform) align() geom.Alignment {
	if w.Alignment == nil {
		return geom.TopLeft
	}
	return *w.Alignment
}
func (w Transform) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.TransformBox{Transform: w.Transform, Alignment: w.align()}
}
func (w Transform) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	t := b.(*rendering.TransformBox)
	t.Transform, t.Alignment = w.Transform, w.align()
}

// Rotate поворачивает ребёнка на угол (в радианах, по часовой стрелке) вокруг центра.
func Rotate(angle float64, child Widget) Widget {
	return Transform{Transform: geom.Rotation(angle), Alignment: &geom.Center, Child: child}
}

// Scale масштабирует ребёнка относительно центра.
func Scale(s float64, child Widget) Widget {
	return Transform{Transform: geom.Scaling(s, s), Alignment: &geom.Center, Child: child}
}

// Translate сдвигает ребёнка.
func Translate(dx, dy float64, child Widget) Widget {
	return Transform{Transform: geom.Translation(dx, dy), Child: child}
}

// Offstage раскладывает ребёнка, но не показывает его и не занимает места.
type Offstage struct {
	Base
	Offstage bool
	Child    Widget
}

func (w Offstage) ChildWidget() Widget { return w.Child }
func (w Offstage) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.OffstageBox{Offstage: w.Offstage}
}
func (w Offstage) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	b.(*rendering.OffstageBox).Offstage = w.Offstage
}

// Visibility показывает ребёнка, если Visible, иначе — Replacement (по умолчанию пустое место).
type Visibility struct {
	Base
	Visible     bool
	Replacement Widget
	Child       Widget
}

func (w Visibility) Build(BuildContext) Widget {
	if w.Visible {
		return w.Child
	}
	if w.Replacement != nil {
		return w.Replacement
	}
	return SizedBoxShrink()
}

// CustomPaint — своя отрисовка: Painter рисует под ребёнком, Foreground — поверх.
// Size — размер без ребёнка.
type CustomPaint struct {
	Base
	Painter    rendering.PaintFunc
	Foreground rendering.PaintFunc
	Size       geom.Size
	Child      Widget
}

func (w CustomPaint) ChildWidget() Widget { return w.Child }
func (w CustomPaint) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.CustomPaintBox{Painter: w.Painter, Foreground: w.Foreground, Preferred: w.Size}
}
func (w CustomPaint) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	c := b.(*rendering.CustomPaintBox)
	c.Painter, c.Foreground, c.Preferred = w.Painter, w.Foreground, w.Size
}

// Builder строит ребёнка функцией — удобно, когда нужен BuildContext внутри дерева.
type Builder struct {
	Base
	Builder func(ctx BuildContext) Widget
}

func (w Builder) Build(ctx BuildContext) Widget { return w.Builder(ctx) }

// Placeholder — серый прямоугольник с крестом: «здесь будет виджет».
type Placeholder struct {
	Base
	Color          gfx.Color // по умолчанию серо-синий, как во Flutter
	FallbackWidth  float64   // размер, если место не ограничено (по умолчанию 400×400)
	FallbackHeight float64
}

func (w Placeholder) Build(BuildContext) Widget {
	col := w.Color
	if col.IsTransparent() {
		col = gfx.ARGB(0xFF455A64)
	}
	fw, fh := w.FallbackWidth, w.FallbackHeight
	if fw == 0 {
		fw = 400
	}
	if fh == 0 {
		fh = 400
	}
	return LimitedBox{MaxWidth: &fw, MaxHeight: &fh, Child: CustomPaint{
		Size: geom.Size{Width: Infinity, Height: Infinity},
		Painter: func(c *gfx.Canvas, s geom.Size) {
			p := gfx.Paint{Color: col, StrokeWidth: 2, Style: gfx.Stroke}
			c.DrawRect(geom.RectFromSize(geom.Zero, s).Inflate(-1), p)
			c.DrawLine(geom.Zero, geom.Offset{X: s.Width, Y: s.Height}, p)
			c.DrawLine(geom.Offset{X: s.Width}, geom.Offset{Y: s.Height}, p)
		},
	}}
}

// ErrorBox — красный прямоугольник с текстом ошибки на месте виджета, при построении которого случилась паника.
// Текст ошибки печатается в лог (hexago run покажет его в терминале).
type ErrorBox struct {
	Base
	Message string
}

func (w ErrorBox) Build(BuildContext) Widget {
	size := 200.0
	return LimitedBox{MaxWidth: &size, MaxHeight: &size, Child: SizedBoxExpand(ClipRect{Child: ColoredBox{
		Color: gfx.ARGB(0xFFD32F2F),
		Child: Padding{Padding: geom.All(8), Child: Align{Alignment: geom.TopLeft, Child: Text{
			Data:  w.Message,
			Style: text.Style{FontSize: 12, Color: gfx.White},
		}}},
	}})}
}

// view — корень любого дерева: даёт корневой render-бокс.
type view struct {
	Base
	Child Widget
}

func (w view) ChildWidget() Widget                            { return w.Child }
func (w view) CreateRenderObject(BuildContext) rendering.Box  { return &rendering.Proxy{} }
func (w view) UpdateRenderObject(BuildContext, rendering.Box) {}
