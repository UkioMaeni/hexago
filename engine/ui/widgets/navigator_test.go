package widgets_test

import (
	"image"
	"testing"
	"time"

	"hexago/ui/animation"
	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/rendering"
	w "hexago/ui/widgets"
)

// navScreen — экран 200×300 dp (плотность 2) с ручным временем; каждый кадр рисуется только
// в повреждённой области и сверяется с полной отрисовкой того же состояния.
type navScreen struct {
	t      *testing.T
	b      *w.Binding
	canvas *gfx.Canvas
	now    time.Duration
	media  w.MediaQueryData
	back   []bool // что сообщал SetHandlesBack
}

func newNavScreen(t *testing.T, home w.Widget) *navScreen {
	s := &navScreen{t: t, b: w.NewBinding(w.App{Home: home}), canvas: gfx.NewImageCanvas(400, 600),
		media: w.MediaQueryData{Size: geom.Size{Width: 200, Height: 300}, DevicePixelRatio: 2}}
	s.b.SetHandlesBack = func(h bool) { s.back = append(s.back, h) }
	s.frame()
	return s
}

func (s *navScreen) frame() image.Rectangle {
	s.t.Helper()
	s.b.Tick(s.now)
	damage := s.b.Layout(s.media)
	s.b.Paint(s.canvas, damage)
	full := gfx.NewImageCanvas(400, 600)
	s.b.Paint(full, image.Rect(0, 0, 400, 600))
	got, want := s.canvas.ToNRGBA(), full.ToNRGBA()
	for i := range got.Pix {
		if d := int(got.Pix[i]) - int(want.Pix[i]); d > 1 || d < -1 {
			s.t.Fatalf("кадр %v: частичная перерисовка отличается от полной (пиксель %d)", s.now, i/4)
		}
	}
	return damage
}

func (s *navScreen) wait(d time.Duration) {
	s.t.Helper()
	for end := s.now + d; s.now < end; {
		s.now += 16 * time.Millisecond
		s.frame()
	}
}

func (s *navScreen) tap(x, y float64) {
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerDown, Pointer: 1, Position: geom.Offset{X: x, Y: y}})
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerUp, Pointer: 1, Position: geom.Offset{X: x, Y: y}})
}

// color — цвет пикселя в точке (x, y) dp.
func (s *navScreen) color(x, y float64) gfx.Color {
	c := s.canvas.ToNRGBA().NRGBAAt(int(x*2), int(y*2))
	return gfx.Color{R: c.R, G: c.G, B: c.B, A: c.A}
}

// homeScreen — синий экран (после десятка нажатий — бирюзовый); касание открывает красный экран.
type homeScreen struct {
	w.State
	taps int
}

func (h *homeScreen) Build(ctx w.BuildContext) w.Widget {
	return w.GestureDetector{
		OnTap: func() { h.taps++; w.Push(ctx, redScreen{}) },
		Child: w.ColoredBox{Color: h.color()},
	}
}

func (h *homeScreen) color() gfx.Color {
	if h.taps >= 10 {
		return gfx.Teal
	}
	return gfx.Blue
}

type redScreen struct{ w.Base }

func (redScreen) Build(ctx w.BuildContext) w.Widget {
	return w.GestureDetector{OnTap: func() { w.Pop(ctx) }, Child: w.ColoredBox{Color: gfx.Red}}
}

func TestPushAndPop(t *testing.T) {
	s := newNavScreen(t, homeScreen{})
	home := findState[*homeScreen](s.b.Root())
	if s.b.HandleBack() {
		t.Error("на первом экране «Назад» закрыл что-то")
	}

	s.tap(100, 150)
	s.frame()
	s.wait(150 * time.Millisecond)
	if c := s.color(100, 150); c == gfx.Blue || c == gfx.Red {
		t.Errorf("посреди перехода экран уже %v — перехода нет", c)
	}
	s.wait(300 * time.Millisecond)
	if c := s.color(100, 150); c != gfx.Red {
		t.Errorf("после перехода экран %v, а не красный", c)
	}
	if len(s.back) != 1 || !s.back[0] {
		t.Errorf("SetHandlesBack: %v, ждали [true]", s.back)
	}

	// Первый экран закрыт вторым: его изменения не перерисовывают экран.
	home.SetState(func() { home.taps = 10 }) // экран стал бирюзовым, но его не видно
	if d := s.frame(); !d.Empty() {
		t.Errorf("закрытый экран перерисовал %v", d)
	}

	if !s.b.HandleBack() {
		t.Fatal("«Назад» не закрыл второй экран")
	}
	s.wait(400 * time.Millisecond)
	if c := s.color(100, 150); c != gfx.Teal {
		t.Errorf("после Pop экран %v, а не бирюзовый", c)
	}
	if findState[*homeScreen](s.b.Root()) != home || home.taps != 10 {
		t.Error("первый экран не сохранил состояние")
	}
	if hasWidget[redScreen](s.b.Root()) {
		t.Error("закрытый экран остался в дереве")
	}
	if len(s.back) != 2 || s.back[1] {
		t.Errorf("SetHandlesBack: %v, ждали [true false]", s.back)
	}
}

// resultScreen открывает экран выбора и запоминает результат.
type resultScreen struct {
	w.State
	got any
}

func (r *resultScreen) Build(ctx w.BuildContext) w.Widget {
	return w.GestureDetector{
		OnTap: func() {
			w.PushRoute(ctx, w.Route{Page: w.Builder{Builder: func(ctx w.BuildContext) w.Widget {
				return w.GestureDetector{OnTap: func() { w.PopWith(ctx, "зелёный") }, Child: w.ColoredBox{Color: gfx.Green}}
			}}, OnPop: func(v any) { r.got = v }})
		},
		Child: w.ColoredBox{Color: gfx.Blue},
	}
}

func TestPopWithResult(t *testing.T) {
	s := newNavScreen(t, resultScreen{})
	s.tap(10, 10)
	s.wait(400 * time.Millisecond)
	s.tap(10, 10)
	if got := findState[*resultScreen](s.b.Root()).got; got != "зелёный" {
		t.Errorf("OnPop получил %v", got)
	}
	s.wait(400 * time.Millisecond)
	if c := s.color(10, 10); c != gfx.Blue {
		t.Errorf("после PopWith экран %v", c)
	}
}

// dialogScreen открывает окно 50×50 по центру над полупрозрачным затемнением.
type dialogScreen struct {
	w.State
	popped int
}

func (d *dialogScreen) Build(ctx w.BuildContext) w.Widget {
	return w.GestureDetector{
		OnTap: func() {
			w.PushRoute(ctx, w.Route{
				Page:        w.Center{Child: w.SizedBox{Width: new(50.0), Height: new(50.0), Child: w.ColoredBox{Color: gfx.Green}}},
				Popup:       true,
				Barrier:     gfx.ARGB(0x80000000),
				Dismissible: true,
				OnPop:       func(any) { d.popped++ },
			})
		},
		Child: w.ColoredBox{Color: gfx.Blue},
	}
}

func TestPopupOverScreen(t *testing.T) {
	s := newNavScreen(t, dialogScreen{})
	d := findState[*dialogScreen](s.b.Root())
	s.tap(10, 10)
	s.wait(400 * time.Millisecond)
	if c := s.color(100, 150); c != gfx.Green {
		t.Errorf("окно %v, а не зелёное", c)
	}
	if c := s.color(10, 10); c.B < 100 || c.B > 150 {
		t.Errorf("экран под затемнением %v: должен быть виден наполовину", c)
	}
	s.tap(100, 150) // по окну — не закрывается
	s.wait(100 * time.Millisecond)
	if d.popped != 0 {
		t.Error("окно закрылось касанием по нему самому")
	}
	s.tap(10, 10) // по затемнению — закрывается, а экран под ним касания не получает
	if d.popped != 1 {
		t.Fatal("касание затемнения не закрыло окно")
	}
	s.wait(400 * time.Millisecond)
	if c := s.color(10, 10); c != gfx.Blue {
		t.Errorf("после закрытия окна экран %v", c)
	}
	if findState[*dialogScreen](s.b.Root()).popped != 1 {
		t.Error("касание затемнения дошло до экрана под окном")
	}
}

// loginScreen заменяет себя главным экраном.
type loginScreen struct{ w.Base }

func (loginScreen) Build(ctx w.BuildContext) w.Widget {
	return w.GestureDetector{OnTap: func() { w.PushReplacement(ctx, redScreen{}) }, Child: w.ColoredBox{Color: gfx.Blue}}
}

func TestPushReplacement(t *testing.T) {
	s := newNavScreen(t, loginScreen{})
	s.tap(10, 10)
	s.wait(500 * time.Millisecond)
	if hasWidget[loginScreen](s.b.Root()) {
		t.Error("заменённый экран остался в дереве")
	}
	if s.b.HandleBack() {
		t.Error("«Назад» вернул к заменённому экрану")
	}
	if c := s.color(10, 10); c != gfx.Red {
		t.Errorf("экран %v, а не красный", c)
	}
}

// hasWidget сообщает, есть ли в дереве виджет типа T.
func hasWidget[T w.Widget](e *w.Element) bool {
	if _, ok := e.Widget().(T); ok {
		return true
	}
	for _, c := range e.Children() {
		if hasWidget[T](c) {
			return true
		}
	}
	return false
}

// changingScreen меняет цвет и добавляет полосу по команде — прямо во время перехода.
type changingScreen struct {
	w.State
	color gfx.Color
	band  bool
}

func (c *changingScreen) Build(w.BuildContext) w.Widget {
	kids := []w.Widget{w.Expanded{Child: w.ColoredBox{Color: c.color}}}
	if c.band {
		kids = append(kids, w.SizedBox{Height: new(50.0), Child: w.ColoredBox{Color: gfx.Orange}})
	}
	return w.Column{CrossAxisAlignment: rendering.CrossAxisStretch, Children: kids}
}

// Во время перехода экран рисуется снимком; изменения на нём обязаны попасть на экран сразу.
func TestSnapshotFollowsChanges(t *testing.T) {
	s := newNavScreen(t, homeScreen{})
	home := findState[*homeScreen](s.b.Root())
	slide := w.Transition{Build: func(anim, _ animation.Animation, child w.Widget) w.Widget {
		return w.SlideTransition{Position: anim, Begin: geom.Offset{X: 1}, Child: child} // непрозрачный сдвиг
	}}
	w.PushRoute(home.Context(), w.Route{Page: changingScreen{color: gfx.Red}, Transition: slide})
	s.frame()
	s.wait(100 * time.Millisecond) // экран уже непрозрачен, но ещё едет
	if !s.b.Animating() {
		t.Fatal("переход закончился слишком рано для проверки")
	}
	c := findState[*changingScreen](s.b.Root())
	c.SetState(func() { c.color = gfx.Green })
	s.frame()
	if got := s.color(190, 100); got != gfx.Green {
		t.Errorf("после смены цвета посреди перехода экран %v", got)
	}
	c.SetState(func() { c.band = true })
	s.frame()
	if got := s.color(190, 290); got != gfx.Orange {
		t.Errorf("полоса, добавленная посреди перехода, не видна: %v", got)
	}
	s.wait(400 * time.Millisecond)
	if got := s.color(10, 290); got != gfx.Orange {
		t.Errorf("после перехода полоса %v", got)
	}
}
