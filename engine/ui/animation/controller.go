package animation

import "time"

// Scheduler — планировщик кадров: пока анимация активна, каждый кадр вызывает её Tick со временем
// кадра. Его реализует widgets.Binding (на Android время кадра — от развёртки экрана).
type Scheduler interface {
	Animate(t Ticker)
}

// Ticker — то, что движется каждый кадр. Tick возвращает false, когда движение закончилось.
type Ticker interface {
	Tick(now time.Duration) bool
}

// Status — где анимация.
type Status uint8

const (
	Dismissed Status = iota // стоит в начале (Lower)
	Forward                 // идёт к концу
	Reverse                 // идёт к началу
	Completed               // стоит в конце (Upper)
)

// Controller ведёт число от Lower до Upper (по умолчанию 0…1) во времени, как AnimationController
// во Flutter: Forward, Reverse, AnimateTo, Repeat, AnimateWith (физика). Каждый кадр, пока значение
// меняется, вызываются слушатели (AddListener) — обычно они перестраивают виджет.
//
// В виджете контроллер создают через widgets.NewAnimation(ctx, длительность): он сам перестраивает
// виджет и останавливается, когда виджет убран из дерева.
type Controller struct {
	Duration        time.Duration
	ReverseDuration time.Duration // 0 — как Duration
	Lower, Upper    float64

	scheduler Scheduler
	value     float64
	status    Status
	sim       Simulation
	direction Status // Forward или Reverse — куда идёт текущее движение
	start     time.Duration
	started   bool // первый кадр движения уже был (start задан)
	active    bool // зарегистрирован у планировщика
	disposed  bool

	listeners       []func()
	statusListeners []func(Status)
}

// NewController создаёт контроллер со значением Lower = 0, Upper = 1.
func NewController(s Scheduler, d time.Duration) *Controller {
	return &Controller{scheduler: s, Duration: d, Upper: 1}
}

// Value — текущее значение.
func (c *Controller) Value() float64 { return c.value }

// Status — состояние анимации.
func (c *Controller) Status() Status { return c.status }

// IsAnimating сообщает, что значение сейчас движется.
func (c *Controller) IsAnimating() bool { return c.sim != nil }

// AddListener — f вызывается при каждом изменении значения.
func (c *Controller) AddListener(f func()) { c.listeners = append(c.listeners, f) }

// AddStatusListener — f вызывается, когда меняется Status (например, анимация закончилась).
func (c *Controller) AddStatusListener(f func(Status)) {
	c.statusListeners = append(c.statusListeners, f)
}

// SetValue ставит значение сразу, без анимации (останавливает текущую).
func (c *Controller) SetValue(v float64) {
	c.sim = nil
	c.value = clamp(v, c.Lower, c.Upper)
	c.settle(Forward)
	c.notify()
}

// Forward — от текущего значения к Upper за оставшуюся часть Duration.
func (c *Controller) Forward() { c.animate(c.Upper, c.Duration, Linear, Forward) }

// Reverse — от текущего значения к Lower.
func (c *Controller) Reverse() {
	d := c.ReverseDuration
	if d == 0 {
		d = c.Duration
	}
	c.animate(c.Lower, d, Linear, Reverse)
}

// AnimateTo — к значению target за d по кривой curve (nil — линейно; d = 0 — за Duration).
func (c *Controller) AnimateTo(target float64, d time.Duration, curve Curve) {
	if d == 0 {
		d = c.Duration
	}
	dir := Forward
	if target < c.value {
		dir = Reverse
	}
	c.start0(newTween(c.value, clamp(target, c.Lower, c.Upper), d, curve), dir)
}

// Repeat — бесконечно от Lower до Upper за Duration; reverse — туда и обратно.
func (c *Controller) Repeat(reverse bool) {
	c.start0(&repeat{from: c.Lower, to: c.Upper, period: c.Duration.Seconds(), reverse: reverse, curve: Linear}, Forward)
}

// AnimateWith ведёт значение по физике sim (например, бросок списка с трением). Значение не ограничено
// Lower и Upper: их ограничивает сама sim (Clamped).
func (c *Controller) AnimateWith(sim Simulation) {
	dir := Forward
	if sim.DX(0) < 0 {
		dir = Reverse
	}
	c.start0(sim, dir)
}

// Stop останавливает движение на текущем значении.
func (c *Controller) Stop() {
	if c.sim != nil {
		c.sim = nil
		c.settle(c.direction)
	}
}

// Dispose останавливает контроллер навсегда.
func (c *Controller) Dispose() {
	c.sim = nil
	c.disposed = true
	c.listeners, c.statusListeners = nil, nil
}

// animate — к target за долю d, пропорциональную оставшемуся пути (как во Flutter: Forward с середины
// идёт вдвое быстрее).
func (c *Controller) animate(target float64, d time.Duration, curve Curve, dir Status) {
	if span := c.Upper - c.Lower; span > 0 {
		remaining := (target - c.value) / span
		if remaining < 0 {
			remaining = -remaining
		}
		d = time.Duration(float64(d) * remaining)
	}
	c.start0(newTween(c.value, target, d, curve), dir)
}

func (c *Controller) start0(sim Simulation, dir Status) {
	if c.disposed {
		return
	}
	c.sim, c.direction, c.started = sim, dir, false
	c.setStatus(dir)
	if !c.active && c.scheduler != nil {
		c.active = true
		c.scheduler.Animate(c)
	}
}

// Tick — очередной кадр (вызывает планировщик).
func (c *Controller) Tick(now time.Duration) bool {
	if c.sim == nil || c.disposed {
		c.active = false
		return false
	}
	if !c.started {
		c.start, c.started = now, true
	}
	t := (now - c.start).Seconds()
	sim := c.sim
	c.value = sim.X(t)
	done := sim.Done(t)
	if done {
		c.sim = nil
		c.settle(c.direction)
	}
	c.notify()
	if c.sim == nil {
		c.active = false
		return false
	}
	return true
}

// settle выставляет Status, когда движение остановилось.
func (c *Controller) settle(dir Status) {
	switch {
	case c.value <= c.Lower:
		c.setStatus(Dismissed)
	case c.value >= c.Upper:
		c.setStatus(Completed)
	default:
		c.setStatus(dir)
	}
}

func (c *Controller) setStatus(s Status) {
	if s == c.status {
		return
	}
	c.status = s
	for _, f := range c.statusListeners {
		f(s)
	}
}

func (c *Controller) notify() {
	for _, f := range c.listeners {
		f()
	}
}
