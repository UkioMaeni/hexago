package material

import (
	"time"

	"hexago/ui/animation"
	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/icons"
	"hexago/ui/painting"
	"hexago/ui/rendering"
	"hexago/ui/widgets"
)

// ScrimOpacity — непрозрачность затемнения под диалогом и нижней панелью (цвет — ColorScheme.Scrim).
const ScrimOpacity = 0.32

// AlertDialog — диалог Material 3: значок, заголовок, текст и кнопки действий справа внизу.
// Кнопки сами решают, что делать, и закрывают диалог:
//
//	material.ShowDialog(ctx, material.AlertDialog{
//		Title:   widgets.Text{Data: "Удалить заметку?"},
//		Content: widgets.Text{Data: "Её нельзя будет вернуть."},
//		Actions: []widgets.Widget{
//			material.TextButton{OnPressed: func() { widgets.Pop(ctx) }, Child: widgets.Text{Data: "Отмена"}},
//			material.TextButton{OnPressed: func() { widgets.Pop(ctx); n.delete() }, Child: widgets.Text{Data: "Удалить"}},
//		},
//	})
type AlertDialog struct {
	widgets.Base
	Icon    *icons.Data // значок над заголовком; с ним заголовок — по центру
	Title   widgets.Widget
	Content widgets.Widget
	Actions []widgets.Widget // обычно TextButton
}

func (d AlertDialog) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	c := t.Colors
	var kids []widgets.Widget
	gap := func(h float64) { kids = append(kids, widgets.SizedBox{Height: new(h)}) }
	if d.Icon != nil {
		kids = append(kids, widgets.Center{Child: widgets.Icon{Icon: d.Icon, Size: 24, Color: c.Secondary}})
		if d.Title != nil || d.Content != nil {
			gap(16)
		}
	}
	if d.Title != nil {
		s := t.Text.HeadlineSmall
		s.Color = c.OnSurface
		var title widgets.Widget = widgets.DefaultTextStyle{Style: s, Child: d.Title}
		if d.Icon != nil {
			title = widgets.Center{Child: title}
		}
		kids = append(kids, title)
		if d.Content != nil {
			gap(16)
		}
	}
	if d.Content != nil {
		s := t.Text.BodyMedium
		s.Color = c.OnSurfaceVariant
		kids = append(kids, widgets.DefaultTextStyle{Style: s, Child: d.Content})
	}
	if len(d.Actions) > 0 {
		if len(kids) > 0 {
			gap(24)
		}
		kids = append(kids, widgets.Row{MainAxisAlignment: rendering.MainAxisEnd, Spacing: 8, Children: d.Actions})
	}
	return Dialog{Child: widgets.Padding{Padding: geom.All(24), Child: widgets.Column{
		MainAxisSize: rendering.MainAxisMin, CrossAxisAlignment: rendering.CrossAxisStretch, Children: kids,
	}}}
}

// Dialog — поверхность диалога Material 3 со своим содержимым: цвет SurfaceContainerHigh,
// скругление 28 dp, ширина от 280 до 560 dp. Диалог с заголовком и кнопками — AlertDialog.
type Dialog struct {
	widgets.Base
	Child widgets.Widget
}

func (d Dialog) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	return widgets.ConstrainedBox{
		Constraints: rendering.Constraints{MinWidth: 280, MaxWidth: 560, MaxHeight: rendering.Inf},
		Child:       Material{Color: t.Colors.SurfaceContainerHigh, Radius: t.Shapes.ExtraLarge, Clip: true, Child: d.Child},
	}
}

// ShowDialog показывает диалог поверх экрана: по центру, над затемнением. Закрывается кнопкой
// диалога (widgets.Pop), касанием затемнения и кнопкой «Назад».
func ShowDialog(ctx widgets.BuildContext, dialog widgets.Widget) {
	widgets.PushRoute(ctx, DialogRoute(ctx, dialog))
}

// DialogRoute — маршрут диалога, как его показывает ShowDialog. Нужен, чтобы поменять что-то
// перед показом: узнать, что диалог закрыли (OnPop), запретить закрытие касанием затемнения.
func DialogRoute(ctx widgets.BuildContext, dialog widgets.Widget) widgets.Route {
	return widgets.Route{
		Page:        dialogFrame{Child: dialog},
		Popup:       true,
		Barrier:     Of(ctx).Colors.Scrim.WithOpacity(ScrimOpacity),
		Dismissible: true,
		Transition:  dialogTransition,
	}
}

// dialogTransition — диалог и затемнение проявляются.
var dialogTransition = widgets.Transition{
	Duration:        200 * time.Millisecond,
	ReverseDuration: 150 * time.Millisecond,
	Curve:           animation.EaseOut,
	Build:           widgets.Fade.Build,
}

// dialogFrame ставит диалог по центру, не ближе 40 dp к бокам и 24 dp к верху и низу экрана.
type dialogFrame struct {
	widgets.Base
	Child widgets.Widget
}

func (f dialogFrame) Build(ctx widgets.BuildContext) widgets.Widget {
	inset := widgets.MediaQueryOf(ctx).Padding.Add(geom.Symmetric(40, 24))
	return widgets.Padding{Padding: inset, Child: widgets.Center{Child: f.Child}}
}

// ShowBottomSheet показывает нижнюю панель Material 3 (modal bottom sheet): она выезжает снизу над
// затемнением, высотой по содержимому (не выше 9/16 экрана). Закрывается движением пальца вниз,
// касанием затемнения, кнопкой «Назад» и widgets.Pop.
func ShowBottomSheet(ctx widgets.BuildContext, content widgets.Widget) {
	widgets.PushRoute(ctx, BottomSheetRoute(ctx, content))
}

// BottomSheetRoute — маршрут нижней панели, как её показывает ShowBottomSheet.
func BottomSheetRoute(ctx widgets.BuildContext, content widgets.Widget) widgets.Route {
	return widgets.Route{
		Page:        bottomSheet{Content: content},
		Popup:       true,
		Barrier:     Of(ctx).Colors.Scrim.WithOpacity(ScrimOpacity),
		Dismissible: true,
		Transition:  sheetTransition,
	}
}

// sheetTransition — панель выезжает сама (bottomSheet): на долю своей высоты, а не экрана,
// чтобы следовать за пальцем. Затемнение проявляется с той же анимацией.
var sheetTransition = widgets.Transition{
	Duration:        300 * time.Millisecond,
	ReverseDuration: 250 * time.Millisecond,
	Curve:           animation.EmphasizedDecelerate,
	ReverseCurve:    animation.EmphasizedAccelerate,
	Build:           func(_, _ animation.Animation, child widgets.Widget) widgets.Widget { return child },
}

// bottomSheet — нижняя панель внизу экрана.
type bottomSheet struct {
	widgets.Base
	Content widgets.Widget
}

func (s bottomSheet) Build(ctx widgets.BuildContext) widgets.Widget {
	mq := widgets.MediaQueryOf(ctx)
	return widgets.Align{Alignment: geom.BottomCenter, Child: widgets.ConstrainedBox{
		Constraints: rendering.Constraints{MaxWidth: 640, MaxHeight: mq.Size.Height * 9 / 16},
		Child: widgets.SlideTransition{Position: widgets.RouteAnimation(ctx), Begin: geom.Offset{Y: 1},
			Child: sheetBody{Content: s.Content}},
	}}
}

// sheetBody — сама панель: ручка сверху, содержимое, перетаскивание вниз.
type sheetBody struct {
	widgets.State
	Content widgets.Widget
}

// Панель закрывается, если её бросили вниз быстрее minFlingVelocity или отпустили ниже середины.
const minFlingVelocity = 700.0

func (s *sheetBody) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	c := t.Colors
	r := geom.Circular(t.Shapes.ExtraLarge)
	handle := widgets.Padding{Padding: geom.Symmetric(0, 22), Child: widgets.Center{Child: widgets.Container{
		Width: new(32.0), Height: new(4.0),
		Decoration: &painting.BoxDecoration{Color: c.OnSurfaceVariant.WithOpacity(0.4), BorderRadius: geom.RadiusAll(2)},
	}}}
	bottom := widgets.MediaQueryOf(ctx).Padding.Bottom // панель уходит под панель навигации системы
	return widgets.GestureDetector{
		OnVerticalDragUpdate: s.drag,
		OnVerticalDragEnd:    s.release,
		Child: widgets.DecoratedBox{
			Decoration: painting.BoxDecoration{Color: c.SurfaceContainerLow, BorderRadius: geom.BorderRadius{TopLeft: r, TopRight: r}},
			Child: widgets.Padding{Padding: geom.EdgeInsets{Bottom: bottom}, Child: widgets.Column{
				MainAxisSize: rendering.MainAxisMin, CrossAxisAlignment: rendering.CrossAxisStretch,
				Children: []widgets.Widget{handle, widgets.Flexible{Child: s.Content}},
			}},
		},
	}
}

// drag — панель едет за пальцем: значение анимации входа — доля видимой высоты.
func (s *sheetBody) drag(d gestures.DragUpdateDetails) {
	a := widgets.RouteAnimation(s.Context())
	if h := widgets.SizeOf(s.Context()).Height; a != nil && h > 0 {
		a.SetValue(a.Value() - d.PrimaryDelta/h)
	}
}

// release — палец отпущен: закрыть или вернуть панель на место.
func (s *sheetBody) release(d gestures.DragEndDetails) {
	a := widgets.RouteAnimation(s.Context())
	if a == nil {
		return
	}
	if d.PrimaryVelocity > minFlingVelocity || (d.PrimaryVelocity > -minFlingVelocity && a.Value() < 0.5) {
		widgets.Pop(s.Context())
		return
	}
	a.AnimateTo(1, time.Duration(float64(sheetTransition.Duration)*(1-a.Value()))+1, sheetTransition.Curve)
}
