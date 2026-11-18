package animation

import (
	"math"
	"time"
)

// Simulation — движение во времени: положение X и скорость DX через t секунд от начала.
// Контроллер ведёт по ней своё значение, пока Done не вернёт true.
type Simulation interface {
	X(t float64) float64
	DX(t float64) float64
	Done(t float64) bool
}

// tween — движение от From к To за Duration по кривой Curve.
type tween struct {
	from, to float64
	duration float64 // секунды
	curve    Curve
}

func newTween(from, to float64, d time.Duration, curve Curve) *tween {
	if curve == nil {
		curve = Linear
	}
	return &tween{from: from, to: to, duration: d.Seconds(), curve: curve}
}

func (s *tween) progress(t float64) float64 {
	if s.duration <= 0 {
		return 1
	}
	return clamp(t/s.duration, 0, 1)
}

func (s *tween) X(t float64) float64 { return Lerp(s.from, s.to, s.curve.Transform(s.progress(t))) }

func (s *tween) DX(t float64) float64 {
	const h = 1e-3
	return (s.X(t+h) - s.X(t-h)) / (2 * h)
}

func (s *tween) Done(t float64) bool { return t >= s.duration }

// repeat — бесконечное движение туда (и, если reverse, обратно) за period.
type repeat struct {
	from, to float64
	period   float64
	reverse  bool
	curve    Curve
}

func (s *repeat) X(t float64) float64 {
	if s.period <= 0 {
		return s.to
	}
	cycles := t / s.period
	f := cycles - math.Floor(cycles)
	if s.reverse && int(math.Floor(cycles))%2 == 1 {
		f = 1 - f
	}
	return Lerp(s.from, s.to, s.curve.Transform(f))
}

func (s *repeat) DX(t float64) float64 { return 0 }
func (s *repeat) Done(float64) bool    { return false }

// Friction — движение с трением: скорость каждую секунду умножается на Drag (0 < Drag < 1).
// Так замедляется брошенный список. Останавливается, когда скорость меньше 1 dp/с.
type Friction struct {
	Drag     float64 // доля скорости, остающаяся через секунду
	Position float64 // начальное положение
	Velocity float64 // начальная скорость, в единицах положения в секунду
}

// ScrollFriction — трение броска списка (как FrictionSimulation в прокрутке Flutter).
const ScrollFriction = 0.135

func (s Friction) X(t float64) float64 {
	k := math.Log(s.Drag)
	return s.Position + s.Velocity*(math.Pow(s.Drag, t)-1)/k
}

func (s Friction) DX(t float64) float64 { return s.Velocity * math.Pow(s.Drag, t) }

func (s Friction) Done(t float64) bool { return math.Abs(s.DX(t)) < 1 }

// FinalX — где движение остановится.
func (s Friction) FinalX() float64 { return s.Position - s.Velocity/math.Log(s.Drag) }

// Clamped — движение sim, остановленное на краях [Min, Max]: бросок списка упирается в начало или конец.
type Clamped struct {
	Sim      Simulation
	Min, Max float64
}

func (s Clamped) X(t float64) float64  { return clamp(s.Sim.X(t), s.Min, s.Max) }
func (s Clamped) DX(t float64) float64 { return s.Sim.DX(t) }

func (s Clamped) Done(t float64) bool {
	x := s.Sim.X(t)
	return s.Sim.Done(t) || x <= s.Min || x >= s.Max
}

// TweenSimulation — движение от from к to за d по кривой curve, как Simulation: так прокрутка плавно
// едет к нужному месту (ScrollController.AnimateTo, доводка страницы PageView).
func TweenSimulation(from, to float64, d time.Duration, curve Curve) Simulation {
	return newTween(from, to, d, curve)
}
