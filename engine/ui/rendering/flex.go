package rendering

import (
	"hexago/ui/geom"
)

// Axis — направление: по горизонтали (Row) или по вертикали (Column).
type Axis uint8

const (
	Horizontal Axis = iota
	Vertical
)

// MainAxisAlignment — как распределить свободное место вдоль главной оси.
type MainAxisAlignment uint8

const (
	MainAxisStart        MainAxisAlignment = iota // прижать к началу
	MainAxisEnd                                   // прижать к концу
	MainAxisCenter                                // по центру
	MainAxisSpaceBetween                          // поровну между детьми, края без отступа
	MainAxisSpaceAround                           // поровну вокруг каждого (по краям — половина)
	MainAxisSpaceEvenly                           // поровну между детьми и по краям
)

// MainAxisSize — сколько места занять вдоль главной оси: всё (Max) или по детям (Min).
type MainAxisSize uint8

const (
	MainAxisMax MainAxisSize = iota
	MainAxisMin
)

// CrossAxisAlignment — как ставить детей поперёк главной оси.
type CrossAxisAlignment uint8

const (
	CrossAxisCenter  CrossAxisAlignment = iota // по центру (по умолчанию, как во Flutter)
	CrossAxisStart                             // к началу
	CrossAxisEnd                               // к концу
	CrossAxisStretch                           // растянуть на всю ширину/высоту
)

// FlexFit — должен ли гибкий ребёнок занять всю свою долю (Tight, Expanded)
// или может быть меньше (Loose, Flexible).
type FlexFit uint8

const (
	FlexTight FlexFit = iota
	FlexLoose
)

// FlexParentData — данные гибкого ребёнка: доля свободного места и как её занимать.
type FlexParentData struct {
	Flex int
	Fit  FlexFit
}

// FlexBox раскладывает детей в ряд (Row) или столбец (Column) — алгоритм Flutter RenderFlex:
//  1. Негибких детей раскладывают без ограничения по главной оси.
//  2. Оставшееся место делят между гибкими детьми (Expanded, Flexible) по их долям.
//  3. Свободное место распределяют по MainAxisAlignment, поперёк ставят по CrossAxisAlignment.
type FlexBox struct {
	Node
	Direction          Axis
	MainAxisAlignment  MainAxisAlignment
	MainAxisSize       MainAxisSize
	CrossAxisAlignment CrossAxisAlignment
	Spacing            float64 // постоянный зазор между детьми
}

// main/cross — размер вдоль и поперёк главной оси.
func (b *FlexBox) main(s geom.Size) float64 {
	if b.Direction == Horizontal {
		return s.Width
	}
	return s.Height
}

func (b *FlexBox) cross(s geom.Size) float64 {
	if b.Direction == Horizontal {
		return s.Height
	}
	return s.Width
}

func (b *FlexBox) size(main, cross float64) geom.Size {
	if b.Direction == Horizontal {
		return geom.Size{Width: main, Height: cross}
	}
	return geom.Size{Width: cross, Height: main}
}

// childConstraints — ограничения ребёнка: по главной оси от minMain до maxMain.
func (b *FlexBox) childConstraints(c Constraints, minMain, maxMain float64) Constraints {
	if b.Direction == Horizontal {
		cc := Constraints{MinWidth: minMain, MaxWidth: maxMain, MaxHeight: c.MaxHeight}
		if b.CrossAxisAlignment == CrossAxisStretch {
			cc.MinHeight = c.MaxHeight
		}
		return cc
	}
	cc := Constraints{MinHeight: minMain, MaxHeight: maxMain, MaxWidth: c.MaxWidth}
	if b.CrossAxisAlignment == CrossAxisStretch {
		cc.MinWidth = c.MaxWidth
	}
	return cc
}

func flexOf(ch Box) (int, FlexFit) {
	if pd, ok := ch.node().ParentData.(*FlexParentData); ok && pd.Flex > 0 {
		return pd.Flex, pd.Fit
	}
	return 0, FlexTight
}

func (b *FlexBox) PerformLayout(c Constraints) geom.Size {
	children := Children(b)
	maxMain := c.MaxWidth
	if b.Direction == Vertical {
		maxMain = c.MaxHeight
	}
	canFlex := maxMain < Inf
	gaps := b.Spacing * float64(max(0, len(children)-1))

	// 1. Негибкие дети.
	totalFlex, allocated, crossSize := 0, gaps, 0.0
	for _, ch := range children {
		flex, _ := flexOf(ch)
		if flex > 0 && canFlex {
			totalFlex += flex
			continue
		}
		s := Layout(ch, b.childConstraints(c, 0, Inf))
		allocated += b.main(s)
		crossSize = max(crossSize, b.cross(s))
	}

	// 2. Гибкие дети делят оставшееся место.
	if totalFlex > 0 {
		perFlex := max(0, maxMain-allocated) / float64(totalFlex)
		for _, ch := range children {
			flex, fit := flexOf(ch)
			if flex == 0 {
				continue
			}
			extent := perFlex * float64(flex)
			minExtent := 0.0
			if fit == FlexTight {
				minExtent = extent
			}
			s := Layout(ch, b.childConstraints(c, minExtent, extent))
			allocated += b.main(s)
			crossSize = max(crossSize, b.cross(s))
		}
	}

	// 3. Размер самого ряда.
	idealMain := allocated
	if canFlex && b.MainAxisSize == MainAxisMax {
		idealMain = maxMain
	}
	size := c.Constrain(b.size(idealMain, crossSize))
	mainSize, actualCross := b.main(size), b.cross(size)

	// 4. Свободное место и расстановка.
	free := max(0, mainSize-allocated)
	n := float64(len(children))
	leading, between := 0.0, 0.0
	switch b.MainAxisAlignment {
	case MainAxisEnd:
		leading = free
	case MainAxisCenter:
		leading = free / 2
	case MainAxisSpaceBetween:
		if n > 1 {
			between = free / (n - 1)
		}
	case MainAxisSpaceAround:
		if n > 0 {
			between = free / n
			leading = between / 2
		}
	case MainAxisSpaceEvenly:
		if n > 0 {
			between = free / (n + 1)
			leading = between
		}
	}

	pos := leading
	for _, ch := range children {
		s := SizeOf(ch)
		crossPos := 0.0
		switch b.CrossAxisAlignment {
		case CrossAxisCenter:
			crossPos = (actualCross - b.cross(s)) / 2
		case CrossAxisEnd:
			crossPos = actualCross - b.cross(s)
		}
		if b.Direction == Horizontal {
			SetOffset(ch, geom.Offset{X: pos, Y: crossPos})
		} else {
			SetOffset(ch, geom.Offset{X: crossPos, Y: pos})
		}
		pos += b.main(s) + between + b.Spacing
	}
	return size
}

func (b *FlexBox) Paint(ctx *PaintContext, offset geom.Offset) { PaintChildren(ctx, b, offset) }
