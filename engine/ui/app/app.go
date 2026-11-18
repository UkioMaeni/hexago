// Package app запускает приложение Hexago: app.Run(widget), как runApp во Flutter.
package app

import "hexago/ui/widgets"

// Run запускает приложение с корневым виджетом — обычно widgets.App.
// На Android функция возвращается сразу: дальше кадры рисуются по событиям экрана.
func Run(root widgets.Widget) {
	// start у каждой платформы свой: app_android.go, app_other.go.
	start(root)
}

// Post выполняет f в потоке интерфейса. Виджеты, SetState и обработчики жестов живут в этом
// потоке; из другой горутины (загрузка из сети, таймер) менять состояние можно только через Post:
//
//	go func() {
//		data := load()
//		app.Post(func() { c.SetState(func() { c.data = data }) })
//	}()
func Post(f func()) { post(f) }
