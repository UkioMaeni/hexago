package material

import (
	"math"
	"testing"

	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/hexagotest"
	"hexago/ui/icons"
	"hexago/ui/rendering"
	"hexago/ui/widgets"
)

// screenSample — экран из всех компонентов.
func screenSample() widgets.Widget {
	tile := func(icon *icons.Data, title, sub string, trailing widgets.Widget) widgets.Widget {
		t := ListTile{Leading: widgets.Icon{Icon: icon}, Title: widgets.Text{Data: title}, Trailing: trailing, OnTap: noop}
		if sub != "" {
			t.Subtitle = widgets.Text{Data: sub}
		}
		return t
	}
	radio := func(v int, name string) widgets.Widget {
		return widgets.Row{MainAxisSize: rendering.MainAxisMin, Children: []widgets.Widget{
			Radio[int]{Value: v, GroupValue: 1, OnChanged: func(int) {}}, widgets.Text{Data: name},
		}}
	}
	return Scaffold{
		AppBar: AppBar{
			Leading: IconButton{Icon: icons.Menu, OnPressed: noop},
			Title:   widgets.Text{Data: "Настройки"},
			Actions: []widgets.Widget{IconButton{Icon: icons.Search, OnPressed: noop}, IconButton{Icon: icons.MoreVert, OnPressed: noop}},
		},
		Body: widgets.Column{CrossAxisAlignment: rendering.CrossAxisStretch, Children: []widgets.Widget{
			widgets.Padding{Padding: geom.Symmetric(12, 0), Child: Card{Child: widgets.Padding{Padding: geom.All(16),
				Child: widgets.Text{Data: "Карточка: связанное содержимое на приподнятой поверхности."}}}},
			tile(icons.Person, "Анна", "Привет! Как дела?", Checkbox{Value: true, OnChanged: func(bool) {}}),
			Divider{Indent: 16, EndIndent: 16},
			tile(icons.Notifications, "Уведомления", "", Switch{Value: true, OnChanged: func(bool) {}}),
			tile(icons.Lock, "Только по Wi-Fi", "", Switch{OnChanged: func(bool) {}}),
			widgets.Padding{Padding: geom.Symmetric(12, 0), Child: widgets.Row{Children: []widgets.Widget{
				radio(1, "Чай"), radio(2, "Кофе"), Checkbox{}, Switch{Value: true},
			}}},
			widgets.Padding{Padding: geom.Symmetric(4, 0), Child: Slider{Value: 0.4, OnChanged: func(float64) {}}},
			widgets.Padding{Padding: geom.Symmetric(16, 8), Child: widgets.Row{Spacing: 16, Children: []widgets.Widget{
				widgets.Expanded{Child: LinearProgressIndicator{Value: 0.6}}, CircularProgressIndicator{Value: 0.7},
			}}},
		}},
		FloatingActionButton: FloatingActionButton{Icon: icons.Edit, Label: "Написать", OnPressed: noop},
		BottomBar: NavigationBar{Selected: 1, OnSelected: func(int) {}, Destinations: []NavigationDestination{
			{Icon: icons.Home, Label: "Главная"}, {Icon: icons.Settings, Label: "Настройки"}, {Icon: icons.Person, Label: "Профиль"},
		}},
	}
}

// renderPhone рисует как на телефоне: 360×760 dp, плотность 2, строка состояния и панель навигации по 24 dp.
func renderPhone(w widgets.Widget) *gfx.Canvas {
	b := widgets.NewBinding(w)
	c := gfx.NewImageCanvas(720, 1520)
	c.Clear(gfx.White)
	b.DrawFrame(c, widgets.MediaQueryData{
		Size: geom.Size{Width: 360, Height: 760}, DevicePixelRatio: 2,
		Padding: geom.EdgeInsets{Top: 24, Bottom: 24},
	})
	return c
}

func TestScreenGolden(t *testing.T) {
	hexagotest.Golden(t, "screen_light", renderPhone(App{Mode: ThemeLight, Home: screenSample()}).ToNRGBA())
	hexagotest.Golden(t, "screen_dark", renderPhone(App{Mode: ThemeDark, Home: screenSample()}).ToNRGBA())
}

// controls — переключатели, которые меняют своё значение по нажатию.
type controls struct {
	widgets.State
	checked bool
	radio   string
	on      bool
	value   float64
	tab     int
}

func (c *controls) Build(widgets.BuildContext) widgets.Widget {
	return widgets.Column{CrossAxisAlignment: rendering.CrossAxisStart, Children: []widgets.Widget{
		Checkbox{Value: c.checked, OnChanged: func(v bool) { c.SetState(func() { c.checked = v }) }},                     // y 0…48
		Radio[string]{Value: "б", GroupValue: c.radio, OnChanged: func(v string) { c.SetState(func() { c.radio = v }) }}, // 48…96
		Switch{Value: c.on, OnChanged: func(v bool) { c.SetState(func() { c.on = v }) }},                                 // 96…144
		widgets.SizedBox{Width: new(224.0), Child: Slider{Value: c.value, Divisions: 4,
			OnChanged: func(v float64) { c.SetState(func() { c.value = v }) }}}, // 144…192
		widgets.SizedBox{Width: new(300.0), Child: NavigationBar{Selected: c.tab, OnSelected: func(i int) { c.SetState(func() { c.tab = i }) },
			Destinations: []NavigationDestination{{Icon: icons.Home, Label: "1"}, {Icon: icons.Mail, Label: "2"}, {Icon: icons.Person, Label: "3"}}}}, // 192…272
	}}
}

func TestControlsRespondToTaps(t *testing.T) {
	b := widgets.NewBinding(App{Mode: ThemeLight, Home: controls{}})
	frame := func() {
		b.DrawFrame(gfx.NewImageCanvas(320, 320), widgets.MediaQueryData{Size: geom.Size{Width: 320, Height: 320}, DevicePixelRatio: 1})
	}
	frame()
	n := 0
	tap := func(x, y float64) {
		n++
		b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerDown, Pointer: n, Position: geom.Offset{X: x, Y: y}})
		b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerUp, Pointer: n, Position: geom.Offset{X: x, Y: y}})
		frame()
	}
	tap(24, 24)   // флажок
	tap(24, 72)   // «один из»
	tap(30, 120)  // выключатель
	tap(162, 168) // ползунок: дорожка 12…212, точка 162 → 0.75
	tap(250, 232) // третий пункт навигации

	c := findControls(b.Root())
	if !c.checked || c.radio != "б" || !c.on || math.Abs(c.value-0.75) > 1e-9 || c.tab != 2 {
		t.Errorf("после нажатий: checked=%v radio=%q on=%v value=%v tab=%d", c.checked, c.radio, c.on, c.value, c.tab)
	}
}

func findControls(e *widgets.Element) *controls {
	if c, ok := e.State().(*controls); ok {
		return c
	}
	for _, ch := range e.Children() {
		if c := findControls(ch); c != nil {
			return c
		}
	}
	return nil
}
