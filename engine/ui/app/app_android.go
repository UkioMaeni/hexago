//go:build android

package app

import (
	"hexago/platform/android"
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/widgets"
)

// start на Android: дерево виджетов рисуется в буфер экрана каждый раз,
// когда платформа просит кадр.
//
// Файл собирается только под Android — так говорит строка //go:build android.
func start(root widgets.Widget) {
	binding := widgets.NewBinding(root)
	android.SetDraw(func(f android.Frame) {
		c := gfx.NewCanvas(f.Pixels, f.Width, f.Height, f.Stride)
		c.Clear(gfx.White)
		binding.DrawFrame(c, widgets.MediaQueryData{
			Size:             geom.Size{Width: float64(f.Width) / f.Scale, Height: float64(f.Height) / f.Scale},
			DevicePixelRatio: f.Scale,
			Padding: geom.EdgeInsets{
				Left: float64(f.Insets[0]) / f.Scale, Top: float64(f.Insets[1]) / f.Scale,
				Right: float64(f.Insets[2]) / f.Scale, Bottom: float64(f.Insets[3]) / f.Scale,
			},
		})
	})
}
