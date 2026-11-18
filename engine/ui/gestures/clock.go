package gestures

import (
	"slices"
	"time"
)

// Clock даёт распознавателям таймеры: долгое нажатие, ожидание второго нажатия.
// Функция таймера должна выполняться в потоке интерфейса — там же, где события касаний.
type Clock interface {
	// AfterFunc вызывает f через d; stop отменяет вызов, если он ещё не случился.
	AfterFunc(d time.Duration, f func()) (stop func())
}

// FakeClock — часы для тестов: время идёт только по Advance, таймеры срабатывают в нём же.
type FakeClock struct {
	now    time.Duration
	timers []*fakeTimer
}

type fakeTimer struct {
	at      time.Duration
	f       func()
	stopped bool
}

// Now — сколько прошло с начала теста.
func (c *FakeClock) Now() time.Duration { return c.now }

func (c *FakeClock) AfterFunc(d time.Duration, f func()) func() {
	t := &fakeTimer{at: c.now + d, f: f}
	c.timers = append(c.timers, t)
	return func() { t.stopped = true }
}

// Advance переводит часы на d вперёд и по порядку вызывает таймеры, время которых пришло.
func (c *FakeClock) Advance(d time.Duration) {
	end := c.now + d
	for {
		slices.SortStableFunc(c.timers, func(a, b *fakeTimer) int { return int(a.at - b.at) })
		if len(c.timers) == 0 || c.timers[0].at > end {
			break
		}
		t := c.timers[0]
		c.timers = c.timers[1:]
		if !t.stopped {
			c.now = t.at
			t.f()
		}
	}
	c.now = end
}
