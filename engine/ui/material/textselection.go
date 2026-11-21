package material

import (
	"math"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/icons"
	"hexago/ui/rendering"
	"hexago/ui/text"
	"hexago/ui/widgets"
)

// TextSelectionControls — ручки и меню выделения текста Material 3: ручки-капли цвета Primary
// и меню «Вырезать, Копировать, Вставить, Выбрать все» на поверхности. Их ставит TextField; своему
// EditableText их можно отдать через Controls.
type TextSelectionControls struct{}

const (
	handleSize  = 22.0 // диаметр капли
	handleTouch = 11.0 // запас вокруг ручки, чтобы её было легко взять пальцем
)

func (TextSelectionControls) Handle(ctx widgets.BuildContext, kind widgets.HandleKind) (widgets.Widget, geom.Size, geom.Offset) {
	col := Of(ctx).Colors.Primary
	r := handleSize / 2
	size := geom.Size{Width: handleSize, Height: handleSize}
	var anchor geom.Offset
	var paint func(c *gfx.Canvas)
	switch kind {
	case widgets.HandleLeft: // острый угол справа сверху — у начала выделения
		anchor = geom.Offset{X: handleSize}
		paint = func(c *gfx.Canvas) {
			c.DrawCircle(geom.Offset{X: r, Y: r}, r, gfx.Paint{Color: col})
			c.DrawRect(geom.Rect{Left: r, Right: handleSize, Bottom: r}, gfx.Paint{Color: col})
		}
	case widgets.HandleRight: // острый угол слева сверху — у конца
		paint = func(c *gfx.Canvas) {
			c.DrawCircle(geom.Offset{X: r, Y: r}, r, gfx.Paint{Color: col})
			c.DrawRect(geom.Rect{Right: r, Bottom: r}, gfx.Paint{Color: col})
		}
	default: // под курсором: остриём вверх
		d := r * math.Sqrt2
		size.Height = r + d
		anchor = geom.Offset{X: r}
		paint = func(c *gfx.Canvas) {
			center := geom.Offset{X: r, Y: d}
			c.DrawCircle(center, r, gfx.Paint{Color: col})
			var p gfx.Path
			p.MoveTo(r, 0)
			p.LineTo(r+d/2, d/2)
			p.LineTo(r, d)
			p.LineTo(r-d/2, d/2)
			p.Close()
			c.DrawPath(&p, gfx.Paint{Color: col})
		}
	}
	w := widgets.Padding{Padding: geom.All(handleTouch), Child: widgets.CustomPaint{Size: size,
		Painter: func(c *gfx.Canvas, _ geom.Size) { paint(c) }}}
	return w, geom.Size{Width: size.Width + 2*handleTouch, Height: size.Height + 2*handleTouch},
		anchor.Add(geom.Offset{X: handleTouch, Y: handleTouch})
}

func (TextSelectionControls) Toolbar(_ widgets.BuildContext, actions []widgets.SelectionAction) widgets.Widget {
	return selectionToolbar{Actions: actions}
}

// selectionToolbar — меню выделения: кнопки в ряд; что не помещается в ширину экрана, уходит под «⋮»
// и открывается списком.
type selectionToolbar struct {
	widgets.State
	Actions []widgets.SelectionAction
	more    bool // открыт список того, что не поместилось
}

func (s *selectionToolbar) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	c := t.Colors
	label := t.Text.LabelLarge
	label.Color = c.OnSurface
	const pad = 12.0
	avail := widgets.MediaQueryOf(ctx).Size.Width - 16
	widths := make([]float64, len(s.Actions))
	total := 0.0
	for i, a := range s.Actions {
		widths[i] = math.Ceil(text.Layout(a.Label, label, math.Inf(1)).Width) + 2*pad
		total += widths[i]
	}
	shown := len(s.Actions)
	if total > avail { // оставить место под «⋮»
		used := 48.0
		shown = 0
		for shown < len(s.Actions) && used+widths[shown] <= avail {
			used += widths[shown]
			shown++
		}
	}
	button := func(a widgets.SelectionAction, height float64) widgets.Widget {
		return InkWell{OnTap: a.Do, Child: widgets.SizedBox{Height: new(height), Child: widgets.Padding{
			Padding: geom.Symmetric(pad, 0), Child: widgets.Align{Alignment: geom.CenterLeft, WidthFactor: new(1.0),
				Child: widgets.Text{Data: a.Label, Style: label}},
		}}}
	}
	surface := func(child widgets.Widget) widgets.Widget {
		return Material{Color: c.SurfaceContainer, Radius: t.Shapes.Small, Elevation: t.Elevation.Level2, Clip: true, Child: child}
	}
	icon := func(ic *icons.Data, tap func()) widgets.Widget {
		return InkWell{OnTap: tap, Child: widgets.SizedBox{Width: new(48.0), Height: new(44.0),
			Child: widgets.Center{Child: widgets.Icon{Icon: ic, Size: 24, Color: c.OnSurfaceVariant}}}}
	}
	if s.more {
		items := []widgets.Widget{widgets.Align{Alignment: geom.CenterRight, Child: icon(icons.ArrowBack,
			func() { s.SetState(func() { s.more = false }) })}}
		for _, a := range s.Actions[shown:] {
			items = append(items, button(a, 48))
		}
		return surface(widgets.SizedBox{Width: new(200.0), Child: widgets.Column{MainAxisSize: rendering.MainAxisMin,
			CrossAxisAlignment: rendering.CrossAxisStretch, Children: items}})
	}
	row := []widgets.Widget{}
	for _, a := range s.Actions[:shown] {
		row = append(row, button(a, 44))
	}
	if shown < len(s.Actions) {
		row = append(row, icon(icons.MoreVert, func() { s.SetState(func() { s.more = true }) }))
	}
	return surface(widgets.Row{MainAxisSize: rendering.MainAxisMin, Children: row})
}
