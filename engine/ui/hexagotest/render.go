package hexagotest

import (
	"image"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/widgets"
)

// Render рисует виджет на белом холсте width×height физических пикселей с плотностью dpr —
// как на экране телефона, только в памяти. Результат удобно сравнить с эталоном через Golden.
func Render(w widgets.Widget, width, height int, dpr float64) *image.NRGBA {
	b := widgets.NewBinding(w)
	c := gfx.NewImageCanvas(width, height)
	c.Clear(gfx.White)
	b.DrawFrame(c, widgets.MediaQueryData{
		Size:             geom.Size{Width: float64(width) / dpr, Height: float64(height) / dpr},
		DevicePixelRatio: dpr,
	})
	return c.ToNRGBA()
}
