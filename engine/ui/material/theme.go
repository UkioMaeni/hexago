package material

import (
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/widgets"
)

// Shapes — шкала скруглений Material 3, радиусы углов в dp. Full — «полностью скруглённый»:
// радиус больше любой стороны, углы ограничиваются половиной высоты (капсула, круг).
type Shapes struct {
	None, ExtraSmall, Small, Medium, Large, ExtraLarge, Full float64
}

// Corners — скругление всех углов радиусом r: Corners(t.Shapes.Medium).
func Corners(r float64) geom.BorderRadius { return geom.RadiusAll(r) }

// DefaultShapes — скругления из спецификации Material 3.
func DefaultShapes() Shapes {
	return Shapes{None: 0, ExtraSmall: 4, Small: 8, Medium: 12, Large: 16, ExtraLarge: 28, Full: 10000}
}

// Elevation — уровни высоты Material 3 в dp: 0, 1, 3, 6, 8, 12. В Material 3 высоту показывают
// не только тенью, но и цветом поверхности: чем выше слой, тем выше его SurfaceContainer.
type Elevation struct {
	Level0, Level1, Level2, Level3, Level4, Level5 float64
}

// DefaultElevation — уровни высоты из спецификации Material 3.
func DefaultElevation() Elevation {
	return Elevation{Level0: 0, Level1: 1, Level2: 3, Level3: 6, Level4: 8, Level5: 12}
}

// ThemeData — тема Material 3: цвета, текст, скругления, высоты. Её получают компоненты через Of(ctx).
type ThemeData struct {
	Colors    ColorScheme
	Text      TextTheme
	Shapes    Shapes
	Elevation Elevation
}

// Brightness — светлая тема или тёмная (по схеме цветов).
func (t ThemeData) Brightness() gfx.Brightness { return t.Colors.Brightness }

// Light — стандартная светлая тема Material 3.
func Light() ThemeData { return FromColors(LightColors()) }

// Dark — стандартная тёмная тема Material 3.
func Dark() ThemeData { return FromColors(DarkColors()) }

// FromColors — тема со своими цветами; текст, скругления и высоты — стандартные.
func FromColors(c ColorScheme) ThemeData {
	return ThemeData{Colors: c, Text: NewTextTheme(c.OnSurface), Shapes: DefaultShapes(), Elevation: DefaultElevation()}
}

// Theme передаёт тему поддереву. Его ставит material.App; вложенный Theme меняет тему части экрана.
type Theme struct {
	widgets.Base
	Data  ThemeData
	Child widgets.Widget
}

func (w Theme) ChildWidget() widgets.Widget { return w.Child }

func (w Theme) UpdateShouldNotify(old widgets.Widget) bool { return old.(Theme).Data != w.Data }

// Of возвращает тему в этом месте дерева и подписывает виджет на её изменения.
// Без material.App и Theme — стандартная светлая тема.
func Of(ctx widgets.BuildContext) ThemeData {
	if t, ok := widgets.DependOn[Theme](ctx); ok {
		return t.Data
	}
	return Light()
}

// themed — Theme вместе со стилем текста по умолчанию (BodyMedium цвета OnSurface), как Material во Flutter.
func themed(t ThemeData, child widgets.Widget) widgets.Widget {
	return Theme{Data: t, Child: widgets.DefaultTextStyle{Style: t.Text.BodyMedium, Child: child}}
}
