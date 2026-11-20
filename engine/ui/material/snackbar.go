package material

import (
	"time"

	"hexago/ui/animation"
	"hexago/ui/geom"
	"hexago/ui/icons"
	"hexago/ui/rendering"
	"hexago/ui/widgets"
)

// SnackBar — короткое сообщение внизу экрана Material 3: «Заметка удалена · Отменить».
// Показывается поверх содержимого Scaffold, над нижней панелью, и уходит само через Duration.
//
//	material.ShowSnackBar(ctx, material.SnackBar{
//		Content: widgets.Text{Data: "Заметка удалена"},
//		Action:  &material.SnackBarAction{Label: "Отменить", OnPressed: n.undo},
//	})
type SnackBar struct {
	Content   widgets.Widget  // обычно widgets.Text
	Action    *SnackBarAction // кнопка справа
	ShowClose bool            // крестик «закрыть»
	Duration  time.Duration   // сколько показывать; 0 — 4 секунды
}

// SnackBarAction — кнопка в SnackBar. Нажатие закрывает сообщение и вызывает OnPressed.
type SnackBarAction struct {
	Label     string
	OnPressed func()
}

// Сколько SnackBar появляется, уходит и висит по умолчанию.
const (
	snackEnter    = 250 * time.Millisecond
	snackExit     = 150 * time.Millisecond
	snackDuration = 4 * time.Second
)

// ShowSnackBar показывает сообщение. Если другое ещё на экране, новое ждёт своей очереди.
// SnackBar принадлежит приложению, а не экрану: при переходе на другой экран он остаётся.
func ShowSnackBar(ctx widgets.BuildContext, bar SnackBar) { messengerOf(ctx).show(bar) }

// HideSnackBar убирает сообщение, которое сейчас на экране (следующее в очереди появится за ним).
func HideSnackBar(ctx widgets.BuildContext) { messengerOf(ctx).hide() }

// messenger — очередь сообщений приложения, как ScaffoldMessenger во Flutter. Его ставит App над
// Navigator. Сам он не перестраивается (иначе перестраивалось бы всё приложение): об изменениях
// узнают нижние области Scaffold, которые на него подписаны.
type messenger struct {
	widgets.State
	Child widgets.Widget

	queue   []SnackBar
	current *snackEntry
	hosts   map[*scaffoldBottom]bool
	stop    func() // отмена таймера: когда уйти текущему сообщению
}

// snackEntry — сообщение на экране.
type snackEntry struct {
	bar    SnackBar
	hiding bool // уходит
}

func (m *messenger) Init(widgets.BuildContext) { m.hosts = map[*scaffoldBottom]bool{} }

func (m *messenger) Build(widgets.BuildContext) widgets.Widget {
	return messengerScope{m: m, Child: m.Child}
}

func (m *messenger) show(bar SnackBar) {
	m.queue = append(m.queue, bar)
	if m.current == nil {
		m.next()
	}
}

// next показывает следующее сообщение из очереди (или ничего).
func (m *messenger) next() {
	m.current = nil
	if len(m.queue) > 0 {
		m.current = &snackEntry{bar: m.queue[0]}
		m.queue = m.queue[1:]
		d := m.current.bar.Duration
		if d <= 0 {
			d = snackDuration
		}
		m.stop = widgets.AfterFunc(m.Context(), snackEnter+d, m.hide)
	}
	m.notify()
}

func (m *messenger) hide() {
	if m.current == nil || m.current.hiding {
		return
	}
	if m.stop != nil {
		m.stop()
	}
	m.current.hiding = true
	m.notify()
	m.stop = widgets.AfterFunc(m.Context(), snackExit, m.next)
}

func (m *messenger) notify() {
	for h := range m.hosts {
		if h.Mounted() {
			h.sync()
		}
	}
}

// messengerScope — очередь сообщений для поддерева.
type messengerScope struct {
	widgets.Base
	m     *messenger
	Child widgets.Widget
}

func (s messengerScope) ChildWidget() widgets.Widget { return s.Child }
func (s messengerScope) UpdateShouldNotify(old widgets.Widget) bool {
	return old.(messengerScope).m != s.m
}

func messengerOf(ctx widgets.BuildContext) *messenger {
	s, ok := widgets.DependOn[messengerScope](ctx)
	if !ok {
		panic("material: SnackBar показывают внутри material.App — очередь сообщений ставит он")
	}
	return s.m
}

// scaffoldBottom — нижний край Scaffold: плавающая кнопка и под ней SnackBar. Когда сообщение
// появляется, кнопка плавно поднимается над ним. Перестраивается только эта область.
type scaffoldBottom struct {
	widgets.State
	FAB         widgets.Widget
	Left, Right float64 // системные отступы по бокам

	m     *messenger
	shown *snackEntry
	anim  *animation.Controller
}

func (b *scaffoldBottom) Init(ctx widgets.BuildContext) {
	b.anim = widgets.NewAnimation(ctx, snackEnter)
	if s, ok := widgets.DependOn[messengerScope](ctx); ok {
		b.m = s.m
		b.m.hosts[b] = true
		if cur := b.m.current; cur != nil && !cur.hiding {
			b.shown = cur // экран открылся, когда сообщение уже было на экране
			b.anim.SetValue(1)
		}
	}
}

func (b *scaffoldBottom) Dispose() {
	if b.m != nil {
		delete(b.m.hosts, b)
	}
}

// sync — очередь сообщений изменилась.
func (b *scaffoldBottom) sync() {
	cur := b.m.current
	switch {
	case cur == nil:
		b.shown = nil
		b.anim.SetValue(0)
	case cur != b.shown:
		b.shown = cur
		b.anim.SetValue(0)
		b.anim.AnimateTo(1, snackEnter, animation.EmphasizedDecelerate)
	case cur.hiding:
		b.anim.AnimateTo(0, snackExit, animation.EmphasizedAccelerate)
	}
	b.SetState(nil)
}

func (b *scaffoldBottom) Build(widgets.BuildContext) widgets.Widget {
	var kids []widgets.Widget
	if b.FAB != nil {
		kids = append(kids, widgets.Padding{Padding: geom.EdgeInsets{Right: 16 + b.Right}, Child: b.FAB})
	}
	if b.shown != nil {
		v := b.anim.Value()
		top := 0.0
		if b.FAB != nil {
			top = 16
		}
		// Место под сообщение растёт вместе с анимацией — кнопка над ним плавно поднимается,
		// а само сообщение проявляется на своём месте.
		kids = append(kids, widgets.Align{Alignment: geom.BottomCenter, HeightFactor: &v, Child: widgets.Opacity{
			Opacity: v,
			Child: widgets.Padding{Padding: geom.EdgeInsets{Top: top, Left: 16 + b.Left, Right: 16 + b.Right},
				Child: snackBarView{bar: b.shown.bar, hide: b.m.hide}},
		}})
	}
	return widgets.Padding{Padding: geom.EdgeInsets{Bottom: 16}, Child: widgets.Column{
		MainAxisSize: rendering.MainAxisMin, CrossAxisAlignment: rendering.CrossAxisEnd, Children: kids,
	}}
}

// snackBarView — само сообщение: InverseSurface, скругление 4 dp, текст, кнопка, крестик.
type snackBarView struct {
	widgets.Base
	bar  SnackBar
	hide func()
}

func (v snackBarView) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	c := t.Colors
	text := t.Text.BodyMedium
	text.Color = c.OnInverseSurface
	row := []widgets.Widget{widgets.Expanded{Child: widgets.Padding{Padding: geom.Symmetric(0, 14),
		Child: widgets.DefaultTextStyle{Style: text, Child: v.bar.Content}}}}
	right := 16.0
	if a := v.bar.Action; a != nil {
		label := t.Text.LabelLarge
		label.Color = c.InversePrimary
		row = append(row, InkWell{Radius: t.Shapes.Full, OverlayColor: c.InversePrimary,
			OnTap: func() {
				v.hide()
				if a.OnPressed != nil {
					a.OnPressed()
				}
			},
			Child: widgets.Padding{Padding: geom.Symmetric(12, 10), Child: widgets.Text{Data: a.Label, Style: label}},
		})
		right = 8
	}
	if v.bar.ShowClose {
		row = append(row, InkWell{Radius: t.Shapes.Full, OverlayColor: c.OnInverseSurface, OnTap: v.hide,
			Child: widgets.SizedBox{Width: new(48.0), Height: new(48.0),
				Child: widgets.Center{Child: widgets.Icon{Icon: icons.Close, Size: 24, Color: c.OnInverseSurface}}}})
		right = 4
	}
	return Material{Color: c.InverseSurface, Radius: t.Shapes.ExtraSmall, Elevation: t.Elevation.Level3,
		Child: widgets.ConstrainedBox{Constraints: rendering.Constraints{MinHeight: 48, MaxWidth: rendering.Inf, MaxHeight: rendering.Inf},
			Child: widgets.Padding{Padding: geom.EdgeInsets{Left: 16, Right: right},
				Child: widgets.Row{Children: row}}}}
}
