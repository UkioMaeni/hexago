package text

import (
	"image"
	"image/color"
	"math"
	"testing"

	"hexago/ui/geom"
	"hexago/ui/gfx"
)

// squares — запасной шрифт для тестов: каждый символ — красный квадрат размером в кегль.
type squares struct{ calls int }

func (s *squares) Measure(t string, size float64) float64 { return size * float64(clusters(t)) }

func (s *squares) Render(t string, size float64, _ gfx.Color) (*image.NRGBA, float64) {
	s.calls++
	n := int(math.Ceil(size))
	img := image.NewNRGBA(image.Rect(0, 0, n*clusters(t), n))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+3] = 255, 255
	}
	return img, size * 0.8
}

func withSquares(t *testing.T) *squares {
	s := &squares{}
	SetFallback(s)
	t.Cleanup(func() { SetFallback(nil) })
	return s
}

func TestClusters(t *testing.T) {
	for _, c := range []struct {
		s string
		n int
	}{
		{"👨‍👩‍👧", 1}, // семья через ZWJ
		{"🇷🇺🇫🇷", 2},  // два флага
		{"👍🏽", 1},    // оттенок кожи
		{"1️⃣", 1},   // цифра в рамке
		{"е́ж", 2},   // буква с ударением и ещё одна
	} {
		if n := clusters(c.s); n != c.n {
			t.Errorf("%q: символов %d, ждали %d", c.s, n, c.n)
		}
	}
}

func TestFallbackRunsAndCarets(t *testing.T) {
	withSquares(t)
	if r := runs("ab😀🇷🇺cd"); len(r) != 3 || !r[1].fallback || r[1].s != "😀🇷🇺" {
		t.Errorf("куски: %+v", r)
	}
	// «a», семья (5 рун), «b»: внутри семьи курсор не встаёт.
	st := Style{FontSize: 20}
	l := LayoutEditable("a👨‍👩‍👧b", st, math.Inf(1), false)
	x1, _, _ := l.Caret(1)
	x6, _, _ := l.Caret(6)
	if math.Abs(x6-x1-20) > 0.01 {
		t.Errorf("ширина эмодзи %v, ждали 20", x6-x1)
	}
	for i := 2; i <= 5; i++ {
		if x, _, _ := l.Caret(i); x != x1 {
			t.Errorf("курсор внутри эмодзи (%d) не прижат к его началу", i)
		}
	}
	for px := 0.0; px < x6+10; px += 1 {
		if i := l.IndexAt(geom.Offset{X: px, Y: 5}); i >= 2 && i <= 5 {
			t.Fatalf("касание в %v попало внутрь эмодзи: %d", px, i)
		}
	}
	if s, e := WordAt([]rune("a 👨‍👩‍👧"), 3); s != 2 || e != 7 {
		t.Errorf("слово-эмодзи: [%d, %d)", s, e)
	}
}

func TestFallbackPaints(t *testing.T) {
	sq := withSquares(t)
	l := LayoutEditable("a😀", Style{FontSize: 20, Color: gfx.Black}, math.Inf(1), false)
	c := gfx.NewImageCanvas(100, 40)
	c.Clear(gfx.White)
	l.Paint(c, geom.Zero)
	x, _, _ := l.Caret(1)
	img := c.ToNRGBA()
	if got := img.NRGBAAt(int(x)+10, 10); got != (color.NRGBA{255, 0, 0, 255}) {
		t.Errorf("на месте эмодзи %v, ждали красный квадрат", got)
	}
	l.Paint(c, geom.Zero)
	if sq.calls != 1 {
		t.Errorf("картинка эмодзи рисовалась %d раз — кеш не работает", sq.calls)
	}
}
