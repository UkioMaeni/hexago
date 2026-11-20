package material

import (
	"testing"
	"time"

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

// BenchmarkPushTransition — кадр посреди перехода экранов: оба экрана видны и сдвинуты.
func BenchmarkPushTransition(b *testing.B) {
	bind := widgets.NewBinding(App{Mode: ThemeLight, Home: screenSample()})
	c := gfx.NewImageCanvas(1080, 2400)
	media := widgets.MediaQueryData{Size: geom.Size{Width: 1080 / 2.625, Height: 2400 / 2.625}, DevicePixelRatio: 2.625,
		Padding: geom.EdgeInsets{Top: 24, Bottom: 24}}
	now := time.Duration(0)
	frame := func() {
		now += 16 * time.Millisecond
		bind.Tick(now)
		bind.Paint(c, bind.Layout(media))
	}
	frame()
	nav := findNavigatorChild(bind.Root())
	b.ResetTimer()
	for i := range b.N {
		if i%20 == 0 {
			b.StopTimer()
			for bind.Animating() || widgets.NavigatorOf(nav).CanPop() {
				widgets.NavigatorOf(nav).Pop(nil)
				frame()
			}
			widgets.Push(nav, screenSample())
			frame()
			b.StartTimer()
		}
		frame()
	}
}

// findNavigatorChild — элемент под Navigator (его контекст видит Navigator).
func findNavigatorChild(e *widgets.Element) widgets.BuildContext {
	if _, ok := e.Widget().(widgets.Navigator); ok {
		return e.Children()[0]
	}
	for _, c := range e.Children() {
		if r := findNavigatorChild(c); r != nil {
			return r
		}
	}
	return nil
}
