package material

import (
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/icons"
	"hexago/ui/painting"
	"hexago/ui/rendering"
	"hexago/ui/text"
	"hexago/ui/widgets"
)

// Кнопки Material 3. Общее у всех: OnPressed — что делать по нажатию (nil — кнопка выключена),
// OnLongPress — по долгому нажатию, Icon — значок перед подписью, Child — подпись (обычно
// widgets.Text без стиля: цвет и шрифт кнопка задаёт сама).
//
//	material.FilledButton{OnPressed: save, Icon: icons.Check, Child: widgets.Text{Data: "Сохранить"}}

// FilledButton — главная кнопка экрана: залита цветом Primary.
type FilledButton struct {
	widgets.Base
	OnPressed, OnLongPress func()
	Icon                   *icons.Data
	Child                  widgets.Widget
}

func (b FilledButton) Build(ctx widgets.BuildContext) widgets.Widget {
	c := Of(ctx).Colors
	return button(ctx, buttonSpec{container: c.Primary, content: c.OnPrimary}, b.OnPressed, b.OnLongPress, b.Icon, b.Child)
}

// FilledTonalButton — менее заметная залитая кнопка (SecondaryContainer).
type FilledTonalButton struct {
	widgets.Base
	OnPressed, OnLongPress func()
	Icon                   *icons.Data
	Child                  widgets.Widget
}

func (b FilledTonalButton) Build(ctx widgets.BuildContext) widgets.Widget {
	c := Of(ctx).Colors
	return button(ctx, buttonSpec{container: c.SecondaryContainer, content: c.OnSecondaryContainer},
		b.OnPressed, b.OnLongPress, b.Icon, b.Child)
}

// ElevatedButton — приподнятая кнопка с тенью: когда кнопке нужно отделиться от пёстрого фона.
type ElevatedButton struct {
	widgets.Base
	OnPressed, OnLongPress func()
	Icon                   *icons.Data
	Child                  widgets.Widget
}

func (b ElevatedButton) Build(ctx widgets.BuildContext) widgets.Widget {
	c := Of(ctx).Colors
	return button(ctx, buttonSpec{container: c.SurfaceContainerLow, content: c.Primary, elevation: 1},
		b.OnPressed, b.OnLongPress, b.Icon, b.Child)
}

// OutlinedButton — кнопка с обводкой: второстепенное действие рядом с главным.
type OutlinedButton struct {
	widgets.Base
	OnPressed, OnLongPress func()
	Icon                   *icons.Data
	Child                  widgets.Widget
}

func (b OutlinedButton) Build(ctx widgets.BuildContext) widgets.Widget {
	c := Of(ctx).Colors
	return button(ctx, buttonSpec{content: c.Primary, border: &painting.BorderSide{Color: c.Outline, Width: 1}},
		b.OnPressed, b.OnLongPress, b.Icon, b.Child)
}

// TextButton — кнопка без фона: самое незаметное действие (в диалогах, карточках).
type TextButton struct {
	widgets.Base
	OnPressed, OnLongPress func()
	Icon                   *icons.Data
	Child                  widgets.Widget
}

func (b TextButton) Build(ctx widgets.BuildContext) widgets.Widget {
	c := Of(ctx).Colors
	return button(ctx, buttonSpec{content: c.Primary, text: true}, b.OnPressed, b.OnLongPress, b.Icon, b.Child)
}

// buttonSpec — чем различаются кнопки.
type buttonSpec struct {
	container gfx.Color // фон; прозрачный — без фона
	content   gfx.Color // подпись и значок
	border    *painting.BorderSide
	elevation float64
	text      bool // TextButton: поля меньше
}

// button собирает кнопку по спецификации Material 3: высота 40 (место для касания — 48),
// ширина не меньше 64, поля 24 (со значком — 16 слева), значок 18, форма — капсула.
func button(ctx widgets.BuildContext, s buttonSpec, onPressed, onLongPress func(), icon *icons.Data, child widgets.Widget) widgets.Widget {
	t := Of(ctx)
	if onPressed == nil && onLongPress == nil {
		s = disabled(t.Colors, s)
	}
	left, right := 24.0, 24.0
	switch {
	case s.text && icon != nil:
		left, right = 12, 16
	case s.text:
		left, right = 12, 12
	case icon != nil:
		left = 16
	}
	content := child
	if icon != nil {
		row := []widgets.Widget{widgets.Icon{Icon: icon, Size: 18}}
		if child != nil {
			row = append(row, child)
		}
		content = widgets.Row{MainAxisSize: rendering.MainAxisMin, Spacing: 8, Children: row}
	}
	label := t.Text.LabelLarge
	label.Color = s.content
	shape := t.Shapes.Full
	body := widgets.ConstrainedBox{
		Constraints: rendering.Constraints{MinWidth: 64, MaxWidth: rendering.Inf, MinHeight: 40, MaxHeight: 40},
		Child: widgets.Padding{Padding: geom.EdgeInsets{Left: left, Right: right}, Child: widgets.Row{
			MainAxisSize: rendering.MainAxisMin, MainAxisAlignment: rendering.MainAxisCenter,
			Children: []widgets.Widget{content},
		}},
	}
	return widgets.Padding{Padding: geom.Symmetric(0, 4), Child: Material{
		Color: s.container, Radius: shape, Border: s.border, Elevation: s.elevation, Clip: true,
		Child: InkWell{OnTap: onPressed, OnLongPress: onLongPress, Radius: shape, OverlayColor: s.content,
			Child: contentStyle(label, s.content, 18, body)},
	}}
}

// disabled — вид выключенной кнопки: фон и подпись цвета OnSurface, прозрачные, без тени.
func disabled(c ColorScheme, s buttonSpec) buttonSpec {
	if !s.container.IsTransparent() {
		s.container = c.OnSurface.WithOpacity(DisabledSurface)
	}
	s.content = c.OnSurface.WithOpacity(DisabledContent)
	if s.border != nil {
		s.border = &painting.BorderSide{Color: c.OnSurface.WithOpacity(DisabledSurface), Width: s.border.Width}
	}
	s.elevation = 0
	return s
}

// contentStyle задаёт содержимому стиль текста и значков: так подпись без стиля и значок
// получают цвет кнопки.
func contentStyle(label text.Style, color gfx.Color, iconSize float64, child widgets.Widget) widgets.Widget {
	return widgets.DefaultTextStyle{Style: label, Child: widgets.IconTheme{
		Data: widgets.IconThemeData{Size: iconSize, Color: color}, Child: child,
	}}
}
