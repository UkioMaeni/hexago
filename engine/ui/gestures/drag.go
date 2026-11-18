package gestures

import (
	"math"

	"hexago/ui/geom"
)

// DragAxis — в каком направлении перетаскивают.
type DragAxis uint8

const (
	DragFree       DragAxis = iota // в любом направлении (pan)
	DragVertical                   // только вверх-вниз
	DragHorizontal                 // только влево-вправо
)

// DragRecognizer распознаёт перетаскивание: палец сдвинулся дальше порога (TouchSlop вдоль оси,
// PanSlop для свободного). Вертикальное и горизонтальное перетаскивания соревнуются: побеждает то,
// вдоль чьей оси палец ушёл дальше, — так список прокручивается, а страницы рядом листаются.
//
// OnStart получает точку, где палец опустился, а первый OnUpdate — весь путь до порога:
// перетаскиваемый элемент точно следует за пальцем.
type DragRecognizer struct {
	Env      Env
	Axis     DragAxis
	OnStart  func(DragStartDetails)
	OnUpdate func(DragUpdateDetails)
	OnEnd    func(DragEndDetails)

	tracking bool
	accepted bool
	pointer  int
	entry    *ArenaEntry
	down     PointerEvent
	last     PointerEvent
	pending  geom.Offset // путь до победы в арене
	velocity VelocityTracker
}

func (r *DragRecognizer) AddPointer(e PointerEvent) {
	if r.tracking {
		return // перетаскивают одним пальцем; несколько пальцев — это ScaleRecognizer
	}
	r.tracking, r.accepted, r.pointer, r.down, r.last, r.pending = true, false, e.Pointer, e, e, geom.Zero
	r.velocity.Reset()
	r.velocity.Add(e.Time, e.Position)
	r.entry = r.Env.Arena.Add(e.Pointer, r)
}

func (r *DragRecognizer) HandleEvent(e PointerEvent) {
	if !r.tracking || e.Pointer != r.pointer {
		return
	}
	r.velocity.Add(e.Time, e.Position)
	switch e.Kind {
	case PointerMove:
		delta := e.Position.Sub(r.last.Position)
		r.last = e
		if r.accepted {
			r.update(e, delta)
			return
		}
		r.pending = r.pending.Add(delta)
		if r.passedSlop() {
			r.entry.Resolve(true)
		}
	case PointerUp:
		if !r.accepted {
			r.giveUp()
			return
		}
		r.tracking, r.accepted = false, false
		if r.OnEnd != nil {
			v := r.velocity.Velocity()
			end := DragEndDetails{Velocity: r.project(v)}
			end.PrimaryVelocity = r.primary(v)
			r.OnEnd(end)
		}
	case PointerCancel:
		wasAccepted := r.accepted
		r.giveUp()
		if wasAccepted && r.OnEnd != nil {
			r.OnEnd(DragEndDetails{})
		}
	}
}

func (r *DragRecognizer) AcceptGesture(int) {
	if !r.tracking || r.accepted {
		return
	}
	r.accepted = true
	if r.OnStart != nil {
		r.OnStart(DragStartDetails{Position: r.down.Position, Local: r.down.Local})
	}
	if r.pending != geom.Zero {
		r.update(r.last, r.pending)
	}
}

func (r *DragRecognizer) RejectGesture(int) {
	if r.tracking && !r.accepted {
		r.tracking = false
	}
}

func (r *DragRecognizer) Dispose() { r.giveUp() }

func (r *DragRecognizer) giveUp() {
	entry := r.entry
	r.tracking, r.accepted, r.entry = false, false, nil
	entry.Resolve(false)
}

func (r *DragRecognizer) passedSlop() bool {
	switch r.Axis {
	case DragVertical:
		return math.Abs(r.pending.Y) > TouchSlop
	case DragHorizontal:
		return math.Abs(r.pending.X) > TouchSlop
	}
	return r.pending.Distance() > PanSlop
}

func (r *DragRecognizer) update(e PointerEvent, delta geom.Offset) {
	if r.OnUpdate == nil {
		return
	}
	d := r.project(delta)
	r.OnUpdate(DragUpdateDetails{Position: e.Position, Local: e.Local, Delta: d, PrimaryDelta: r.primary(delta)})
}

// project оставляет у вектора только составляющую вдоль оси.
func (r *DragRecognizer) project(v geom.Offset) geom.Offset {
	switch r.Axis {
	case DragVertical:
		return geom.Offset{Y: v.Y}
	case DragHorizontal:
		return geom.Offset{X: v.X}
	}
	return v
}

func (r *DragRecognizer) primary(v geom.Offset) float64 {
	switch r.Axis {
	case DragVertical:
		return v.Y
	case DragHorizontal:
		return v.X
	}
	return 0
}
