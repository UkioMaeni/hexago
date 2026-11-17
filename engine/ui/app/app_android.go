//go:build android

package app

import (
	"hexago/platform/android"
	"hexago/ui/widgets"
)

// start на Android: сообщает платформе, как рисовать кадр.
//
// Файл собирается только под Android — так говорит строка //go:build android.
func start(a widgets.App) {
	android.SetDraw(func(f android.Frame) {
		// Пока рисуем только фон — весь экран голубым #00ADD8.
		// Отрисовка дерева виджетов (a.Home) — следующий этап.
		for i := range f.Pixels {
			f.Pixels[i] = 0xFFD8AD00
		}
	})
}
