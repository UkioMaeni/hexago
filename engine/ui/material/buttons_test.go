package material

import (
	"testing"

	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/hexagotest"
	"hexago/ui/icons"
	"hexago/ui/rendering"
	"hexago/ui/widgets"
)

func label(s string) widgets.Widget { return widgets.Text{Data: s} }

func noop() {}

// buttonsSample — все кнопки: включённые и выключенные, со значком и без.
func buttonsSample() widgets.Widget {
	row := func(children ...widgets.Widget) widgets.Widget {
		return widgets.Padding{Padding: geom.Symmetric(12, 0), Child: widgets.Row{Spacing: 8, Children: children}}
	}
	return widgets.Column{CrossAxisAlignment: rendering.CrossAxisStart, Children: []widgets.Widget{
		row(FilledButton{OnPressed: noop, Child: label("Filled")}, FilledButton{Child: label("Выключена")}),
		row(FilledTonalButton{OnPressed: noop, Icon: icons.Add, Child: label("Tonal")}, FilledTonalButton{Child: label("Выключена")}),
		row(ElevatedButton{OnPressed: noop, Child: label("Elevated")}, ElevatedButton{Child: label("Выключена")}),
		row(OutlinedButton{OnPressed: noop, Icon: icons.Share, Child: label("Outlined")}, OutlinedButton{Child: label("Выключена")}),
		row(TextButton{OnPressed: noop, Child: label("Text")}, TextButton{Icon: icons.Send, OnPressed: noop, Child: label("Со значком")}),
		row(IconButton{Icon: icons.Favorite, OnPressed: noop}, IconButton{Icon: icons.Settings, OnPressed: noop, Variant: IconFilled},
			IconButton{Icon: icons.Edit, OnPressed: noop, Variant: IconTonal}, IconButton{Icon: icons.Delete, OnPressed: noop, Variant: IconOutlined},
			IconButton{Icon: icons.Delete, Variant: IconFilled}),
		row(FloatingActionButton{Icon: icons.Add, OnPressed: noop, Size: FABSmall}, FloatingActionButton{Icon: icons.Edit, OnPressed: noop},
			FloatingActionButton{Icon: icons.Add, OnPressed: noop, Label: "Создать"}),
		widgets.SizedBox{Height: new(12.0)},
		row(FloatingActionButton{Icon: icons.PhotoCamera, OnPressed: noop, Size: FABLarge}),
	}}
}

func TestButtonsGolden(t *testing.T) {
	for _, m := range []struct {
		name string
		mode ThemeMode
	}{{"buttons_light", ThemeLight}, {"buttons_dark", ThemeDark}} {
		img := hexagotest.Render(App{Mode: m.mode, Home: widgets.Padding{Padding: geom.Symmetric(0, 8), Child: buttonsSample()}},
			380*2, 540*2, 2)
		hexagotest.Golden(t, m.name, img)
	}
}

func TestButtonPressAndDisabled(t *testing.T) {
	pressed := 0
	app := App{Mode: ThemeLight, Home: widgets.Column{CrossAxisAlignment: rendering.CrossAxisStart, Children: []widgets.Widget{
		FilledButton{OnPressed: func() { pressed++ }, Child: label("Да")}, // y 4…44
		FilledButton{Child: label("Нет")},                                 // y 52…92, выключена
	}}}
	b := widgets.NewBinding(app)
	frame := func() *gfx.Canvas {
		c := gfx.NewImageCanvas(200, 200)
		b.DrawFrame(c, widgets.MediaQueryData{Size: geom.Size{Width: 200, Height: 200}, DevicePixelRatio: 1})
		return c
	}
	frame()
	send := func(kind gestures.PointerKind, p int, y float64) {
		b.HandlePointer(gestures.PointerEvent{Kind: kind, Pointer: p, Position: geom.Offset{X: 20, Y: y}})
	}

	// Палец опущен — поверх кнопки подложка нажатия: цвет темнее/светлее обычного.
	before := frame().ToNRGBA().NRGBAAt(10, 24)
	send(gestures.PointerDown, 1, 24)
	during := frame().ToNRGBA().NRGBAAt(10, 24)
	send(gestures.PointerUp, 1, 24)
	after := frame().ToNRGBA().NRGBAAt(10, 24)
	if pressed != 1 {
		t.Errorf("нажатий %d, ожидали 1", pressed)
	}
	if before == during || before != after {
		t.Errorf("подложка нажатия: до %v, во время %v, после %v", before, during, after)
	}

	send(gestures.PointerDown, 2, 72)
	send(gestures.PointerUp, 2, 72)
	if pressed != 1 {
		t.Error("выключенная кнопка сработала")
	}
}
