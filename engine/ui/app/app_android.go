//go:build android

package app

import (
	"time"

	"hexago/platform/android"
	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/widgets"
)

// start на Android: дерево виджетов рисуется в буфер экрана, когда платформа просит кадр
// (поверхность появилась, изменилась или приложение попросило кадр после SetState),
// а касания уходят в Binding.
//
// Файл собирается только под Android — так говорит строка //go:build android.
func start(root widgets.Widget) {
	binding := widgets.NewBinding(root)
	binding.RequestFrame = android.RequestFrame
	binding.SetSystemBars = android.SetSystemBars
	binding.SetClock(uiClock{})

	scale := 1.0 // плотность экрана из последнего кадра: касания приходят в пикселях, а нужны dp
	android.SetDraw(func(f android.Frame) {
		scale = f.Scale
		brightness := gfx.Light
		if f.DarkTheme {
			brightness = gfx.Dark
		}
		binding.Tick(f.Time) // анимации — на время этого кадра
		c := gfx.NewCanvas(f.Pixels, f.Width, f.Height, f.Stride)
		c.Clear(gfx.White)
		binding.DrawFrame(c, widgets.MediaQueryData{
			Size:             geom.Size{Width: float64(f.Width) / f.Scale, Height: float64(f.Height) / f.Scale},
			DevicePixelRatio: f.Scale,
			Padding: geom.EdgeInsets{
				Left: float64(f.Insets[0]) / f.Scale, Top: float64(f.Insets[1]) / f.Scale,
				Right: float64(f.Insets[2]) / f.Scale, Bottom: float64(f.Insets[3]) / f.Scale,
			},
			PlatformBrightness: brightness,
		})
	})
	android.SetPointer(func(p android.PointerEvent) {
		binding.HandlePointer(gestures.PointerEvent{
			Kind:     gestures.PointerKind(p.Kind),
			Pointer:  p.Pointer,
			Position: geom.Offset{X: p.X / scale, Y: p.Y / scale},
			Time:     p.Time,
		})
	})
}

func post(f func()) { android.Post(f) }

// uiClock — таймеры жестов: отсчитывает время горутина, а функция выполняется в потоке интерфейса.
type uiClock struct{}

func (uiClock) AfterFunc(d time.Duration, f func()) func() {
	stopped := false // читается и пишется только в потоке интерфейса
	t := time.AfterFunc(d, func() {
		android.Post(func() {
			if !stopped {
				f()
			}
		})
	})
	return func() {
		stopped = true
		t.Stop()
	}
}
