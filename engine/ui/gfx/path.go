package gfx

import (
	"math"

	"hexago/ui/geom"
)

// pathOp — вид команды пути.
type pathOp uint8

const (
	opMove pathOp = iota
	opLine
	opQuad
	opCubic
	opClose
)

// points — сколько точек у команды.
func (op pathOp) points() int {
	switch op {
	case opMove, opLine:
		return 1
	case opQuad:
		return 2
	case opCubic:
		return 3
	}
	return 0
}

// segment — одна команда пути и её точки (в локальных координатах).
type segment struct {
	op  pathOp
	pts [3]geom.Offset
}

// Path — контур из отрезков и кривых Безье, как Path во Flutter.
// Внутренние контуры, идущие в обратную сторону, становятся дырками.
type Path struct {
	segs []segment
}

// MoveTo начинает новый контур в точке.
func (p *Path) MoveTo(x, y float64) { p.add(opMove, geom.Offset{X: x, Y: y}) }

// LineTo — отрезок до точки.
func (p *Path) LineTo(x, y float64) { p.add(opLine, geom.Offset{X: x, Y: y}) }

// QuadTo — квадратичная кривая Безье с контрольной точкой (cx, cy).
func (p *Path) QuadTo(cx, cy, x, y float64) {
	p.add(opQuad, geom.Offset{X: cx, Y: cy}, geom.Offset{X: x, Y: y})
}

// CubicTo — кубическая кривая Безье с двумя контрольными точками.
func (p *Path) CubicTo(c1x, c1y, c2x, c2y, x, y float64) {
	p.add(opCubic, geom.Offset{X: c1x, Y: c1y}, geom.Offset{X: c2x, Y: c2y}, geom.Offset{X: x, Y: y})
}

// Close замыкает текущий контур.
func (p *Path) Close() { p.segs = append(p.segs, segment{op: opClose}) }

// IsEmpty сообщает, что в пути нет команд.
func (p *Path) IsEmpty() bool { return len(p.segs) == 0 }

func (p *Path) add(op pathOp, pts ...geom.Offset) {
	s := segment{op: op}
	copy(s.pts[:], pts)
	p.segs = append(p.segs, s)
}

// AddRect добавляет прямоугольник. reverse — обход против часовой стрелки (для дырок).
func (p *Path) AddRect(r geom.Rect, reverse bool) {
	pts := []geom.Offset{{X: r.Left, Y: r.Top}, {X: r.Right, Y: r.Top}, {X: r.Right, Y: r.Bottom}, {X: r.Left, Y: r.Bottom}}
	if reverse {
		pts[1], pts[3] = pts[3], pts[1]
	}
	p.MoveTo(pts[0].X, pts[0].Y)
	for _, q := range pts[1:] {
		p.LineTo(q.X, q.Y)
	}
	p.Close()
}

// kappa — коэффициент, с которым кубическая кривая Безье точно приближает четверть окружности.
const kappa = 0.5522847498307936

// AddRRect добавляет прямоугольник со скруглёнными углами. reverse — обход против часовой стрелки.
func (p *Path) AddRRect(rr geom.RRect, reverse bool) {
	rr = rr.Clamped()
	if rr.Radius.IsZero() {
		p.AddRect(rr.Rect, reverse)
		return
	}
	r, b := rr.Rect, rr.Radius
	// Обход по часовой стрелке (ось Y вниз): верх → правый край → низ → левый край.
	// Каждый угол — четверть эллипса из одной кубической кривой.
	type corner struct {
		start, c1, c2, end geom.Offset
	}
	tl, tr, br, bl := b.TopLeft, b.TopRight, b.BottomRight, b.BottomLeft
	corners := []corner{
		{ // правый верхний
			geom.Offset{X: r.Right - tr.X, Y: r.Top},
			geom.Offset{X: r.Right - tr.X + tr.X*kappa, Y: r.Top},
			geom.Offset{X: r.Right, Y: r.Top + tr.Y - tr.Y*kappa},
			geom.Offset{X: r.Right, Y: r.Top + tr.Y},
		},
		{ // правый нижний
			geom.Offset{X: r.Right, Y: r.Bottom - br.Y},
			geom.Offset{X: r.Right, Y: r.Bottom - br.Y + br.Y*kappa},
			geom.Offset{X: r.Right - br.X + br.X*kappa, Y: r.Bottom},
			geom.Offset{X: r.Right - br.X, Y: r.Bottom},
		},
		{ // левый нижний
			geom.Offset{X: r.Left + bl.X, Y: r.Bottom},
			geom.Offset{X: r.Left + bl.X - bl.X*kappa, Y: r.Bottom},
			geom.Offset{X: r.Left, Y: r.Bottom - bl.Y + bl.Y*kappa},
			geom.Offset{X: r.Left, Y: r.Bottom - bl.Y},
		},
		{ // левый верхний
			geom.Offset{X: r.Left, Y: r.Top + tl.Y},
			geom.Offset{X: r.Left, Y: r.Top + tl.Y - tl.Y*kappa},
			geom.Offset{X: r.Left + tl.X - tl.X*kappa, Y: r.Top},
			geom.Offset{X: r.Left + tl.X, Y: r.Top},
		},
	}
	if !reverse {
		p.MoveTo(corners[3].end.X, corners[3].end.Y)
		for _, c := range corners {
			p.LineTo(c.start.X, c.start.Y)
			p.CubicTo(c.c1.X, c.c1.Y, c.c2.X, c.c2.Y, c.end.X, c.end.Y)
		}
		p.Close()
		return
	}
	// Тот же контур в обратную сторону: углы в обратном порядке, кривые задом наперёд.
	p.MoveTo(corners[0].start.X, corners[0].start.Y)
	for i := 3; i >= 0; i-- {
		c := corners[i]
		p.LineTo(c.end.X, c.end.Y)
		p.CubicTo(c.c2.X, c.c2.Y, c.c1.X, c.c1.Y, c.start.X, c.start.Y)
	}
	p.Close()
}

// AddOval добавляет эллипс, вписанный в прямоугольник.
func (p *Path) AddOval(r geom.Rect, reverse bool) {
	rx, ry := r.Width()/2, r.Height()/2
	p.AddRRect(geom.RRect{Rect: r, Radius: geom.BorderRadius{
		TopLeft: geom.Radius{X: rx, Y: ry}, TopRight: geom.Radius{X: rx, Y: ry},
		BottomRight: geom.Radius{X: rx, Y: ry}, BottomLeft: geom.Radius{X: rx, Y: ry},
	}}, reverse)
}

// AddCircle добавляет окружность.
func (p *Path) AddCircle(c geom.Offset, radius float64) {
	p.AddOval(geom.Rect{Left: c.X - radius, Top: c.Y - radius, Right: c.X + radius, Bottom: c.Y + radius}, false)
}

// AddArc добавляет дугу окружности как открытый контур: от угла start на угол sweep (в радианах,
// по часовой стрелке; 0 — направление вправо). Нужна для обводки, например индикатора загрузки.
func (p *Path) AddArc(c geom.Offset, radius, start, sweep float64) {
	steps := int(math.Ceil(math.Abs(sweep) / (math.Pi / 2)))
	if steps == 0 {
		return
	}
	step := sweep / float64(steps)
	// Длина касательных для дуги угла step: 4/3 · tan(step/4) · r.
	k := 4.0 / 3.0 * math.Tan(step/4) * radius
	a := start
	sin, cos := math.Sincos(a)
	p.MoveTo(c.X+radius*cos, c.Y+radius*sin)
	for range steps {
		b := a + step
		sinB, cosB := math.Sincos(b)
		p.CubicTo(
			c.X+radius*cos-k*sin, c.Y+radius*sin+k*cos,
			c.X+radius*cosB+k*sinB, c.Y+radius*sinB-k*cosB,
			c.X+radius*cosB, c.Y+radius*sinB,
		)
		a, sin, cos = b, sinB, cosB
	}
}

// Bounds — прямоугольник, охватывающий все точки пути (с контрольными точками кривых).
func (p *Path) Bounds() geom.Rect {
	first := true
	var r geom.Rect
	for _, s := range p.segs {
		for _, q := range s.pts[:s.op.points()] {
			if first {
				r = geom.Rect{Left: q.X, Top: q.Y, Right: q.X, Bottom: q.Y}
				first = false
				continue
			}
			r.Left, r.Top = min(r.Left, q.X), min(r.Top, q.Y)
			r.Right, r.Bottom = max(r.Right, q.X), max(r.Bottom, q.Y)
		}
	}
	return r
}

// flatten превращает путь в ломаные: кривые заменяются отрезками.
// Каждая ломаная — отдельный контур; closed — замкнут ли он.
func (p *Path) flatten(m geom.Affine, tolerance float64) (polys [][]geom.Offset, closed []bool) {
	var cur []geom.Offset
	finish := func(isClosed bool) {
		if len(cur) > 1 {
			polys = append(polys, cur)
			closed = append(closed, isClosed)
		}
		cur = nil
	}
	for _, s := range p.segs {
		switch s.op {
		case opMove:
			finish(false)
			cur = []geom.Offset{m.Apply(s.pts[0])}
		case opLine:
			cur = append(cur, m.Apply(s.pts[0]))
		case opQuad, opCubic:
			if len(cur) == 0 {
				continue
			}
			start := cur[len(cur)-1]
			c1 := m.Apply(s.pts[0])
			c2, end := c1, m.Apply(s.pts[1])
			if s.op == opCubic {
				c2, end = m.Apply(s.pts[1]), m.Apply(s.pts[2])
			}
			// Число отрезков — по длине контрольной ломаной.
			length := start.Sub(c1).Distance() + c1.Sub(c2).Distance() + c2.Sub(end).Distance()
			n := max(2, int(math.Ceil(math.Sqrt(length/tolerance))))
			for i := 1; i <= n; i++ {
				t := float64(i) / float64(n)
				u := 1 - t
				var q geom.Offset
				if s.op == opQuad {
					q = geom.Offset{
						X: u*u*start.X + 2*u*t*c1.X + t*t*end.X,
						Y: u*u*start.Y + 2*u*t*c1.Y + t*t*end.Y,
					}
				} else {
					q = geom.Offset{
						X: u*u*u*start.X + 3*u*u*t*c1.X + 3*u*t*t*c2.X + t*t*t*end.X,
						Y: u*u*u*start.Y + 3*u*u*t*c1.Y + 3*u*t*t*c2.Y + t*t*t*end.Y,
					}
				}
				cur = append(cur, q)
			}
		case opClose:
			first := []geom.Offset(nil)
			if len(cur) > 0 {
				first = cur[:1]
			}
			finish(true)
			if first != nil {
				cur = []geom.Offset{first[0]} // следующий контур продолжается из начала
			}
		}
	}
	finish(false)
	return polys, closed
}
