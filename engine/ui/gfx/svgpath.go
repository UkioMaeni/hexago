package gfx

import (
	"fmt"
	"math"
	"strconv"

	"hexago/ui/geom"
)

// ParseSVGPath разбирает путь в формате SVG — значение атрибута d элемента <path>:
// команды M, L, H, V, C, S, Q, T, A, Z, большими буквами (абсолютные координаты) и маленькими
// (относительные). Дуги (A) переводятся в кривые Безье. Так рисуются значки Material и любые
// свои значки, нарисованные в редакторе векторной графики.
func ParseSVGPath(d string) (*Path, error) {
	s := svgScanner{src: d}
	p := &Path{}
	var (
		cmd                 byte
		cur, start, ctrl    geom.Offset // текущая точка, начало контура, прошлая контрольная точка
		prevCubic, prevQuad bool
	)
	for {
		s.skipSeparators()
		if s.done() {
			break
		}
		if c := s.peek(); isCommand(c) {
			cmd = c
			s.pos++
		} else if cmd == 0 {
			return nil, s.errorf("путь должен начинаться с команды M")
		} else if cmd|0x20 == 'z' {
			return nil, s.errorf("после Z нужна команда, а не число")
		}
		rel := cmd >= 'a'
		base := geom.Zero
		if rel {
			base = cur
		}
		point := func() (geom.Offset, error) {
			x, err := s.number()
			if err != nil {
				return geom.Zero, err
			}
			y, err := s.number()
			if err != nil {
				return geom.Zero, err
			}
			return geom.Offset{X: x, Y: y}.Add(base), nil
		}
		isCubic, isQuad := false, false
		switch cmd | 0x20 { // в нижний регистр
		case 'm':
			pt, err := point()
			if err != nil {
				return nil, err
			}
			p.MoveTo(pt.X, pt.Y)
			cur, start = pt, pt
			// Следующие пары чисел после M — это L (после m — l).
			if rel {
				cmd = 'l'
			} else {
				cmd = 'L'
			}
		case 'l':
			pt, err := point()
			if err != nil {
				return nil, err
			}
			p.LineTo(pt.X, pt.Y)
			cur = pt
		case 'h':
			x, err := s.number()
			if err != nil {
				return nil, err
			}
			if rel {
				x += cur.X
			}
			cur.X = x
			p.LineTo(cur.X, cur.Y)
		case 'v':
			y, err := s.number()
			if err != nil {
				return nil, err
			}
			if rel {
				y += cur.Y
			}
			cur.Y = y
			p.LineTo(cur.X, cur.Y)
		case 'c':
			c1, err := point()
			if err != nil {
				return nil, err
			}
			c2, err := point()
			if err != nil {
				return nil, err
			}
			pt, err := point()
			if err != nil {
				return nil, err
			}
			p.CubicTo(c1.X, c1.Y, c2.X, c2.Y, pt.X, pt.Y)
			cur, ctrl, isCubic = pt, c2, true
		case 's':
			c1 := cur
			if prevCubic {
				c1 = cur.Add(cur.Sub(ctrl)) // отражение прошлой контрольной точки
			}
			c2, err := point()
			if err != nil {
				return nil, err
			}
			pt, err := point()
			if err != nil {
				return nil, err
			}
			p.CubicTo(c1.X, c1.Y, c2.X, c2.Y, pt.X, pt.Y)
			cur, ctrl, isCubic = pt, c2, true
		case 'q':
			c, err := point()
			if err != nil {
				return nil, err
			}
			pt, err := point()
			if err != nil {
				return nil, err
			}
			p.QuadTo(c.X, c.Y, pt.X, pt.Y)
			cur, ctrl, isQuad = pt, c, true
		case 't':
			c := cur
			if prevQuad {
				c = cur.Add(cur.Sub(ctrl))
			}
			pt, err := point()
			if err != nil {
				return nil, err
			}
			p.QuadTo(c.X, c.Y, pt.X, pt.Y)
			cur, ctrl, isQuad = pt, c, true
		case 'a':
			rx, err := s.number()
			if err != nil {
				return nil, err
			}
			ry, err := s.number()
			if err != nil {
				return nil, err
			}
			rot, err := s.number()
			if err != nil {
				return nil, err
			}
			large, err := s.flag()
			if err != nil {
				return nil, err
			}
			sweep, err := s.flag()
			if err != nil {
				return nil, err
			}
			pt, err := point()
			if err != nil {
				return nil, err
			}
			arcTo(p, cur, pt, rx, ry, rot, large, sweep)
			cur = pt
		case 'z':
			p.Close()
			cur = start
		default:
			return nil, s.errorf("неизвестная команда %q", cmd)
		}
		prevCubic, prevQuad = isCubic, isQuad
	}
	return p, nil
}

// MustParseSVGPath — как ParseSVGPath, но паникует на ошибке: для путей, вшитых в код.
func MustParseSVGPath(d string) *Path {
	p, err := ParseSVGPath(d)
	if err != nil {
		panic(err)
	}
	return p
}

func isCommand(c byte) bool {
	switch c | 0x20 {
	case 'm', 'l', 'h', 'v', 'c', 's', 'q', 't', 'a', 'z':
		return true
	}
	return false
}

// svgScanner читает числа и флаги пути. Числа могут идти без разделителей: "1.41-1.41", ".5.5".
type svgScanner struct {
	src string
	pos int
}

func (s *svgScanner) done() bool { return s.pos >= len(s.src) }
func (s *svgScanner) peek() byte { return s.src[s.pos] }

func (s *svgScanner) skipSeparators() {
	for !s.done() {
		switch s.peek() {
		case ' ', '\t', '\n', '\r', ',':
			s.pos++
		default:
			return
		}
	}
}

func (s *svgScanner) errorf(format string, args ...any) error {
	return fmt.Errorf("путь SVG, позиция %d: %s", s.pos, fmt.Sprintf(format, args...))
}

func (s *svgScanner) number() (float64, error) {
	s.skipSeparators()
	begin := s.pos
	if !s.done() && (s.peek() == '-' || s.peek() == '+') {
		s.pos++
	}
	dot, digits := false, false
	for !s.done() {
		c := s.peek()
		switch {
		case c >= '0' && c <= '9':
			digits = true
		case c == '.' && !dot:
			dot = true
		case (c == 'e' || c == 'E') && digits:
			s.pos++
			if !s.done() && (s.peek() == '-' || s.peek() == '+') {
				s.pos++
			}
			continue
		default:
			goto end
		}
		s.pos++
	}
end:
	if !digits {
		return 0, s.errorf("ожидалось число")
	}
	return strconv.ParseFloat(s.src[begin:s.pos], 64)
}

// flag читает флаг дуги — одну цифру 0 или 1 (флаги могут стоять вплотную: "011").
func (s *svgScanner) flag() (bool, error) {
	s.skipSeparators()
	if s.done() || (s.peek() != '0' && s.peek() != '1') {
		return false, s.errorf("ожидался флаг дуги 0 или 1")
	}
	f := s.peek() == '1'
	s.pos++
	return f, nil
}

// arcTo добавляет эллиптическую дугу от from до to (параметры SVG) кривыми Безье — по алгоритму
// из приложения F.6 спецификации SVG: найти центр и углы, разбить на куски не больше 90°.
func arcTo(p *Path, from, to geom.Offset, rx, ry, rotDeg float64, large, sweep bool) {
	if from == to {
		return
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx == 0 || ry == 0 {
		p.LineTo(to.X, to.Y)
		return
	}
	sinR, cosR := math.Sincos(rotDeg * math.Pi / 180)
	// Середина хорды в повёрнутых координатах.
	dx, dy := (from.X-to.X)/2, (from.Y-to.Y)/2
	x1 := cosR*dx + sinR*dy
	y1 := -sinR*dx + cosR*dy
	// Радиусы слишком малы — увеличить, чтобы дуга существовала.
	if l := x1*x1/(rx*rx) + y1*y1/(ry*ry); l > 1 {
		k := math.Sqrt(l)
		rx, ry = rx*k, ry*k
	}
	num := rx*rx*ry*ry - rx*rx*y1*y1 - ry*ry*x1*x1
	den := rx*rx*y1*y1 + ry*ry*x1*x1
	coef := math.Sqrt(math.Max(0, num/den))
	if large == sweep {
		coef = -coef
	}
	cx1, cy1 := coef*rx*y1/ry, -coef*ry*x1/rx
	cx := cosR*cx1 - sinR*cy1 + (from.X+to.X)/2
	cy := sinR*cx1 + cosR*cy1 + (from.Y+to.Y)/2

	angle := func(ux, uy, vx, vy float64) float64 {
		a := math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
		return a
	}
	ux, uy := (x1-cx1)/rx, (y1-cy1)/ry
	vx, vy := (-x1-cx1)/rx, (-y1-cy1)/ry
	theta := angle(1, 0, ux, uy)
	delta := angle(ux, uy, vx, vy)
	if !sweep && delta > 0 {
		delta -= 2 * math.Pi
	} else if sweep && delta < 0 {
		delta += 2 * math.Pi
	}

	n := int(math.Ceil(math.Abs(delta) / (math.Pi / 2)))
	step := delta / float64(n)
	k := 4.0 / 3 * math.Tan(step/4) // длина «ручек» Безье для дуги step
	point := func(t float64) (geom.Offset, geom.Offset) {
		s, c := math.Sincos(t)
		// Точка и производная на эллипсе, повёрнутом на rot.
		px, py := rx*c, ry*s
		tx, ty := -rx*s, ry*c
		return geom.Offset{X: cx + cosR*px - sinR*py, Y: cy + sinR*px + cosR*py},
			geom.Offset{X: cosR*tx - sinR*ty, Y: sinR*tx + cosR*ty}
	}
	t := theta
	a, da := point(t)
	for i := range n {
		t2 := t + step
		b, db := point(t2)
		if i == n-1 {
			b = to // последняя точка — ровно конец дуги, без накопленной погрешности
		}
		c1, c2 := a.Add(da.Scale(k)), b.Sub(db.Scale(k))
		p.CubicTo(c1.X, c1.Y, c2.X, c2.Y, b.X, b.Y)
		t, a, da = t2, b, db
	}
}
