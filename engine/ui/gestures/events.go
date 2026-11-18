// Package gestures — касания и жесты, как gestures-библиотека Flutter.
//
// Платформа присылает сырые события касаний (PointerEvent): палец коснулся, двигается,
// отпущен. Распознаватели (Recognizer) превращают их в жесты: нажатие, двойное нажатие,
// долгое нажатие, перетаскивание, масштаб. Если на одно касание претендуют несколько
// распознавателей (нажатие и перетаскивание), победителя выбирает арена (Arena).
//
// Пакет ничего не знает о боксах и виджетах: события ему передаёт widgets.Binding, а
// таймеры он берёт у Clock — в тестах их заменяют FakeClock, и жесты проверяются без устройства.
package gestures

import (
	"time"

	"hexago/ui/geom"
)

// PointerKind — что произошло с пальцем.
type PointerKind uint8

const (
	PointerDown   PointerKind = iota // палец коснулся экрана
	PointerMove                      // палец двигается
	PointerUp                        // палец отпущен
	PointerCancel                    // касание отменила система (например, открылась шторка)
)

func (k PointerKind) String() string {
	return [...]string{"Down", "Move", "Up", "Cancel"}[k]
}

// PointerEvent — событие одного пальца.
type PointerEvent struct {
	Kind     PointerKind
	Pointer  int           // номер касания: у каждого пальца свой, пока он на экране
	Position geom.Offset   // где палец, в dp, в координатах экрана
	Local    geom.Offset   // то же в координатах бокса, которому доставлено событие
	Delta    geom.Offset   // сдвиг с прошлого события этого пальца, в dp
	Time     time.Duration // время события (отсчёт от включения устройства)
}

// Пороги жестов, в dp и миллисекундах — те же, что у Flutter и Android.
const (
	TouchSlop        = 18.0  // палец сдвинулся меньше — это ещё нажатие, а не перетаскивание
	PanSlop          = 36.0  // свободное перетаскивание начинается после такого сдвига
	ScaleSlop        = 18.0  // масштаб начинается, когда расстояние между пальцами изменилось на столько
	DoubleTapSlop    = 100.0 // второе нажатие двойного — не дальше этого от первого
	MinFlingVelocity = 50.0  // dp/с: медленнее — не бросок
	MaxFlingVelocity = 8000.0

	PressTimeout     = 100 * time.Millisecond // через столько нажатие считается начавшимся (OnTapDown)
	LongPressTimeout = 500 * time.Millisecond // столько держать палец для долгого нажатия
	DoubleTapTimeout = 300 * time.Millisecond // столько ждать второго нажатия
)

// Recognizer — распознаватель жеста. GestureDetector отдаёт ему каждое новое касание
// (AddPointer с событием PointerDown), а затем все остальные события этого касания (HandleEvent).
type Recognizer interface {
	AddPointer(e PointerEvent)
	HandleEvent(e PointerEvent)
	// Dispose — распознаватель больше не нужен: отказаться от касаний, остановить таймеры.
	Dispose()
}

// Env — то, чем пользуются распознаватели: арена и часы.
type Env struct {
	Arena *Arena
	Clock Clock
}
