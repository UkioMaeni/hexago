package gfx

import (
	"math"

	"hexago/ui/geom"
)

// PaintStyle — залить фигуру или обвести её контур.
type PaintStyle uint8

const (
	Fill   PaintStyle = iota // заливка
	Stroke                   // обводка линией толщины StrokeWidth
)

// StrokeCap — как выглядят концы незамкнутой линии.
type StrokeCap uint8

const (
	ButtCap   StrokeCap = iota // обрезаны ровно по концу
	RoundCap                   // скруглены полукругом
	SquareCap                  // продлены квадратом на половину толщины
)

// Paint — чем рисовать: цвет или градиент, заливка или обводка.
type Paint struct {
	Color       Color
	Style       PaintStyle
	StrokeWidth float64
	StrokeCap   StrokeCap
	// Shader, если задан, даёт цвет каждого пикселя вместо Color (градиенты).
	Shader Shader
}

// Shader вычисляет цвет в каждой точке фигуры — так устроены градиенты.
type Shader interface {
	// at готовит функцию цвета для пикселей экрана: m переводит локальные координаты
	// фигуры в координаты экрана.
	at(m geom.Affine) func(x, y float64) Color
}

// LinearGradient — линейный градиент от точки From к точке To (в координатах фигуры).
// Stops — положения цветов от 0 до 1; если не заданы, цвета распределяются равномерно.
type LinearGradient struct {
	From, To geom.Offset
	Colors   []Color
	Stops    []float64
}

func (g LinearGradient) at(m geom.Affine) func(x, y float64) Color {
	inv, ok := m.Invert()
	d := g.To.Sub(g.From)
	length2 := d.X*d.X + d.Y*d.Y
	return func(x, y float64) Color {
		if !ok || length2 == 0 {
			return sampleStops(g.Colors, g.Stops, 0)
		}
		p := inv.Apply(geom.Offset{X: x, Y: y}).Sub(g.From)
		return sampleStops(g.Colors, g.Stops, (p.X*d.X+p.Y*d.Y)/length2)
	}
}

// RadialGradient — круговой градиент от центра (цвет 0) к окружности радиуса Radius (последний цвет).
type RadialGradient struct {
	Center geom.Offset
	Radius float64
	Colors []Color
	Stops  []float64
}

func (g RadialGradient) at(m geom.Affine) func(x, y float64) Color {
	inv, ok := m.Invert()
	return func(x, y float64) Color {
		if !ok || g.Radius <= 0 {
			return sampleStops(g.Colors, g.Stops, 0)
		}
		p := inv.Apply(geom.Offset{X: x, Y: y}).Sub(g.Center)
		return sampleStops(g.Colors, g.Stops, math.Hypot(p.X, p.Y)/g.Radius)
	}
}

// sampleStops возвращает цвет градиента в положении t (за пределами 0..1 — крайние цвета).
func sampleStops(colors []Color, stops []float64, t float64) Color {
	switch len(colors) {
	case 0:
		return Transparent
	case 1:
		return colors[0]
	}
	stop := func(i int) float64 {
		if len(stops) == len(colors) {
			return stops[i]
		}
		return float64(i) / float64(len(colors)-1)
	}
	if t <= stop(0) {
		return colors[0]
	}
	for i := 1; i < len(colors); i++ {
		if a, b := stop(i-1), stop(i); t <= b {
			if b == a {
				return colors[i]
			}
			return LerpColor(colors[i-1], colors[i], (t-a)/(b-a))
		}
	}
	return colors[len(colors)-1]
}
