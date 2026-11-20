package gfx

import (
	"testing"

	"hexago/ui/geom"
)

// Тень, которая плавно едет и чуть меняет размер (переход экрана), берёт маски из кеша:
// у соседних положений один ключ, а маска обязана подходить по размеру.
func TestShadowCacheWhileMoving(t *testing.T) {
	c := NewImageCanvas(200, 120)
	for i := range 2000 {
		x := 10 + float64(i)*0.0137
		w := 56 + float64(i%97)*0.0091
		c.DrawShadow(geom.RRect{Rect: geom.Rect{Left: x, Top: 20, Right: x + w, Bottom: 76},
			Radius: geom.RadiusAll(16)}, ARGB(0x40000000), 3)
	}
}

// mix (два канала за раз) совпадает с поканальным div255 на всех значениях.
func TestMixMatchesPerChannel(t *testing.T) {
	ref := func(d, r, g, b, a uint32) uint32 {
		inv := 255 - a
		return div255(255*a+(d>>24)*inv)<<24 | div255(b*a+(d>>16&0xFF)*inv)<<16 |
			div255(g*a+(d>>8&0xFF)*inv)<<8 | div255(r*a+(d&0xFF)*inv)
	}
	for a := uint32(0); a < 255; a++ {
		for v := uint32(0); v < 256; v += 3 {
			d := v | (255-v)<<8 | (v/2)<<16 | (255-v/3)<<24
			r, g, b := (v*7)&0xFF, (v*13)&0xFF, 255-v
			if got, want := mix(d, r, g, b, a), ref(d, r, g, b, a); got != want {
				t.Fatalf("mix(%08x, %d, %d, %d, a=%d) = %08x, ждали %08x", d, r, g, b, a, got, want)
			}
		}
	}
}
