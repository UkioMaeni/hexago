package widgets_test

import (
	"strings"
	"testing"
	"time"

	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	w "hexago/ui/widgets"
)

// screen — приложение 200×200 dp для проверки касаний.
type screen struct {
	b      *w.Binding
	clk    *gestures.FakeClock
	frames int // сколько раз приложение попросило кадр
	next   int // номер следующего касания
}

func newScreen(app w.Widget) *screen {
	s := &screen{b: w.NewBinding(app), clk: &gestures.FakeClock{}}
	s.b.SetClock(s.clk)
	s.b.RequestFrame = func() { s.frames++ }
	s.draw()
	return s
}

func (s *screen) draw() {
	s.b.DrawFrame(gfx.NewImageCanvas(200, 200), w.MediaQueryData{Size: geom.Size{Width: 200, Height: 200}, DevicePixelRatio: 1})
}

func (s *screen) send(kind gestures.PointerKind, p int, x, y float64) {
	s.b.HandlePointer(gestures.PointerEvent{Kind: kind, Pointer: p, Position: geom.Offset{X: x, Y: y}, Time: s.clk.Now()})
}

// tap — палец опустился и поднялся в точке (x, y).
func (s *screen) tap(x, y float64) {
	s.next++
	s.send(gestures.PointerDown, s.next, x, y)
	s.send(gestures.PointerUp, s.next, x, y)
}

// tapCounter — счётчик нажатий на виджете с состоянием.
type tapCounter struct {
	w.State
	taps int
}

func (c *tapCounter) Build(w.BuildContext) w.Widget {
	return w.GestureDetector{
		OnTap: func() { c.SetState(func() { c.taps++ }) },
		Child: w.SizedBox{Width: new(100.0), Height: new(100.0), Child: w.ColoredBox{Color: gfx.Blue}},
	}
}

func TestTapChangesState(t *testing.T) {
	s := newScreen(w.Center{Child: tapCounter{}})
	c := findState[*tapCounter](s.b.Root())
	s.tap(100, 100)
	if c.taps != 1 || s.frames != 1 {
		t.Fatalf("после нажатия taps=%d, запросов кадра %d; ожидали 1 и 1", c.taps, s.frames)
	}
	s.draw()
	s.tap(10, 10) // мимо синего квадрата
	if c.taps != 1 {
		t.Errorf("нажатие мимо засчитано: taps=%d", c.taps)
	}
}

// findState — живой экземпляр виджета с состоянием типа T.
func findState[T any](e *w.Element) T {
	v, _ := lookState[T](e)
	return v
}

func lookState[T any](e *w.Element) (T, bool) {
	if v, ok := e.State().(T); ok {
		return v, true
	}
	for _, c := range e.Children() {
		if v, ok := lookState[T](c); ok {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// layered — фон и передний слой во весь экран; front оборачивает передний слой.
func layered(log *[]string, front func(w.Widget) w.Widget) w.Widget {
	detector := func(name string) w.Widget {
		return w.GestureDetector{OnTap: func() { *log = append(*log, name) }}
	}
	return w.Stack{Children: []w.Widget{
		w.PositionedFill(detector("фон")),
		w.PositionedFill(front(detector("слой"))),
	}}
}

func TestIgnoreAndAbsorbPointer(t *testing.T) {
	cases := map[string]struct {
		front func(w.Widget) w.Widget
		want  string
	}{
		"без обёртки":        {func(c w.Widget) w.Widget { return c }, "слой"},
		"IgnorePointer":      {func(c w.Widget) w.Widget { return w.IgnorePointer{Child: c} }, "фон"},
		"AbsorbPointer":      {func(c w.Widget) w.Widget { return w.AbsorbPointer{Child: c} }, ""},
		"выключенный Ignore": {func(c w.Widget) w.Widget { return w.IgnorePointer{Disabled: true, Child: c} }, "слой"},
	}
	for name, c := range cases {
		var log []string
		s := newScreen(layered(&log, c.front))
		s.tap(50, 50)
		if got := strings.Join(log, ","); got != c.want {
			t.Errorf("%s: нажатие получил %q, ожидали %q", name, got, c.want)
		}
	}
}

func TestLocalPositionThroughPaddingAndScale(t *testing.T) {
	var got geom.Offset
	app := w.Padding{Padding: geom.All(20), Child: w.Scale(2, w.Listener{
		OnPointerDown: func(e gestures.PointerEvent) { got = e.Local },
		Child:         w.ColoredBox{Color: gfx.Red},
	})}
	s := newScreen(app)
	// Квадрат 160×160 начинается в (20, 20) и увеличен вдвое вокруг центра (100, 100):
	// точка экрана (120, 140) — это (80+20/2, 80+40/2) = (90, 100) в координатах квадрата.
	s.send(gestures.PointerDown, 1, 120, 140)
	if got != (geom.Offset{X: 90, Y: 100}) {
		t.Errorf("Local = %v, ожидали {90 100}", got)
	}
}

func TestLongPressAndDragInDetector(t *testing.T) {
	var log []string
	app := w.GestureDetector{
		OnLongPress:          func() { log = append(log, "long") },
		OnVerticalDragUpdate: func(d gestures.DragUpdateDetails) { log = append(log, "drag") },
	}
	s := newScreen(app)
	s.send(gestures.PointerDown, 1, 50, 50)
	s.clk.Advance(600 * time.Millisecond)
	s.send(gestures.PointerUp, 1, 50, 50)

	s.send(gestures.PointerDown, 2, 50, 50)
	s.send(gestures.PointerMove, 2, 50, 90)
	s.send(gestures.PointerUp, 2, 50, 90)
	if got := strings.Join(log, ","); got != "long,drag" {
		t.Errorf("жесты: %q, ожидали long,drag", got)
	}
}

func TestPanAndScaleTogetherPanics(t *testing.T) {
	app := w.GestureDetector{OnPanUpdate: func(gestures.DragUpdateDetails) {}, OnScaleUpdate: func(gestures.ScaleUpdateDetails) {}}
	s := newScreen(w.Center{Child: app})
	// Ошибка в Build показывается ErrorBox, а не роняет приложение: в центре — красный.
	img := gfx.NewImageCanvas(200, 200)
	s.b.DrawFrame(img, w.MediaQueryData{Size: geom.Size{Width: 200, Height: 200}, DevicePixelRatio: 1})
	if c := img.ToNRGBA().NRGBAAt(100, 100); c.R < 200 || c.G > 80 {
		t.Errorf("ожидался ErrorBox, в центре %v", c)
	}
}
