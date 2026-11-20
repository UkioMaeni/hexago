package material

import (
	"math"
	"time"

	"hexago/ui/animation"
	"hexago/ui/geom"
	"hexago/ui/icons"
	"hexago/ui/rendering"
	"hexago/ui/text"
	"hexago/ui/widgets"
)

// MenuButton — кнопка со значком (по умолчанию «⋮»), которая открывает меню Items под собой.
//
//	material.AppBar{Actions: []widgets.Widget{material.MenuButton{Items: []widgets.Widget{
//		material.MenuItem{Leading: icons.Share, Label: "Поделиться", OnTap: n.share},
//		material.Divider{},
//		material.MenuItem{Leading: icons.Delete, Label: "Удалить", OnTap: n.delete},
//	}}}}
type MenuButton struct {
	widgets.Base
	Icon  *icons.Data // nil — MoreVert
	Items []widgets.Widget
}

func (b MenuButton) Build(ctx widgets.BuildContext) widgets.Widget {
	icon := b.Icon
	if icon == nil {
		icon = icons.MoreVert
	}
	return IconButton{Icon: icon, OnPressed: func() { ShowMenu(ctx, b.Items) }}
}

// MenuItem — пункт меню: значок, подпись и подсказка справа (например, «Ctrl+C»). Нажатие закрывает
// меню и вызывает OnTap. Без OnTap пункт выключен.
type MenuItem struct {
	widgets.Base
	Leading  *icons.Data
	Label    string
	Trailing string
	OnTap    func()
}

func (m MenuItem) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	c := t.Colors
	label, hint := t.Text.LabelLarge, t.Text.LabelLarge
	label.Color, hint.Color = c.OnSurface, c.OnSurfaceVariant
	iconColor := c.OnSurfaceVariant
	var tap func()
	if m.OnTap != nil {
		tap = func() {
			widgets.Pop(ctx) // сначала закрыть меню: OnTap может открыть новый экран
			m.OnTap()
		}
	} else {
		label.Color = label.Color.WithOpacity(DisabledContent)
		hint.Color, iconColor = label.Color, label.Color
	}
	row := []widgets.Widget{}
	if m.Leading != nil {
		row = append(row, widgets.Icon{Icon: m.Leading, Size: 24, Color: iconColor})
	}
	row = append(row, widgets.Expanded{Child: widgets.Text{Data: m.Label, Style: label}})
	if m.Trailing != "" {
		row = append(row, widgets.Text{Data: m.Trailing, Style: hint})
	}
	return InkWell{OnTap: tap, Child: widgets.SizedBox{Height: new(48.0), Child: widgets.Padding{
		Padding: geom.Symmetric(12, 0), Child: widgets.Row{Spacing: 12, Children: row},
	}}}
}

// width — сколько места пункту нужно по ширине (меню берёт ширину самого широкого пункта).
func (m MenuItem) width(t ThemeData) float64 {
	w := 24 + text.Layout(m.Label, t.Text.LabelLarge, math.Inf(1)).Width
	if m.Leading != nil {
		w += 24 + 12
	}
	if m.Trailing != "" {
		w += 12 + text.Layout(m.Trailing, t.Text.LabelLarge, math.Inf(1)).Width
	}
	return math.Ceil(w)
}

// ShowMenu открывает меню Material 3 у виджета ctx: под ним, а если внизу нет места — над ним;
// по горизонтали — от его левого края или, у правого края экрана, до его правого края.
// Закрывается выбором пункта, касанием мимо меню и кнопкой «Назад».
func ShowMenu(ctx widgets.BuildContext, items []widgets.Widget) {
	widgets.PushRoute(ctx, widgets.Route{
		Page:        menuFrame{anchor: widgets.BoundsOf(ctx), items: items},
		Popup:       true,
		Dismissible: true,
		Transition:  menuTransition,
	})
}

// menuTransition — меню проявляется и раскрывается сверху вниз (раскрытие — в menuLayoutBox).
var menuTransition = widgets.Transition{
	Duration:        250 * time.Millisecond,
	ReverseDuration: 120 * time.Millisecond,
	Curve:           animation.EmphasizedDecelerate,
	Build:           widgets.Fade.Build,
}

// menuFrame — меню на своём месте у кнопки.
type menuFrame struct {
	widgets.Base
	anchor geom.Rect
	items  []widgets.Widget
}

func (f menuFrame) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	width := 112.0 // ширина меню — по самому широкому пункту, от 112 до 280 dp
	for _, it := range f.items {
		if m, ok := it.(MenuItem); ok {
			width = math.Max(width, m.width(t))
		}
	}
	width = math.Min(width, 280)
	mq := widgets.MediaQueryOf(ctx)
	return menuLayout{
		anchor: f.anchor,
		margin: mq.Padding.Add(geom.All(8)),
		reveal: widgets.RouteAnimation(ctx),
		Child: widgets.SizedBox{Width: new(width), Child: Material{
			Color: t.Colors.SurfaceContainer, Radius: t.Shapes.ExtraSmall, Elevation: t.Elevation.Level2, Clip: true,
			Child: widgets.Padding{Padding: geom.Symmetric(0, 8), Child: widgets.Column{
				MainAxisSize: rendering.MainAxisMin, CrossAxisAlignment: rendering.CrossAxisStretch, Children: f.items,
			}},
		}},
	}
}

// menuLayout ставит меню у прямоугольника anchor (не ближе margin к краям экрана) и раскрывает его.
type menuLayout struct {
	widgets.Base
	anchor geom.Rect
	margin geom.EdgeInsets
	reveal animation.Animation
	Child  widgets.Widget
}

func (w menuLayout) ChildWidget() widgets.Widget { return w.Child }
func (w menuLayout) CreateRenderObject(widgets.BuildContext) rendering.Box {
	return &menuLayoutBox{anchor: w.anchor, margin: w.margin, reveal: w.reveal}
}
func (w menuLayout) UpdateRenderObject(_ widgets.BuildContext, b rendering.Box) {
	m := b.(*menuLayoutBox)
	m.anchor, m.margin, m.reveal = w.anchor, w.margin, w.reveal
}

// menuLayoutBox — бокс во весь экран: меню (ребёнок) стоит у anchor, а видна его часть сверху
// (снизу, если меню открылось вверх) — по значению анимации reveal.
type menuLayoutBox struct {
	rendering.Proxy
	anchor geom.Rect
	margin geom.EdgeInsets
	reveal animation.Animation
	up     bool    // меню над кнопкой
	shown  float64 // какая доля была видна в прошлом кадре
}

func (b *menuLayoutBox) PerformLayout(c rendering.Constraints) geom.Size {
	size := c.Biggest()
	ch := rendering.Child(b)
	if ch == nil {
		return size
	}
	m := b.margin
	s := rendering.Layout(ch, rendering.Constraints{
		MaxWidth: math.Max(0, size.Width-m.Horizontal()), MaxHeight: math.Max(0, size.Height-m.Vertical()),
	})
	x := b.anchor.Left
	if x+s.Width > size.Width-m.Right {
		x = b.anchor.Right - s.Width // не помещается вправо — выравнивание по правому краю кнопки
	}
	x = math.Max(m.Left, math.Min(x, size.Width-m.Right-s.Width))
	y := b.anchor.Bottom
	b.up = false
	if y+s.Height > size.Height-m.Bottom {
		if b.anchor.Top-s.Height >= m.Top {
			y, b.up = b.anchor.Top-s.Height, true
		} else {
			y = math.Max(m.Top, size.Height-m.Bottom-s.Height)
		}
	}
	rendering.SetOffset(ch, geom.Offset{X: x, Y: y})
	return size
}

func (b *menuLayoutBox) value() float64 {
	if b.reveal == nil {
		return 1
	}
	return math.Max(0, math.Min(1, b.reveal.Value()))
}

func (b *menuLayoutBox) Paint(ctx *rendering.PaintContext, offset geom.Offset) {
	ch := rendering.Child(b)
	if ch == nil {
		return
	}
	r := geom.RectFromSize(offset.Add(rendering.OffsetOf(ch)), rendering.SizeOf(ch)).Inflate(16) // с тенью
	h := r.Height() * b.value()
	if b.up {
		r.Top = r.Bottom - h
	} else {
		r.Bottom = r.Top + h
	}
	ctx.Canvas.Save()
	ctx.Canvas.ClipRect(r)
	ctx.PaintChild(ch, offset)
	ctx.Canvas.Restore()
}

// PaintChanged — раскрытие изменилось с прошлого кадра.
func (b *menuLayoutBox) PaintChanged() bool {
	v := b.value()
	changed := v != b.shown
	b.shown = v
	return changed
}
