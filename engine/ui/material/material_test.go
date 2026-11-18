package material

import (
	"testing"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/hexagotest"
	"hexago/ui/painting"
	"hexago/ui/rendering"
	"hexago/ui/text"
	"hexago/ui/widgets"
)

// sample — образец темы: пары цветов «фон / содержимое» и шкала текста.
type sample struct{ widgets.Base }

func (sample) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	c := t.Colors
	swatch := func(bg, fg gfx.Color, name string) widgets.Widget {
		label := t.Text.LabelMedium
		label.Color = fg
		return widgets.Expanded{Child: widgets.Container{
			Height: new(44.0), Alignment: &geom.Center,
			Decoration: &painting.BoxDecoration{Color: bg, BorderRadius: Corners(t.Shapes.Small)},
			Child:      widgets.Text{Data: name, Style: label},
		}}
	}
	row := func(children ...widgets.Widget) widgets.Widget {
		return widgets.Padding{Padding: geom.Symmetric(12, 4), Child: widgets.Row{Spacing: 8, Children: children}}
	}
	line := func(s text.Style, name string) widgets.Widget {
		return widgets.Padding{Padding: geom.Symmetric(12, 0), Child: widgets.Text{Data: name, Style: s}}
	}
	return widgets.Column{CrossAxisAlignment: rendering.CrossAxisStart, Children: []widgets.Widget{
		row(swatch(c.Primary, c.OnPrimary, "Primary"), swatch(c.PrimaryContainer, c.OnPrimaryContainer, "Container")),
		row(swatch(c.Secondary, c.OnSecondary, "Secondary"), swatch(c.SecondaryContainer, c.OnSecondaryContainer, "Container")),
		row(swatch(c.Tertiary, c.OnTertiary, "Tertiary"), swatch(c.TertiaryContainer, c.OnTertiaryContainer, "Container")),
		row(swatch(c.Error, c.OnError, "Error"), swatch(c.ErrorContainer, c.OnErrorContainer, "Container")),
		row(swatch(c.SurfaceContainerLowest, c.OnSurface, "Lowest"), swatch(c.SurfaceContainer, c.OnSurface, "Container"),
			swatch(c.SurfaceContainerHighest, c.OnSurfaceVariant, "Highest")),
		line(t.Text.DisplaySmall, "Display"),
		line(t.Text.HeadlineMedium, "Headline"),
		line(t.Text.TitleLarge, "Title large"),
		line(t.Text.TitleMedium, "Title medium"),
		line(t.Text.LabelLarge, "Label large"),
		// Без стиля: цвет и размер — из темы (DefaultTextStyle, BodyMedium).
		widgets.Padding{Padding: geom.Symmetric(12, 0), Child: widgets.Text{Data: "Основной текст — Body medium"}},
	}}
}

func TestThemeGolden(t *testing.T) {
	for _, mode := range []struct {
		name string
		mode ThemeMode
	}{{"theme_light", ThemeLight}, {"theme_dark", ThemeDark}} {
		img := hexagotest.Render(App{Mode: mode.mode, Home: sample{}}, 360*2, 520*2, 2)
		hexagotest.Golden(t, mode.name, img)
	}
}

func TestThemeChoice(t *testing.T) {
	custom := Light()
	custom.Colors.Primary = gfx.ColorFromHex("#006A6A")
	cases := []struct {
		app      App
		platform gfx.Brightness
		want     gfx.Color
	}{
		{App{}, gfx.Light, LightColors().Primary},
		{App{}, gfx.Dark, DarkColors().Primary}, // по умолчанию — как в системе
		{App{Mode: ThemeLight}, gfx.Dark, LightColors().Primary},
		{App{Mode: ThemeDark}, gfx.Light, DarkColors().Primary},
		{App{Theme: &custom}, gfx.Light, custom.Colors.Primary}, // свой вариант
		{App{Theme: &custom}, gfx.Dark, DarkColors().Primary},   // своей тёмной нет — стандартная
	}
	for i, c := range cases {
		if got := c.app.theme(c.platform).Colors.Primary; got != c.want {
			t.Errorf("случай %d: Primary = %v, ожидали %v", i, got, c.want)
		}
	}
}

func TestOfWithoutAppIsLight(t *testing.T) {
	var got ThemeData
	hexagotest.Render(widgets.Builder{Builder: func(ctx widgets.BuildContext) widgets.Widget {
		got = Of(ctx)
		return widgets.SizedBox{}
	}}, 10, 10, 1)
	if got != Light() {
		t.Error("без material.App тема должна быть стандартной светлой")
	}
}
