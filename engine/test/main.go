// test — приложение на Hexago: проверка виджетов раскладки на устройстве.
package main

import (
	"hexago/ui/app"
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/painting"
	"hexago/ui/rendering"
	w "hexago/ui/widgets"
)

func main() {
	app.Run(w.App{
		Title: "Appuru",
		Home: w.SafeArea{Child: w.Column{
			CrossAxisAlignment: rendering.CrossAxisStretch,
			Children: []w.Widget{
				header(),
				w.Padding{Padding: geom.All(16), Child: w.Row{Spacing: 12, Children: []w.Widget{
					w.Expanded{Child: card()},
					w.Expanded{Flex: 2, Child: w.Container{Height: new(80.0), Decoration: &painting.BoxDecoration{
						BorderRadius: geom.RadiusAll(12),
						Gradient:     painting.LinearGradient{Colors: []gfx.Color{gfx.Pink, gfx.Orange}},
					}}},
				}}},
				w.Padding{Padding: geom.Symmetric(16, 0), Child: w.Row{
					MainAxisAlignment: rendering.MainAxisSpaceBetween,
					Children:          []w.Widget{circle(gfx.Red), circle(gfx.Green), circle(gfx.Blue)},
				}},
				w.Expanded{Child: w.Padding{Padding: geom.All(16), Child: w.Stack{Children: []w.Widget{
					w.Container{Decoration: &painting.BoxDecoration{Border: painting.BorderAll(gfx.Grey, 2), BorderRadius: geom.RadiusAll(16)}},
					w.Center{Child: w.Rotate(0.3, w.Container{Width: new(96.0), Height: new(96.0),
						Decoration: &painting.BoxDecoration{Color: gfx.ColorFromHex("#111111"), BorderRadius: geom.RadiusAll(20)}})},
					w.Positioned{Right: new(16.0), Bottom: new(16.0), Width: new(56.0), Height: new(56.0), Child: fab()},
				}}}},
			},
		}},
	})
}

func header() w.Widget {
	return w.Container{
		Height: new(56.0), Color: gfx.Indigo, Padding: geom.Symmetric(16, 0), Alignment: &geom.CenterLeft,
		Child: w.Container{Width: new(140.0), Height: new(14.0),
			Decoration: &painting.BoxDecoration{Color: gfx.White.WithOpacity(0.85), BorderRadius: geom.RadiusAll(7)}},
	}
}

func card() w.Widget {
	return w.Container{Height: new(80.0), Decoration: &painting.BoxDecoration{
		Color: gfx.White, BorderRadius: geom.RadiusAll(12),
		BoxShadow: []painting.BoxShadow{{Color: gfx.Black.WithOpacity(0.25), Offset: geom.Offset{Y: 3}, BlurRadius: 8}},
	}}
}

func circle(c gfx.Color) w.Widget {
	return w.Container{Width: new(48.0), Height: new(48.0), Decoration: &painting.BoxDecoration{Color: c, Shape: painting.Circle}}
}

func fab() w.Widget {
	return w.Container{Decoration: &painting.BoxDecoration{Color: gfx.Amber, Shape: painting.Circle,
		BoxShadow: []painting.BoxShadow{{Color: gfx.Black.WithOpacity(0.3), Offset: geom.Offset{Y: 4}, BlurRadius: 10}}}}
}
