package material

import (
	"testing"
	"time"

	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/widgets"
)

// motionScreen — экран 200×200 с ручным временем кадров.
type motionScreen struct {
	b   *widgets.Binding
	now time.Duration
	img *gfx.Canvas
}

func newMotionScreen(app widgets.Widget) *motionScreen {
	s := &motionScreen{b: widgets.NewBinding(App{Mode: ThemeLight, Home: app})}
	s.frame()
	return s
}

func (s *motionScreen) frame() {
	s.b.Tick(s.now)
	s.img = gfx.NewImageCanvas(200, 200)
	s.b.DrawFrame(s.img, widgets.MediaQueryData{Size: geom.Size{Width: 200, Height: 200}, DevicePixelRatio: 1})
}

func (s *motionScreen) wait(d time.Duration) {
	for end := s.now + d; s.now < end; {
		s.now += 16 * time.Millisecond
		s.frame()
	}
}

func (s *motionScreen) pixel(x, y int) [4]uint8 {
	c := s.img.ToNRGBA().NRGBAAt(x, y)
	return [4]uint8{c.R, c.G, c.B, c.A}
}

// toggle — выключатель, который включается нажатием.
type toggle struct {
	widgets.State
	on bool
}

func (t *toggle) Build(widgets.BuildContext) widgets.Widget {
	return widgets.Align{Alignment: geom.TopLeft,
		Child: Switch{Value: t.on, OnChanged: func(v bool) { t.SetState(func() { t.on = v }) }}}
}

func TestSwitchSlides(t *testing.T) {
	s := newMotionScreen(toggle{})
	// Дорожка 52×32 начинается в (4, 8); середина ползунка: выключен — x 20, включён — x 40.
	// Центр включённого ползунка: середина белых (OnPrimary) точек на линии y = 24.
	thumbX := func() float64 {
		sum, n := 0, 0
		for x := 6; x < 54; x++ {
			if p := s.pixel(x, 24); p[0] > 250 && p[1] > 250 && p[2] > 250 {
				sum, n = sum+x, n+1
			}
		}
		if n == 0 {
			return 0
		}
		return float64(sum) / float64(n)
	}
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerDown, Pointer: 1, Position: geom.Offset{X: 30, Y: 24}})
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerUp, Pointer: 1, Position: geom.Offset{X: 30, Y: 24}})
	s.frame() // перестройка: анимация начата
	s.frame() // время её начала
	s.wait(100 * time.Millisecond)
	if !s.b.Animating() {
		t.Fatal("выключатель переключился без анимации")
	}
	s.wait(600 * time.Millisecond)
	if s.b.Animating() {
		t.Error("анимация выключателя не закончилась")
	}
	if x := thumbX(); x < 34 {
		t.Errorf("после включения ползунок у x=%v, ожидали справа (около 40)", x)
	}
}

func TestRippleShowsAndFades(t *testing.T) {
	s := newMotionScreen(widgets.Align{Alignment: geom.TopLeft, Child: widgets.SizedBox{Width: new(120.0), Height: new(80.0),
		Child: InkWell{OnTap: func() {}, OverlayColor: gfx.Black}}})
	before := s.pixel(100, 60)
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerDown, Pointer: 1, Position: geom.Offset{X: 10, Y: 10}})
	s.frame()
	s.wait(300 * time.Millisecond)
	during := s.pixel(100, 60) // далеко от точки касания: туда дошла волна
	s.b.HandlePointer(gestures.PointerEvent{Kind: gestures.PointerUp, Pointer: 1, Position: geom.Offset{X: 10, Y: 10}})
	s.wait(600 * time.Millisecond)
	after := s.pixel(100, 60)
	if during == before || after != before {
		t.Errorf("волна: до %v, во время %v, после %v", before, during, after)
	}
}

func TestIndeterminateSpinnerAnimates(t *testing.T) {
	s := newMotionScreen(widgets.Center{Child: CircularProgressIndicator{}})
	s.wait(100 * time.Millisecond)
	if !s.b.Animating() {
		t.Error("бесконечный индикатор не крутится")
	}
	s2 := newMotionScreen(widgets.Center{Child: CircularProgressIndicator{Value: new(0.5)}})
	s2.wait(100 * time.Millisecond)
	if s2.b.Animating() {
		t.Error("индикатор со значением не должен анимироваться")
	}
}
