package rendering

import (
	"hexago/ui/geom"
)

// StackFit — какие ограничения получают непозиционированные дети Stack.
type StackFit uint8

const (
	StackLoose       StackFit = iota // любой размер до размера Stack
	StackExpand                      // ровно размер Stack
	StackPassthrough                 // те же ограничения, что у Stack
)

// StackParentData — позиция ребёнка Stack (Positioned): расстояния до краёв и размеры.
// nil — не задано.
type StackParentData struct {
	Left, Top, Right, Bottom *float64
	Width, Height            *float64
}

func (p *StackParentData) positioned() bool {
	return p != nil && (p.Left != nil || p.Top != nil || p.Right != nil || p.Bottom != nil || p.Width != nil || p.Height != nil)
}

// StackBox кладёт детей друг на друга: обычных — по Alignment, Positioned — по краям.
// Размер — по самому большому обычному ребёнку (или всё место, если таких нет).
type StackBox struct {
	Node
	Alignment geom.Alignment
	Fit       StackFit
	Clip      bool // обрезать детей, выходящих за пределы
}

func stackData(ch Box) *StackParentData {
	pd, _ := ch.node().ParentData.(*StackParentData)
	return pd
}

func (b *StackBox) PerformLayout(c Constraints) geom.Size {
	var inner Constraints
	switch b.Fit {
	case StackLoose:
		inner = c.Loosen()
	case StackExpand:
		inner = Tight(c.Biggest())
	default:
		inner = c
	}

	hasNormal := false
	width, height := c.MinWidth, c.MinHeight
	for _, ch := range Children(b) {
		if stackData(ch).positioned() {
			continue
		}
		hasNormal = true
		s := Layout(ch, inner)
		width, height = max(width, s.Width), max(height, s.Height)
	}
	size := geom.Size{Width: width, Height: height}
	if !hasNormal {
		size = c.Biggest()
	}
	size = c.Constrain(size)

	for _, ch := range Children(b) {
		pd := stackData(ch)
		if !pd.positioned() {
			s := SizeOf(ch)
			SetOffset(ch, b.Alignment.Along(geom.Size{Width: size.Width - s.Width, Height: size.Height - s.Height}))
			continue
		}
		cc := Constraints{MaxWidth: Inf, MaxHeight: Inf}
		if pd.Left != nil && pd.Right != nil {
			w := max(0, size.Width-*pd.Right-*pd.Left)
			cc.MinWidth, cc.MaxWidth = w, w
		} else if pd.Width != nil {
			cc.MinWidth, cc.MaxWidth = *pd.Width, *pd.Width
		}
		if pd.Top != nil && pd.Bottom != nil {
			h := max(0, size.Height-*pd.Bottom-*pd.Top)
			cc.MinHeight, cc.MaxHeight = h, h
		} else if pd.Height != nil {
			cc.MinHeight, cc.MaxHeight = *pd.Height, *pd.Height
		}
		s := Layout(ch, cc)

		var x, y float64
		switch {
		case pd.Left != nil:
			x = *pd.Left
		case pd.Right != nil:
			x = size.Width - *pd.Right - s.Width
		default:
			x = b.Alignment.Along(geom.Size{Width: size.Width - s.Width}).X
		}
		switch {
		case pd.Top != nil:
			y = *pd.Top
		case pd.Bottom != nil:
			y = size.Height - *pd.Bottom - s.Height
		default:
			y = b.Alignment.Along(geom.Size{Height: size.Height - s.Height}).Y
		}
		SetOffset(ch, geom.Offset{X: x, Y: y})
	}
	return size
}

func (b *StackBox) Paint(ctx *PaintContext, offset geom.Offset) {
	if b.Clip {
		ctx.Canvas.Save()
		ctx.Canvas.ClipRect(geom.RectFromSize(offset, SizeOf(b)))
		defer ctx.Canvas.Restore()
	}
	PaintChildren(ctx, b, offset)
}

// WrapBox раскладывает детей в ряд и переносит на новую строку, когда место кончается.
type WrapBox struct {
	Node
	Direction  Axis
	Spacing    float64 // между детьми в строке
	RunSpacing float64 // между строками
	Alignment  MainAxisAlignment
}

func (b *WrapBox) PerformLayout(c Constraints) geom.Size {
	horizontal := b.Direction == Horizontal
	limit := c.MaxWidth
	childC := Constraints{MaxWidth: c.MaxWidth, MaxHeight: Inf}
	if !horizontal {
		limit = c.MaxHeight
		childC = Constraints{MaxWidth: Inf, MaxHeight: c.MaxHeight}
	}
	mainOf := func(s geom.Size) float64 {
		if horizontal {
			return s.Width
		}
		return s.Height
	}
	crossOf := func(s geom.Size) float64 {
		if horizontal {
			return s.Height
		}
		return s.Width
	}

	type run struct {
		children    []Box
		main, cross float64
	}
	var runs []run
	cur := run{}
	for _, ch := range Children(b) {
		s := Layout(ch, childC)
		need := mainOf(s)
		if len(cur.children) > 0 {
			need += b.Spacing
		}
		if len(cur.children) > 0 && cur.main+need > limit {
			runs = append(runs, cur)
			cur = run{}
			need = mainOf(s)
		}
		cur.children = append(cur.children, ch)
		cur.main += need
		cur.cross = max(cur.cross, crossOf(s))
	}
	if len(cur.children) > 0 {
		runs = append(runs, cur)
	}

	mainSize, crossSize := 0.0, 0.0
	for i, r := range runs {
		mainSize = max(mainSize, r.main)
		crossSize += r.cross
		if i > 0 {
			crossSize += b.RunSpacing
		}
	}
	var size geom.Size
	if horizontal {
		size = c.Constrain(geom.Size{Width: mainSize, Height: crossSize})
	} else {
		size = c.Constrain(geom.Size{Width: crossSize, Height: mainSize})
	}
	full := mainOf(size)

	crossPos := 0.0
	for _, r := range runs {
		free := max(0, full-r.main)
		pos, between := 0.0, b.Spacing
		switch b.Alignment {
		case MainAxisEnd:
			pos = free
		case MainAxisCenter:
			pos = free / 2
		case MainAxisSpaceBetween:
			if len(r.children) > 1 {
				between += free / float64(len(r.children)-1)
			}
		case MainAxisSpaceAround:
			between += free / float64(len(r.children))
			pos = (between - b.Spacing) / 2
		case MainAxisSpaceEvenly:
			between += free / float64(len(r.children)+1)
			pos = between - b.Spacing
		}
		for _, ch := range r.children {
			s := SizeOf(ch)
			if horizontal {
				SetOffset(ch, geom.Offset{X: pos, Y: crossPos})
			} else {
				SetOffset(ch, geom.Offset{X: crossPos, Y: pos})
			}
			pos += mainOf(s) + between
		}
		crossPos += r.cross + b.RunSpacing
	}
	return size
}

func (b *WrapBox) Paint(ctx *PaintContext, offset geom.Offset) { PaintChildren(ctx, b, offset) }
