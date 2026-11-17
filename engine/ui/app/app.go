// Package app запускает приложение Hexago: app.Run(widget), как runApp во Flutter.
package app

import "hexago/ui/widgets"

// Run запускает приложение с корневым виджетом — обычно widgets.App.
// На Android функция возвращается сразу: дальше кадры рисуются по событиям экрана.
func Run(root widgets.Widget) {
	// start у каждой платформы свой: app_android.go, app_other.go.
	start(root)
}
