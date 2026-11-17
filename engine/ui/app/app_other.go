//go:build !android

package app

import "hexago/ui/widgets"

// start на остальных платформах пока ничего не делает:
// окна для Windows, macOS и других появятся позже.
func start(a widgets.App) {}
