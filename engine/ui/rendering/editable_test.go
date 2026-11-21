package rendering

import (
	"testing"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/text"
)

// Набираемое слово подчёркнуто: картинки с ним и без него отличаются только тонкой полосой
// у нижнего края строки под этим словом.
func TestEditableBoxUnderlinesComposingWord(t *testing.T) {
	draw := func(composing bool) *gfx.Canvas {
		b := &EditableBox{Text: "привет мир", Style: text.Style{FontSize: 20, Color: gfx.Black}}
		if composing {
			b.ComposeStart, b.ComposeEnd = 0, 6
		}
		Layout(b, Loose(geom.Size{Width: 300, Height: 100}))
		c := gfx.NewImageCanvas(300, 100)
		c.Clear(gfx.White)
		b.Paint(&PaintContext{Canvas: c}, geom.Zero)
		return c
	}
	plain, under := draw(false).ToNRGBA(), draw(true).ToNRGBA()
	b := &EditableBox{Text: "привет мир", Style: text.Style{FontSize: 20}}
	Layout(b, Loose(geom.Size{Width: 300, Height: 100}))
	word := b.CaretRect(6).Left // где кончается «привет»
	line := b.CaretRect(0).Bottom
	minY, maxY, maxX := 1000, -1, -1
	for y := 0; y < 100; y++ {
		for x := 0; x < 300; x++ {
			if plain.NRGBAAt(x, y) != under.NRGBAAt(x, y) {
				minY, maxY, maxX = min(minY, y), max(maxY, y), max(maxX, x)
			}
		}
	}
	if maxY < 0 {
		t.Fatal("подчёркивания нет")
	}
	if float64(minY) < line-3 || float64(maxY) > line || float64(maxX) > word+1 {
		t.Errorf("подчёркивание в y %d…%d, x до %d; ждали у низа строки (%.1f) под словом (до %.1f)", minY, maxY, maxX, line, word)
	}
}
