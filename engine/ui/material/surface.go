package material

import (
	"math"
	"time"

	"hexago/ui/animation"
	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/painting"
	"hexago/ui/rendering"
	"hexago/ui/widgets"
)

// Material — поверхность Material: цвет, скругление, обводка, высота (тень) и обрезка содержимого
// по форме. Из неё сделаны кнопки, карточки, панели.
type Material struct {
	widgets.Base
	Color     gfx.Color
	Radius    float64              // скругление углов; t.Shapes.Full — капсула
	Border    *painting.BorderSide // обводка (например, у OutlinedButton)
	Elevation float64              // высота в dp: тень по уровням Material 3 (0, 1, 3, 6, 8, 12)
	Clip      bool                 // обрезать содержимое по форме (картинки в карточке)
	Child     widgets.Widget
}

func (m Material) Build(ctx widgets.BuildContext) widgets.Widget {
	d := &painting.BoxDecoration{
		Color:        m.Color,
		BorderRadius: geom.RadiusAll(m.Radius),
		BoxShadow:    Shadows(m.Elevation, Of(ctx).Colors.Shadow),
	}
	if m.Border != nil {
		d.Border = &painting.Border{Top: *m.Border, Right: *m.Border, Bottom: *m.Border, Left: *m.Border}
	}
	child := m.Child
	if m.Clip && child != nil {
		child = widgets.ClipRRect{BorderRadius: geom.RadiusAll(m.Radius), Child: child}
	}
	return widgets.DecoratedBox{Decoration: *d, Child: child}
}

// Shadows — тень Material 3 для высоты elevation (dp): две тени, «ключевая» и «окружающая»,
// как в Material Web (md-elevation). Цвет — ColorScheme.Shadow.
func Shadows(elevation float64, color gfx.Color) []painting.BoxShadow {
	type pair struct{ keyY, keyBlur, ambY, ambBlur, ambSpread float64 }
	var p pair
	switch {
	case elevation <= 0:
		return nil
	case elevation <= 1:
		p = pair{1, 2, 1, 3, 1}
	case elevation <= 3:
		p = pair{1, 2, 2, 6, 2}
	case elevation <= 6:
		p = pair{1, 3, 4, 8, 3}
	case elevation <= 8:
		p = pair{2, 3, 6, 10, 4}
	default:
		p = pair{4, 4, 8, 12, 6}
	}
	return []painting.BoxShadow{
		{Color: color.WithOpacity(0.3), Offset: geom.Offset{Y: p.keyY}, BlurRadius: p.keyBlur},
		{Color: color.WithOpacity(0.15), Offset: geom.Offset{Y: p.ambY}, BlurRadius: p.ambBlur, SpreadRadius: p.ambSpread},
	}
}

// Прозрачность «подложки состояния» Material 3: поверх элемента цветом его содержимого.
const (
	PressedOpacity  = 0.10 // нажат
	DisabledContent = 0.38 // содержимое выключенного элемента (цвет OnSurface)
	DisabledSurface = 0.12 // фон выключенного элемента (цвет OnSurface)
)

// RippleOpacity — прозрачность волны нажатия (цвет — как у подложки).
const RippleOpacity = 0.12

// InkWell делает область нажимаемой и показывает нажатие, как в Material 3: полупрозрачную подложку
// цвета OverlayColor поверх содержимого и волну — круг, который расходится из точки касания и гаснет,
// когда палец отпущен. Всё обрезается по форме Radius. Без обработчиков InkWell выключен и касаний не ловит.
type InkWell struct {
	widgets.State
	OnTap        func()
	OnLongPress  func()
	OnDoubleTap  func()
	Radius       float64   // форма подложки и волны
	OverlayColor gfx.Color // цвет подложки и волны; прозрачный — OnSurface темы
	Child        widgets.Widget

	pressed bool
	origin  geom.Offset           // откуда идёт волна
	grow    *animation.Controller // радиус волны: 0…1 от наибольшего
	fade    *animation.Controller // видимость волны: 1 → 0 после отпускания
}

func (w *InkWell) Init(ctx widgets.BuildContext) {
	w.grow = widgets.NewAnimation(ctx, 300*time.Millisecond)
	w.fade = widgets.NewAnimation(ctx, 250*time.Millisecond)
}

// Форма дерева под InkWell не зависит ни от нажатия, ни от того, включён ли он: иначе содержимое
// пересоздавалось бы при каждом нажатии и теряло своё состояние.
func (w *InkWell) Build(ctx widgets.BuildContext) widgets.Widget {
	enabled := w.OnTap != nil || w.OnLongPress != nil || w.OnDoubleTap != nil
	if !enabled {
		w.pressed = false
	}
	overlay := w.OverlayColor
	if overlay.IsTransparent() {
		overlay = Of(ctx).Colors.OnSurface
	}
	pressed, origin := w.pressed, w.origin
	grow, fade := w.grow.Value(), w.fade.Value()
	paint := func(c *gfx.Canvas, s geom.Size) {
		rr := geom.RRect{Rect: geom.RectFromSize(geom.Zero, s), Radius: geom.RadiusAll(w.Radius)}.Clamped()
		if pressed {
			c.DrawRRect(rr, gfx.Paint{Color: overlay.WithOpacity(PressedOpacity)})
		}
		if fade > 0 && grow > 0 {
			// Наибольший радиус — до самого дальнего угла.
			far := 0.0
			for _, p := range []geom.Offset{{}, {X: s.Width}, {Y: s.Height}, {X: s.Width, Y: s.Height}} {
				far = math.Max(far, p.Sub(origin).Distance())
			}
			c.ClipRRect(rr)
			c.DrawCircle(origin, grow*far, gfx.Paint{Color: overlay.WithOpacity(RippleOpacity * fade)})
		}
	}
	d := widgets.GestureDetector{
		Behavior: rendering.DeferToChild,
		Child:    widgets.CustomPaint{Foreground: paint, Child: w.Child},
	}
	if !enabled {
		return d
	}
	down := func(at geom.Offset) {
		w.SetState(func() { w.pressed, w.origin = true, at })
		w.grow.SetValue(0)
		w.grow.AnimateTo(1, 0, animation.Standard)
		w.fade.SetValue(1)
	}
	up := func() {
		if w.Mounted() && w.pressed {
			w.SetState(func() { w.pressed = false })
		}
		w.fade.AnimateTo(0, 0, animation.Linear)
	}
	d.Behavior = rendering.Opaque
	d.OnTapDown = func(e gestures.TapDownDetails) { down(e.Local) }
	d.OnTapUp = func(gestures.TapUpDetails) { up() }
	d.OnTapCancel = up
	d.OnTap = w.OnTap
	d.OnDoubleTap = w.OnDoubleTap
	if w.OnLongPress != nil {
		d.OnLongPress = w.OnLongPress
		d.OnLongPressUp = up
	}
	return d
}
