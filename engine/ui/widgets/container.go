package widgets

import "hexago/ui/gfx"

type Container struct {
	Base
	Width      *float64
	Height     *float64
	Background gfx.Color
	Child      Widget
}
