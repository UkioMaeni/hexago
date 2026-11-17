package widgets_test

import (
	"testing"

	"hexago/ui/gfx"
	"hexago/ui/hexagotest"
	w "hexago/ui/widgets"
)

func TestTextGolden(t *testing.T) {
	tree := w.App{Home: w.Column{
		MainAxisAlignment: w.MainAxisCenter,
		Children: []w.Widget{
			w.Text{Data: "hexago", Style: w.TextStyle{FontSize: 32, FontWeight: w.FontWeightBold}},
			w.SizedBox{Height: new(12.0)},
			w.Text{Data: "Привет, мир! Hello, world!", Style: w.TextStyle{FontSize: 16, Color: gfx.Indigo}},
			w.SizedBox{Height: new(12.0)},
			w.Padding{Padding: w.Symmetric(24, 0), Child: w.Text{
				Data:      "Длинный текст переносится по словам, если не помещается в ширину экрана.",
				TextAlign: w.TextAlignCenter,
			}},
		},
	}}
	hexagotest.Golden(t, "text", hexagotest.Render(tree, 600, 500, 2))
}

func TestErrorBoxShowsMessage(t *testing.T) {
	broken := w.Builder{Builder: func(w.BuildContext) w.Widget { panic("сломалось") }}
	hexagotest.Golden(t, "error_box", hexagotest.Render(w.Center{Child: broken}, 400, 400, 2))
}
