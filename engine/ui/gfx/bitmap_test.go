package gfx_test

import (
	"image"
	"image/color"
	"math"
	"testing"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/hexagotest"
)

func nrgba(w, h int, px ...color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i, p := range px {
		img.SetNRGBA(i%w, i/w, p)
	}
	return img
}

var (
	black = color.NRGBA{A: 255}
	white = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
)

func TestNewBitmapFormats(t *testing.T) {
	b := gfx.NewBitmap(nrgba(2, 1, color.NRGBA{R: 255, A: 255}, color.NRGBA{G: 200, A: 100}))
	if b.Opaque() || b.At(0, 0) != (gfx.Color{R: 255, A: 255}) || b.At(1, 0) != (gfx.Color{G: 200, A: 100}) {
		t.Errorf("NRGBA: opaque=%v %v %v", b.Opaque(), b.At(0, 0), b.At(1, 0))
	}
	pre := image.NewRGBA(image.Rect(0, 0, 1, 1))
	pre.SetRGBA(0, 0, color.RGBA{R: 100, A: 128}) // предумножено: на самом деле R ≈ 199
	if got := gfx.NewBitmap(pre).At(0, 0); got.R < 198 || got.R > 200 || got.A != 128 {
		t.Errorf("RGBA с предумножением: %v", got)
	}
	ycc := image.NewYCbCr(image.Rect(0, 0, 4, 4), image.YCbCrSubsampleRatio420)
	for i := range ycc.Y {
		ycc.Y[i] = 235
	}
	for i := range ycc.Cb {
		ycc.Cb[i], ycc.Cr[i] = 128, 128
	}
	if b := gfx.NewBitmap(ycc); !b.Opaque() || b.At(3, 3).R < 230 {
		t.Errorf("YCbCr: opaque=%v %v", b.Opaque(), b.At(3, 3))
	}
}

func canvasPixel(c *gfx.Canvas, x, y int) color.NRGBA { return c.ToNRGBA().NRGBAAt(x, y) }

func TestDrawBitmapExactCopy(t *testing.T) {
	src := nrgba(3, 2, color.NRGBA{R: 10, A: 255}, color.NRGBA{G: 20, A: 255}, color.NRGBA{B: 30, A: 255},
		color.NRGBA{R: 40, A: 255}, color.NRGBA{G: 50, A: 255}, color.NRGBA{B: 60, A: 255})
	c := gfx.NewImageCanvas(5, 4)
	c.Clear(gfx.White)
	c.DrawBitmap(gfx.NewBitmap(src), geom.Rect{}, geom.RectXYWH(1, 1, 3, 2), 1)
	for y := range 2 {
		for x := range 3 {
			if got, want := canvasPixel(c, x+1, y+1), src.NRGBAAt(x, y); got != want {
				t.Errorf("(%d,%d): %v, ожидали %v", x, y, got, want)
			}
		}
	}
	if got := canvasPixel(c, 0, 0); got != white {
		t.Errorf("вне картинки: %v", got)
	}
}

// Уменьшение вдвое усредняет пиксели, а не выбирает каждый второй (иначе мелкий узор даёт «рябь»).
func TestDrawBitmapDownscaleAverages(t *testing.T) {
	var px []color.NRGBA
	for i := range 16 {
		if (i%4+i/4)%2 == 0 {
			px = append(px, black)
		} else {
			px = append(px, white)
		}
	}
	c := gfx.NewImageCanvas(2, 2)
	c.DrawBitmap(gfx.NewBitmap(nrgba(4, 4, px...)), geom.Rect{}, geom.RectXYWH(0, 0, 2, 2), 1)
	for y := range 2 {
		for x := range 2 {
			if p := canvasPixel(c, x, y); p.R < 126 || p.R > 129 || p.A != 255 {
				t.Errorf("(%d,%d): %v, ожидали серый 128", x, y, p)
			}
		}
	}
}

// Прозрачные пиксели не затемняют край: красный + прозрачный = полупрозрачный красный, а не тёмно-красный.
func TestDrawBitmapNoDarkFringe(t *testing.T) {
	c := gfx.NewImageCanvas(1, 1)
	c.Clear(gfx.White)
	c.DrawBitmap(gfx.NewBitmap(nrgba(2, 1, color.NRGBA{R: 255, A: 255}, color.NRGBA{})), geom.Rect{}, geom.RectXYWH(0, 0, 1, 1), 1)
	if p := canvasPixel(c, 0, 0); p.R != 255 || p.G < 126 || p.G > 129 {
		t.Errorf("половина красного на белом: %v, ожидали (255, 128, 128)", p)
	}
}

func TestDrawBitmapUpscaleBilinear(t *testing.T) {
	c := gfx.NewImageCanvas(4, 1)
	c.DrawBitmap(gfx.NewBitmap(nrgba(2, 1, black, white)), geom.Rect{}, geom.RectXYWH(0, 0, 4, 1), 1)
	want := []uint8{0, 64, 191, 255}
	for x, w := range want {
		if p := canvasPixel(c, x, 0); int(p.R)-int(w) > 1 || int(w)-int(p.R) > 1 {
			t.Errorf("x=%d: %d, ожидали %d", x, p.R, w)
		}
	}
}

func TestDrawBitmapOpacityClipAndPart(t *testing.T) {
	c := gfx.NewImageCanvas(4, 4)
	c.Clear(gfx.Black)
	c.Save()
	c.ClipRect(geom.RectXYWH(0, 0, 2, 4))
	c.DrawBitmap(gfx.NewBitmap(nrgba(1, 1, white)), geom.Rect{}, geom.RectXYWH(0, 0, 4, 4), 0.5)
	c.Restore()
	if p := canvasPixel(c, 1, 1); p.R < 127 || p.R > 129 {
		t.Errorf("полупрозрачный белый на чёрном: %v", p)
	}
	if p := canvasPixel(c, 3, 1); p.R != 0 {
		t.Errorf("за обрезкой: %v", p)
	}

	// Правая половина картинки 2×1 (белая) — на весь холст.
	c.DrawBitmap(gfx.NewBitmap(nrgba(2, 1, black, white)), geom.Rect{Left: 1, Right: 2, Bottom: 1}, geom.RectXYWH(0, 0, 4, 4), 1)
	if p := canvasPixel(c, 0, 0); p.R != 255 {
		t.Errorf("часть картинки: %v", p)
	}
}

// photo — картинка с плавными переходами, кругами и мелкими полосами (на них видна «рябь»).
func photo(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			fx, fy := float64(x)/float64(w), float64(y)/float64(h)
			c := color.NRGBA{R: uint8(255 * fx), G: uint8(255 * fy), B: uint8(255 * (1 - fx)), A: 255}
			if d := math.Hypot(fx-0.3, fy-0.5); d < 0.25 && int(d*200)%2 == 0 {
				c = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			}
			if fx > 0.6 && x%2 == 0 {
				c = color.NRGBA{A: 255}
			}
			if fy > 0.85 { // прозрачная полоса внизу — проверка края
				c.A = uint8(255 * (1 - fy) / 0.15)
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func TestDrawBitmapGolden(t *testing.T) {
	b := gfx.NewBitmap(photo(400, 300))
	c := gfx.NewImageCanvas(400, 200)
	c.Clear(gfx.ColorFromHex("#EEEEEE"))
	c.DrawBitmap(b, geom.Rect{}, geom.RectXYWH(10, 10, 120, 90), 1)                                              // уменьшение в 3,3 раза
	c.DrawBitmap(b, geom.Rect{Left: 100, Top: 120, Right: 140, Bottom: 150}, geom.RectXYWH(140, 10, 120, 90), 1) // кусок ×3
	c.Save()
	c.Translate(330, 55)
	c.Transform(geom.Rotation(math.Pi / 8))
	c.DrawBitmap(b, geom.Rect{}, geom.RectXYWH(-60, -45, 120, 90), 1) // с поворотом
	c.Restore()
	c.Scale(2, 2) // плотность 2: 60×45 dp — это 120×90 пикселей
	c.DrawBitmap(b, geom.Rect{}, geom.RectXYWH(5, 52, 60, 45), 0.6)
	hexagotest.Golden(t, "bitmap_scaled", c.ToNRGBA())
}

func BenchmarkBlitOpaqueScreen(b *testing.B) {
	bm := gfx.NewBitmap(photo(1080, 2400))
	c := gfx.NewImageCanvas(1080, 2400)
	for b.Loop() {
		c.DrawBitmap(bm, geom.Rect{}, geom.RectXYWH(0, 0, 1080, 2400), 1)
	}
}

func BenchmarkResamplePhoto(b *testing.B) {
	bm := gfx.NewBitmap(photo(4000, 3000))
	c := gfx.NewImageCanvas(1080, 810)
	size := 1080.0
	for b.Loop() {
		size-- // каждый раз новый размер — копия считается заново
		c.DrawBitmap(bm, geom.Rect{}, geom.RectXYWH(0, 0, size, size*3/4), 1)
	}
}
