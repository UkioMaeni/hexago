//go:build !android

package app

import "hexago/ui/widgets"

// start на остальных платформах пока ничего не делает:
// окна для Windows, macOS и других появятся позже.
func start(root widgets.Widget) {}

// post без потока интерфейса выполняет f сразу.
func post(f func()) { f() }
