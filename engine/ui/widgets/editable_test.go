package widgets_test

import (
	"testing"
	"time"

	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/rendering"
	"hexago/ui/text"
	w "hexago/ui/widgets"
)

// fakeKeyboard — клавиатура для тестов: запоминает, что ей сказали.
type fakeKeyboard struct {
	shown   bool
	cfg     w.TextInputConfig
	value   w.TextEditingValue
	shows   int
	updates int
	hides   int
}

func (k *fakeKeyboard) Show(cfg w.TextInputConfig, v w.TextEditingValue) {
	k.shown, k.cfg, k.value = true, cfg, v
	k.shows++
}
func (k *fakeKeyboard) Update(v w.TextEditingValue) { k.value, k.updates = v, k.updates+1 }
func (k *fakeKeyboard) Hide()                       { k.shown, k.hides = false, k.hides+1 }

// form — два поля: имя («Далее») и почта («Готово»).
type form struct {
	w.State
	name, email *w.TextEditingController
	changed     []string
	submitted   string
}

func (f *form) Init(w.BuildContext) {
	f.name, f.email = &w.TextEditingController{}, w.NewTextController("a@b.c")
}

func (f *form) Build(w.BuildContext) w.Widget {
	st := text.Style{FontSize: 16}
	return w.Column{CrossAxisAlignment: rendering.CrossAxisStretch, Children: []w.Widget{
		w.SizedBox{Height: new(40.0), Child: w.EditableText{Controller: f.name, Style: st, Action: w.ActionNext,
			OnChanged: func(s string) { f.changed = append(f.changed, s) }}},
		w.SizedBox{Height: new(40.0), Child: w.EditableText{Controller: f.email, Style: st, Keyboard: w.KeyboardEmail,
			OnSubmitted: func(s string) { f.submitted = s }}},
	}}
}

// formScreen — экран 300×200 dp с клавиатурой и часами.
type formScreen struct {
	b     *w.Binding
	kb    *fakeKeyboard
	clock *gestures.FakeClock
	now   time.Duration
}

func newFormScreen(app w.Widget) *formScreen {
	s := &formScreen{b: w.NewBinding(app), kb: &fakeKeyboard{}, clock: &gestures.FakeClock{}}
	s.b.SetTextInput(s.kb)
	s.b.SetClock(s.clock)
	s.frame()
	return s
}

func (s *formScreen) frame() {
	s.b.DrawFrame(gfx.NewImageCanvas(300, 200), w.MediaQueryData{Size: geom.Size{Width: 300, Height: 200}, DevicePixelRatio: 1})
}

func (s *formScreen) tap(x, y float64) {
	s.now += 400 * time.Millisecond
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerDown, Pointer: 1, Position: geom.Offset{X: x, Y: y}, Time: s.now})
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerUp, Pointer: 1, Position: geom.Offset{X: x, Y: y}, Time: s.now})
	s.frame()
}

func TestEditableTextWithKeyboard(t *testing.T) {
	s := newFormScreen(form{})
	f := findState[*form](s.b.Root())
	if s.kb.shown {
		t.Fatal("клавиатура открылась без фокуса")
	}

	s.tap(100, 20) // первое поле
	if !s.kb.shown || s.kb.cfg.Action != w.ActionNext {
		t.Fatalf("клавиатура: %+v", s.kb)
	}

	// Клавиатура набирает слово (подчёркнутое), потом подтверждает его.
	s.b.HandleTextEditing(w.TextEditingValue{Text: "Ан", Selection: w.Collapsed(2), Composing: w.TextRange{Start: 0, End: 2}})
	s.b.HandleTextEditing(w.TextEditingValue{Text: "Анна", Selection: w.Collapsed(4)})
	s.frame()
	if f.name.Text() != "Анна" || len(f.changed) != 2 || f.changed[1] != "Анна" {
		t.Fatalf("после набора: %q, OnChanged %v", f.name.Text(), f.changed)
	}
	if s.kb.updates != 0 {
		t.Error("набор с клавиатуры отправлен ей же обратно")
	}

	// «Далее» — фокус у второго поля, клавиатура не прячется, а переключается на почту.
	s.b.HandleTextAction(w.ActionNext)
	s.frame()
	if !s.kb.shown || s.kb.hides != 0 || s.kb.cfg.Keyboard != w.KeyboardEmail || s.kb.value.Text != "a@b.c" {
		t.Fatalf("после «Далее»: %+v", s.kb)
	}

	// Текст из кода приложения уходит клавиатуре.
	f.email.SetText("x@y.z")
	s.frame()
	if s.kb.updates != 1 || s.kb.value.Text != "x@y.z" {
		t.Errorf("клавиатура не узнала о новом тексте: %+v", s.kb)
	}

	// «Готово» — OnSubmitted, фокус снят, клавиатура спрятана.
	s.b.HandleTextAction(w.ActionDone)
	s.frame()
	if f.submitted != "x@y.z" || s.kb.shown {
		t.Errorf("после «Готово»: submitted %q, клавиатура %v", f.submitted, s.kb.shown)
	}
}

func TestEditableTextTapAndDoubleTap(t *testing.T) {
	s := newFormScreen(form{})
	f := findState[*form](s.b.Root())
	f.name.SetText("один два")
	s.frame()

	s.tap(2, 20) // у левого края — курсор в начало
	if sel := f.name.Selection(); sel != w.Collapsed(0) {
		t.Errorf("касание у левого края: %+v", sel)
	}
	s.tap(290, 20) // правее текста — курсор в конец
	if sel := f.name.Selection(); sel != w.Collapsed(8) {
		t.Errorf("касание правее текста: %+v", sel)
	}

	// Два быстрых касания по второму слову выделяют его.
	b := findBox[*rendering.EditableBox](s.b.Root())
	x := b.CaretRect(6).Left
	s.tap(x, 20)
	s.now -= 300 * time.Millisecond // второе касание через 100 мс
	s.tap(x, 20)
	if sel := f.name.Selection(); sel.Start() != 5 || sel.End() != 8 {
		t.Errorf("двойное касание: %+v", sel)
	}
}

// findBox — первый бокс типа T в дереве.
func findBox[T rendering.Box](e *w.Element) T {
	if b, ok := e.Box().(T); ok {
		return b
	}
	for _, c := range e.Children() {
		if b := findBox[T](c); any(b) != any(*new(T)) {
			return b
		}
	}
	return *new(T)
}
