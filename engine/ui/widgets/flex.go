package widgets

import (
	"hexago/ui/geom"
	"hexago/ui/rendering"
)

// Row раскладывает детей в ряд слева направо.
type Row struct {
	Base
	MainAxisAlignment  rendering.MainAxisAlignment
	MainAxisSize       rendering.MainAxisSize
	CrossAxisAlignment rendering.CrossAxisAlignment
	Spacing            float64 // зазор между детьми
	Children           []Widget
}

func (w Row) ChildWidgets() []Widget { return w.Children }
func (w Row) CreateRenderObject(ctx BuildContext) rendering.Box {
	b := &rendering.FlexBox{Direction: rendering.Horizontal}
	w.UpdateRenderObject(ctx, b)
	return b
}
func (w Row) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	f := b.(*rendering.FlexBox)
	f.MainAxisAlignment, f.MainAxisSize, f.CrossAxisAlignment, f.Spacing =
		w.MainAxisAlignment, w.MainAxisSize, w.CrossAxisAlignment, w.Spacing
}

// Column раскладывает детей в столбец сверху вниз.
type Column struct {
	Base
	MainAxisAlignment  rendering.MainAxisAlignment
	MainAxisSize       rendering.MainAxisSize
	CrossAxisAlignment rendering.CrossAxisAlignment
	Spacing            float64
	Children           []Widget
}

func (w Column) ChildWidgets() []Widget { return w.Children }
func (w Column) CreateRenderObject(ctx BuildContext) rendering.Box {
	b := &rendering.FlexBox{Direction: rendering.Vertical}
	w.UpdateRenderObject(ctx, b)
	return b
}
func (w Column) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	f := b.(*rendering.FlexBox)
	f.MainAxisAlignment, f.MainAxisSize, f.CrossAxisAlignment, f.Spacing =
		w.MainAxisAlignment, w.MainAxisSize, w.CrossAxisAlignment, w.Spacing
}

// Flex — ряд или столбец с направлением Direction.
type Flex struct {
	Base
	Direction          rendering.Axis
	MainAxisAlignment  rendering.MainAxisAlignment
	MainAxisSize       rendering.MainAxisSize
	CrossAxisAlignment rendering.CrossAxisAlignment
	Spacing            float64
	Children           []Widget
}

func (w Flex) ChildWidgets() []Widget { return w.Children }
func (w Flex) CreateRenderObject(ctx BuildContext) rendering.Box {
	b := &rendering.FlexBox{}
	w.UpdateRenderObject(ctx, b)
	return b
}
func (w Flex) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	f := b.(*rendering.FlexBox)
	f.Direction, f.MainAxisAlignment, f.MainAxisSize, f.CrossAxisAlignment, f.Spacing =
		w.Direction, w.MainAxisAlignment, w.MainAxisSize, w.CrossAxisAlignment, w.Spacing
}

// Expanded растягивает ребёнка Row/Column на свою долю свободного места.
// Flex — доля (по умолчанию 1): при Flex 2 и Flex 1 ребёнок получит две трети.
type Expanded struct {
	Base
	Flex  int
	Child Widget
}

func (w Expanded) ChildWidget() Widget { return w.Child }
func (w Expanded) ApplyParentData(b rendering.Box) {
	setFlex(b, w.Flex, rendering.FlexTight)
}

// Flexible — как Expanded, но ребёнок может быть меньше своей доли (Fit: FlexLoose по умолчанию).
type Flexible struct {
	Base
	Flex  int
	Fit   rendering.FlexFit
	Child Widget
}

func (w Flexible) ChildWidget() Widget { return w.Child }
func (w Flexible) ApplyParentData(b rendering.Box) {
	fit := w.Fit
	if fit == rendering.FlexTight {
		fit = rendering.FlexLoose // нулевое значение — по умолчанию Flexible свободный
	}
	setFlex(b, w.Flex, fit)
}

func setFlex(b rendering.Box, flex int, fit rendering.FlexFit) {
	if flex <= 0 {
		flex = 1
	}
	pd, ok := parentData(b).(*rendering.FlexParentData)
	if !ok {
		pd = &rendering.FlexParentData{}
		setParentData(b, pd)
	}
	pd.Flex, pd.Fit = flex, fit
}

// Spacer — пустое растягивающееся место между детьми Row/Column.
type Spacer struct {
	Base
	Flex int
}

func (w Spacer) Build(BuildContext) Widget {
	return Expanded{Flex: w.Flex, Child: SizedBoxShrink()}
}

// Stack кладёт детей друг на друга. Alignment — куда ставить обычных детей
// (nil — левый верхний угол, как во Flutter); Positioned-дети ставятся по краям.
type Stack struct {
	Base
	Alignment *geom.Alignment
	Fit       rendering.StackFit
	Clip      bool
	Children  []Widget
}

func (w Stack) ChildWidgets() []Widget { return w.Children }
func (w Stack) CreateRenderObject(ctx BuildContext) rendering.Box {
	b := &rendering.StackBox{}
	w.UpdateRenderObject(ctx, b)
	return b
}
func (w Stack) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	s := b.(*rendering.StackBox)
	s.Alignment = geom.TopLeft
	if w.Alignment != nil {
		s.Alignment = *w.Alignment
	}
	s.Fit, s.Clip = w.Fit, w.Clip
}

// Positioned ставит ребёнка Stack по расстояниям до краёв и/или размерам. nil — не задано.
type Positioned struct {
	Base
	Left, Top, Right, Bottom *float64
	Width, Height            *float64
	Child                    Widget
}

func (w Positioned) ChildWidget() Widget { return w.Child }
func (w Positioned) ApplyParentData(b rendering.Box) {
	setParentData(b, &rendering.StackParentData{
		Left: w.Left, Top: w.Top, Right: w.Right, Bottom: w.Bottom, Width: w.Width, Height: w.Height,
	})
}

// PositionedFill растягивает ребёнка Stack на весь Stack.
func PositionedFill(child Widget) Widget {
	zero := 0.0
	return Positioned{Left: &zero, Top: &zero, Right: &zero, Bottom: &zero, Child: child}
}

// Wrap раскладывает детей в ряд и переносит на новую строку, когда место кончается.
type Wrap struct {
	Base
	Direction  rendering.Axis
	Spacing    float64
	RunSpacing float64
	Alignment  rendering.MainAxisAlignment
	Children   []Widget
}

func (w Wrap) ChildWidgets() []Widget { return w.Children }
func (w Wrap) CreateRenderObject(ctx BuildContext) rendering.Box {
	b := &rendering.WrapBox{}
	w.UpdateRenderObject(ctx, b)
	return b
}
func (w Wrap) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	r := b.(*rendering.WrapBox)
	r.Direction, r.Spacing, r.RunSpacing, r.Alignment = w.Direction, w.Spacing, w.RunSpacing, w.Alignment
}

func parentData(b rendering.Box) any { return rendering.ParentDataOf(b) }

func setParentData(b rendering.Box, pd any) { rendering.SetParentData(b, pd) }
