package widgets

import (
	"hexago/ui/animation"
	"hexago/ui/geom"
	"hexago/ui/rendering"
)

// Виджеты переходов следят за анимацией сами: значение берётся в каждом кадре при отрисовке, а ребёнок
// не перестраивается. Анимация — Controller (из NewAnimation или маршрута) или animation.Curved от него:
//
//	widgets.FadeTransition{Opacity: animation.Curved(c.fade, animation.EaseOut), Child: ...}
//
// Для анимации, которая меняет что-то кроме прозрачности и сдвига, — NewAnimation и перестройка
// (или неявные анимации: AnimatedContainer и другие).

// FadeTransition — прозрачность ребёнка следует за анимацией Opacity: 0 — невидим, 1 — как есть.
type FadeTransition struct {
	Base
	Opacity animation.Animation
	Child   Widget
}

func (w FadeTransition) ChildWidget() Widget { return w.Child }
func (w FadeTransition) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.FadeBox{Opacity: w.Opacity}
}
func (w FadeTransition) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	b.(*rendering.FadeBox).Opacity = w.Opacity
}

// SlideTransition сдвигает ребёнка на долю его размера: от Begin (значение анимации 0) до End
// (значение 1). Begin: geom.Offset{X: 1} — ребёнок въезжает справа на всю свою ширину.
type SlideTransition struct {
	Base
	Position   animation.Animation
	Begin, End geom.Offset
	Child      Widget
}

func (w SlideTransition) ChildWidget() Widget { return w.Child }
func (w SlideTransition) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.SlideBox{Position: w.Position, Begin: w.Begin, End: w.End}
}
func (w SlideTransition) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	s := b.(*rendering.SlideBox)
	s.Position, s.Begin, s.End = w.Position, w.Begin, w.End
}
