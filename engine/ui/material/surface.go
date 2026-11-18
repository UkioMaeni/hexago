package material

import (
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

// InkWell делает область нажимаемой и показывает подложку нажатия — полупрозрачный слой цвета
// OverlayColor поверх содержимого, как state layer в Material 3. Волна (ripple) появится вместе с анимациями.
// Без обработчиков InkWell выключен и касаний не ловит.
type InkWell struct {
	widgets.State
	OnTap        func()
	OnLongPress  func()
	OnDoubleTap  func()
	Radius       float64   // форма подложки
	OverlayColor gfx.Color // цвет подложки; прозрачный — OnSurface темы
	Child        widgets.Widget

	pressed bool
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
	layer := gfx.Transparent
	if w.pressed {
		layer = overlay.WithOpacity(PressedOpacity)
	}
	press := func(p bool) {
		if w.Mounted() && w.pressed != p {
			w.SetState(func() { w.pressed = p })
		}
	}
	d := widgets.GestureDetector{
		Behavior: rendering.DeferToChild,
		Child: widgets.DecoratedBox{
			Position:   widgets.DecorationForeground,
			Decoration: painting.BoxDecoration{Color: layer, BorderRadius: geom.RadiusAll(w.Radius)},
			Child:      w.Child,
		},
	}
	if !enabled {
		return d
	}
	d.Behavior = rendering.Opaque
	d.OnTapDown = func(gestures.TapDownDetails) { press(true) }
	d.OnTapUp = func(gestures.TapUpDetails) { press(false) }
	d.OnTapCancel = func() { press(false) }
	d.OnTap = w.OnTap
	d.OnDoubleTap = w.OnDoubleTap
	if w.OnLongPress != nil {
		d.OnLongPressStart = func(gestures.LongPressStartDetails) { press(true) }
		d.OnLongPress = w.OnLongPress
		d.OnLongPressUp = func() { press(false) }
	}
	return d
}
