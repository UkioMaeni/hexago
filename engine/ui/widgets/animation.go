package widgets

import (
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
