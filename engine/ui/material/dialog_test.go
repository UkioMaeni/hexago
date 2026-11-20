package material

import (
	"image"
	"testing"
	"time"

	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/hexagotest"
	"hexago/ui/icons"
	"hexago/ui/rendering"
	"hexago/ui/widgets"
)

// phone — телефон 360×760 dp (плотность 2, системные панели по 24 dp) с ручным временем кадров.
// Кадры рисуются только в повреждённой области, как на устройстве.
type phone struct {
	t      *testing.T
	b      *widgets.Binding
	canvas *gfx.Canvas
	now    time.Duration
}

func newPhone(t *testing.T, app widgets.Widget) *phone {
	p := &phone{t: t, b: widgets.NewBinding(app), canvas: gfx.NewImageCanvas(720, 1520)}
	p.frame()
	return p
}

func (p *phone) frame() {
	p.b.Tick(p.now)
	p.b.Paint(p.canvas, p.b.Layout(widgets.MediaQueryData{
		Size: geom.Size{Width: 360, Height: 760}, DevicePixelRatio: 2,
		Padding: geom.EdgeInsets{Top: 24, Bottom: 24},
	}))
}

func (p *phone) wait(d time.Duration) {
	for end := p.now + d; p.now < end; {
		p.now += 16 * time.Millisecond
		p.frame()
	}
}

func (p *phone) send(kind gestures.PointerKind, x, y float64) {
	p.b.HandlePointer(gestures.PointerEvent{Kind: kind, Pointer: 1, Position: geom.Offset{X: x, Y: y}, Time: p.now})
}

func (p *phone) tap(x, y float64) {
	p.send(gestures.PointerDown, x, y)
	p.send(gestures.PointerUp, x, y)
	p.frame()
}

// image — кадр целиком (частичная перерисовка обязана с ним совпадать).
func (p *phone) image() *image.NRGBA {
	full := gfx.NewImageCanvas(720, 1520)
	p.b.Paint(full, image.Rect(0, 0, 720, 1520))
	got, want := p.canvas.ToNRGBA(), full.ToNRGBA()
	for i := range got.Pix {
		if d := int(got.Pix[i]) - int(want.Pix[i]); d > 1 || d < -1 {
			p.t.Fatalf("частичная перерисовка отличается от полной (пиксель %d)", i/4)
		}
	}
	return want
}

// notes — экран со списком; ctx нужен тестам, чтобы открывать окна.
type notes struct {
	widgets.State
	deleted bool
}

func (n *notes) Build(widgets.BuildContext) widgets.Widget {
	tiles := []widgets.Widget{}
	for _, s := range []string{"Купить хлеб", "Позвонить маме", "Дописать отчёт"} {
		tiles = append(tiles, ListTile{Leading: widgets.Icon{Icon: icons.Edit}, Title: widgets.Text{Data: s}, OnTap: noop})
	}
	return Scaffold{AppBar: AppBar{Title: widgets.Text{Data: "Заметки"}},
		Body: widgets.Column{Children: tiles}}
}

func findNotes(e *widgets.Element) *notes {
	if n, ok := e.State().(*notes); ok {
		return n
	}
	for _, c := range e.Children() {
		if n := findNotes(c); n != nil {
			return n
		}
	}
	return nil
}

func (n *notes) confirm() widgets.Widget {
	ctx := n.Context()
	return AlertDialog{
		Icon:    icons.Delete,
		Title:   widgets.Text{Data: "Удалить заметки?"},
		Content: widgets.Text{Data: "Все заметки будут удалены, вернуть их будет нельзя."},
		Actions: []widgets.Widget{
			TextButton{OnPressed: func() { widgets.Pop(ctx) }, Child: widgets.Text{Data: "Отмена"}},
			TextButton{OnPressed: func() { widgets.Pop(ctx); n.SetState(func() { n.deleted = true }) }, Child: widgets.Text{Data: "Удалить"}},
		},
	}
}

func TestAlertDialog(t *testing.T) {
	for _, mode := range []ThemeMode{ThemeLight, ThemeDark} {
		p := newPhone(t, App{Mode: mode, Home: notes{}})
		n := findNotes(p.b.Root())
		ShowDialog(n.Context(), n.confirm())
		p.wait(300 * time.Millisecond)
		name := "dialog_light"
		if mode == ThemeDark {
			name = "dialog_dark"
		}
		hexagotest.Golden(t, name, p.image())
		if !p.b.HandleBack() {
			t.Fatal("«Назад» не закрыл диалог")
		}
		p.wait(300 * time.Millisecond)
		if p.b.HandleBack() {
			t.Fatal("диалог не закрылся")
		}
	}

	// Кнопка «Удалить» закрывает диалог и делает своё дело.
	p := newPhone(t, App{Mode: ThemeLight, Home: notes{}})
	n := findNotes(p.b.Root())
	ShowDialog(n.Context(), n.confirm())
	p.wait(300 * time.Millisecond)
	// Диалог шириной 280 (360 − 2·40) по центру; кнопки — в правом нижнем углу, «Удалить» — крайняя.
	img := p.image()
	bottom := 0 // нижний край диалога: последний ряд цвета SurfaceContainerHigh в середине
	for y := 0; y < 1520; y++ {
		if c := img.NRGBAAt(360, y); c.R == 0xEC && c.G == 0xE6 && c.B == 0xF0 {
			bottom = y / 2
		}
	}
	p.tap(316-30, float64(bottom-24-20))
	p.wait(300 * time.Millisecond)
	if !n.deleted || p.b.HandleBack() {
		t.Errorf("после «Удалить»: deleted=%v, диалог закрыт=%v", n.deleted, !p.b.HandleBack())
	}
}

func TestBottomSheet(t *testing.T) {
	p := newPhone(t, App{Mode: ThemeLight, Home: notes{}})
	n := findNotes(p.b.Root())
	ShowBottomSheet(n.Context(), widgets.Column{MainAxisSize: rendering.MainAxisMin, Children: []widgets.Widget{
		ListTile{Leading: widgets.Icon{Icon: icons.Share}, Title: widgets.Text{Data: "Поделиться"}, OnTap: noop},
		ListTile{Leading: widgets.Icon{Icon: icons.Edit}, Title: widgets.Text{Data: "Переименовать"}, OnTap: noop},
		ListTile{Leading: widgets.Icon{Icon: icons.Delete}, Title: widgets.Text{Data: "Удалить"}, OnTap: noop},
	}})
	p.wait(400 * time.Millisecond)
	hexagotest.Golden(t, "sheet_light", p.image())

	// Потянуть немного и отпустить — панель возвращается.
	p.send(gestures.PointerDown, 180, 600)
	for i := 1; i <= 5; i++ {
		p.now += 16 * time.Millisecond
		p.send(gestures.PointerMove, 180, 600+float64(i)*8)
		p.frame()
	}
	p.wait(200 * time.Millisecond) // палец остановился
	p.send(gestures.PointerUp, 180, 640)
	p.wait(400 * time.Millisecond)
	if !p.b.HandleBack() {
		t.Fatal("панель закрылась, хотя её потянули чуть-чуть")
	}
	p.wait(400 * time.Millisecond)

	// Открыть снова и бросить вниз — закрывается.
	ShowBottomSheet(n.Context(), ListTile{Title: widgets.Text{Data: "Одна строка"}})
	p.wait(400 * time.Millisecond)
	p.send(gestures.PointerDown, 180, 700)
	for i := 1; i <= 4; i++ {
		p.now += 16 * time.Millisecond
		p.send(gestures.PointerMove, 180, 700+float64(i)*15)
		p.frame()
	}
	p.send(gestures.PointerUp, 180, 760)
	p.wait(400 * time.Millisecond)
	if p.b.HandleBack() {
		t.Error("брошенная вниз панель не закрылась")
	}
}

// detail — второй экран со своей верхней панелью.
type detail struct{ widgets.Base }

func (detail) Build(widgets.BuildContext) widgets.Widget {
	return Scaffold{AppBar: AppBar{Title: widgets.Text{Data: "Заметка"}},
		Body: widgets.Padding{Padding: geom.All(16), Child: widgets.Text{Data: "Купить хлеб и молоко."}}}
}

func TestPageTransitionAndBackArrow(t *testing.T) {
	p := newPhone(t, App{Mode: ThemeLight, Home: notes{}})
	n := findNotes(p.b.Root())
	widgets.Push(n.Context(), detail{})
	p.frame()
	p.wait(100 * time.Millisecond)
	p.image() // посреди перехода частичная перерисовка тоже верна
	p.wait(400 * time.Millisecond)
	hexagotest.Golden(t, "page_pushed", p.image())
	p.tap(28, 24+32) // стрелка «Назад» в верхней панели
	p.wait(400 * time.Millisecond)
	if p.b.HandleBack() {
		t.Error("стрелка «Назад» не вернула на первый экран")
	}
}
