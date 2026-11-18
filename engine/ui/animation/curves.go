// Package animation — анимации, как animation-библиотека Flutter: кривые (Curve), контроллер,
// который ведёт значение во времени (Controller), и физика движения (Simulation) — бросок
// списка с замедлением.
//
// Время анимациям даёт планировщик кадров (Scheduler) — его реализует widgets.Binding: каждый кадр
// активные анимации получают время кадра. Пакет не знает о виджетах; привязать контроллер к виджету —
// widgets.NewAnimation.
package animation

import "math"

// Curve — кривая скорости: переводит долю времени t (0…1) в долю пути (0…1, может выходить
// за края у «пружинящих» кривых).
type Curve interface {
	Transform(t float64) float64
}

// CurveFunc — кривая из функции.
type CurveFunc func(t float64) float64

func (f CurveFunc) Transform(t float64) float64 { return f(t) }

// Cubic — кубическая кривая Безье через точки (0,0), (X1,Y1), (X2,Y2), (1,1), как cubic-bezier в CSS.
type Cubic struct{ X1, Y1, X2, Y2 float64 }

func (c Cubic) Transform(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	// Найти параметр u, при котором x(u) = t (x(u) монотонна), — делением пополам, затем y(u).
	lo, hi := 0.0, 1.0
	for range 40 {
		mid := (lo + hi) / 2
		if bezier(c.X1, c.X2, mid) < t {
			lo = mid
		} else {
			hi = mid
		}
	}
	return bezier(c.Y1, c.Y2, (lo+hi)/2)
}

// bezier — координата кубической кривой с концами 0 и 1 и контрольными a, b в точке u.
func bezier(a, b, u float64) float64 {
	v := 1 - u
	return 3*v*v*u*a + 3*v*u*u*b + u*u*u
}

// Стандартные кривые (как Curves во Flutter) и кривые движения Material 3.
var (
	Linear Curve = CurveFunc(func(t float64) float64 { return t })

	Ease          Curve = Cubic{0.25, 0.1, 0.25, 1}
	EaseIn        Curve = Cubic{0.42, 0, 1, 1}
	EaseOut       Curve = Cubic{0, 0, 0.58, 1}
	EaseInOut     Curve = Cubic{0.42, 0, 0.58, 1}
	FastOutSlowIn Curve = Cubic{0.4, 0, 0.2, 1}
	// Decelerate — быстрое начало, плавная остановка: 1 − (1 − t)².
	Decelerate Curve = CurveFunc(func(t float64) float64 { return 1 - (1-t)*(1-t) })

	// Кривые движения Material 3 (motion easing tokens).
	Standard             Curve = Cubic{0.2, 0, 0, 1}
	StandardDecelerate   Curve = Cubic{0, 0, 0, 1}
	StandardAccelerate   Curve = Cubic{0.3, 0, 1, 1}
	EmphasizedDecelerate Curve = Cubic{0.05, 0.7, 0.1, 1}
	EmphasizedAccelerate Curve = Cubic{0.3, 0, 0.8, 0.15}
)

// Lerp — линейная интерполяция чисел: t=0 → a, t=1 → b.
func Lerp(a, b, t float64) float64 { return a + (b-a)*t }

// clamp ограничивает x отрезком [lo, hi].
func clamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, x)) }
