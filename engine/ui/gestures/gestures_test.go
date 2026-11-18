package gestures

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"hexago/ui/geom"
)

// harness разыгрывает касания так же, как widgets.Binding: PointerDown — всем распознавателям
// (AddPointer), потом арена закрывается; остальное — HandleEvent; после PointerUp арену подметают.
type harness struct {
	env  Env
	clk  *FakeClock
	recs []Recognizer
	last map[int]geom.Offset
	log  []string
}

func newHarness() *harness {
	clk := &FakeClock{}
	return &harness{env: Env{Arena: &Arena{}, Clock: clk}, clk: clk, last: map[int]geom.Offset{}}
}

func (h *harness) say(s string, args ...any) { h.log = append(h.log, fmt.Sprintf(s, args...)) }

func (h *harness) event(kind PointerKind, p int, x, y float64) PointerEvent {
	pos := geom.Offset{X: x, Y: y}
	e := PointerEvent{Kind: kind, Pointer: p, Position: pos, Local: pos, Delta: pos.Sub(h.last[p]), Time: h.clk.Now()}
	h.last[p] = pos
	return e
}

func (h *harness) down(p int, x, y float64) {
	e := h.event(PointerDown, p, x, y)
	for _, r := range h.recs {
		r.AddPointer(e)
	}
	h.env.Arena.Close(p)
}

func (h *harness) move(p int, x, y float64) {
	e := h.event(PointerMove, p, x, y)
	for _, r := range h.recs {
		r.HandleEvent(e)
	}
}

func (h *harness) up(p int, x, y float64) {
	e := h.event(PointerUp, p, x, y)
	for _, r := range h.recs {
		r.HandleEvent(e)
	}
	h.env.Arena.Sweep(p)
}

func (h *harness) wait(d time.Duration) { h.clk.Advance(d) }

func (h *harness) tap(name string) *TapRecognizer {
	r := &TapRecognizer{Env: h.env,
		OnTapDown:   func(TapDownDetails) { h.say("%s.down", name) },
		OnTapUp:     func(TapUpDetails) { h.say("%s.up", name) },
		OnTap:       func() { h.say("%s.tap", name) },
		OnTapCancel: func() { h.say("%s.cancel", name) },
	}
	h.recs = append(h.recs, r)
	return r
}

func (h *harness) drag(name string, axis DragAxis) *DragRecognizer {
	r := &DragRecognizer{Env: h.env, Axis: axis,
		OnStart:  func(d DragStartDetails) { h.say("%s.start %v", name, d.Position) },
		OnUpdate: func(d DragUpdateDetails) { h.say("%s.update %v", name, d.Delta) },
		OnEnd:    func(DragEndDetails) { h.say("%s.end", name) },
	}
	h.recs = append(h.recs, r)
	return r
}

func (h *harness) expect(t *testing.T, want string) {
	t.Helper()
	if got := strings.Join(h.log, ", "); got != want {
		t.Errorf("события:\n  %s\nожидали:\n  %s", got, want)
	}
	h.log = nil
}

func TestTap(t *testing.T) {
	h := newHarness()
	h.tap("t")
	h.down(1, 10, 10)
	h.up(1, 12, 11)
	h.expect(t, "t.down, t.up, t.tap")
}

func TestTapCancelledByMove(t *testing.T) {
	h := newHarness()
	h.tap("t")
	h.down(1, 10, 10)
	h.move(1, 10, 60)
	h.up(1, 10, 60)
	h.expect(t, "t.down, t.cancel")
}

func TestTapVersusDrag(t *testing.T) {
	// Палец не сдвинулся — нажатие; перетаскивание отказывается, когда палец отпущен.
	h := newHarness()
	h.tap("t")
	h.drag("v", DragVertical)
	h.down(1, 100, 100)
	h.up(1, 100, 102)
	h.expect(t, "t.down, t.up, t.tap")

	// Палец ушёл вниз — перетаскивание; нажатие молча уступает (OnTapDown ещё не был).
	// Нажатие отказывается раньше, чем перетаскивание видит то же движение, поэтому путь 40 dp
	// приходит частями 10 (до порога) + 20 + 10 — ничего не теряется.
	h.down(2, 100, 100)
	h.move(2, 100, 110)
	h.move(2, 100, 130)
	h.move(2, 100, 140)
	h.up(2, 100, 140)
	h.expect(t, "v.start {100 100}, v.update {0 10}, v.update {0 20}, v.update {0 10}, v.end")
}

func TestTapDownAfterPressTimeout(t *testing.T) {
	h := newHarness()
	h.tap("t")
	h.drag("v", DragVertical) // арена не решится сразу
	h.down(1, 0, 0)
	h.wait(50 * time.Millisecond)
	h.expect(t, "")
	h.wait(60 * time.Millisecond)
	h.expect(t, "t.down")
	h.move(1, 0, 40) // ушёл в перетаскивание — нажатие отменяется
	h.expect(t, "t.cancel, v.start {0 0}, v.update {0 40}")
}

func TestDoubleTap(t *testing.T) {
	h := newHarness()
	h.tap("t")
	h.recs = append(h.recs, &DoubleTapRecognizer{Env: h.env, OnDoubleTap: func() { h.say("double") }})

	h.down(1, 50, 50)
	h.up(1, 50, 50)
	h.wait(100 * time.Millisecond)
	h.down(2, 55, 52)
	h.up(2, 55, 52)
	h.expect(t, "t.down, t.cancel, double")

	// Одиночное нажатие — срабатывает, когда истекло ожидание второго.
	h.down(3, 50, 50)
	h.up(3, 50, 50)
	h.expect(t, "")
	h.wait(DoubleTapTimeout)
	h.expect(t, "t.down, t.up, t.tap")
}

func TestLongPress(t *testing.T) {
	h := newHarness()
	h.tap("t")
	h.recs = append(h.recs, &LongPressRecognizer{Env: h.env,
		OnLongPressStart:      func(LongPressStartDetails) { h.say("long.start") },
		OnLongPress:           func() { h.say("long") },
		OnLongPressMoveUpdate: func(d LongPressMoveUpdateDetails) { h.say("long.move %v", d.OffsetFromOrigin) },
		OnLongPressEnd:        func(LongPressEndDetails) { h.say("long.end") },
		OnLongPressUp:         func() { h.say("long.up") },
	})
	h.down(1, 10, 10)
	h.wait(LongPressTimeout + 10*time.Millisecond)
	h.move(1, 40, 10)
	h.up(1, 40, 10)
	h.expect(t, "t.down, t.cancel, long.start, long, long.move {30 0}, long.end, long.up")

	// Отпустили раньше — обычное нажатие.
	h.down(2, 10, 10)
	h.wait(200 * time.Millisecond)
	h.up(2, 10, 10)
	h.expect(t, "t.down, t.up, t.tap")
}

func TestDragAxesCompete(t *testing.T) {
	h := newHarness()
	h.drag("v", DragVertical)
	h.drag("h", DragHorizontal)
	h.down(1, 0, 0)
	h.move(1, 25, 5)
	h.up(1, 25, 5)
	h.expect(t, "h.start {0 0}, h.update {25 0}, h.end")
}

func TestLoneRecognizerWinsAtOnce(t *testing.T) {
	// Единственный участник побеждает, как только арену закрыли, — как во Flutter.
	h := newHarness()
	h.drag("p", DragFree)
	h.down(1, 0, 0)
	h.expect(t, "p.start {0 0}")
}

func TestPanVersusHorizontal(t *testing.T) {
	// Свободное перетаскивание внутри горизонтального: побеждает то, чей порог пройден первым.
	h := newHarness()
	h.drag("p", DragFree)
	h.drag("h", DragHorizontal)
	h.down(1, 0, 0)
	h.move(1, 10, 30) // по X 10 < 18, всего 31.6 < 36 — никто
	h.expect(t, "")
	h.move(1, 10, 40) // всего 41 > 36 — свободное
	h.expect(t, "p.start {0 0}, p.update {10 40}")
}

func TestScale(t *testing.T) {
	h := newHarness()
	var last ScaleUpdateDetails
	h.recs = append(h.recs, &ScaleRecognizer{Env: h.env,
		OnStart:  func(d ScaleStartDetails) { h.say("start %d", d.PointerCount) },
		OnUpdate: func(d ScaleUpdateDetails) { last = d },
		OnEnd:    func(ScaleEndDetails) { h.say("end") },
	})
	h.down(1, 100, 100)
	h.down(2, 200, 100)
	h.move(2, 300, 100) // расстояние 100 → 200
	h.move(1, 0, 100)   // 200 → 300, середина на месте
	if math.Abs(last.Scale-3) > 1e-9 {
		t.Errorf("Scale = %v, ожидали 3", last.Scale)
	}
	h.up(1, 0, 100)
	h.up(2, 300, 100)
	h.expect(t, "start 1, end") // единственный участник начал сразу, с одним пальцем
}

func TestInnermostWinsOnSweep(t *testing.T) {
	// Два нажатия друг в друге: при подметании побеждает первый участник — внутренний виджет.
	h := newHarness()
	h.tap("inner")
	h.tap("outer")
	h.down(1, 0, 0)
	h.up(1, 0, 0)
	h.expect(t, "inner.down, inner.up, inner.tap")
}

func TestVelocity(t *testing.T) {
	var v VelocityTracker
	for i := range 10 {
		v.Add(time.Duration(i)*10*time.Millisecond, geom.Offset{Y: float64(i) * 10}) // 1000 dp/с
	}
	if got := v.Velocity(); math.Abs(got.Y-1000) > 1 || math.Abs(got.X) > 1e-9 {
		t.Errorf("скорость %v, ожидали {0 1000}", got)
	}
}
