package gestures

import (
	"math"

	"hexago/ui/geom"
)

// ScaleRecognizer распознаёт масштаб, поворот и перемещение одним или несколькими пальцами,
// как ScaleGestureRecognizer во Flutter. Одним пальцем он работает как свободное перетаскивание
// (Scale = 1), поэтому вместе с DragFree его не используют.
type ScaleRecognizer struct {
	Env      Env
	OnStart  func(ScaleStartDetails)
	OnUpdate func(ScaleUpdateDetails)
	OnEnd    func(ScaleEndDetails)

	pointers map[int]*scalePointer
	order    []int // порядок касаний: поворот считается по первым двум
	entries  map[int]*ArenaEntry
	accepted bool
	started  bool

	// Начало: по ним считаются Scale и Rotation. При смене числа пальцев пересчитываются,
	// чтобы масштаб не прыгал.
	initialSpan, initialH, initialV, initialAngle float64
	baseScale, baseH, baseV, baseRotation         float64
	initialFocal, focal                           geom.Offset
	lastScale, lastH, lastV, lastRotation         float64
	velocity                                      VelocityTracker
}

type scalePointer struct {
	pos   geom.Offset
	local geom.Offset
}

func (r *ScaleRecognizer) AddPointer(e PointerEvent) {
	if r.pointers == nil {
		r.pointers, r.entries = map[int]*scalePointer{}, map[int]*ArenaEntry{}
	}
	r.pointers[e.Pointer] = &scalePointer{e.Position, e.Local}
	r.order = append(r.order, e.Pointer)
	r.entries[e.Pointer] = r.Env.Arena.Add(e.Pointer, r)
	if len(r.pointers) == 1 {
		r.velocity.Reset()
		r.lastScale, r.lastH, r.lastV, r.lastRotation = 1, 1, 1, 0
		r.baseScale, r.baseH, r.baseV, r.baseRotation = 1, 1, 1, 0
	}
	r.rebase()
	if r.accepted {
		r.entries[e.Pointer].Resolve(true)
	}
}

func (r *ScaleRecognizer) HandleEvent(e PointerEvent) {
	p, ok := r.pointers[e.Pointer]
	if !ok {
		return
	}
	switch e.Kind {
	case PointerMove:
		p.pos, p.local = e.Position, e.Local
		prevFocal := r.focal
		r.measure()
		r.velocity.Add(e.Time, r.focal)
		if !r.accepted {
			if math.Abs(r.span()-r.initialSpan) > ScaleSlop || r.focal.Sub(r.initialFocal).Distance() > PanSlop {
				r.acceptAll()
			}
			return
		}
		r.update(prevFocal, e)
	case PointerUp, PointerCancel:
		r.remove(e.Pointer)
		if len(r.pointers) > 0 {
			r.rebase()
			return
		}
		started := r.started
		r.reset()
		if started && r.OnEnd != nil {
			end := ScaleEndDetails{PointerCount: 0}
			if e.Kind == PointerUp {
				end.Velocity = r.velocity.Velocity()
			}
			r.OnEnd(end)
		}
	}
}

func (r *ScaleRecognizer) AcceptGesture(int) {
	if r.accepted || len(r.pointers) == 0 {
		return
	}
	r.accepted = true
	r.start()
}

func (r *ScaleRecognizer) RejectGesture(pointer int) {
	if r.accepted {
		return
	}
	r.remove(pointer)
	if len(r.pointers) == 0 {
		r.reset()
	}
}

func (r *ScaleRecognizer) Dispose() {
	for p, e := range r.entries {
		delete(r.entries, p)
		e.Resolve(false)
	}
	r.reset()
}

func (r *ScaleRecognizer) acceptAll() {
	r.accepted = true
	for _, e := range r.entries {
		e.Resolve(true)
	}
	r.start()
}

func (r *ScaleRecognizer) start() {
	if r.started {
		return
	}
	r.started = true
	if r.OnStart != nil {
		r.OnStart(ScaleStartDetails{FocalPoint: r.focal, LocalFocalPoint: r.localFocal(), PointerCount: len(r.pointers)})
	}
}

func (r *ScaleRecognizer) update(prevFocal geom.Offset, e PointerEvent) {
	if !r.started {
		r.start()
	}
	r.lastScale = r.ratio(r.span(), r.initialSpan, r.baseScale)
	r.lastH = r.ratio(r.spanAlong(true), r.initialH, r.baseH)
	r.lastV = r.ratio(r.spanAlong(false), r.initialV, r.baseV)
	r.lastRotation = r.baseRotation
	if a, ok := r.angle(); ok {
		r.lastRotation += a - r.initialAngle
	}
	if r.OnUpdate != nil {
		r.OnUpdate(ScaleUpdateDetails{
			FocalPoint: r.focal, LocalFocalPoint: r.localFocal(), FocalPointDelta: r.focal.Sub(prevFocal),
			Scale: r.lastScale, HorizontalScale: r.lastH, VerticalScale: r.lastV,
			Rotation: r.lastRotation, PointerCount: len(r.pointers),
		})
	}
}

// rebase начинает отсчёт заново с текущих пальцев, сохранив уже набранные масштаб и поворот.
func (r *ScaleRecognizer) rebase() {
	r.measure()
	r.baseScale, r.baseH, r.baseV, r.baseRotation = r.lastScale, r.lastH, r.lastV, r.lastRotation
	r.initialSpan, r.initialH, r.initialV = r.span(), r.spanAlong(true), r.spanAlong(false)
	r.initialAngle, _ = r.angle()
	r.initialFocal = r.focal
}

func (r *ScaleRecognizer) ratio(now, initial, base float64) float64 {
	if initial == 0 {
		return base
	}
	return base * now / initial
}

func (r *ScaleRecognizer) measure() {
	var sum geom.Offset
	for _, p := range r.pointers {
		sum = sum.Add(p.pos)
	}
	if n := len(r.pointers); n > 0 {
		r.focal = sum.Scale(1 / float64(n))
	}
}

func (r *ScaleRecognizer) localFocal() geom.Offset {
	var sum geom.Offset
	for _, p := range r.pointers {
		sum = sum.Add(p.local)
	}
	if n := len(r.pointers); n > 0 {
		return sum.Scale(1 / float64(n))
	}
	return geom.Zero
}

// span — среднее расстояние пальцев до их середины.
func (r *ScaleRecognizer) span() float64 {
	var s float64
	for _, p := range r.pointers {
		s += p.pos.Sub(r.focal).Distance()
	}
	return s / float64(max(len(r.pointers), 1))
}

func (r *ScaleRecognizer) spanAlong(horizontal bool) float64 {
	var s float64
	for _, p := range r.pointers {
		d := p.pos.Sub(r.focal)
		if horizontal {
			s += math.Abs(d.X)
		} else {
			s += math.Abs(d.Y)
		}
	}
	return s / float64(max(len(r.pointers), 1))
}

// angle — направление от первого пальца ко второму; ok=false, если пальцев меньше двух.
func (r *ScaleRecognizer) angle() (float64, bool) {
	if len(r.order) < 2 {
		return 0, false
	}
	a, b := r.pointers[r.order[0]], r.pointers[r.order[1]]
	d := b.pos.Sub(a.pos)
	return math.Atan2(d.Y, d.X), true
}

func (r *ScaleRecognizer) remove(pointer int) {
	delete(r.pointers, pointer)
	delete(r.entries, pointer)
	for i, p := range r.order {
		if p == pointer {
			r.order = append(r.order[:i:i], r.order[i+1:]...)
			break
		}
	}
}

func (r *ScaleRecognizer) reset() {
	r.pointers, r.entries, r.order = map[int]*scalePointer{}, map[int]*ArenaEntry{}, nil
	r.accepted, r.started = false, false
}
