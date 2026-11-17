package app

import "hexago/ui/widgets"

func Run(app widgets.App) {
	// start у каждой платформы свой: app_android.go, app_other.go.
	start(app)
}
