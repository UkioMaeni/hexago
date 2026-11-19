package widgets_test

import (
	"fmt"
	"image"
	"testing"
	"time"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/rendering"
	"hexago/ui/text"
	w "hexago/ui/widgets"
)

// damageScreen — экран 200×300 dp при плотности 2: кадры рисуются в один и тот же холст,
// как на телефоне, но только в повреждённой области.
type damageScreen struct {
	b      *w.Binding
	canvas *gfx.Canvas
	now    time.Duration
	media  w.MediaQueryData
}

func newDamageScreen(app w.Widget) *damageScreen {
	s := &damageScreen{b: w.NewBinding(app), canvas: gfx.NewImageCanvas(400, 600),
		media: w.MediaQueryData{Size: geom.Size{Width: 200, Height: 300}, DevicePixelRatio: 2}}
	s.frame()
	return s
}

// frame — кадр с перерисовкой только повреждённого; возвращает повреждённую область.
func (s *damageScreen) frame() image.Rectangle {
	s.b.Tick(s.now)
	damage := s.b.Layout(s.media)
	s.b.Paint(s.canvas, damage)
	return damage
}

// sameAsFull сравнивает холст с кадром того же дерева, нарисованным с нуля.
func (s *damageScreen) sameAsFull(t *testing.T, app w.Widget, label string) {
	t.Helper()
	full := gfx.NewImageCanvas(400, 600)
	b := w.NewBinding(app)
	b.DrawFrame(full, s.media)
	got, want := s.canvas.ToNRGBA(), full.ToNRGBA()
	bad := 0
	for i := range got.Pix {
		d := int(got.Pix[i]) - int(want.Pix[i])
		if d > 1 || d < -1 {
			bad++
		}
	}
	if bad > 0 {
		t.Errorf("%s: частичная перерисовка отличается от полной в %d каналах", label, bad)
	}
}

// counterScreen — цветные карточки и счётчик в углу.
type counterScreen struct {
	w.State
	n      int
	wide   bool
	scroll *w.ScrollController
	hidden bool
}

func (c *counterScreen) Build(w.BuildContext) w.Widget {
	width := 60.0
	if c.wide {
		width = 120
	}
	kids := []w.Widget{
		w.Text{Data: fmt.Sprint("Счёт: ", c.n), Style: text.Style{FontSize: 14}},
		w.Container{Width: new(width), Height: new(40.0), Color: gfx.Teal},
	}
	if !c.hidden {
		kids = append(kids, w.Container{Height: new(30.0), Color: gfx.Orange})
	}
	list := make([]w.Widget, 30)
	for i := range list {
		list[i] = w.Container{Height: new(20.0), Color: []gfx.Color{gfx.Red, gfx.Green, gfx.Blue}[i%3]}
	}
	kids = append(kids, w.SizedBox{Height: new(120.0), Child: w.ListView{Controller: c.scroll, Children: list}})
	return w.Column{CrossAxisAlignment: rendering.CrossAxisStart, Children: kids}
}

func TestPartialRepaint(t *testing.T) {
	ctrl := &w.ScrollController{}
	s := newDamageScreen(counterScreen{scroll: ctrl})
	c := findState[*counterScreen](s.b.Root())
	snapshot := func() counterScreen {
		return counterScreen{n: c.n, wide: c.wide, hidden: c.hidden, scroll: &w.ScrollController{InitialOffset: ctrl.Offset()}}
	}

	if d := s.frame(); !d.Empty() {
		t.Errorf("ничего не менялось, а повреждено %v", d)
	}

	c.SetState(func() { c.n = 7 })
	d := s.frame()
	if d.Empty() || d.Dy() > 60 || d.Dx() > 200 {
		t.Errorf("изменился только текст счётчика, а повреждено %v", d)
	}
	s.sameAsFull(t, snapshot(), "текст")

	c.SetState(func() { c.wide = true })
	s.frame()
	s.sameAsFull(t, snapshot(), "ширина контейнера")

	ctrl.JumpTo(55)
	s.frame()
	s.sameAsFull(t, snapshot(), "прокрутка")

	c.SetState(func() { c.hidden = true }) // контейнер убран, всё ниже сдвинулось вверх
	s.frame()
	s.sameAsFull(t, snapshot(), "удалённый виджет")
}
