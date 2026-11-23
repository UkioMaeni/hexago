package gfx

import (
	"image"
	"testing"

	"hexago/ui/geom"
)

// Копия под размер на экране считается один раз; хранятся две последние.
func TestBitmapKeepsRenditions(t *testing.T) {
	b := NewBitmap(image.NewNRGBA(image.Rect(0, 0, 100, 100)))
	c := NewImageCanvas(100, 100)
	draw := func(size float64) { c.DrawBitmap(b, geom.Rect{}, geom.RectXYWH(0, 0, size, size), 1) }
	draw(50)
	first := &b.scaled[0].pix[0]
	draw(50)
	if len(b.scaled) != 1 || &b.scaled[0].pix[0] != first {
		t.Fatal("копию того же размера посчитали заново")
	}
	draw(100) // свой размер — без копии
	if len(b.scaled) != 1 {
		t.Fatal("для рисования в свой размер сделана копия")
	}
	draw(30)
	draw(40)
	if len(b.scaled) != 2 || b.scaled[0].w != 30 || b.scaled[1].w != 40 {
		t.Fatalf("хранятся копии %d×…, нужны две последние", len(b.scaled))
	}
	draw(30)
	if b.scaled[1].w != 30 {
		t.Error("использованная копия не стала свежей")
	}
}
