package widgets

import (
	"reflect"
	"slices"
	"time"

	"hexago/ui/animation"
	"hexago/ui/gfx"
	"hexago/ui/rendering"
)

// Навигация, как Navigator во Flutter: стопка маршрутов — экранов и окон поверх них. Navigator ставит
// App, его первый экран — Home. Открыть экран и вернуться:
//
//	widgets.Push(ctx, Details{ID: id}) // экран въезжает поверх текущего
//	widgets.Pop(ctx)                   // верхний маршрут уходит, под ним — прежний экран с его состоянием
//
// Кнопка «Назад» Android тоже закрывает верхний маршрут; на первом экране она сворачивает приложение.
// Окна поверх экрана (диалоги, нижние панели) — тоже маршруты: material.ShowDialog, material.ShowBottomSheet.

// Route — то, что кладут в Navigator: экран или окно поверх экрана.
//
//	widgets.PushRoute(ctx, widgets.Route{Page: ColorPicker{}, OnPop: func(r any) { ... }})
type Route struct {
	// Page — содержимое маршрута. Экран получает весь размер Navigator.
	Page Widget
	// Popup — окно поверх экрана (диалог, нижняя панель): экран под ним остаётся виден и не уезжает.
	// По умолчанию маршрут — экран, который закрывает собой предыдущий.
	Popup bool
	// Barrier — затемнение под окном; прозрачный — без затемнения. Касания к экрану под окном
	// не проходят в любом случае.
	Barrier gfx.Color
	// Dismissible — касание затемнения закрывает окно.
	Dismissible bool
	// Transition — как маршрут входит и уходит. Пустой — переход экранов Navigator (у окна — Fade).
	Transition Transition
	// OnPop вызывается, когда маршрут закрывают: result — то, что передали в PopWith (у Pop — nil).
	OnPop func(result any)
}

// Transition — как маршрут входит и уходит.
type Transition struct {
	Duration        time.Duration   // вход; 0 — 300 мс
	ReverseDuration time.Duration   // уход; 0 — как Duration
	Curve           animation.Curve // кривая входа; nil — равномерно
	ReverseCurve    animation.Curve // кривая ухода; nil — как Curve
	// Build оборачивает содержимое маршрута в виджеты анимации. anim — вход: 0 → 1 при открытии,
	// обратно при закрытии; secondary — уход под следующий экран: 0 → 1, когда поверх открывают экран.
	// Build вызывается при перестройке маршрута, а не в каждом кадре, поэтому анимацию показывают виджеты,
	// которые берут её значение сами: FadeTransition, SlideTransition.
	Build func(anim, secondary animation.Animation, child Widget) Widget
}

func (t Transition) duration() time.Duration {
	if t.Duration > 0 {
		return t.Duration
	}
	return defaultDuration
}

func (t Transition) reverseDuration() time.Duration {
	if t.ReverseDuration > 0 {
		return t.ReverseDuration
	}
	return t.duration()
}

func (t Transition) reverseCurve() animation.Curve {
	if t.ReverseCurve != nil {
		return t.ReverseCurve
	}
	return t.Curve
}

// Fade — переход, в котором маршрут проявляется и гаснет.
var Fade = Transition{Build: func(anim, _ animation.Animation, child Widget) Widget {
	return FadeTransition{Opacity: anim, Child: child}
}}

// Navigator — стопка маршрутов. Обычно его ставит App; свой Navigator нужен для отдельной стопки
// экранов внутри части интерфейса.
type Navigator struct {
	State
	Home       Widget     // первый экран
	Transition Transition // переход экранов по умолчанию; пустой — Fade

	routes []*routeEntry // снизу вверх, вместе с уходящими
}

// routeEntry — маршрут в стопке.
type routeEntry struct {
	Route
	nav        *Navigator
	ctrl       *animation.Controller // вход и уход: 0 — маршрута нет, 1 — на месте
	secondary  secondaryAnimation
	home       bool        // первый экран: содержимое — Navigator.Home
	popping    bool        // уходит
	replacedBy *routeEntry // заменён этим маршрутом (PushReplacement), уйдёт, когда тот доедет
}

// active — маршрут в стопке, а не уходит из неё.
func (e *routeEntry) active() bool { return !e.popping && e.replacedBy == nil }

// opaqueNow — экран на месте и закрывает всё под собой: нижние экраны можно не рисовать.
func (e *routeEntry) opaqueNow() bool {
	return !e.Popup && !e.popping && e.ctrl.Status() == animation.Completed
}

// secondaryAnimation — уход маршрута под следующий экран: это анимация входа того экрана.
type secondaryAnimation struct{ parent *animation.Controller }

func (s *secondaryAnimation) Value() float64 {
	if s.parent == nil {
		return 0
	}
	return s.parent.Value()
}

func (n *Navigator) Init(BuildContext) {
	home := n.newEntry(Route{})
	home.home = true
	home.ctrl.SetValue(1)
	n.routes = []*routeEntry{home}
	o := n.element.owner
	o.navigators = append(o.navigators, n)
}

func (n *Navigator) Dispose() {
	for _, e := range n.routes {
		e.ctrl.Dispose()
	}
	o := n.element.owner
	if i := slices.Index(o.navigators, n); i >= 0 {
		o.navigators = slices.Delete(o.navigators, i, i+1)
	}
}

func (n *Navigator) newEntry(r Route) *routeEntry {
	e := &routeEntry{Route: r, nav: n}
	o := n.element.owner
	e.ctrl = animation.NewController(o.scheduler, n.transitionOf(e).duration())
	// Боксы переходов берут значение сами — при каждом изменении нужен только новый кадр, не перестройка.
	e.ctrl.AddListener(o.requestFrame)
	e.ctrl.AddStatusListener(func(s animation.Status) { n.statusChanged(e, s) })
	return e
}

// transitionOf — как входит и уходит маршрут e.
func (n *Navigator) transitionOf(e *routeEntry) Transition {
	switch {
	case e.Transition.Build != nil:
		return e.Transition
	case !e.Popup && n.Transition.Build != nil:
		return n.Transition
	}
	return Fade
}

// Push кладёт маршрут поверх остальных.
func (n *Navigator) Push(r Route) {
	if r.Page == nil {
		panic("widgets: Push без содержимого: Route.Page = nil")
	}
	e := n.newEntry(r)
	n.routes = append(n.routes, e)
	n.relink()
	n.animate(e, 1)
	n.changed()
}

// PushReplacement кладёт маршрут вместо верхнего: прежний уходит из стопки, когда новый доедет
// (вход после входа в приложение: экран входа заменяется главным, и «Назад» к нему не вернёт).
func (n *Navigator) PushReplacement(r Route) {
	if r.Page == nil {
		panic("widgets: PushReplacement без содержимого: Route.Page = nil")
	}
	old := n.top()
	e := n.newEntry(r)
	if old != nil {
		old.replacedBy = e
	}
	n.routes = append(n.routes, e)
	n.relink()
	n.animate(e, 1)
	n.changed()
	if old != nil && old.OnPop != nil {
		old.OnPop(nil)
	}
}

// Pop закрывает верхний маршрут; result получит его OnPop. Первый экран не закрывается.
func (n *Navigator) Pop(result any) {
	if !n.CanPop() {
		return
	}
	n.pop(n.top(), result)
}

// PopToFirst закрывает всё до первого экрана: верхний маршрут уходит с анимацией, остальные — сразу.
func (n *Navigator) PopToFirst() {
	var active []*routeEntry
	for _, e := range n.routes {
		if e.active() {
			active = append(active, e)
		}
	}
	if len(active) < 2 {
		return
	}
	for _, e := range active[1 : len(active)-1] {
		e.popping = true
		n.remove(e)
		if e.OnPop != nil {
			e.OnPop(nil)
		}
	}
	n.pop(active[len(active)-1], nil)
}

// CanPop сообщает, есть ли что закрывать: над первым экраном есть маршруты.
func (n *Navigator) CanPop() bool {
	count := 0
	for _, e := range n.routes {
		if e.active() {
			count++
		}
	}
	return count > 1
}

// top — верхний маршрут, который не уходит.
func (n *Navigator) top() *routeEntry {
	for i := len(n.routes) - 1; i >= 0; i-- {
		if n.routes[i].active() {
			return n.routes[i]
		}
	}
	return nil
}

func (n *Navigator) pop(e *routeEntry, result any) {
	e.popping = true
	n.animate(e, 0)
	n.changed()
	if e.OnPop != nil {
		e.OnPop(result)
	}
}

// animate ведёт маршрут к target (1 — вход, 0 — уход) за ту часть длительности, что осталась.
func (n *Navigator) animate(e *routeEntry, target float64) {
	t := n.transitionOf(e)
	d, curve := t.duration(), t.Curve
	if target < e.ctrl.Value() {
		d, curve = t.reverseDuration(), t.reverseCurve()
	}
	left := e.ctrl.Value() - target
	if left < 0 {
		left = -left
	}
	e.ctrl.AnimateTo(target, max(time.Duration(float64(d)*left), 1), curve)
}

func (n *Navigator) statusChanged(e *routeEntry, s animation.Status) {
	switch {
	case s == animation.Dismissed && e.popping:
		n.remove(e) // ушёл
	case s == animation.Completed && e.active():
		for _, r := range slices.Clone(n.routes) {
			if r.replacedBy == e {
				n.remove(r) // новый экран доехал — заменённый больше не виден
			}
		}
	}
}

// remove убирает маршрут из стопки сразу (и те, что ждали, пока он доедет).
func (n *Navigator) remove(e *routeEntry) {
	i := slices.Index(n.routes, e)
	if i < 0 {
		return
	}
	n.routes = slices.Delete(n.routes, i, i+1)
	e.ctrl.Dispose()
	for _, r := range slices.Clone(n.routes) {
		if r.replacedBy == e {
			n.remove(r)
		}
	}
	n.relink()
	n.changed()
}

// relink связывает уход каждого маршрута со входом ближайшего экрана над ним.
func (n *Navigator) relink() {
	for i, e := range n.routes {
		e.secondary.parent = nil
		for _, above := range n.routes[i+1:] {
			if !above.Popup {
				e.secondary.parent = above.ctrl
				break
			}
		}
	}
}

func (n *Navigator) changed() {
	if n.Mounted() {
		n.SetState(nil)
	}
}

func (n *Navigator) Build(BuildContext) Widget {
	layers := make([]Widget, len(n.routes))
	activeBelow := false
	for i, e := range n.routes {
		page := e.Page
		if e.home {
			page = n.Home
		}
		layers[i] = Keyed{Key: e, Child: theatreLayer{opaque: e.opaqueNow, page: !e.Popup, Child: routeView{
			entry: e, first: !activeBelow, page: page, transition: n.transitionOf(e),
		}}}
		if e.active() {
			activeBelow = true
		}
	}
	return navigatorScope{nav: n, Child: theatre{Children: layers}}
}

// routeView — один маршрут: содержимое в переходе, у окна — ещё и затемнение под ним.
type routeView struct {
	Base
	entry      *routeEntry
	first      bool // под маршрутом нет других: возвращаться некуда
	page       Widget
	transition Transition
}

func (v routeView) Build(BuildContext) Widget {
	e := v.entry
	var child Widget = routeScope{entry: e, first: v.first, Child: v.page}
	if !e.Popup {
		// Пока идёт переход, экран рисуется один раз в снимок, а в кадрах снимок только едет.
		child = snapshot{active: e.nav.animating, Child: child}
	}
	child = v.transition.Build(e.ctrl, &e.secondary, child)
	if e.Popup {
		child = Stack{Fit: rendering.StackExpand, Children: []Widget{modalBarrier{entry: e}, child}}
	}
	// Уходящий маршрут касаний не получает: они достаются экрану, который под ним открывается.
	return IgnorePointer{Disabled: !e.popping, Child: child}
}

// modalBarrier — затемнение под окном: не пропускает касания к экрану, по касанию закрывает окно.
type modalBarrier struct {
	Base
	entry *routeEntry
}

func (b modalBarrier) Build(BuildContext) Widget {
	e := b.entry
	var tap func()
	if e.Dismissible {
		tap = func() {
			if e.active() && e.nav.top() == e {
				e.nav.pop(e, nil)
			}
		}
	}
	return GestureDetector{OnTap: tap, Behavior: rendering.Opaque,
		Child: SizedBoxExpand(FadeTransition{Opacity: e.ctrl, Child: ColoredBox{Color: e.Barrier}})}
}

// theatre — слои маршрутов: экран на месте закрывает те, что под ним.
type theatre struct {
	Base
	Children []Widget
}

func (w theatre) ChildWidgets() []Widget                       { return w.Children }
func (theatre) CreateRenderObject(BuildContext) rendering.Box  { return &rendering.TheatreBox{} }
func (theatre) UpdateRenderObject(BuildContext, rendering.Box) {}

// theatreLayer сообщает слою, закрывает ли он сейчас всё под собой.
type theatreLayer struct {
	Base
	opaque func() bool
	page   bool
	Child  Widget
}

func (w theatreLayer) ChildWidget() Widget { return w.Child }
func (w theatreLayer) ApplyParentData(b rendering.Box) {
	rendering.SetParentData(b, &rendering.TheatreParentData{Opaque: w.opaque, Page: w.page})
}

// navigatorScope — Navigator для своего поддерева.
type navigatorScope struct {
	Base
	nav   *Navigator
	Child Widget
}

func (s navigatorScope) ChildWidget() Widget { return s.Child }
func (s navigatorScope) UpdateShouldNotify(old Widget) bool {
	return old.(navigatorScope).nav != s.nav
}

// routeScope — маршрут для своего содержимого.
type routeScope struct {
	Base
	entry *routeEntry
	first bool
	Child Widget
}

func (s routeScope) ChildWidget() Widget { return s.Child }
func (s routeScope) UpdateShouldNotify(old Widget) bool {
	o := old.(routeScope)
	return o.entry != s.entry || o.first != s.first
}

// find — ближайший предок-InheritedWidget типа T, без подписки на его изменения.
func find[T InheritedWidget](ctx BuildContext) (T, bool) {
	var zero T
	ie := ctx.element().inherited[reflect.TypeFor[T]()]
	if ie == nil {
		return zero, false
	}
	return ie.widget.(T), true
}

// NavigatorOf — ближайший Navigator над ctx.
func NavigatorOf(ctx BuildContext) *Navigator {
	s, ok := find[navigatorScope](ctx)
	if !ok {
		panic("widgets: над этим виджетом нет Navigator — его ставит widgets.App (и material.App)")
	}
	return s.nav
}

// Push открывает экран page поверх текущего.
func Push(ctx BuildContext, page Widget) { NavigatorOf(ctx).Push(Route{Page: page}) }

// PushRoute кладёт маршрут: экран со своим переходом, окно, экран с результатом (Route.OnPop).
func PushRoute(ctx BuildContext, r Route) { NavigatorOf(ctx).Push(r) }

// PushReplacement открывает экран page вместо текущего: вернуться к прежнему «Назад» нельзя.
func PushReplacement(ctx BuildContext, page Widget) {
	NavigatorOf(ctx).PushReplacement(Route{Page: page})
}

// Pop закрывает верхний маршрут: экран или окно.
func Pop(ctx BuildContext) { NavigatorOf(ctx).Pop(nil) }

// PopWith закрывает верхний маршрут с результатом — его получит Route.OnPop.
func PopWith(ctx BuildContext, result any) { NavigatorOf(ctx).Pop(result) }

// PopToFirst возвращает к первому экрану, закрывая всё над ним.
func PopToFirst(ctx BuildContext) { NavigatorOf(ctx).PopToFirst() }

// CanPop сообщает, есть ли куда вернуться с экрана ctx: под его маршрутом есть другой.
// Так AppBar решает, показывать ли стрелку «Назад».
func CanPop(ctx BuildContext) bool {
	s, ok := DependOn[routeScope](ctx)
	return ok && !s.first
}

// RouteAnimation — анимация входа маршрута, в котором стоит ctx (nil вне Navigator). Её можно вести
// самому: нижняя панель, которую тянут пальцем вниз, уменьшает значение, а отпущенная — закрывается
// (Pop) или возвращается (AnimateTo(1, ...)).
func RouteAnimation(ctx BuildContext) *animation.Controller {
	s, ok := find[routeScope](ctx)
	if !ok {
		return nil
	}
	return s.entry.ctrl
}

// animating — какой-то маршрут сейчас входит или уходит.
func (n *Navigator) animating() bool {
	for _, e := range n.routes {
		if e.ctrl.IsAnimating() {
			return true
		}
	}
	return false
}

// snapshot — экран рисуется снимком, пока active.
type snapshot struct {
	Base
	active func() bool
	Child  Widget
}

func (w snapshot) ChildWidget() Widget { return w.Child }
func (w snapshot) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.SnapshotBox{Active: w.active}
}
func (w snapshot) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	b.(*rendering.SnapshotBox).Active = w.active
}
