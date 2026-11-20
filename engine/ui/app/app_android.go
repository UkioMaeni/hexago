//go:build android

package app

import (
	"image"
	"time"

	"hexago/platform/android"
	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/widgets"
)

// start на Android: дерево виджетов рисуется в буфер экрана, когда платформа просит кадр
// (поверхность появилась, изменилась или приложение попросило кадр после SetState и анимаций),
// а касания уходят в Binding. Перерисовывается только изменившаяся часть экрана.
//
// Файл собирается только под Android — так говорит строка //go:build android.
func start(root widgets.Widget) {
	binding := widgets.NewBinding(root)
	binding.RequestFrame = android.RequestFrame
	binding.SetSystemBars = android.SetSystemBars
	binding.SetHandlesBack = android.SetHandlesBack
	android.SetBack(binding.HandleBack)
	binding.SetClock(uiClock{})

	r := &renderer{binding: binding, scale: 1}
	android.SetRenderer(r)
	android.SetPointer(func(p android.PointerEvent) {
		binding.HandlePointer(gestures.PointerEvent{
			Kind:     gestures.PointerKind(p.Kind),
			Pointer:  p.Pointer,
			Position: geom.Offset{X: p.X / r.scale, Y: p.Y / r.scale}, // касания приходят в пикселях, нужны dp
			Time:     p.Time,
		})
	})
}

// renderer рисует кадры Binding в окно Android.
type renderer struct {
	binding   *widgets.Binding
	scale     float64 // плотность экрана из последнего кадра
	unpainted bool    // Prepare нашёл изменения, а Paint не случился (окно не заблокировалось)
}

func (r *renderer) Prepare(f android.FrameInfo) image.Rectangle {
	r.scale = f.Scale
	r.binding.Tick(f.Time) // анимации — на время этого кадра
	brightness := gfx.Light
	if f.DarkTheme {
		brightness = gfx.Dark
	}
	dirty := r.binding.Layout(widgets.MediaQueryData{
		Size:             geom.Size{Width: float64(f.Width) / f.Scale, Height: float64(f.Height) / f.Scale},
		DevicePixelRatio: f.Scale,
		Padding: geom.EdgeInsets{
			Left: float64(f.Insets[0]) / f.Scale, Top: float64(f.Insets[1]) / f.Scale,
			Right: float64(f.Insets[2]) / f.Scale, Bottom: float64(f.Insets[3]) / f.Scale,
		},
		PlatformBrightness: brightness,
	})
	if r.unpainted {
		// Прошлые изменения так и не нарисованы — рисуем всё, чтобы на экране не остались старые куски.
		dirty = image.Rect(0, 0, f.Width, f.Height)
	}
	r.unpainted = !dirty.Empty()
	return dirty
}

func (r *renderer) Paint(f android.Frame, dirty image.Rectangle) {
	r.unpainted = false
	r.binding.Paint(gfx.NewCanvas(f.Pixels, f.Width, f.Height, f.Stride), dirty)
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
