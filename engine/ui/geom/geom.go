// Package geom — геометрия Hexago: точки, размеры, прямоугольники, отступы,
// выравнивание, скругления и аффинные преобразования.
//
// Все величины — float64 в логических пикселях (dp), как double во Flutter (ADR-0013).
package geom

import "math"

// Offset — точка или сдвиг на плоскости. Ось Y направлена вниз.
type Offset struct {
	X, Y float64
}

// Zero — нулевой сдвиг, начало координат.
var Zero = Offset{}

// Add складывает сдвиги.
func (o Offset) Add(p Offset) Offset { return Offset{o.X + p.X, o.Y + p.Y} }

// Sub вычитает сдвиги.
func (o Offset) Sub(p Offset) Offset { return Offset{o.X - p.X, o.Y - p.Y} }

// Scale умножает сдвиг на число.
func (o Offset) Scale(k float64) Offset { return Offset{o.X * k, o.Y * k} }

// Distance — длина сдвига.
func (o Offset) Distance() float64 { return math.Hypot(o.X, o.Y) }

// Size — ширина и высота.
type Size struct {
	Width, Height float64
}

// IsEmpty сообщает, что у размера нет площади.
func (s Size) IsEmpty() bool { return s.Width <= 0 || s.Height <= 0 }

// Contains сообщает, лежит ли точка внутри прямоугольника (0, 0, Width, Height).
func (s Size) Contains(p Offset) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < s.Width && p.Y < s.Height
}

// Rect — прямоугольник по краям: левый, верхний, правый, нижний.
type Rect struct {
	Left, Top, Right, Bottom float64
}

// RectXYWH строит прямоугольник по левому верхнему углу и размеру.
func RectXYWH(x, y, w, h float64) Rect { return Rect{x, y, x + w, y + h} }

// RectFromSize строит прямоугольник с началом в точке o.
func RectFromSize(o Offset, s Size) Rect { return RectXYWH(o.X, o.Y, s.Width, s.Height) }

// Width — ширина прямоугольника.
func (r Rect) Width() float64 { return r.Right - r.Left }

// Height — высота прямоугольника.
func (r Rect) Height() float64 { return r.Bottom - r.Top }

// Size — размер прямоугольника.
func (r Rect) Size() Size { return Size{r.Width(), r.Height()} }

// TopLeft — левый верхний угол.
func (r Rect) TopLeft() Offset { return Offset{r.Left, r.Top} }

// Center — центр прямоугольника.
func (r Rect) Center() Offset { return Offset{(r.Left + r.Right) / 2, (r.Top + r.Bottom) / 2} }

// IsEmpty сообщает, что у прямоугольника нет площади.
func (r Rect) IsEmpty() bool { return r.Right <= r.Left || r.Bottom <= r.Top }

// Translate сдвигает прямоугольник.
func (r Rect) Translate(o Offset) Rect {
	return Rect{r.Left + o.X, r.Top + o.Y, r.Right + o.X, r.Bottom + o.Y}
}

// Inflate расширяет прямоугольник на d во все стороны (d < 0 — сужает).
func (r Rect) Inflate(d float64) Rect { return Rect{r.Left - d, r.Top - d, r.Right + d, r.Bottom + d} }

// Deflate сужает прямоугольник на отступы.
func (r Rect) Deflate(e EdgeInsets) Rect {
	return Rect{r.Left + e.Left, r.Top + e.Top, r.Right - e.Right, r.Bottom - e.Bottom}
}

// Intersect — общая часть двух прямоугольников (может оказаться пустой).
func (r Rect) Intersect(o Rect) Rect {
	return Rect{max(r.Left, o.Left), max(r.Top, o.Top), min(r.Right, o.Right), min(r.Bottom, o.Bottom)}
}

// Union — наименьший прямоугольник, содержащий оба.
func (r Rect) Union(o Rect) Rect {
	if r.IsEmpty() {
		return o
	}
	if o.IsEmpty() {
		return r
	}
	return Rect{min(r.Left, o.Left), min(r.Top, o.Top), max(r.Right, o.Right), max(r.Bottom, o.Bottom)}
}

// Contains сообщает, лежит ли точка внутри.
func (r Rect) Contains(p Offset) bool {
	return p.X >= r.Left && p.X < r.Right && p.Y >= r.Top && p.Y < r.Bottom
}

// EdgeInsets — отступы с четырёх сторон, как EdgeInsets во Flutter.
type EdgeInsets struct {
	Left, Top, Right, Bottom float64
}

// All — одинаковый отступ со всех сторон: All(16).
func All(v float64) EdgeInsets { return EdgeInsets{v, v, v, v} }

// Symmetric — отступы по горизонтали (слева и справа) и по вертикали (сверху и снизу).
func Symmetric(horizontal, vertical float64) EdgeInsets {
	return EdgeInsets{horizontal, vertical, horizontal, vertical}
}

// Horizontal — сумма левого и правого отступов.
func (e EdgeInsets) Horizontal() float64 { return e.Left + e.Right }

// Vertical — сумма верхнего и нижнего отступов.
func (e EdgeInsets) Vertical() float64 { return e.Top + e.Bottom }

// TopLeft — сдвиг содержимого: левый и верхний отступы.
func (e EdgeInsets) TopLeft() Offset { return Offset{e.Left, e.Top} }

// Add складывает отступы.
func (e EdgeInsets) Add(o EdgeInsets) EdgeInsets {
	return EdgeInsets{e.Left + o.Left, e.Top + o.Top, e.Right + o.Right, e.Bottom + o.Bottom}
}

// Alignment — точка внутри прямоугольника: X и Y от -1 (лево, верх) до 1 (право, низ),
// 0 — середина. Как Alignment во Flutter.
type Alignment struct {
	X, Y float64
}

// Стандартные выравнивания.
var (
	TopLeft      = Alignment{-1, -1}
	TopCenter    = Alignment{0, -1}
	TopRight     = Alignment{1, -1}
	CenterLeft   = Alignment{-1, 0}
	Center       = Alignment{0, 0}
	CenterRight  = Alignment{1, 0}
	BottomLeft   = Alignment{-1, 1}
	BottomCenter = Alignment{0, 1}
	BottomRight  = Alignment{1, 1}
)

// Along возвращает сдвиг ребёнка, когда свободного места free:
// для Center — половина свободного места, для TopLeft — ноль.
func (a Alignment) Along(free Size) Offset {
	return Offset{(a.X + 1) / 2 * free.Width, (a.Y + 1) / 2 * free.Height}
}

// Point возвращает точку выравнивания внутри прямоугольника.
func (a Alignment) Point(r Rect) Offset {
	return Offset{r.Left + (a.X+1)/2*r.Width(), r.Top + (a.Y+1)/2*r.Height()}
}

// Radius — радиус скругления угла по горизонтали и вертикали (эллиптический угол).
type Radius struct {
	X, Y float64
}

// Circular — круглый угол радиуса r.
func Circular(r float64) Radius { return Radius{r, r} }

// BorderRadius — скругления четырёх углов.
type BorderRadius struct {
	TopLeft, TopRight, BottomRight, BottomLeft Radius
}

// RadiusAll — одинаковое круглое скругление всех углов: RadiusAll(12).
func RadiusAll(r float64) BorderRadius {
	c := Circular(r)
	return BorderRadius{c, c, c, c}
}

// IsZero сообщает, что углы не скруглены.
func (b BorderRadius) IsZero() bool { return b == BorderRadius{} }

// RRect — прямоугольник со скруглёнными углами.
type RRect struct {
	Rect
	Radius BorderRadius
}

// Clamped возвращает скругления, уменьшенные так, чтобы соседние углы не перекрывались
// (как в CSS и Skia): если сумма радиусов на стороне больше стороны, все радиусы
// пропорционально уменьшаются.
func (r RRect) Clamped() RRect {
	w, h := r.Width(), r.Height()
	b := r.Radius
	k := 1.0
	fit := func(sum, side float64) {
		if sum > side && sum > 0 {
			k = min(k, side/sum)
		}
	}
	fit(b.TopLeft.X+b.TopRight.X, w)
	fit(b.BottomLeft.X+b.BottomRight.X, w)
	fit(b.TopLeft.Y+b.BottomLeft.Y, h)
	fit(b.TopRight.Y+b.BottomRight.Y, h)
	if k < 1 {
		scale := func(r Radius) Radius { return Radius{r.X * k, r.Y * k} }
		b = BorderRadius{scale(b.TopLeft), scale(b.TopRight), scale(b.BottomRight), scale(b.BottomLeft)}
	}
	return RRect{r.Rect, b}
}

// Inflate расширяет прямоугольник со скруглениями на d (радиусы растут вместе с ним).
func (r RRect) Inflate(d float64) RRect {
	grow := func(c Radius) Radius {
		if c.X == 0 && c.Y == 0 {
			return c
		}
		return Radius{max(0, c.X+d), max(0, c.Y+d)}
	}
	b := r.Radius
	return RRect{r.Rect.Inflate(d), BorderRadius{grow(b.TopLeft), grow(b.TopRight), grow(b.BottomRight), grow(b.BottomLeft)}}
}

// Affine — аффинное преобразование: x' = A*x + C*y + E, y' = B*x + D*y + F.
// Как матрица 3×3 с последней строкой (0, 0, 1).
type Affine struct {
	A, B, C, D, E, F float64
}

// Identity — тождественное преобразование.
var Identity = Affine{A: 1, D: 1}

// Translation — сдвиг.
func Translation(dx, dy float64) Affine { return Affine{A: 1, D: 1, E: dx, F: dy} }

// Scaling — масштаб.
func Scaling(sx, sy float64) Affine { return Affine{A: sx, D: sy} }

// Rotation — поворот на угол в радианах (по часовой стрелке при оси Y вниз).
func Rotation(angle float64) Affine {
	s, c := math.Sincos(angle)
	return Affine{A: c, B: s, C: -s, D: c}
}

// Mul возвращает преобразование «сначала o, потом m».
func (m Affine) Mul(o Affine) Affine {
	return Affine{
		A: m.A*o.A + m.C*o.B,
		B: m.B*o.A + m.D*o.B,
		C: m.A*o.C + m.C*o.D,
		D: m.B*o.C + m.D*o.D,
		E: m.A*o.E + m.C*o.F + m.E,
		F: m.B*o.E + m.D*o.F + m.F,
	}
}

// Apply применяет преобразование к точке.
func (m Affine) Apply(p Offset) Offset {
	return Offset{m.A*p.X + m.C*p.Y + m.E, m.B*p.X + m.D*p.Y + m.F}
}

// Invert возвращает обратное преобразование; ok=false, если его нет (вырожденная матрица).
func (m Affine) Invert() (Affine, bool) {
	det := m.A*m.D - m.B*m.C
	if det == 0 {
		return Affine{}, false
	}
	a, b, c, d := m.D/det, -m.B/det, -m.C/det, m.A/det
	return Affine{A: a, B: b, C: c, D: d, E: -(a*m.E + c*m.F), F: -(b*m.E + d*m.F)}, true
}

// IsTranslation сообщает, что преобразование — только сдвиг (без поворота и масштаба).
func (m Affine) IsTranslation() bool { return m.A == 1 && m.B == 0 && m.C == 0 && m.D == 1 }

// MapRect возвращает прямоугольник, охватывающий преобразованный прямоугольник.
func (m Affine) MapRect(r Rect) Rect {
	p := [4]Offset{
		m.Apply(Offset{r.Left, r.Top}), m.Apply(Offset{r.Right, r.Top}),
		m.Apply(Offset{r.Right, r.Bottom}), m.Apply(Offset{r.Left, r.Bottom}),
	}
	out := Rect{p[0].X, p[0].Y, p[0].X, p[0].Y}
	for _, q := range p[1:] {
		out.Left, out.Top = min(out.Left, q.X), min(out.Top, q.Y)
		out.Right, out.Bottom = max(out.Right, q.X), max(out.Bottom, q.Y)
	}
	return out
}

// Lerp — линейная интерполяция между a и b: t=0 → a, t=1 → b.
func Lerp(a, b, t float64) float64 { return a + (b-a)*t }
