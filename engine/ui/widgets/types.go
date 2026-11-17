package widgets

import (
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/painting"
	"hexago/ui/rendering"
)

// Типы и значения соседних пакетов, чтобы в коде приложения хватало импорта widgets —
// как во Flutter хватает одного package:flutter/widgets.dart.
type (
	Color          = gfx.Color
	Offset         = geom.Offset
	Size           = geom.Size
	Rect           = geom.Rect
	EdgeInsets     = geom.EdgeInsets
	Alignment      = geom.Alignment
	Radius         = geom.Radius
	BorderRadius   = geom.BorderRadius
	BoxDecoration  = painting.BoxDecoration
	Border         = painting.Border
	BorderSide     = painting.BorderSide
	BoxShadow      = painting.BoxShadow
	LinearGradient = painting.LinearGradient
	RadialGradient = painting.RadialGradient
	BoxConstraints = rendering.Constraints
)

var (
	// Отступы: All(16), Symmetric(24, 8).
	All       = geom.All
	Symmetric = geom.Symmetric

	// Скругления и рамки: RadiusAll(12), BorderAll(gfx.Grey, 1).
	RadiusAll = geom.RadiusAll
	Circular  = geom.Circular
	BorderAll = painting.BorderAll

	// Цвет из строки: ColorFromHex("#2196F3").
	ColorFromHex = gfx.ColorFromHex

	// Выравнивания (как Alignment.center и прочие во Flutter).
	AlignTopLeft      = geom.TopLeft
	AlignTopCenter    = geom.TopCenter
	AlignTopRight     = geom.TopRight
	AlignCenterLeft   = geom.CenterLeft
	AlignCenter       = geom.Center
	AlignCenterRight  = geom.CenterRight
	AlignBottomLeft   = geom.BottomLeft
	AlignBottomCenter = geom.BottomCenter
	AlignBottomRight  = geom.BottomRight
)

// Значения для Row, Column, Flex, Stack, Wrap и BoxDecoration.
const (
	MainAxisStart        = rendering.MainAxisStart
	MainAxisEnd          = rendering.MainAxisEnd
	MainAxisCenter       = rendering.MainAxisCenter
	MainAxisSpaceBetween = rendering.MainAxisSpaceBetween
	MainAxisSpaceAround  = rendering.MainAxisSpaceAround
	MainAxisSpaceEvenly  = rendering.MainAxisSpaceEvenly

	MainAxisMax = rendering.MainAxisMax
	MainAxisMin = rendering.MainAxisMin

	CrossAxisCenter  = rendering.CrossAxisCenter
	CrossAxisStart   = rendering.CrossAxisStart
	CrossAxisEnd     = rendering.CrossAxisEnd
	CrossAxisStretch = rendering.CrossAxisStretch

	Horizontal = rendering.Horizontal
	Vertical   = rendering.Vertical

	FlexTight = rendering.FlexTight
	FlexLoose = rendering.FlexLoose

	StackLoose       = rendering.StackLoose
	StackExpand      = rendering.StackExpand
	StackPassthrough = rendering.StackPassthrough

	ShapeRectangle = painting.Rectangle
	ShapeCircle    = painting.Circle
)
