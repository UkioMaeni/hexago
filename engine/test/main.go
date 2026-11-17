// test — приложение на Hexago.
//
// Пока движок не умеет запускать Go-код на Android, здесь заглушка.
package main

import (
	"hexago/ui/app"
	"hexago/ui/gfx"
	w "hexago/ui/widgets"
)

func main() {

	application:=w.App{
		Title: "Appuru",
		Home: w.Container{
			Width: new(200.0),
			Height: new(200.0),
			Background: gfx.ColorFromHex("#111111"),
		},
	}
	
	app.Run(application)
}
