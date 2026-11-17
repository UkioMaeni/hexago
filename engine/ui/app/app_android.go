//go:build android

package app

import (
	"fmt"

	"hexago/platform/android"
	"hexago/ui/widgets"
)

// start на Android: сообщает платформе, как рисовать кадр.
//
// Файл собирается только под Android — так говорит строка //go:build android.
func start(a widgets.App) {
	android.Log("app.Run: приложение «" + a.Title + "»")

	android.SetDraw(func(f android.Frame) {
		android.Log(fmt.Sprintf("кадр: %d×%d пикселей, плотность %.3f", f.Width, f.Height, f.Scale))

		// Пока рисуем только фон — весь экран голубым #00ADD8.
		// Сюда встанет вызов твоей функции отрисовки дерева виджетов (a.Home).
		for i := range f.Pixels {
			f.Pixels[i] = 0xFFD8AD00
		}
	})
}
