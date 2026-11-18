package animation

import (
	"math"
	"testing"
	"time"
)

// frames — планировщик для тестов: кадры идут только по Advance.
type frames struct {
	now     time.Duration
	tickers []Ticker
}

func (f *frames) Animate(t Ticker) { f.tickers = append(f.tickers, t) }

// Advance — кадры каждые 16 мс в течение d.
func (f *frames) Advance(d time.Duration) {
	for end := f.now + d; f.now < end; {
		f.now += 16 * time.Millisecond
		active := f.tickers[:0]
		for _, t := range f.tickers {
			if t.Tick(f.now) {
				active = append(active, t)
			}
		}
		f.tickers = active
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestCubicCurves(t *testing.T) {
	for name, c := range map[string]Curve{"Ease": Ease, "FastOutSlowIn": FastOutSlowIn, "Standard": Standard} {
		if !near(c.Transform(0), 0) || !near(c.Transform(1), 1) {
			t.Errorf("%s: концы %v, %v", name, c.Transform(0), c.Transform(1))
		}
		prev := 0.0
		for i := 1; i <= 100; i++ {
			v := c.Transform(float64(i) / 100)
			if v < prev-1e-9 {
				t.Errorf("%s убывает в %v", name, float64(i)/100)
			}
			prev = v
		}
	}
	// EaseIn в середине медленнее линейной, EaseOut — быстрее.
	if EaseIn.Transform(0.5) >= 0.5 || EaseOut.Transform(0.5) <= 0.5 {
		t.Errorf("EaseIn(0.5)=%v, EaseOut(0.5)=%v", EaseIn.Transform(0.5), EaseOut.Transform(0.5))
	}
}

func TestControllerForwardReverse(t *testing.T) {
	f := &frames{}
	c := NewController(f, 300*time.Millisecond)
	var statuses []Status
	c.AddStatusListener(func(s Status) { statuses = append(statuses, s) })
	changes := 0
	c.AddListener(func() { changes++ })

	c.Forward()
	f.Advance(150 * time.Millisecond)
	if v := c.Value(); v <= 0.2 || v >= 0.8 {
		t.Errorf("на полпути значение %v", v)
	}
	f.Advance(300 * time.Millisecond)
	if c.Value() != 1 || c.Status() != Completed || c.IsAnimating() {
		t.Errorf("в конце: %v %v %v", c.Value(), c.Status(), c.IsAnimating())
	}
	c.Reverse()
	f.Advance(400 * time.Millisecond)
	if c.Value() != 0 || c.Status() != Dismissed {
		t.Errorf("после Reverse: %v %v", c.Value(), c.Status())
	}
	if len(statuses) != 4 || statuses[0] != Forward || statuses[1] != Completed || statuses[2] != Reverse || statuses[3] != Dismissed {
		t.Errorf("статусы %v", statuses)
	}
	if changes < 10 {
		t.Errorf("слушатели вызваны %d раз — ожидали каждый кадр", changes)
	}
	if len(f.tickers) != 0 {
		t.Error("остановленная анимация продолжает просить кадры")
	}
}

func TestAnimateToAndRepeat(t *testing.T) {
	f := &frames{}
	c := NewController(f, time.Second)
	c.AnimateTo(0.5, 200*time.Millisecond, EaseOut)
	f.Advance(250 * time.Millisecond)
	if !near(c.Value(), 0.5) {
		t.Errorf("AnimateTo(0.5): %v", c.Value())
	}
	c.Repeat(true)
	f.Advance(5 * time.Second)
	if !c.IsAnimating() {
		t.Error("Repeat остановился сам")
	}
	c.Stop()
	f.Advance(100 * time.Millisecond)
	if len(f.tickers) != 0 {
		t.Error("после Stop кадры всё ещё нужны")
	}
}

func TestFrictionFling(t *testing.T) {
	s := Friction{Drag: ScrollFriction, Position: 100, Velocity: 2000}
	end := s.FinalX()
	if end <= 100 || !near(s.X(60), end) {
		t.Errorf("бросок: конец %v, X(60) %v", end, s.X(60))
	}
	// Бросок упирается в край и останавливается на нём.
	f := &frames{}
	c := NewController(f, 0)
	c.AnimateWith(Clamped{Sim: s, Min: 0, Max: 300})
	f.Advance(3 * time.Second)
	if c.Value() != 300 || c.IsAnimating() {
		t.Errorf("бросок за край: значение %v, идёт %v", c.Value(), c.IsAnimating())
	}
}
