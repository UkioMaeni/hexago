package gestures

import (
	"time"

	"hexago/ui/geom"
)

// VelocityTracker оценивает скорость пальца по последним событиям — для броска (fling)
// в конце перетаскивания.
type VelocityTracker struct {
	samples []sample
}

type sample struct {
	t time.Duration
	p geom.Offset
}

// horizon — скорость считается по событиям за последние 100 мс, как во Flutter.
const horizon = 100 * time.Millisecond

// Add запоминает положение пальца в момент t.
func (v *VelocityTracker) Add(t time.Duration, p geom.Offset) {
	v.samples = append(v.samples, sample{t, p})
	for len(v.samples) > 1 && t-v.samples[0].t > horizon {
		v.samples = v.samples[1:]
	}
}

// Reset забывает все события.
func (v *VelocityTracker) Reset() { v.samples = v.samples[:0] }

// Velocity — скорость в dp/с методом наименьших квадратов по последним событиям.
// Медленнее MinFlingVelocity — ноль (палец остановился перед отпусканием), быстрее
// MaxFlingVelocity — обрезается.
func (v *VelocityTracker) Velocity() geom.Offset {
	n := len(v.samples)
	if n < 2 {
		return geom.Zero
	}
	last := v.samples[n-1].t
	var st, sx, sy, stt, stx, sty float64
	for _, s := range v.samples {
		t := (s.t - last).Seconds()
		st += t
		sx += s.p.X
		sy += s.p.Y
		stt += t * t
		stx += t * s.p.X
		sty += t * s.p.Y
	}
	k := float64(n)
	den := k*stt - st*st
	if den == 0 {
		return geom.Zero
	}
	vel := geom.Offset{X: (k*stx - st*sx) / den, Y: (k*sty - st*sy) / den}
	speed := vel.Distance()
	switch {
	case speed < MinFlingVelocity:
		return geom.Zero
	case speed > MaxFlingVelocity:
		return vel.Scale(MaxFlingVelocity / speed)
	}
	return vel
}
