package gestures

// TapRecognizer распознаёт нажатие: палец опустился и поднялся, почти не сдвинувшись.
//
// OnTapDown приходит, когда нажатие выиграло арену или через PressTimeout (чтобы при прокрутке
// кнопки не мигали), OnTapUp и OnTap — когда палец отпущен, OnTapCancel — если после OnTapDown
// жест достался другому (например, палец пошёл в перетаскивание).
type TapRecognizer struct {
	Env         Env
	OnTapDown   func(TapDownDetails)
	OnTapUp     func(TapUpDetails)
	OnTap       func()
	OnTapCancel func()

	tracking bool
	pointer  int
	entry    *ArenaEntry
	down     PointerEvent
	up       *PointerEvent
	won      bool
	sentDown bool
	stop     func()
}

func (r *TapRecognizer) AddPointer(e PointerEvent) {
	if r.tracking {
		return // второй палец во время нажатия не участвует
	}
	r.tracking, r.pointer, r.down = true, e.Pointer, e
	r.entry = r.Env.Arena.Add(e.Pointer, r)
	r.stop = r.Env.Clock.AfterFunc(PressTimeout, func() {
		if r.tracking && !r.sentDown {
			r.sendDown()
		}
	})
}

func (r *TapRecognizer) HandleEvent(e PointerEvent) {
	if !r.tracking || e.Pointer != r.pointer {
		return
	}
	switch e.Kind {
	case PointerMove:
		if e.Position.Sub(r.down.Position).Distance() > TouchSlop {
			r.cancel()
		}
	case PointerUp:
		r.up = &e
		if r.won {
			r.finish()
		}
	case PointerCancel:
		r.cancel()
	}
}

func (r *TapRecognizer) AcceptGesture(int) {
	if !r.tracking {
		return
	}
	r.won = true
	if !r.sentDown {
		r.sendDown()
	}
	if r.up != nil {
		r.finish()
	}
}

func (r *TapRecognizer) RejectGesture(int) {
	if r.tracking {
		r.cancel()
	}
}

func (r *TapRecognizer) Dispose() {
	if r.tracking {
		r.entry.Resolve(false)
		r.reset()
	}
}

func (r *TapRecognizer) sendDown() {
	r.sentDown = true
	if r.OnTapDown != nil {
		r.OnTapDown(TapDownDetails{Position: r.down.Position, Local: r.down.Local})
	}
}

func (r *TapRecognizer) finish() {
	up := *r.up
	r.reset()
	if r.OnTapUp != nil {
		r.OnTapUp(TapUpDetails{Position: up.Position, Local: up.Local})
	}
	if r.OnTap != nil {
		r.OnTap()
	}
}

// cancel — нажатия не будет: если OnTapDown уже был, сообщить об отмене, и отказаться от арены.
// Сначала отмена, потом отказ: иначе жест-победитель начался бы раньше, чем нажатие отменилось.
func (r *TapRecognizer) cancel() {
	sent, entry := r.sentDown, r.entry
	r.reset()
	if sent && r.OnTapCancel != nil {
		r.OnTapCancel()
	}
	entry.Resolve(false)
}

func (r *TapRecognizer) reset() {
	if r.stop != nil {
		r.stop()
	}
	r.tracking, r.won, r.sentDown, r.up, r.entry, r.stop = false, false, false, nil, nil, nil
}

// DoubleTapRecognizer распознаёт двойное нажатие. Пока он ждёт второго нажатия (DoubleTapTimeout),
// арена первого придержана, поэтому одиночное нажатие рядом с ним срабатывает с задержкой — как во Flutter.
type DoubleTapRecognizer struct {
	Env         Env
	OnDoubleTap func()

	first   *tapTrack // первое нажатие, уже отпущенное
	current *tapTrack // нажатие, которое сейчас под пальцем
	stop    func()
}

type tapTrack struct {
	pointer int
	entry   *ArenaEntry
	down    PointerEvent
	up      PointerEvent
}

func (r *DoubleTapRecognizer) AddPointer(e PointerEvent) {
	if r.current != nil {
		return
	}
	if r.first != nil && e.Position.Sub(r.first.up.Position).Distance() > DoubleTapSlop {
		r.reset() // второе нажатие слишком далеко — это уже не двойное
	}
	r.current = &tapTrack{pointer: e.Pointer, down: e, entry: r.Env.Arena.Add(e.Pointer, r)}
}

func (r *DoubleTapRecognizer) HandleEvent(e PointerEvent) {
	t := r.current
	if t == nil || e.Pointer != t.pointer {
		return
	}
	switch e.Kind {
	case PointerMove:
		if e.Position.Sub(t.down.Position).Distance() > TouchSlop {
			r.reset()
		}
	case PointerCancel:
		r.reset()
	case PointerUp:
		t.up = e
		r.current = nil
		if r.first == nil {
			// Первое нажатие: придержать его арену и ждать второго.
			r.first = t
			r.Env.Arena.Hold(t.pointer)
			r.stop = r.Env.Clock.AfterFunc(DoubleTapTimeout, r.reset)
			return
		}
		first := r.first
		r.first = nil
		r.stopTimer()
		first.entry.Resolve(true)
		t.entry.Resolve(true)
		r.Env.Arena.Release(first.pointer)
		if r.OnDoubleTap != nil {
			r.OnDoubleTap()
		}
	}
}

func (r *DoubleTapRecognizer) AcceptGesture(int) {}

func (r *DoubleTapRecognizer) RejectGesture(pointer int) {
	if (r.first != nil && r.first.pointer == pointer) || (r.current != nil && r.current.pointer == pointer) {
		r.reset()
	}
}

func (r *DoubleTapRecognizer) Dispose() { r.reset() }

// reset отказывается от всех касаний и отпускает придержанную арену.
func (r *DoubleTapRecognizer) reset() {
	r.stopTimer()
	first, current := r.first, r.current
	r.first, r.current = nil, nil
	if current != nil {
		current.entry.Resolve(false)
	}
	if first != nil {
		first.entry.Resolve(false)
		r.Env.Arena.Release(first.pointer)
	}
}

func (r *DoubleTapRecognizer) stopTimer() {
	if r.stop != nil {
		r.stop()
		r.stop = nil
	}
}
