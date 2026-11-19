package material

import (
	"testing"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/widgets"
)

// BenchmarkScreenFrame — кадр экрана Material (как на эмуляторе: 1080×2400, плотность 2.625):
// построение, раскладка и отрисовка всего экрана.
func BenchmarkScreenFrame(b *testing.B) {
	bind := widgets.NewBinding(App{Mode: ThemeLight, Home: screenSample()})
	c := gfx.NewImageCanvas(1080, 2400)
	media := widgets.MediaQueryData{Size: geom.Size{Width: 1080 / 2.625, Height: 2400 / 2.625}, DevicePixelRatio: 2.625,
		Padding: geom.EdgeInsets{Top: 24, Bottom: 24}}
	bind.DrawFrame(c, media)
	b.ResetTimer()
	for range b.N {
		bind.SetApp(App{Mode: ThemeLight, Home: screenSample()}) // как перестройка всего экрана
		c.Clear(gfx.White)
		bind.DrawFrame(c, media)
	}
}
