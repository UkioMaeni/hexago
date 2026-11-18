package widgets_test

import (
	"fmt"
	"testing"
	"time"

	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	w "hexago/ui/widgets"
)

// phone — экран 200×200 dp с ручным временем: кадры и касания идут по шагам 16 мс.
type phone struct {
	b   *w.Binding
	now time.Duration
	n   int
}

func newPhone(app w.Widget) *phone {
	p := &phone{b: w.NewBinding(app)}
	p.frame()
	return p
}

func (p *phone) frame() {
	p.b.Tick(p.now)
	p.b.DrawFrame(gfx.NewImageCanvas(200, 200), w.MediaQueryData{Size: geom.Size{Width: 200, Height: 200}, DevicePixelRatio: 1})
}

// wait — кадры каждые 16 мс в течение d.
func (p *phone) wait(d time.Duration) {
	for end := p.now + d; p.now < end; {
		p.now += 16 * time.Millisecond
		p.frame()
	}
}

func (p *phone) send(kind gestures.PointerKind, x, y float64) {
	p.b.HandlePointer(gestures.PointerEvent{Kind: kind, Pointer: p.n, Position: geom.Offset{X: x, Y: y}, Time: p.now})
}

// drag — палец от (x, y) сдвигается на (dx, dy) за steps шагов по 16 мс и отпускается.
func (p *phone) drag(x, y, dx, dy float64, steps int) { p.dragHold(x, y, dx, dy, steps, 0) }

// dragHold — как drag, но перед отпусканием палец стоит на месте hold: так список не бросают.
func (p *phone) dragHold(x, y, dx, dy float64, steps int, hold time.Duration) {
	p.n++
	p.send(gestures.PointerDown, x, y)
	for i := 1; i <= steps; i++ {
		p.now += 16 * time.Millisecond
		p.send(gestures.PointerMove, x+dx*float64(i)/float64(steps), y+dy*float64(i)/float64(steps))
		p.frame()
	}
	for end := p.now + hold; p.now < end; {
		p.now += 16 * time.Millisecond
		p.send(gestures.PointerMove, x+dx, y+dy)
		p.frame()
	}
	p.send(gestures.PointerUp, x+dx, y+dy)
	p.frame()
}

func (p *phone) tap(x, y float64) {
	p.n++
	p.send(gestures.PointerDown, x, y)
	p.send(gestures.PointerUp, x, y)
	p.frame()
}

func rows(n int, tapped *int) []w.Widget {
	out := make([]w.Widget, n)
	for i := range out {
		out[i] = w.GestureDetector{OnTap: func() { *tapped = i }, Child: w.SizedBox{Height: new(40.0), Child: w.ColoredBox{Color: gfx.Blue}}}
	}
	return out
}

func TestListViewDragAndTap(t *testing.T) {
	tapped := -1
	ctrl := &w.ScrollController{}
	p := newPhone(w.ListView{Controller: ctrl, Children: rows(50, &tapped)})
	// Тянем вверх на 100 dp и останавливаем палец перед отпусканием — полёта нет.
	p.dragHold(100, 150, 0, -100, 25, 150*time.Millisecond)
	p.wait(500 * time.Millisecond)
	if got := ctrl.Offset(); got < 99 || got > 101 {
		t.Errorf("после перетаскивания смещение %v, ожидали 100", got)
	}
	p.tap(100, 10) // экранная y=10 → содержимое y=110 → строка 2
	if tapped != 2 {
		t.Errorf("нажата строка %d, ожидали 2", tapped)
	}
}

func TestListViewFlingStopsAtEnd(t *testing.T) {
	ctrl := &w.ScrollController{}
	var tapped int
	p := newPhone(w.ListView{Controller: ctrl, Children: rows(50, &tapped)})
	p.drag(100, 190, 0, -150, 5) // быстро: 150 dp за 80 мс
	after := ctrl.Offset()
	p.wait(300 * time.Millisecond)
	if ctrl.Offset() <= after+50 {
		t.Errorf("брошенный список не полетел: %v → %v", after, ctrl.Offset())
	}
	p.wait(5 * time.Second)
	if p.b.Animating() {
		t.Error("полёт не закончился")
	}
	ctrl.AnimateTo(5000, 0, nil) // дальше конца — остановится на конце
	p.wait(time.Second)
	if got, max := ctrl.Offset(), ctrl.MaxOffset(); got != max || max != 50*40-200 {
		t.Errorf("в конце смещение %v, максимум %v; ожидали %v", got, max, 50*40-200)
	}
}

func TestLazyListBuildsOnlyVisible(t *testing.T) {
	built := map[int]bool{}
	ctrl := &w.ScrollController{}
	p := newPhone(w.ListView{Controller: ctrl, ItemCount: 10000, ItemExtent: 50,
		ItemBuilder: func(_ w.BuildContext, i int) w.Widget {
			built[i] = true
			return w.Text{Data: fmt.Sprint("Строка ", i)}
		}})
	if len(built) > 20 {
		t.Errorf("построено %d строк — ленивый список строит только видимые и запас", len(built))
	}
	clear(built)
	ctrl.JumpTo(250000) // строка 5000
	p.frame()
	if !built[5000] || built[0] || len(built) > 20 {
		t.Errorf("после прокрутки в середину построены: %d строк, 5000 — %v, 0 — %v", len(built), built[5000], built[0])
	}
}

func TestPageViewSnaps(t *testing.T) {
	var changed []int
	pages := []w.Widget{w.ColoredBox{Color: gfx.Red}, w.ColoredBox{Color: gfx.Green}, w.ColoredBox{Color: gfx.Blue}}
	ctrl := &w.PageController{}
	p := newPhone(w.PageView{Pages: pages, Controller: ctrl, OnPageChanged: func(i int) { changed = append(changed, i) }})

	p.drag(150, 100, -60, 0, 30) // медленно на треть страницы — возвращается
	p.wait(500 * time.Millisecond)
	if ctrl.Page() != 0 {
		t.Errorf("после короткого медленного сдвига страница %d, ожидали 0", ctrl.Page())
	}
	p.drag(150, 100, -60, 0, 3) // быстро — перелистывается
	p.wait(500 * time.Millisecond)
	if ctrl.Page() != 1 || len(changed) != 1 || changed[0] != 1 {
		t.Errorf("после броска страница %d, OnPageChanged %v; ожидали 1 и [1]", ctrl.Page(), changed)
	}
	ctrl.AnimateTo(5) // дальше последней — последняя
	p.wait(500 * time.Millisecond)
	if ctrl.Page() != 2 {
		t.Errorf("AnimateTo(5): страница %d, ожидали 2", ctrl.Page())
	}
}
