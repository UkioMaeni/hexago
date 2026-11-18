package gestures

// Member — участник арены: распознаватель, который претендует на касание.
type Member interface {
	// AcceptGesture — касание pointer досталось этому распознавателю.
	AcceptGesture(pointer int)
	// RejectGesture — касание досталось другому (или распознаватель сам отказался).
	RejectGesture(pointer int)
}

// Arena выбирает, какому распознавателю достанется касание, как GestureArenaManager во Flutter.
//
// Для каждого касания своя арена. Пока палец опускается, в неё входят все распознаватели под ним
// (Add). После этого арену закрывают (Close): новых участников нет, и если участник один — он победил.
// Дальше участник может сам объявить победу или отказаться (ArenaEntry.Resolve): победа одного —
// поражение остальных. Когда палец отпущен, арену «подметают» (Sweep): если победителя всё ещё нет,
// побеждает первый участник — самый глубокий виджет под пальцем. Двойное нажатие может придержать
// арену (Hold), чтобы она не решилась по отпусканию первого пальца.
type Arena struct {
	arenas map[int]*arena
}

type arena struct {
	members      []Member
	open         bool // ещё можно входить
	held         bool // подметание отложено (Hold)
	sweepPending bool // Sweep пришёл, пока арена была придержана
	eagerWinner  Member
}

// ArenaEntry — участие одного распознавателя в арене одного касания.
type ArenaEntry struct {
	arena   *Arena
	pointer int
	member  Member
}

// Resolve объявляет победу (accept = true) или отказ. После решения арены ничего не делает.
func (e *ArenaEntry) Resolve(accept bool) {
	if e != nil {
		e.arena.resolve(e.pointer, e.member, accept)
	}
}

// Add вводит участника в арену касания pointer.
func (a *Arena) Add(pointer int, m Member) *ArenaEntry {
	if a.arenas == nil {
		a.arenas = map[int]*arena{}
	}
	s := a.arenas[pointer]
	if s == nil {
		s = &arena{open: true}
		a.arenas[pointer] = s
	}
	s.members = append(s.members, m)
	return &ArenaEntry{arena: a, pointer: pointer, member: m}
}

// Close закрывает арену: вызывается после того, как событие PointerDown дошло до всех.
func (a *Arena) Close(pointer int) {
	s := a.arenas[pointer]
	if s == nil {
		return
	}
	s.open = false
	a.tryResolve(pointer, s)
}

// Sweep вызывается, когда палец отпущен: нерешённая арена отдаётся первому участнику.
func (a *Arena) Sweep(pointer int) {
	s := a.arenas[pointer]
	if s == nil {
		return
	}
	if s.held {
		s.sweepPending = true
		return
	}
	delete(a.arenas, pointer)
	if len(s.members) > 0 {
		for _, m := range s.members[1:] {
			m.RejectGesture(pointer)
		}
		s.members[0].AcceptGesture(pointer)
	}
}

// Cancel отказывает всем участникам: касание отменила система.
func (a *Arena) Cancel(pointer int) {
	s := a.arenas[pointer]
	if s == nil {
		return
	}
	delete(a.arenas, pointer)
	for _, m := range s.members {
		m.RejectGesture(pointer)
	}
}

// Hold откладывает подметание арены до Release.
func (a *Arena) Hold(pointer int) {
	if s := a.arenas[pointer]; s != nil {
		s.held = true
	}
}

// Release снимает Hold; если пока арена была придержана, пришёл Sweep, — подметает её.
func (a *Arena) Release(pointer int) {
	s := a.arenas[pointer]
	if s == nil {
		return
	}
	s.held = false
	if s.sweepPending {
		a.Sweep(pointer)
	}
}

func (a *Arena) resolve(pointer int, m Member, accept bool) {
	s := a.arenas[pointer]
	if s == nil {
		return // арена уже решена
	}
	i := -1
	for j, x := range s.members {
		if x == m {
			i = j
			break
		}
	}
	if i < 0 {
		return
	}
	if !accept {
		s.members = append(s.members[:i:i], s.members[i+1:]...)
		m.RejectGesture(pointer)
		if !s.open {
			a.tryResolve(pointer, s)
		}
		return
	}
	if s.open {
		if s.eagerWinner == nil {
			s.eagerWinner = m // победит, когда арену закроют
		}
		return
	}
	a.resolveInFavorOf(pointer, s, m)
}

func (a *Arena) tryResolve(pointer int, s *arena) {
	switch {
	case len(s.members) == 0:
		delete(a.arenas, pointer)
	case len(s.members) == 1:
		a.resolveInFavorOf(pointer, s, s.members[0])
	case s.eagerWinner != nil:
		a.resolveInFavorOf(pointer, s, s.eagerWinner)
	}
}

func (a *Arena) resolveInFavorOf(pointer int, s *arena, winner Member) {
	delete(a.arenas, pointer)
	for _, m := range s.members {
		if m != winner {
			m.RejectGesture(pointer)
		}
	}
	winner.AcceptGesture(pointer)
}
