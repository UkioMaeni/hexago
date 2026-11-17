package widgets

import (
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/painting"
	"hexago/ui/rendering"
)

// Container — универсальный бокс, как во Flutter: размер, отступы внутри и снаружи,
// выравнивание ребёнка, цвет или украшение, ограничения, преобразование.
//
// Сам он ничего не рисует — он собирается из простых виджетов (Padding, Align,
// ColoredBox, DecoratedBox, ConstrainedBox, Transform) ровно в том порядке, что и во Flutter.
// Без ребёнка и без размера Container занимает всё доступное место.
type Container struct {
	Base
	Width                *float64
	Height               *float64
	Padding              geom.EdgeInsets // отступ внутри, между краем и ребёнком
	Margin               geom.EdgeInsets // отступ снаружи
	Alignment            *geom.Alignment
	Color                gfx.Color
	Decoration           *painting.BoxDecoration
	ForegroundDecoration *painting.BoxDecoration
	Constraints          *rendering.Constraints
	Transform            *geom.Affine
	Child                Widget
}

func (c Container) Build(BuildContext) Widget {
	current := c.Child
	if current == nil && (c.Constraints == nil || !c.Constraints.IsTight()) {
		// Пустой Container растягивается на всё место, но внутри списка (без ограничения) схлопывается.
		zero := 0.0
		current = LimitedBox{MaxWidth: &zero, MaxHeight: &zero, Child: ConstrainedBox{Constraints: rendering.Expand()}}
	}
	if c.Alignment != nil {
		current = Align{Alignment: *c.Alignment, Child: current}
	}
	if c.Padding != (geom.EdgeInsets{}) {
		current = Padding{Padding: c.Padding, Child: current}
	}
	if !c.Color.IsTransparent() {
		current = ColoredBox{Color: c.Color, Child: current}
	}
	if c.Decoration != nil {
		current = DecoratedBox{Decoration: *c.Decoration, Child: current}
	}
	if c.ForegroundDecoration != nil {
		current = DecoratedBox{Decoration: *c.ForegroundDecoration, Position: DecorationForeground, Child: current}
	}
	if cons, ok := c.constraints(); ok {
		current = ConstrainedBox{Constraints: cons, Child: current}
	}
	if c.Margin != (geom.EdgeInsets{}) {
		current = Padding{Padding: c.Margin, Child: current}
	}
	if c.Transform != nil {
		current = Transform{Transform: *c.Transform, Child: current}
	}
	return current
}

// constraints объединяет Width/Height и Constraints, как во Flutter:
// заданные размеры «затягивают» ограничения до точного значения.
func (c Container) constraints() (rendering.Constraints, bool) {
	if c.Width == nil && c.Height == nil {
		if c.Constraints != nil {
			return *c.Constraints, true
		}
		return rendering.Constraints{}, false
	}
	if c.Constraints == nil {
		return rendering.TightFor(c.Width, c.Height), true
	}
	cons := *c.Constraints
	if c.Width != nil {
		w := min(max(*c.Width, cons.MinWidth), cons.MaxWidth)
		cons.MinWidth, cons.MaxWidth = w, w
	}
	if c.Height != nil {
		h := min(max(*c.Height, cons.MinHeight), cons.MaxHeight)
		cons.MinHeight, cons.MaxHeight = h, h
	}
	return cons, true
}
