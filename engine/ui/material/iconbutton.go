package material

import (
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/icons"
	"hexago/ui/painting"
	"hexago/ui/rendering"
	"hexago/ui/widgets"
)

// IconVariant — вид кнопки-значка.
type IconVariant uint8

const (
	IconStandard IconVariant = iota // без фона: панели, строки списка
	IconFilled                      // залита Primary: главное действие
	IconTonal                       // залита SecondaryContainer
	IconOutlined                    // с обводкой
)

// IconButton — кнопка-значок Material 3: круг 40 dp (место для касания — 48), значок 24 dp.
// OnPressed nil — кнопка выключена.
type IconButton struct {
	widgets.Base
	Icon        *icons.Data
	OnPressed   func()
	OnLongPress func()
	Variant     IconVariant
}

func (b IconButton) Build(ctx widgets.BuildContext) widgets.Widget {
	c := Of(ctx).Colors
	var bg, fg gfx.Color
	var border *painting.BorderSide
	switch b.Variant {
	case IconFilled:
		bg, fg = c.Primary, c.OnPrimary
	case IconTonal:
		bg, fg = c.SecondaryContainer, c.OnSecondaryContainer
	case IconOutlined:
		fg, border = c.OnSurfaceVariant, &painting.BorderSide{Color: c.OutlineVariant, Width: 1}
	default:
		fg = c.OnSurfaceVariant
	}
	if b.OnPressed == nil && b.OnLongPress == nil {
		if !bg.IsTransparent() {
			bg = c.OnSurface.WithOpacity(DisabledSurface)
		}
		fg = c.OnSurface.WithOpacity(DisabledContent)
		if border != nil {
			border = &painting.BorderSide{Color: c.OnSurface.WithOpacity(DisabledSurface), Width: 1}
		}
	}
	return widgets.Padding{Padding: geom.All(4), Child: widgets.SizedBox{Width: new(40.0), Height: new(40.0), Child: Material{
		Color: bg, Radius: 20, Border: border, Clip: true,
		Child: InkWell{OnTap: b.OnPressed, OnLongPress: b.OnLongPress, Radius: 20, OverlayColor: fg,
			Child: widgets.Center{Child: widgets.Icon{Icon: b.Icon, Size: 24, Color: fg}}},
	}}}
}

// FABSize — размер плавающей кнопки.
type FABSize uint8

const (
	FABRegular FABSize = iota // 56 dp
	FABSmall                  // 40 dp
	FABLarge                  // 96 dp
)

// FloatingActionButton — плавающая кнопка главного действия экрана (Scaffold ставит её в правый
// нижний угол). С Label — расширенная: значок и подпись.
type FloatingActionButton struct {
	widgets.Base
	Icon      *icons.Data
	Label     string // подпись: кнопка становится расширенной (Extended FAB)
	OnPressed func()
	Size      FABSize
}

func (b FloatingActionButton) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	c := t.Colors
	bg, fg, elevation := c.PrimaryContainer, c.OnPrimaryContainer, 6.0
	if b.OnPressed == nil {
		bg, fg, elevation = c.OnSurface.WithOpacity(DisabledSurface), c.OnSurface.WithOpacity(DisabledContent), 0
	}
	side, radius, iconSize := 56.0, t.Shapes.Large, 24.0
	switch b.Size {
	case FABSmall:
		side, radius = 40, t.Shapes.Medium
	case FABLarge:
		side, radius, iconSize = 96, t.Shapes.ExtraLarge, 36
	}
	var content widgets.Widget = widgets.Center{Child: widgets.Icon{Icon: b.Icon}}
	box := widgets.SizedBox{Width: new(side), Height: new(side)}
	if b.Label != "" {
		row := []widgets.Widget{widgets.Text{Data: b.Label}}
		if b.Icon != nil {
			row = append([]widgets.Widget{widgets.Icon{Icon: b.Icon}}, row...)
		}
		content = widgets.Padding{Padding: geom.Symmetric(16, 0), Child: widgets.Row{
			MainAxisSize: rendering.MainAxisMin, MainAxisAlignment: rendering.MainAxisCenter, Spacing: 12, Children: row,
		}}
		box = widgets.SizedBox{Height: new(56.0)}
		radius, iconSize = t.Shapes.Large, 24
	}
	label := t.Text.LabelLarge
	label.Color = fg
	box.Child = Material{Color: bg, Radius: radius, Elevation: elevation, Clip: true,
		Child: InkWell{OnTap: b.OnPressed, Radius: radius, OverlayColor: fg, Child: contentStyle(label, fg, iconSize, content)}}
	return box
}
