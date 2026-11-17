// Package rendering — render-дерево: лэйаут, отрисовка, попадание касаний.
//
// Правило лэйаута, как во Flutter: ограничения идут вниз, размеры — вверх,
// позицию ребёнка выставляет родитель. Родитель передаёт ребёнку Constraints
// (диапазон допустимых ширины и высоты), ребёнок выбирает свой размер в этом диапазоне,
// родитель решает, где ребёнок будет лежать.
//
// Go без наследования (ADR-0003): общие данные любого бокса лежат во встроенной структуре
// Node, а общие алгоритмы (Layout, PaintChild, HitTest) — функции пакета, принимающие Box.
package rendering

import (
	"fmt"
	"math"

	"hexago/ui/geom"
)

// Inf — «без ограничения» по ширине или высоте.
var Inf = math.Inf(1)

// Constraints — допустимые размеры бокса, как BoxConstraints во Flutter.
type Constraints struct {
	MinWidth, MaxWidth   float64
	MinHeight, MaxHeight float64
}

// Tight — ровно такой размер.
func Tight(s geom.Size) Constraints { return Constraints{s.Width, s.Width, s.Height, s.Height} }

// Loose — любой размер от нуля до s.
func Loose(s geom.Size) Constraints { return Constraints{0, s.Width, 0, s.Height} }

// Expand — занять всё место (ширина и высота — бесконечность, их сожмёт родитель).
func Expand() Constraints { return Constraints{Inf, Inf, Inf, Inf} }

// TightFor — ровно заданные ширина и/или высота; nil — без ограничения по этой оси.
func TightFor(width, height *float64) Constraints {
	c := Constraints{0, Inf, 0, Inf}
	if width != nil {
		c.MinWidth, c.MaxWidth = *width, *width
	}
	if height != nil {
		c.MinHeight, c.MaxHeight = *height, *height
	}
	return c
}

// Constrain возвращает ближайший к s размер, который допускают ограничения.
func (c Constraints) Constrain(s geom.Size) geom.Size {
	return geom.Size{Width: clampF(s.Width, c.MinWidth, c.MaxWidth), Height: clampF(s.Height, c.MinHeight, c.MaxHeight)}
}

// Biggest — наибольший допустимый размер (бесконечность сжимается до минимума).
func (c Constraints) Biggest() geom.Size {
	w, h := c.MaxWidth, c.MaxHeight
	if math.IsInf(w, 1) {
		w = c.MinWidth
	}
	if math.IsInf(h, 1) {
		h = c.MinHeight
	}
	return geom.Size{Width: w, Height: h}
}

// Smallest — наименьший допустимый размер.
func (c Constraints) Smallest() geom.Size { return geom.Size{Width: c.MinWidth, Height: c.MinHeight} }

// Loosen убирает минимумы: ребёнок может быть меньше.
func (c Constraints) Loosen() Constraints { return Constraints{0, c.MaxWidth, 0, c.MaxHeight} }

// Deflate уменьшает ограничения на отступы (для ребёнка внутри Padding).
func (c Constraints) Deflate(e geom.EdgeInsets) Constraints {
	h, v := e.Horizontal(), e.Vertical()
	minW := max(0, c.MinWidth-h)
	minH := max(0, c.MinHeight-v)
	return Constraints{minW, max(minW, c.MaxWidth-h), minH, max(minH, c.MaxHeight-v)}
}

// Enforce подгоняет ограничения c под ограничения родителя outer (outer важнее).
func (c Constraints) Enforce(outer Constraints) Constraints {
	return Constraints{
		MinWidth:  clampF(c.MinWidth, outer.MinWidth, outer.MaxWidth),
		MaxWidth:  clampF(c.MaxWidth, outer.MinWidth, outer.MaxWidth),
		MinHeight: clampF(c.MinHeight, outer.MinHeight, outer.MaxHeight),
		MaxHeight: clampF(c.MaxHeight, outer.MinHeight, outer.MaxHeight),
	}
}

// IsTight сообщает, что размер задан точно.
func (c Constraints) IsTight() bool { return c.MinWidth >= c.MaxWidth && c.MinHeight >= c.MaxHeight }

// HasBoundedWidth сообщает, что ширина ограничена сверху.
func (c Constraints) HasBoundedWidth() bool { return !math.IsInf(c.MaxWidth, 1) }

// HasBoundedHeight сообщает, что высота ограничена сверху.
func (c Constraints) HasBoundedHeight() bool { return !math.IsInf(c.MaxHeight, 1) }

func (c Constraints) String() string {
	return fmt.Sprintf("w %g..%g, h %g..%g", c.MinWidth, c.MaxWidth, c.MinHeight, c.MaxHeight)
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		v = lo
	}
	if v > hi {
		v = hi
	}
	return v
}
