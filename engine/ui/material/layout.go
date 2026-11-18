package material

import (
	"hexago/ui/geom"
	"hexago/ui/painting"
	"hexago/ui/rendering"
	"hexago/ui/widgets"
)

// CardVariant — вид карточки.
type CardVariant uint8

const (
	CardElevated CardVariant = iota // приподнятая, с тенью (по умолчанию)
	CardFilled                      // залитая, без тени
	CardOutlined                    // с обводкой
)

// Card — карточка Material 3: группа связанного содержимого. Скругление 12 dp, отступ снаружи 4 dp,
// содержимое обрезается по форме. OnTap — карточка нажимается целиком.
type Card struct {
	widgets.Base
	Variant CardVariant
	OnTap   func()
	Child   widgets.Widget
}

func (c Card) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	m := Material{Radius: t.Shapes.Medium, Clip: true}
	switch c.Variant {
	case CardFilled:
		m.Color = t.Colors.SurfaceContainerHighest
	case CardOutlined:
		m.Color, m.Border = t.Colors.Surface, &painting.BorderSide{Color: t.Colors.OutlineVariant, Width: 1}
	default:
		m.Color, m.Elevation = t.Colors.SurfaceContainerLow, 1
	}
	m.Child = InkWell{OnTap: c.OnTap, Radius: t.Shapes.Medium, Child: c.Child}
	return widgets.Padding{Padding: geom.All(4), Child: m}
}

// Divider — горизонтальная линия-разделитель 1 dp цвета OutlineVariant. Height — сколько места
// занимает разделитель (линия посередине), Indent и EndIndent — отступы слева и справа.
type Divider struct {
	widgets.Base
	Height, Indent, EndIndent float64 // Height: 0 — 16
}

func (d Divider) Build(ctx widgets.BuildContext) widgets.Widget {
	h := d.Height
	if h <= 0 {
		h = 16
	}
	return widgets.SizedBox{Height: new(h), Child: widgets.Center{Child: widgets.Padding{
		Padding: geom.EdgeInsets{Left: d.Indent, Right: d.EndIndent},
		Child:   widgets.SizedBox{Height: new(1.0), Width: new(widgets.Infinity), Child: widgets.ColoredBox{Color: Of(ctx).Colors.OutlineVariant}},
	}}}
}

// VerticalDivider — вертикальная линия-разделитель (например, между кнопками в строке).
type VerticalDivider struct {
	widgets.Base
	Width float64 // сколько места занимает; 0 — 16
}

func (d VerticalDivider) Build(ctx widgets.BuildContext) widgets.Widget {
	w := d.Width
	if w <= 0 {
		w = 16
	}
	return widgets.SizedBox{Width: new(w), Child: widgets.Center{Child: widgets.SizedBox{
		Width: new(1.0), Height: new(widgets.Infinity), Child: widgets.ColoredBox{Color: Of(ctx).Colors.OutlineVariant},
	}}}
}

// ListTile — строка списка Material 3: Leading (значок, аватар) слева, Title и Subtitle посередине,
// Trailing (значок, переключатель) справа. Высота 56 dp, с подзаголовком — 72. OnTap — строка нажимается.
type ListTile struct {
	widgets.Base
	Leading  widgets.Widget
	Title    widgets.Widget // обычно widgets.Text без стиля
	Subtitle widgets.Widget
	Trailing widgets.Widget
	OnTap    func()
}

func (l ListTile) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	c := t.Colors
	title, sub := t.Text.BodyLarge, t.Text.BodyMedium
	title.Color, sub.Color = c.OnSurface, c.OnSurfaceVariant
	text := []widgets.Widget{widgets.DefaultTextStyle{Style: title, Child: l.Title}}
	height := 56.0
	if l.Subtitle != nil {
		text = append(text, widgets.DefaultTextStyle{Style: sub, Child: l.Subtitle})
		height = 72
	}
	row := []widgets.Widget{}
	if l.Leading != nil {
		row = append(row, l.Leading)
	}
	row = append(row, widgets.Expanded{Child: widgets.Column{
		MainAxisSize: rendering.MainAxisMin, CrossAxisAlignment: rendering.CrossAxisStart, Children: text,
	}})
	if l.Trailing != nil {
		row = append(row, l.Trailing)
	}
	return InkWell{OnTap: l.OnTap, Child: widgets.ConstrainedBox{
		Constraints: rendering.Constraints{MaxWidth: rendering.Inf, MinHeight: height, MaxHeight: rendering.Inf},
		Child: widgets.Padding{Padding: geom.EdgeInsets{Left: 16, Right: 24, Top: 8, Bottom: 8},
			Child: widgets.IconTheme{Data: widgets.IconThemeData{Size: 24, Color: c.OnSurfaceVariant},
				Child: widgets.Row{Spacing: 16, Children: row}}},
	}}
}
