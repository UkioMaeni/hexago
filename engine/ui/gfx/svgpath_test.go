package gfx

import (
	"math"
	"testing"

	"hexago/ui/geom"
)

func near(a, b geom.Rect) bool {
	const eps = 1e-6
	return math.Abs(a.Left-b.Left) < eps && math.Abs(a.Top-b.Top) < eps &&
		math.Abs(a.Right-b.Right) < eps && math.Abs(a.Bottom-b.Bottom) < eps
}

func TestParseSVGPath(t *testing.T) {
	cases := []struct {
		name, d string
		bounds  geom.Rect
	}{
		// Числа без разделителей и неявный L после M.
		{"слитные числа", "M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z", geom.Rect{Left: 5, Top: 5, Right: 19, Bottom: 19}},
		{"неявный L", "M1 1 5 1 5 5z", geom.Rect{Left: 1, Top: 1, Right: 5, Bottom: 5}},
		{"относительные и .5.5", "m2 2 l.5.5 h3 v-1z", geom.Rect{Left: 2, Top: 1.5, Right: 5.5, Bottom: 2.5}},
		// Две полуокружности дугами — круг радиуса 10 вокруг (12, 12).
		{"дуги", "M2 12a10 10 0 1 0 20 0a10 10 0 1 0-20 0z", geom.Rect{Left: 2, Top: 2, Right: 22, Bottom: 22}},
		// Флаги дуги вплотную: «011».
		{"флаги вплотную", "M2 12a10 10 0 0120 0z", geom.Rect{Left: 2, Top: 2, Right: 22, Bottom: 12}},
	}
	for _, c := range cases {
		p, err := ParseSVGPath(c.d)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got := p.Bounds(); !near(got, c.bounds) {
			t.Errorf("%s: границы %v, ожидали %v", c.name, got, c.bounds)
		}
	}
}

func TestParseSVGPathErrors(t *testing.T) {
	for _, d := range []string{"10 10", "M1", "M1 1 Z 5 5", "M1 1 A1 1 0 2 0 3 3", "M1 1 X2 2"} {
		if _, err := ParseSVGPath(d); err == nil {
			t.Errorf("путь %q разобран без ошибки", d)
		}
	}
}
