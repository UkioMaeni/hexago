package gfx_test

import (
	"image"
	"math"
	"testing"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/hexagotest"
)

// drawScene рисует все примитивы холста — по одному эталону видно, что ничего не сломалось.
func drawScene(c *gfx.Canvas) {
	c.Clear(gfx.White)
	fill := func(col gfx.Color) gfx.Paint { return gfx.Paint{Color: col} }

	// Прямоугольник, скруглённый прямоугольник, круг.
	c.DrawRect(geom.RectXYWH(20, 20, 100, 60), fill(gfx.Blue))
	c.DrawRRect(geom.RRect{Rect: geom.RectXYWH(140, 20, 100, 60), Radius: geom.RadiusAll(16)}, fill(gfx.Green))
	c.DrawCircle(geom.Offset{X: 300, Y: 50}, 30, fill(gfx.Red))

	// Обводки: рамка со скруглением, линия с круглыми концами, дуга.
	c.DrawRRect(geom.RRect{Rect: geom.RectXYWH(20, 100, 100, 60), Radius: geom.RadiusAll(12)},
		gfx.Paint{Color: gfx.Purple, Style: gfx.Stroke, StrokeWidth: 4})
	c.DrawLine(geom.Offset{X: 140, Y: 160}, geom.Offset{X: 240, Y: 100},
		gfx.Paint{Color: gfx.Orange, StrokeWidth: 6, StrokeCap: gfx.RoundCap})
	var arc gfx.Path
	arc.AddArc(geom.Offset{X: 300, Y: 130}, 25, -math.Pi/2, 1.5*math.Pi)
	c.DrawPath(&arc, gfx.Paint{Color: gfx.Teal, Style: gfx.Stroke, StrokeWidth: 5})

	// Градиент.
	c.DrawRRect(geom.RRect{Rect: geom.RectXYWH(20, 180, 100, 60), Radius: geom.RadiusAll(8)}, gfx.Paint{
		Shader: gfx.LinearGradient{From: geom.Offset{X: 20, Y: 0}, To: geom.Offset{X: 120, Y: 0},
			Colors: []gfx.Color{gfx.Pink, gfx.Indigo}},
	})

	// Карточка с тенью.
	card := geom.RRect{Rect: geom.RectXYWH(140, 185, 100, 55), Radius: geom.RadiusAll(10)}
	shadow := card
	shadow.Rect = shadow.Rect.Translate(geom.Offset{X: 0, Y: 4})
	c.DrawShadow(shadow, gfx.Black.WithOpacity(0.35), 6)
	c.DrawRRect(card, fill(gfx.White))

	// Обрезка скруглённым прямоугольником, внутри повёрнутый квадрат.
	c.Save()
	c.ClipRRect(geom.RRect{Rect: geom.RectXYWH(265, 180, 70, 70), Radius: geom.RadiusAll(20)})
	c.DrawRect(geom.RectXYWH(265, 180, 70, 70), fill(gfx.Amber))
	c.Translate(300, 215)
	c.Transform(geom.Rotation(math.Pi / 4))
	c.DrawRect(geom.RectXYWH(-20, -20, 40, 40), fill(gfx.Brown))
	c.Restore()

	// Полупрозрачные пересекающиеся круги.
	c.Save()
	c.MultiplyOpacity(0.5)
	c.DrawCircle(geom.Offset{X: 360, Y: 260}, 25, fill(gfx.Blue))
	c.DrawCircle(geom.Offset{X: 380, Y: 270}, 25, fill(gfx.Red))
	c.Restore()
}

func TestCanvasScene(t *testing.T) {
	c := gfx.NewImageCanvas(420, 300)
	drawScene(c)
	hexagotest.Golden(t, "scene", c.ToNRGBA())
}

// TestCanvasScene2x — та же сцена при плотности экрана 2: всё вдвое крупнее и так же чётко.
func TestCanvasScene2x(t *testing.T) {
	c := gfx.NewImageCanvas(840, 600)
	c.Scale(2, 2)
	drawScene(c)
	hexagotest.Golden(t, "scene_2x", c.ToNRGBA())
}

func TestDrawImageScaled(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	copy(src.Pix, []uint8{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 255, 255, 0, 255})
	c := gfx.NewImageCanvas(64, 64)
	c.Clear(gfx.White)
	c.DrawImage(src, geom.RectXYWH(8, 8, 48, 48), 1)
	hexagotest.Golden(t, "image_scaled", c.ToNRGBA())
}

func TestColorFromHex(t *testing.T) {
	cases := map[string]gfx.Color{
		"#111111":   {R: 0x11, G: 0x11, B: 0x11, A: 255},
		"00ADD8":    {R: 0x00, G: 0xAD, B: 0xD8, A: 255},
		"#80FF0000": {R: 255, A: 0x80},
		"#F80":      {R: 0xFF, G: 0x88, A: 255},
	}
	for in, want := range cases {
		if got := gfx.ColorFromHex(in); got != want {
			t.Errorf("ColorFromHex(%q) = %v, ожидали %v", in, got, want)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("ColorFromHex(\"#zz\") должна паниковать")
		}
	}()
	gfx.ColorFromHex("#zz")
}

func TestPixelFormat(t *testing.T) {
	// Android ждёт байты R, G, B, A подряд — число 0xAABBGGRR.
	if p := gfx.ColorFromHex("#00ADD8").Pixel(); p != 0xFFD8AD00 {
		t.Errorf("Pixel() = %#x, ожидали 0xffd8ad00", p)
	}
}
