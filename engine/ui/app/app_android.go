//go:build android

package app

import (
	"image"
	"time"

	"hexago/platform/android"
	"hexago/ui/assets"
	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/text"
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
	binding.Post = android.Post // загрузки картинок и другие горутины возвращаются в поток интерфейса
	binding.SetSystemBars = android.SetSystemBars
	binding.SetHandlesBack = android.SetHandlesBack
	android.SetBack(binding.HandleBack)
	binding.SetTextInput(keyboard{})
	binding.SetClipboard(clipboard{})
	text.SetFallback(systemFont{}) // эмодзи и то, чего нет в шрифте Go, рисует Android
	assets.SetFS(android.Assets()) // папка assets/ проекта — внутри APK
	android.SetEditing(func(s android.EditingState) {
		binding.HandleTextEditing(widgets.TextEditingValue{
			Text:      s.Text,
			Selection: widgets.TextSelection{Base: s.SelStart, Extent: s.SelEnd},
			Composing: widgets.TextRange{Start: max(0, s.ComposeStart), End: max(0, s.ComposeEnd)},
		})
	})
	android.SetTextAction(func(a int) { binding.HandleTextAction(widgets.TextInputAction(a)) })
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
		ViewInsets:         geom.EdgeInsets{Bottom: float64(f.Keyboard) / f.Scale},
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

// keyboard — экранная (и физическая) клавиатура Android для полей ввода.
type keyboard struct{}

func editingState(v widgets.TextEditingValue) android.EditingState {
	return android.EditingState{Text: v.Text, SelStart: v.Selection.Base, SelEnd: v.Selection.Extent,
		ComposeStart: v.Composing.Start, ComposeEnd: v.Composing.End}
}

func (keyboard) Show(cfg widgets.TextInputConfig, v widgets.TextEditingValue) {
	android.ShowKeyboard(android.KeyboardConfig{
		Keyboard: int(cfg.Keyboard), Action: int(cfg.Action), Capitalize: int(cfg.Capitalize),
		Multiline: cfg.Multiline, Obscure: cfg.Obscure, NoAutocorrect: cfg.NoAutocorrect,
	}, editingState(v))
}

func (keyboard) Update(v widgets.TextEditingValue) { android.UpdateEditing(editingState(v)) }

func (keyboard) Hide() { android.HideKeyboard() }

// systemFont — запасной шрифт: системные шрифты Android (цветные эмодзи, иероглифы и прочее, чего нет
// во встроенном шрифте Go).
type systemFont struct{}

func (systemFont) Measure(s string, size float64) float64 { return android.MeasureText(s, size) }

func (systemFont) Render(s string, size float64, col gfx.Color) (*image.NRGBA, float64) {
	argb := uint32(col.A)<<24 | uint32(col.R)<<16 | uint32(col.G)<<8 | uint32(col.B)
	w, h, ascent, pix := android.RenderText(s, size, argb)
	if w == 0 {
		return nil, 0
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i, p := range pix {
		img.Pix[i*4], img.Pix[i*4+1], img.Pix[i*4+2], img.Pix[i*4+3] = uint8(p>>16), uint8(p>>8), uint8(p), uint8(p>>24)
	}
	return img, ascent
}

// clipboard — системный буфер обмена Android.
type clipboard struct{}

func (clipboard) SetText(s string)     { android.SetClipboard(s) }
func (clipboard) Text() (string, bool) { return android.Clipboard() }
func (clipboard) HasText() bool        { return android.HasClipboard() }
