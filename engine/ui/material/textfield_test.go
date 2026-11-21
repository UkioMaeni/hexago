package material

import (
	"testing"
	"time"

	"hexago/ui/geom"
	"hexago/ui/hexagotest"
	"hexago/ui/icons"
	"hexago/ui/rendering"
	"hexago/ui/widgets"
)

// keys — клавиатура для тестов: запоминает, что ей сказали.
type keys struct {
	shown bool
	cfg   widgets.TextInputConfig
	value widgets.TextEditingValue
}

func (k *keys) Show(cfg widgets.TextInputConfig, v widgets.TextEditingValue) {
	k.shown, k.cfg, k.value = true, cfg, v
}
func (k *keys) Update(v widgets.TextEditingValue) { k.value = v }
func (k *keys) Hide()                             { k.shown = false }

// signup — форма со всеми видами полей.
type signup struct {
	widgets.State
	name, email, pass, note *widgets.TextEditingController
	hide                    bool
}

func (s *signup) Init(widgets.BuildContext) {
	s.name = &widgets.TextEditingController{}
	s.email = widgets.NewTextController("anna@")
	s.pass = widgets.NewTextController("секрет")
	s.note = widgets.NewTextController("Первая строка\nвторая строка")
	s.hide = true
}

func (s *signup) Build(widgets.BuildContext) widgets.Widget {
	gap := func() widgets.Widget { return widgets.SizedBox{Height: new(12.0)} }
	return Scaffold{AppBar: AppBar{Title: widgets.Text{Data: "Регистрация"}}, Body: widgets.Padding{Padding: geom.All(16),
		Child: widgets.Column{CrossAxisAlignment: rendering.CrossAxisStretch, Children: []widgets.Widget{
			TextField{Controller: s.name, Label: "Имя", Hint: "Как к вам обращаться", Helper: "Можно псевдоним",
				Leading: widgets.Icon{Icon: icons.Person}, Action: widgets.ActionNext},
			gap(),
			TextField{Controller: s.email, Label: "Почта", Keyboard: widgets.KeyboardEmail, Error: "Нет домена после @"},
			gap(),
			TextField{Controller: s.pass, Outlined: true, Label: "Пароль", Obscure: s.hide,
				Trailing: IconButton{Icon: icons.Visibility, OnPressed: func() { s.SetState(func() { s.hide = !s.hide }) }}},
			gap(),
			TextField{Controller: s.note, Outlined: true, Label: "О себе", Multiline: true},
			gap(),
			TextField{Outlined: true, Label: "Город"},
		}}}}
}

func findSignup(e *widgets.Element) *signup {
	if s, ok := e.State().(*signup); ok {
		return s
	}
	for _, c := range e.Children() {
		if s := findSignup(c); s != nil {
			return s
		}
	}
	return nil
}

func TestTextFields(t *testing.T) {
	p := newPhone(t, App{Mode: ThemeLight, Home: signup{}})
	k := &keys{}
	p.b.SetTextInput(k)
	s := findSignup(p.b.Root())
	hexagotest.Golden(t, "textfields_light", p.image())

	// Касание по полю «Имя»: фокус, подпись поднимается, клавиатура с кнопкой «Далее».
	p.tap(200, 24+64+16+28)
	p.wait(300 * time.Millisecond)
	if !k.shown || k.cfg.Action != widgets.ActionNext {
		t.Fatalf("клавиатура: %+v", k)
	}
	p.b.HandleTextEditing(widgets.TextEditingValue{Text: "Анна", Selection: widgets.Collapsed(4)})
	p.wait(300 * time.Millisecond)
	if s.name.Text() != "Анна" {
		t.Fatalf("набранный текст: %q", s.name.Text())
	}
	hexagotest.Golden(t, "textfields_focused", p.image())

	// «Далее» — фокус у почты, клавиатура для почты с её текстом.
	p.b.HandleTextAction(widgets.ActionNext)
	p.wait(300 * time.Millisecond)
	if !k.shown || k.cfg.Keyboard != widgets.KeyboardEmail || k.value.Text != "anna@" {
		t.Errorf("после «Далее»: %+v", k)
	}
	// Пароль — без подсказок клавиатуры; значок «глаз» показывает его.
	p.b.HandleTextAction(widgets.ActionNext)
	p.wait(300 * time.Millisecond)
	if !k.cfg.Obscure || !k.cfg.NoAutocorrect {
		t.Errorf("клавиатура пароля: %+v", k.cfg)
	}
}

// longForm — форма длиннее экрана: последнее поле внизу.
type longForm struct {
	widgets.State
	last *widgets.FocusNode
}

func (l *longForm) Init(widgets.BuildContext) { l.last = &widgets.FocusNode{} }

func (l *longForm) Build(widgets.BuildContext) widgets.Widget {
	kids := []widgets.Widget{}
	for i := range 7 {
		kids = append(kids, TextField{Label: "Поле " + string(rune('А'+i))})
	}
	kids = append(kids, TextField{Label: "Последнее", Focus: l.last})
	return Scaffold{AppBar: AppBar{Title: widgets.Text{Data: "Анкета"}}, Body: widgets.SingleChildScrollView{
		Padding: geom.All(16), Child: widgets.Column{Spacing: 16, CrossAxisAlignment: rendering.CrossAxisStretch, Children: kids}}}
}

// Клавиатура закрыла нижнее поле: тело экрана укорачивается над ней, а прокрутка показывает поле.
func TestFieldScrollsAboveKeyboard(t *testing.T) {
	p := newPhone(t, App{Mode: ThemeLight, Home: longForm{}})
	k := &keys{}
	p.b.SetTextInput(k)
	var l *longForm
	var find func(e *widgets.Element)
	find = func(e *widgets.Element) {
		if s, ok := e.State().(*longForm); ok {
			l = s
		}
		for _, c := range e.Children() {
			find(c)
		}
	}
	find(p.b.Root())
	l.last.RequestFocus()
	p.wait(300 * time.Millisecond) // поле в фокусе, а клавиатура выезжает чуть позже — как на телефоне
	p.keyboard = 300
	p.wait(500 * time.Millisecond)
	if !k.shown {
		t.Fatal("клавиатура не открылась")
	}
	// Ищем поле «Последнее» по его элементу и проверяем, что оно над клавиатурой.
	var field *widgets.Element
	var walk func(e *widgets.Element)
	walk = func(e *widgets.Element) {
		if tf, ok := e.State().(*TextField); ok && tf.Label == "Последнее" {
			field = e
		}
		for _, c := range e.Children() {
			walk(c)
		}
	}
	walk(p.b.Root())
	r := widgets.BoundsOf(field)
	if r.Bottom > 760-300 || r.Top < 24+64 {
		t.Errorf("поле в %v: должно быть между верхней панелью (88) и клавиатурой (460)", r)
	}
	hexagotest.Golden(t, "textfield_keyboard", p.image())
}
