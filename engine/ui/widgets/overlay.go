package widgets

import (
	"slices"

	"hexago/ui/geom"
	"hexago/ui/rendering"
)

// OverlayEntry — слой поверх всех экранов Navigator, как OverlayEntry во Flutter: ручки и меню
// выделения текста, подсказки. В отличие от окна (Route) он не затемняет экран, не забирает фокус
// и не закрывается кнопкой «Назад». Касания, которые не попали в его виджеты, проходят к экрану.
// Builder обычно возвращает Positioned или Follow.
type OverlayEntry struct {
	Builder func(ctx BuildContext) Widget
	nav     *Navigator
	view    *overlayView
}

// ShowOverlay кладёт слой поверх экранов ближайшего Navigator.
func ShowOverlay(ctx BuildContext, builder func(ctx BuildContext) Widget) *OverlayEntry {
	n := NavigatorOf(ctx)
	e := &OverlayEntry{Builder: builder, nav: n}
	n.overlay = append(n.overlay, e)
	n.changed()
	return e
}

// Remove убирает слой.
func (e *OverlayEntry) Remove() {
	if e.nav == nil {
		return
	}
	n := e.nav
	if i := slices.Index(n.overlay, e); i >= 0 {
		n.overlay = slices.Delete(n.overlay, i, i+1)
		n.changed()
	}
	e.nav = nil
}

// MarkNeedsBuild перестраивает слой (данные, из которых он строится, изменились).
func (e *OverlayEntry) MarkNeedsBuild() {
	if e.view != nil && e.view.Mounted() {
		e.view.SetState(nil)
	}
}

// overlayLayer — все слои Overlay одного Navigator.
type overlayLayer struct {
	Base
	entries []*OverlayEntry
}

func (l overlayLayer) Build(BuildContext) Widget {
	kids := make([]Widget, len(l.entries))
	for i, e := range l.entries {
		kids[i] = Keyed{Key: e, Child: overlayView{entry: e}}
	}
	return Stack{Fit: rendering.StackExpand, Children: kids}
}

// overlayView — один слой: перестраивается сам по MarkNeedsBuild.
type overlayView struct {
	State
	entry *OverlayEntry
}

func (v *overlayView) Init(BuildContext)                { v.entry.view = v }
func (v *overlayView) Update(BuildContext, overlayView) { v.entry.view = v }
func (v *overlayView) Build(ctx BuildContext) Widget    { return v.entry.Builder(ctx) }

// Follow ставит ребёнка рядом с виджетом-целью, где бы тот ни был на экране (в прокрутке, в переходе):
// Place получает прямоугольник цели на экране и размер ребёнка и возвращает, где левый верхний угол
// ребёнка. Цель — тот, чей BuildContext передан в Target. Для слоёв OverlayEntry: ручки выделения
// следуют за полем ввода. Положение цели берётся из прошлого кадра.
type Follow struct {
	Base
	Target BuildContext
	Place  func(target geom.Rect, child geom.Size) geom.Offset
	Child  Widget
}

func (w Follow) ChildWidget() Widget { return w.Child }
func (w Follow) CreateRenderObject(BuildContext) rendering.Box {
	return &followBox{target: w.Target, place: w.Place}
}
func (w Follow) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	f := b.(*followBox)
	f.target, f.place = w.Target, w.Place
}

// followBox — бокс во весь доступный размер: ребёнок стоит там, куда скажет place.
type followBox struct {
	rendering.Proxy
	target BuildContext
	place  func(target geom.Rect, child geom.Size) geom.Offset
}

func (b *followBox) PerformLayout(c rendering.Constraints) geom.Size {
	size := c.Biggest()
	ch := rendering.Child(b)
	if ch == nil {
		return size
	}
	s := rendering.Layout(ch, c.Loosen())
	if b.target != nil && b.place != nil {
		r := BoundsOf(b.target)
		rendering.SetOffset(ch, b.place(r, s))
		// Положение цели — из прошлого кадра. Если в этом кадре она сдвинулась (прокрутка, переход),
		// нужен ещё кадр: иначе после последнего кадра движения ребёнок остался бы чуть в стороне.
		if o := b.target.element().owner; o != nil {
			o.postFrame = append(o.postFrame, func() {
				if BoundsOf(b.target) != r {
					o.requestFrame()
				}
			})
		}
	}
	return size
}
