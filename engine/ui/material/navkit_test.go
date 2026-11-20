package material

import (
	"image"
	"testing"
	"time"

	"hexago/ui/gestures"
	"hexago/ui/hexagotest"
	"hexago/ui/icons"
	"hexago/ui/widgets"
)

// inbox — экран почты: боковая панель с папками, меню «⋮», список, плавающая кнопка.
type inbox struct {
	widgets.State
	folder  int
	menu    string
	undone  int
	actions []string
}

func (b *inbox) Build(ctx widgets.BuildContext) widgets.Widget {
	folders := []string{"Входящие", "Отправленные", "Черновики"}
	tiles := []widgets.Widget{}
	for _, s := range []string{"Анна: Привет!", "Борис: Отчёт готов", "Вера: Встреча в 15:00"} {
		tiles = append(tiles, ListTile{Leading: widgets.Icon{Icon: icons.Mail}, Title: widgets.Text{Data: s}, OnTap: noop})
	}
	pick := func(s string) func() { return func() { b.SetState(func() { b.menu = s }) } }
	return Scaffold{
		AppBar: AppBar{Title: widgets.Text{Data: folders[b.folder]}, Actions: []widgets.Widget{MenuButton{Items: []widgets.Widget{
			MenuItem{Leading: icons.Refresh, Label: "Обновить", OnTap: pick("обновить")},
			MenuItem{Leading: icons.Settings, Label: "Настройки", Trailing: "Ctrl+,", OnTap: pick("настройки")},
			Divider{},
			MenuItem{Leading: icons.Delete, Label: "Очистить корзину"},
		}}}},
		Drawer: NavigationDrawer{
			Header:   widgets.Text{Data: "Почта"},
			Selected: b.folder,
			Destinations: []NavigationDestination{
				{Icon: icons.Mail, Label: folders[0]}, {Icon: icons.Send, Label: folders[1]}, {Icon: icons.Edit, Label: folders[2]},
			},
			OnSelected: func(i int) { b.SetState(func() { b.folder = i }) },
		},
		Body:                 widgets.Column{Children: tiles},
		FloatingActionButton: FloatingActionButton{Icon: icons.Edit, OnPressed: noop},
	}
}

func findInbox(e *widgets.Element) *inbox {
	if b, ok := e.State().(*inbox); ok {
		return b
	}
	for _, c := range e.Children() {
		if b := findInbox(c); b != nil {
			return b
		}
	}
	return nil
}

func TestSnackBar(t *testing.T) {
	p := newPhone(t, App{Mode: ThemeLight, Home: inbox{}})
	b := findInbox(p.b.Root())
	ShowSnackBar(b.Context(), SnackBar{
		Content: widgets.Text{Data: "Письмо удалено"},
		Action:  &SnackBarAction{Label: "Отменить", OnPressed: func() { b.undone++ }},
	})
	ShowSnackBar(b.Context(), SnackBar{Content: widgets.Text{Data: "Второе сообщение"}, ShowClose: true})
	p.wait(400 * time.Millisecond)
	hexagotest.Golden(t, "snackbar_light", p.image())

	// «Отменить» справа в сообщении: сообщение уходит, OnPressed вызван, за ним — следующее.
	p.tap(300, 760-24-16-24)
	if b.undone != 1 {
		t.Fatalf("кнопка сообщения не сработала: %d", b.undone)
	}
	p.wait(500 * time.Millisecond)
	p.image()
	// Второе висит 4 секунды и уходит само; плавающая кнопка возвращается на место.
	p.wait(5 * time.Second)
	if !sameImage(p.image(), newPhone(t, App{Mode: ThemeLight, Home: inbox{}}).image()) {
		t.Error("после ухода сообщений экран не вернулся к исходному")
	}
}

// sameImage — картинки совпадают (с точностью до единицы в канале).
func sameImage(a, b *image.NRGBA) bool {
	for i := range a.Pix {
		if d := int(a.Pix[i]) - int(b.Pix[i]); d > 1 || d < -1 {
			return false
		}
	}
	return true
}

func TestDrawer(t *testing.T) {
	p := newPhone(t, App{Mode: ThemeLight, Home: inbox{}})
	b := findInbox(p.b.Root())
	p.tap(28, 24+32) // кнопка «меню» вместо стрелки «Назад»
	p.wait(400 * time.Millisecond)
	hexagotest.Golden(t, "drawer_light", p.image())

	// Пункт «Отправленные»: папка выбрана, панель закрылась.
	p.tap(100, 24+12+16+18+20+56+28)
	p.wait(400 * time.Millisecond)
	if b.folder != 1 || p.b.HandleBack() {
		t.Fatalf("после выбора: папка %d, панель закрыта=%v", b.folder, !p.b.HandleBack())
	}

	// Движение от левого края выдвигает панель вслед за пальцем; отпущенная дальше середины — остаётся.
	p.send(gestures.PointerDown, 4, 400)
	for i := 1; i <= 12; i++ {
		p.now += 16 * time.Millisecond
		p.send(gestures.PointerMove, 4+float64(i)*20, 400)
		p.frame()
	}
	p.wait(200 * time.Millisecond)
	p.send(gestures.PointerUp, 244, 400)
	p.wait(400 * time.Millisecond)
	if !p.b.HandleBack() {
		t.Fatal("панель не открылась движением от края")
	}
	p.wait(400 * time.Millisecond)

	// Открыть и задвинуть пальцем влево.
	p.tap(28, 24+32)
	p.wait(400 * time.Millisecond)
	p.send(gestures.PointerDown, 250, 600)
	for i := 1; i <= 5; i++ {
		p.now += 16 * time.Millisecond
		p.send(gestures.PointerMove, 250-float64(i)*30, 600)
		p.frame()
	}
	p.send(gestures.PointerUp, 100, 600)
	p.wait(400 * time.Millisecond)
	if p.b.HandleBack() {
		t.Error("панель не задвинулась броском влево")
	}
}

func TestMenu(t *testing.T) {
	p := newPhone(t, App{Mode: ThemeLight, Home: inbox{}})
	b := findInbox(p.b.Root())
	p.tap(360-4-24, 24+32) // «⋮» у правого края верхней панели
	p.wait(400 * time.Millisecond)
	hexagotest.Golden(t, "menu_light", p.image())

	// Меню под кнопкой и прижато к её правому краю: «Настройки» — второй пункт.
	p.tap(250, 24+8+48+48+8+24)
	p.wait(300 * time.Millisecond)
	if b.menu != "настройки" || p.b.HandleBack() {
		t.Fatalf("после выбора пункта: %q, меню закрыто=%v", b.menu, !p.b.HandleBack())
	}

	// Выключенный пункт не закрывает меню; касание мимо — закрывает.
	p.tap(360-4-24, 24+32)
	p.wait(400 * time.Millisecond)
	p.tap(250, 24+8+48+48+8+48+16+24) // «Очистить корзину» (без OnTap)
	p.wait(200 * time.Millisecond)
	if !p.b.HandleBack() {
		t.Fatal("выключенный пункт закрыл меню")
	}
	p.wait(300 * time.Millisecond)
	p.tap(360-4-24, 24+32)
	p.wait(400 * time.Millisecond)
	p.tap(100, 600)
	p.wait(300 * time.Millisecond)
	if p.b.HandleBack() {
		t.Error("касание мимо меню его не закрыло")
	}
}
