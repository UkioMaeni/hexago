package widgets

import (
	"slices"
	"time"

	"hexago/ui/animation"
)

// NewAnimation создаёт контроллер анимации длительностью d, привязанный к виджету ctx: пока анимация
// идёт, виджет перестраивается каждый кадр, а когда виджет убран из дерева, анимация останавливается.
// Создают его в Init виджета с состоянием:
//
//	func (b *Blink) Init(ctx widgets.BuildContext) {
//		b.fade = widgets.NewAnimation(ctx, 600*time.Millisecond)
//		b.fade.Repeat(true)
//	}
//
//	func (b *Blink) Build(ctx widgets.BuildContext) widgets.Widget {
//		return widgets.Opacity{Opacity: b.fade.Value(), Child: b.Child}
//	}
func NewAnimation(ctx BuildContext, d time.Duration) *animation.Controller {
	e := ctx.element()
	c := animation.NewController(e.owner.scheduler, d)
	c.AddListener(e.markNeedsBuild)
	e.animations = append(e.animations, c)
	return c
}

// timer — отложенный вызов, привязанный к элементу.
type timer struct{ stop func() }

// AfterFunc вызывает f через d в потоке интерфейса — таймер, привязанный к виджету ctx: если виджет
// убрали из дерева раньше, f не вызывается. Возвращает функцию отмены.
//
//	widgets.AfterFunc(ctx, 4*time.Second, func() { c.SetState(func() { c.hint = false }) })
func AfterFunc(ctx BuildContext, d time.Duration, f func()) (stop func()) {
	e := ctx.element()
	t := &timer{}
	remove := func() {
		if i := slices.Index(e.timers, t); i >= 0 {
			e.timers = slices.Delete(e.timers, i, i+1)
		}
	}
	cancel := e.owner.gestures.Clock.AfterFunc(d, func() {
		remove()
		if e.mounted {
			f()
		}
	})
	t.stop = func() {
		cancel()
		remove()
	}
	e.timers = append(e.timers, t)
	return t.stop
}
