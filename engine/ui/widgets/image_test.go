package widgets_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"testing"
	"time"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/hexagotest"
	"hexago/ui/painting"
	w "hexago/ui/widgets"
)

// testPicture — PNG 60×30: слева направо от красного к синему, чёрная полоса посередине и белый
// круг слева — по ним видно, какая часть картинки показана.
func testPicture(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 60, 30))
	for y := range 30 {
		for x := range 60 {
			c := color.NRGBA{R: uint8(255 - x*4), B: uint8(x * 4), G: 80, A: 255}
			if x >= 28 && x < 32 {
				c = color.NRGBA{A: 255}
			}
			if math.Hypot(float64(x)-12, float64(y)-15) < 8 {
				c = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// imageScreen — экран в памяти: кадры рисуются на одном холсте, пока не выполнится условие.
type imageScreen struct {
	b *w.Binding
	c *gfx.Canvas
}

func newImageScreen(app w.Widget) *imageScreen {
	s := &imageScreen{b: w.NewBinding(app), c: gfx.NewImageCanvas(400, 300)}
	s.c.Clear(gfx.White)
	s.frame()
	return s
}

func (s *imageScreen) frame() {
	s.b.DrawFrame(s.c, w.MediaQueryData{Size: geom.Size{Width: 200, Height: 150}, DevicePixelRatio: 2})
}

// pumpUntil рисует кадры, пока cond не станет true (картинки грузятся в своих горутинах).
func (s *imageScreen) pumpUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("не дождались за 5 с")
		}
		time.Sleep(time.Millisecond)
		s.frame()
	}
}

// rawImages — все RawImage в дереве.
func rawImages(e *w.Element) []*w.Element {
	var out []*w.Element
	if _, ok := e.Widget().(w.RawImage); ok {
		out = append(out, e)
	}
	for _, c := range e.Children() {
		out = append(out, rawImages(c)...)
	}
	return out
}

func loaded(e *w.Element) bool {
	for _, r := range rawImages(e) {
		if r.Widget().(w.RawImage).Image == nil {
			return false
		}
	}
	return true
}

// blocked — источник, который грузится, пока не закроют gate.
type blocked struct{ gate chan struct{} }

func (p blocked) Key() any { return p.gate }
func (p blocked) Load() (image.Image, error) {
	<-p.gate
	return nil, errors.New("не дождались")
}

func TestImageFitsGolden(t *testing.T) {
	defer painting.ClearImageCache()
	pic := painting.MemoryImage{Bytes: testPicture(t)}
	gate := make(chan struct{})
	defer close(gate)
	cell := func(img w.Widget) w.Widget {
		return w.Padding{Padding: geom.All(3), Child: w.SizedBox{Width: new(44.0), Height: new(44.0),
			Child: w.ColoredBox{Color: gfx.ColorFromHex("#D0D0D0"), Child: img}}}
	}
	img := func(fit painting.BoxFit, a geom.Alignment) w.Widget {
		return cell(w.Image{Source: pic, Fit: fit, Alignment: a})
	}
	app := w.Column{Children: []w.Widget{
		w.Row{Children: []w.Widget{
			img(painting.BoxFitScaleDown, geom.Center), img(painting.BoxFitContain, geom.Center),
			img(painting.BoxFitCover, geom.Center), img(painting.BoxFitFill, geom.Center),
		}},
		w.Row{Children: []w.Widget{
			img(painting.BoxFitWidth, geom.Center), img(painting.BoxFitHeight, geom.Center),
			img(painting.BoxFitNone, geom.Center), img(painting.BoxFitCover, geom.CenterLeft),
		}},
		w.Row{Children: []w.Widget{
			// Без размера: 60×30 dp, как у картинки.
			w.Padding{Padding: geom.All(3), Child: w.Image{Source: pic}},
			cell(w.Image{Source: blocked{gate}, Placeholder: w.ColoredBox{Color: gfx.ColorFromHex("#FFCC80")}}),
			cell(w.Image{Source: painting.MemoryImage{Bytes: []byte("не картинка")},
				Error: func(_ w.BuildContext, err error) w.Widget {
					return w.ColoredBox{Color: gfx.ColorFromHex("#E57373")}
				}}),
		}},
	}}
	s := newImageScreen(app)
	s.pumpUntil(t, func() bool { return len(rawImages(s.b.Root())) == 9 && loaded(s.b.Root()) })
	hexagotest.Golden(t, "image_fits", s.c.ToNRGBA())
}

// Пока картинка грузится, Image занимает Width×Height; загруженная из кеша — показывается сразу.
func TestImageSizeWhileLoadingAndFromCache(t *testing.T) {
	defer painting.ClearImageCache()
	pic := painting.MemoryImage{Bytes: testPicture(t)}
	app := w.Align{Alignment: geom.TopLeft, Child: w.Image{Source: pic, Width: new(50.0), Height: new(40.0)}}
	s := newImageScreen(app)
	r := rawImages(s.b.Root())
	if len(r) != 1 || w.BoundsOf(r[0]).Size() != (geom.Size{Width: 50, Height: 40}) {
		t.Fatalf("пока грузится: %v", w.BoundsOf(r[0]))
	}
	s.pumpUntil(t, func() bool { return loaded(s.b.Root()) })

	s2 := newImageScreen(app)
	if !loaded(s2.b.Root()) {
		t.Error("картинка из кеша не показана в первом кадре")
	}
}

// imageSwitcher — Image, у которого меняют источник.
type imageSwitcher struct {
	w.State
	Src painting.ImageProvider
}

func (s *imageSwitcher) Build(w.BuildContext) w.Widget { return w.Image{Source: s.Src} }

// Пока грузилась первая картинка, источник сменили: показывается вторая, а не та, что пришла позже.
func TestImageIgnoresStaleLoad(t *testing.T) {
	defer painting.ClearImageCache()
	gate := make(chan struct{})
	slow := slowPicture{gate: gate, data: testPicture(t)}
	s := newImageScreen(imageSwitcher{Src: slow})
	sw := findState[*imageSwitcher](s.b.Root())
	second := painting.MemoryImage{Bytes: pngOf(t, 10, 10)}
	sw.SetState(func() { sw.Src = second })
	s.pumpUntil(t, func() bool { return loaded(s.b.Root()) })
	close(gate) // первая догружается после смены
	for range 20 {
		time.Sleep(time.Millisecond)
		s.frame()
	}
	if b := rawImages(s.b.Root())[0].Widget().(w.RawImage).Image; b.Width() != 10 {
		t.Errorf("показана картинка %d×%d, ожидали вторую (10×10)", b.Width(), b.Height())
	}
}

type slowPicture struct {
	gate chan struct{}
	data []byte
}

func (p slowPicture) Key() any { return p.gate }
func (p slowPicture) Load() (image.Image, error) {
	<-p.gate
	return png.Decode(bytes.NewReader(p.data))
}

func pngOf(t *testing.T, wd, h int) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, wd, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestImageErrorKeepsSize(t *testing.T) {
	var got error
	app := w.Align{Alignment: geom.TopLeft, Child: w.Column{Children: []w.Widget{
		w.Image{Source: painting.MemoryImage{Bytes: []byte("плохо")}, Width: new(30.0), Height: new(20.0)},
		w.Image{Source: painting.MemoryImage{Bytes: []byte("тоже плохо")},
			Error: func(_ w.BuildContext, err error) w.Widget { got = err; return w.SizedBox{} }},
	}}}
	s := newImageScreen(app)
	s.pumpUntil(t, func() bool { return got != nil })
	if !strings.Contains(got.Error(), "формат") {
		t.Errorf("ошибка: %v", got)
	}
	if r := rawImages(s.b.Root()); len(r) != 1 || w.BoundsOf(r[0]).Size() != (geom.Size{Width: 30, Height: 20}) {
		t.Errorf("без Error: место под картинку %v", w.BoundsOf(r[0]))
	}
}
