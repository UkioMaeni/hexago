package gfx

import (
	"math"

	"hexago/ui/geom"
)

// stroke превращает линию толщины paint.StrokeWidth в заливаемый контур.
//
// Ломаная (кривые предварительно разбиты на отрезки) обводится так: каждый отрезок —
// прямоугольник толщиной в линию, на изломах — круги (скруглённые соединения), на концах —
// круги или квадраты по StrokeCap. Все части обходятся в одну сторону, поэтому их
// перекрытия при заливке не дают дырок.
func (c *Canvas) stroke(p *Path, paint Paint) *Path {
	scale := max(c.scaleFactor(), 1e-6)
	w := paint.StrokeWidth
	if w <= 0 {
		w = 1 / scale // «волосяная» линия — один пиксель экрана
	}
	hw := w / 2
	polys, closed := p.flatten(geom.Identity, 0.25/scale)

	var out Path
	for i, pts := range polys {
		pts = dedupe(pts)
		if len(pts) < 2 {
			if len(pts) == 1 && paint.StrokeCap == RoundCap {
				out.AddCircle(pts[0], hw)
			}
			continue
		}
		isClosed := closed[i]
		if isClosed && pts[0] == pts[len(pts)-1] {
			pts = pts[:len(pts)-1]
		}
		n := len(pts)
		segments := n - 1
		if isClosed {
			segments = n
		}

		for s := range segments {
			a, b := pts[s], pts[(s+1)%n]
			if !isClosed && paint.StrokeCap == SquareCap {
				// Квадратный конец — продлить крайние отрезки на половину толщины.
				d := b.Sub(a).Scale(hw / b.Sub(a).Distance())
				if s == 0 {
					a = a.Sub(d)
				}
				if s == segments-1 {
					b = b.Add(d)
				}
			}
			addSegmentQuad(&out, a, b, hw)
		}

		// Соединения: круг на изломе, если без него была бы заметная щель.
		for v := range n {
			if !isClosed && (v == 0 || v == n-1) {
				continue
			}
			prev, cur, next := pts[(v-1+n)%n], pts[v], pts[(v+1)%n]
			d1, d2 := cur.Sub(prev), next.Sub(cur)
			angle := math.Abs(math.Atan2(d1.X*d2.Y-d1.Y*d2.X, d1.X*d2.X+d1.Y*d2.Y))
			if hw*scale*angle > 0.1 {
				out.AddCircle(cur, hw)
			}
		}
		if !isClosed && paint.StrokeCap == RoundCap {
			out.AddCircle(pts[0], hw)
			out.AddCircle(pts[n-1], hw)
		}
	}
	return &out
}

// addSegmentQuad добавляет прямоугольник толщиной 2·hw вдоль отрезка ab,
// всегда обходя его в одну сторону (как AddRect без reverse).
func addSegmentQuad(p *Path, a, b geom.Offset, hw float64) {
	d := b.Sub(a)
	length := d.Distance()
	if length == 0 {
		return
	}
	nrm := geom.Offset{X: -d.Y / length * hw, Y: d.X / length * hw}
	q := [4]geom.Offset{a.Add(nrm), b.Add(nrm), b.Sub(nrm), a.Sub(nrm)}
	// Знак площади (формула шнурования) показывает направление обхода.
	area := 0.0
	for i := range q {
		j := (i + 1) % 4
		area += q[i].X*q[j].Y - q[j].X*q[i].Y
	}
	if area < 0 {
		q[1], q[3] = q[3], q[1]
	}
	p.MoveTo(q[0].X, q[0].Y)
	for _, v := range q[1:] {
		p.LineTo(v.X, v.Y)
	}
	p.Close()
}

// dedupe убирает подряд идущие совпадающие точки.
func dedupe(pts []geom.Offset) []geom.Offset {
	out := pts[:0:0]
	for i, q := range pts {
		if i == 0 || q != pts[i-1] {
			out = append(out, q)
		}
	}
	return out
}
