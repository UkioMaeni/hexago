package material

import (
	"math"
	"time"

	"hexago/ui/animation"
	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/painting"
	"hexago/ui/rendering"
	"hexago/ui/widgets"
)

// Drawer — боковая панель Material 3 (modal navigation drawer) со своим содержимым. Её отдают
// Scaffold.Drawer: верхняя панель ставит кнопку «меню», панель выдвигается слева над затемнением,
// закрывается касанием затемнения, кнопкой «Назад» и движением пальца влево. Ширина — 360 dp
// (на узком экране — до 56 dp от правого края). Панель с пунктами навигации — NavigationDrawer.
type Drawer struct {
	widgets.Base
	Child widgets.Widget
}

func (d Drawer) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	mq := widgets.MediaQueryOf(ctx)
	r := geom.Circular(t.Shapes.Large)
	return widgets.SizedBox{Width: new(drawerWidth(mq)), Height: &widgets.Infinity, Child: widgets.DecoratedBox{
		Decoration: painting.BoxDecoration{Color: t.Colors.SurfaceContainerLow, BorderRadius: geom.BorderRadius{TopRight: r, BottomRight: r}},
		Child: widgets.Padding{Padding: geom.EdgeInsets{Left: mq.Padding.Left, Top: mq.Padding.Top, Bottom: mq.Padding.Bottom},
			Child: d.Child},
	}}
}

// drawerWidth — ширина боковой панели на этом экране.
func drawerWidth(mq widgets.MediaQueryData) float64 {
	return math.Max(0, math.Min(360, mq.Size.Width-56)) + mq.Padding.Left
}

// NavigationDrawer — боковая панель с пунктами навигации Material 3: у выбранного пункта —
// «пилюля» SecondaryContainer. Выбор пункта вызывает OnSelected и закрывает панель.
//
//	Drawer: material.NavigationDrawer{
//		Header:   widgets.Text{Data: "Почта"},
//		Selected: c.folder,
//		Destinations: []material.NavigationDestination{
//			{Icon: icons.Mail, Label: "Входящие"}, {Icon: icons.Send, Label: "Отправленные"},
//		},
//		OnSelected: func(i int) { c.SetState(func() { c.folder = i }) },
//	},
type NavigationDrawer struct {
	widgets.Base
	Header       widgets.Widget // заголовок сверху; без стиля — TitleSmall
	Destinations []NavigationDestination
	Selected     int
	OnSelected   func(int)
}

func (n NavigationDrawer) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	c := t.Colors
	_, inDrawer := widgets.DependOn[drawerScope](ctx)
	kids := []widgets.Widget{}
	if n.Header != nil {
		s := t.Text.TitleSmall
		s.Color = c.OnSurfaceVariant
		kids = append(kids, widgets.Padding{Padding: geom.EdgeInsets{Left: 28, Top: 16, Right: 16, Bottom: 18},
			Child: widgets.DefaultTextStyle{Style: s, Child: n.Header}})
	}
	for i, d := range n.Destinations {
		selected := i == n.Selected
		icon, fg, bg := d.Icon, c.OnSurfaceVariant, gfx.Transparent
		label := t.Text.LabelLarge
		if selected {
			fg, bg = c.OnSecondaryContainer, c.SecondaryContainer
			if d.SelectedIcon != nil {
				icon = d.SelectedIcon
			}
		}
		label.Color = fg
		tap := func() {
			if n.OnSelected != nil {
				n.OnSelected(i)
			}
			if inDrawer {
				widgets.Pop(ctx) // в выдвижной панели выбор закрывает её
			}
		}
		kids = append(kids, widgets.Padding{Padding: geom.Symmetric(12, 0), Child: Material{Color: bg, Radius: t.Shapes.Full,
			Child: InkWell{Radius: t.Shapes.Full, OnTap: tap, Child: widgets.SizedBox{Height: new(56.0), Child: widgets.Padding{
				Padding: geom.EdgeInsets{Left: 16, Right: 24},
				Child: widgets.Row{Spacing: 12, Children: []widgets.Widget{
					widgets.Icon{Icon: icon, Size: 24, Color: fg},
					widgets.Expanded{Child: widgets.Text{Data: d.Label, Style: label}},
				}},
			}}},
		}})
	}
	return Drawer{Child: widgets.SingleChildScrollView{Padding: geom.EdgeInsets{Top: 12, Bottom: 12},
		Child: widgets.Column{CrossAxisAlignment: rendering.CrossAxisStretch, Children: kids}}}
}

// OpenDrawer выдвигает боковую панель Scaffold, внутри которого стоит ctx.
func OpenDrawer(ctx widgets.BuildContext) {
	s, ok := widgets.DependOn[scaffoldScope](ctx)
	if !ok || s.drawer == nil {
		panic("material: OpenDrawer вне Scaffold с Drawer")
	}
	widgets.PushRoute(ctx, drawerRoute(ctx, s.drawer))
}

// drawerRoute — маршрут боковой панели.
func drawerRoute(ctx widgets.BuildContext, drawer widgets.Widget) widgets.Route {
	return widgets.Route{
		Page:        drawerFrame{Child: drawer},
		Popup:       true,
		Barrier:     Of(ctx).Colors.Scrim.WithOpacity(ScrimOpacity),
		Dismissible: true,
		Transition:  drawerTransition,
	}
}

// drawerTransition — панель выезжает сама (drawerFrame): на долю своей ширины, чтобы идти за пальцем.
var drawerTransition = widgets.Transition{
	Duration:        300 * time.Millisecond,
	ReverseDuration: 250 * time.Millisecond,
	Curve:           animation.EmphasizedDecelerate,
	ReverseCurve:    animation.EmphasizedAccelerate,
	Build:           func(_, _ animation.Animation, child widgets.Widget) widgets.Widget { return child },
}

// Панель закрывается (или открывается) броском быстрее drawerFling dp/с, иначе — к ближайшему краю.
const drawerFling = 365.0

// drawerFrame ставит панель к левому краю и двигает её вместе с анимацией маршрута.
type drawerFrame struct {
	widgets.Base
	Child widgets.Widget
}

func (f drawerFrame) Build(ctx widgets.BuildContext) widgets.Widget {
	return drawerScope{Child: widgets.Align{Alignment: geom.CenterLeft, Child: widgets.SlideTransition{
		Position: widgets.RouteAnimation(ctx), Begin: geom.Offset{X: -1}, Child: drawerBody{Child: f.Child},
	}}}
}

// drawerScope отмечает, что виджет — внутри выдвижной панели (выбор пункта её закрывает).
type drawerScope struct {
	widgets.Base
	Child widgets.Widget
}

func (s drawerScope) ChildWidget() widgets.Widget            { return s.Child }
func (s drawerScope) UpdateShouldNotify(widgets.Widget) bool { return false }

// drawerBody — панель, которую можно задвинуть пальцем.
type drawerBody struct {
	widgets.State
	Child widgets.Widget
}

func (b *drawerBody) Build(widgets.BuildContext) widgets.Widget {
	return widgets.GestureDetector{OnHorizontalDragUpdate: b.drag, OnHorizontalDragEnd: b.release, Child: b.Child}
}

func (b *drawerBody) drag(d gestures.DragUpdateDetails) {
	a := widgets.RouteAnimation(b.Context())
	if w := widgets.SizeOf(b.Context()).Width; a != nil && w > 0 {
		a.SetValue(a.Value() + d.PrimaryDelta/w)
	}
}

func (b *drawerBody) release(d gestures.DragEndDetails) {
	settleDrawer(b.Context(), widgets.RouteAnimation(b.Context()), d.PrimaryVelocity)
}

// settleDrawer — палец отпущен: панель доезжает до места или закрывается.
func settleDrawer(ctx widgets.BuildContext, a *animation.Controller, velocity float64) {
	if a == nil {
		return
	}
	if velocity < -drawerFling || (velocity < drawerFling && a.Value() < 0.5) {
		widgets.Pop(ctx)
		return
	}
	a.AnimateTo(1, time.Duration(float64(drawerTransition.Duration)*(1-a.Value()))+1, drawerTransition.Curve)
}

// drawerEdge — полоса у левого края Scaffold: движение пальца вправо выдвигает панель вслед за ним.
type drawerEdge struct {
	widgets.State
	Drawer widgets.Widget
	Width  float64

	open *animation.Controller // анимация выдвигаемой панели, пока палец на экране
}

func (e *drawerEdge) Build(widgets.BuildContext) widgets.Widget {
	return widgets.GestureDetector{
		Behavior:               rendering.Translucent, // нажатия на кнопки под полосой проходят к ним
		OnHorizontalDragStart:  e.start,
		OnHorizontalDragUpdate: e.drag,
		OnHorizontalDragEnd:    e.release,
		Child:                  widgets.SizedBox{Width: new(e.Width), Height: &widgets.Infinity},
	}
}

func (e *drawerEdge) start(gestures.DragStartDetails) {
	ctx := e.Context()
	e.open = widgets.NavigatorOf(ctx).Push(drawerRoute(ctx, e.Drawer))
	e.open.SetValue(0) // ведёт палец, а не анимация
}

func (e *drawerEdge) drag(d gestures.DragUpdateDetails) {
	if w := drawerWidth(widgets.MediaQueryOf(e.Context())); e.open != nil && w > 0 {
		e.open.SetValue(e.open.Value() + d.PrimaryDelta/w)
	}
}

func (e *drawerEdge) release(d gestures.DragEndDetails) {
	if e.open != nil {
		settleDrawer(e.Context(), e.open, d.PrimaryVelocity)
		e.open = nil
	}
}

// scaffoldScope — Scaffold для своих детей (AppBar узнаёт, есть ли боковая панель).
type scaffoldScope struct {
	widgets.Base
	drawer widgets.Widget
	Child  widgets.Widget
}

func (s scaffoldScope) ChildWidget() widgets.Widget { return s.Child }
func (s scaffoldScope) UpdateShouldNotify(old widgets.Widget) bool {
	return (old.(scaffoldScope).drawer == nil) != (s.drawer == nil)
}
