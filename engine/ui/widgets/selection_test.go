package widgets_test

import (
	"testing"
	"time"

	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/text"
	w "hexago/ui/widgets"
)

// stubControls — ручки 20×20 и меню, которое запоминает свои кнопки.
type stubControls struct{ actions *[]w.SelectionAction }

func (s stubControls) Handle(_ w.BuildContext, kind w.HandleKind) (w.Widget, geom.Size, geom.Offset) {
	// Ручка без заливки (как капля Material на прозрачном поле): касания должна ловить сама ручка.
	return w.Keyed{Key: kind, Child: w.SizedBox{Width: new(20.0), Height: new(20.0)}},
		geom.Size{Width: 20, Height: 20}, geom.Offset{X: 10}
}

func (s stubControls) Toolbar(_ w.BuildContext, actions []w.SelectionAction) w.Widget {
	*s.actions = actions
	return w.SizedBox{Width: new(100.0), Height: new(30.0), Child: w.ColoredBox{Color: gfx.Blue}}
}

type note struct {
	w.State
	ctrl    *w.TextEditingController
	actions []w.SelectionAction
}

func (n *note) Init(w.BuildContext) { n.ctrl = w.NewTextController("один два") }

func (n *note) Build(w.BuildContext) w.Widget {
	return w.Padding{Padding: geom.All(20), Child: w.Align{Alignment: geom.TopLeft, Child: w.SizedBox{Width: new(200.0),
		Child: w.EditableText{Controller: n.ctrl, Style: text.Style{FontSize: 16}, Controls: stubControls{&n.actions}}}}}
}

// labels — подписи кнопок меню.
func labels(a []w.SelectionAction) []string {
	var out []string
	for _, x := range a {
		out = append(out, x.Label)
	}
	return out
}

func do(t *testing.T, a []w.SelectionAction, label string) {
	t.Helper()
	for _, x := range a {
		if x.Label == label {
			x.Do()
			return
		}
	}
	t.Fatalf("в меню нет «%s»: %v", label, labels(a))
}

// findKeyed — элемент виджета Keyed с ключом key.
func findKeyed(e *w.Element, key any) *w.Element {
	if k, ok := e.Widget().(w.Keyed); ok && k.Key == key {
		return e
	}
	for _, c := range e.Children() {
		if f := findKeyed(c, key); f != nil {
			return f
		}
	}
	return nil
}

func TestSelectionHandlesAndToolbar(t *testing.T) {
	s := newFormScreen(w.App{Home: note{}})
	n := findState[*note](s.b.Root())
	s.frame()

	// Долгое нажатие на «два»: слово выделено, меню без «Вставить» (буфер пуст).
	s.now += time.Second
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerDown, Pointer: 1, Position: geom.Offset{X: 75, Y: 28}, Time: s.now})
	s.clock.Advance(700 * time.Millisecond)
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerUp, Pointer: 1, Position: geom.Offset{X: 75, Y: 28}, Time: s.now})
	s.frame()
	s.frame()
	if sel := n.ctrl.Selection(); sel.Start() != 5 || sel.End() != 8 {
		t.Fatalf("долгое нажатие выделило %+v", sel)
	}
	if got := labels(n.actions); len(got) != 3 || got[0] != "Вырезать" || got[1] != "Копировать" || got[2] != "Выбрать все" {
		t.Fatalf("меню: %v", got)
	}
	if findKeyed(s.b.Root(), w.HandleLeft) == nil || findKeyed(s.b.Root(), w.HandleRight) == nil {
		t.Fatal("нет ручек у краёв выделения")
	}

	// Ручку конца тянут влево — выделение сжимается до «д».
	right := w.BoundsOf(findKeyed(s.b.Root(), w.HandleRight))
	from := geom.Offset{X: right.Left + 10, Y: right.Top + 10}
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerDown, Pointer: 2, Position: from, Time: s.now})
	for i := 1; i <= 6; i++ {
		s.now += 16 * time.Millisecond
		s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerMove, Pointer: 2, Position: from.Add(geom.Offset{X: -float64(i) * 3}), Time: s.now})
		s.frame()
	}
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerUp, Pointer: 2, Position: from.Add(geom.Offset{X: -18}), Time: s.now})
	s.frame()
	if sel := n.ctrl.Selection(); sel.Start() != 5 || sel.End() < 6 || sel.End() > 7 {
		t.Errorf("после перетаскивания ручки: %+v", sel)
	}

	// «Копировать»: в буфере выделенное, выделение снято.
	n.ctrl.SetSelection(w.TextSelection{Base: 5, Extent: 8})
	s.frame()
	do(t, n.actions, "Копировать")
	s.frame()
	if got, _ := w.PasteText(n.Context()); got != "два" || !n.ctrl.Selection().IsCollapsed() {
		t.Fatalf("после «Копировать»: буфер %q, выделение %+v", got, n.ctrl.Selection())
	}
	if findKeyed(s.b.Root(), w.HandleRight) != nil {
		t.Error("ручки остались после «Копировать»")
	}

	// Касание по курсору — меню со «Вставить»; вставка — на место курсора.
	caret := geom.Rect{Left: 20, Top: 20} // поле — с отступом 20 dp
	n.ctrl.SetSelection(w.Collapsed(8))
	s.frame()
	s.tap(caret.Left+190, caret.Top+8) // правее текста — это позиция 8, где курсор
	s.frame()
	do(t, n.actions, "Вставить")
	s.frame()
	if n.ctrl.Text() != "один двадва" {
		t.Errorf("после «Вставить»: %q", n.ctrl.Text())
	}

	// «Выбрать все» и «Вырезать».
	s.tap(caret.Left+190, caret.Top+8)
	s.tap(caret.Left+190, caret.Top+8)
	s.frame()
	do(t, n.actions, "Выбрать все")
	s.frame()
	do(t, n.actions, "Вырезать")
	s.frame()
	if got, _ := w.PasteText(n.Context()); n.ctrl.Text() != "" || got != "один двадва" {
		t.Errorf("после «Вырезать»: текст %q, буфер %q", n.ctrl.Text(), got)
	}
}

// Клавиатура в ответ на выделение присылает то же значение (закрывает набираемое слово) — ручки
// и меню остаются.
func TestSelectionSurvivesKeyboardEcho(t *testing.T) {
	s := newFormScreen(w.App{Home: note{}})
	n := findState[*note](s.b.Root())
	s.frame()
	s.now += time.Second
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerDown, Pointer: 1, Position: geom.Offset{X: 40, Y: 28}, Time: s.now})
	s.clock.Advance(700 * time.Millisecond)
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerUp, Pointer: 1, Position: geom.Offset{X: 40, Y: 28}, Time: s.now})
	s.frame()
	s.b.HandleTextEditing(n.ctrl.Value()) // то же самое — как finishComposingText у Gboard
	s.frame()
	if findKeyed(s.b.Root(), w.HandleLeft) == nil {
		t.Error("ручки пропали от повтора того же значения клавиатурой")
	}
	s.b.HandleTextEditing(w.TextEditingValue{Text: "один два!", Selection: w.Collapsed(9)})
	s.frame()
	if findKeyed(s.b.Root(), w.HandleLeft) != nil {
		t.Error("ручки остались, когда клавиатура изменила текст")
	}
}

// Долгое нажатие и касание по полю без фокуса: клавиатура подключается сразу с новым выделением
// (курсором), а не со старым, которое она потом прислала бы обратно поверх нового.
func TestKeyboardConnectsWithNewSelection(t *testing.T) {
	s := newFormScreen(w.App{Home: note{}})
	s.now += time.Second
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerDown, Pointer: 1, Position: geom.Offset{X: 40, Y: 28}, Time: s.now})
	s.clock.Advance(700 * time.Millisecond)
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerUp, Pointer: 1, Position: geom.Offset{X: 40, Y: 28}, Time: s.now})
	s.frame()
	if want := (w.TextSelection{Base: 0, Extent: 4}); s.kb.shows != 1 || s.kb.updates != 0 || s.kb.value.Selection != want {
		t.Errorf("долгое нажатие: показов %d, обновлений %d, у клавиатуры %+v, а нужно сразу %+v",
			s.kb.shows, s.kb.updates, s.kb.value.Selection, want)
	}

	s = newFormScreen(w.App{Home: note{}})
	s.tap(40, 28)
	if s.kb.shows != 1 || s.kb.updates != 0 || s.kb.value.Selection.End() == 8 {
		t.Errorf("касание: показов %d, обновлений %d, у клавиатуры %+v — курсор должен прийти сразу под пальцем",
			s.kb.shows, s.kb.updates, s.kb.value.Selection)
	}
}

// mover — квадрат-цель, который сдвигается по SetState.
type mover struct {
	w.State
	x float64
}

func (m *mover) Build(w.BuildContext) w.Widget {
	return w.Padding{Padding: geom.EdgeInsets{Left: m.x, Top: 10}, Child: w.Align{Alignment: geom.TopLeft,
		Child: w.Keyed{Key: "цель", Child: w.SizedBox{Width: new(30.0), Height: new(30.0)}}}}
}

// Follow берёт положение цели из прошлого кадра; когда цель сдвинулась, он просит ещё кадр
// и встаёт рядом с ней, даже если больше кадров никто не просит.
func TestFollowCatchesUpWithTarget(t *testing.T) {
	s := newFormScreen(w.App{Home: mover{}})
	m := findState[*mover](s.b.Root())
	target := findKeyed(s.b.Root(), "цель")
	w.ShowOverlay(m.Context(), func(w.BuildContext) w.Widget {
		return w.Follow{Target: target, Place: func(r geom.Rect, _ geom.Size) geom.Offset { return r.TopLeft() },
			Child: w.Keyed{Key: "спутник", Child: w.SizedBox{Width: new(5.0), Height: new(5.0)}}}
	})
	requested := false
	s.b.RequestFrame = func() { requested = true }
	settle := func() {
		for i := 0; i < 10; i++ {
			requested = false
			s.frame()
			if !requested {
				return
			}
		}
		t.Fatal("кадры не кончаются")
	}
	settle()
	m.SetState(func() { m.x = 50 })
	settle()
	got, want := w.BoundsOf(findKeyed(s.b.Root(), "спутник")).TopLeft(), w.BoundsOf(target).TopLeft()
	if got != want || want.X != 50 {
		t.Errorf("спутник в %v, цель в %v", got, want)
	}
}
