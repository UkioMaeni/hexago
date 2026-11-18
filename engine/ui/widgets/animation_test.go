package widgets_test

import (
	"testing"
	"time"

	"hexago/ui/animation"
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/rendering"
	w "hexago/ui/widgets"
)

// grow — квадрат, который по флагу big плавно растёт и краснеет.
type grow struct {
	w.State
	big bool
}

func (g *grow) Build(w.BuildContext) w.Widget {
	size, color := 20.0, gfx.Blue
	if g.big {
		size, color = 100, gfx.Red
	}
	return w.Align{Alignment: geom.TopLeft, Child: w.AnimatedContainer{
		Width: new(size), Height: new(20.0), Color: color, Duration: 300 * time.Millisecond, Curve: animation.Linear,
	}}
}

func TestAnimatedContainer(t *testing.T) {
	b := w.NewBinding(grow{})
	frames := 0
	b.RequestFrame = func() { frames++ }
	now := time.Duration(0)
	draw := func() *gfx.Canvas {
		b.Tick(now)
		c := gfx.NewImageCanvas(200, 50)
		b.DrawFrame(c, w.MediaQueryData{Size: geom.Size{Width: 200, Height: 50}, DevicePixelRatio: 1})
		return c
	}
	width := func() float64 {
		var box rendering.Box
		var find func(e *w.Element)
		find = func(e *w.Element) {
			if _, ok := e.Widget().(w.ColoredBox); ok && box == nil {
				box = e.Box()
			}
			for _, c := range e.Children() {
				find(c)
			}
		}
		find(b.Root())
		return rendering.SizeOf(box).Width
	}
	draw()
	g := findState[*grow](b.Root())
	g.SetState(func() { g.big = true })

	// Кадр с перестройкой запускает анимацию; время её начала — следующий кадр (как во Flutter:
	// время кадра раздаётся до построения, поэтому анимация, начатая при построении, ждёт кадр).
	draw()
	draw()
	now += 150 * time.Millisecond
	img := draw().ToNRGBA()
	if wd := width(); wd < 55 || wd > 65 {
		t.Errorf("на полпути ширина %v, ожидали около 60", wd)
	}
	if c := img.NRGBAAt(5, 5); c.R < 90 || c.B < 90 {
		t.Errorf("на полпути цвет %v — ожидали смесь синего и красного", c)
	}
	now += 200 * time.Millisecond
	draw()
	if wd := width(); wd != 100 {
		t.Errorf("в конце ширина %v, ожидали 100", wd)
	}
	if b.Animating() {
		t.Error("анимация закончилась, а кадры всё ещё нужны")
	}
}

func TestAnimationStopsWhenWidgetRemoved(t *testing.T) {
	var ctrl *animation.Controller
	b := w.NewBinding(w.Center{Child: blinker{ctrl: &ctrl}})
	b.Tick(0)
	b.DrawFrame(gfx.NewImageCanvas(10, 10), w.MediaQueryData{Size: geom.Size{Width: 10, Height: 10}, DevicePixelRatio: 1})
	if !b.Animating() {
		t.Fatal("Repeat не запустил анимацию")
	}
	b.SetApp(w.Center{})
	b.Tick(16 * time.Millisecond)
	b.DrawFrame(gfx.NewImageCanvas(10, 10), w.MediaQueryData{Size: geom.Size{Width: 10, Height: 10}, DevicePixelRatio: 1})
	b.Tick(32 * time.Millisecond)
	if b.Animating() || ctrl.IsAnimating() {
		t.Error("виджет убран, а его анимация идёт")
	}
}

// blinker — бесконечная анимация, созданная через NewAnimation.
type blinker struct {
	w.State
	ctrl **animation.Controller
	a    *animation.Controller
}

func (b *blinker) Init(ctx w.BuildContext) {
	b.a = w.NewAnimation(ctx, 500*time.Millisecond)
	b.a.Repeat(true)
	*b.ctrl = b.a
}

func (b *blinker) Build(w.BuildContext) w.Widget {
	return w.Opacity{Opacity: b.a.Value(), Child: w.ColoredBox{Color: gfx.Red}}
}
