package gestures

// LongPressRecognizer распознаёт долгое нажатие: палец держат LongPressTimeout, почти не сдвигая.
// После начала палец можно двигать (OnLongPressMoveUpdate) — например, чтобы перетащить элемент.
type LongPressRecognizer struct {
	Env                   Env
	OnLongPressStart      func(LongPressStartDetails)
	OnLongPress           func()
	OnLongPressMoveUpdate func(LongPressMoveUpdateDetails)
	OnLongPressEnd        func(LongPressEndDetails)
	OnLongPressUp         func()

	tracking bool
	accepted bool
	pointer  int
	entry    *ArenaEntry
	down     PointerEvent
	start    PointerEvent // где был палец, когда долгое нажатие началось
	velocity VelocityTracker
	stop     func()
}

func (r *LongPressRecognizer) AddPointer(e PointerEvent) {
	if r.tracking {
		return
	}
	r.tracking, r.accepted, r.pointer, r.down, r.start = true, false, e.Pointer, e, e
	r.velocity.Reset()
	r.velocity.Add(e.Time, e.Position)
	r.entry = r.Env.Arena.Add(e.Pointer, r)
	r.stop = r.Env.Clock.AfterFunc(LongPressTimeout, func() {
		if r.tracking && !r.accepted {
			r.entry.Resolve(true)
		}
	})
}

func (r *LongPressRecognizer) HandleEvent(e PointerEvent) {
	if !r.tracking || e.Pointer != r.pointer {
		return
	}
	r.velocity.Add(e.Time, e.Position)
	switch e.Kind {
	case PointerMove:
		if !r.accepted {
			r.down = e
			if e.Position.Sub(r.start.Position).Distance() > TouchSlop {
				r.giveUp()
			}
			return
		}
		if r.OnLongPressMoveUpdate != nil {
			r.OnLongPressMoveUpdate(LongPressMoveUpdateDetails{
				Position: e.Position, Local: e.Local, OffsetFromOrigin: e.Position.Sub(r.start.Position),
			})
		}
	case PointerUp:
		if !r.accepted {
			r.giveUp() // отпустили раньше времени — это не долгое нажатие
			return
		}
		r.reset()
		if r.OnLongPressEnd != nil {
			r.OnLongPressEnd(LongPressEndDetails{Position: e.Position, Local: e.Local, Velocity: r.velocity.Velocity()})
		}
		if r.OnLongPressUp != nil {
			r.OnLongPressUp()
		}
	case PointerCancel:
		r.giveUp()
	}
}

func (r *LongPressRecognizer) AcceptGesture(int) {
	if !r.tracking || r.accepted {
		return
	}
	r.accepted = true
	r.start = r.down
	if r.OnLongPressStart != nil {
		r.OnLongPressStart(LongPressStartDetails{Position: r.down.Position, Local: r.down.Local})
	}
	if r.OnLongPress != nil {
		r.OnLongPress()
	}
}

func (r *LongPressRecognizer) RejectGesture(int) {
	if r.tracking {
		r.reset()
	}
}

func (r *LongPressRecognizer) Dispose() { r.giveUp() }

func (r *LongPressRecognizer) giveUp() {
	entry := r.entry
	r.reset()
	entry.Resolve(false)
}

func (r *LongPressRecognizer) reset() {
	if r.stop != nil {
		r.stop()
	}
	r.tracking, r.accepted, r.entry, r.stop = false, false, nil, nil
}
