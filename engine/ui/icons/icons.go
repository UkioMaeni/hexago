// Package icons — векторные значки: путь SVG в квадрате 24×24 dp. Рисует их виджет widgets.Icon.
//
// Свой значок — любой путь из SVG 24×24 (атрибут d элемента <path>):
//
//	var Rocket = icons.Icon("M12 2.5s4.5 2.04 4.5 10.5c0 2.49-1.04 5.57-1.6 7H9.1c-.56-1.43-1.6-4.51-1.6-7C7.5 4.54 12 2.5 12 2.5z")
package icons

import (
	"sync"

	"hexago/ui/gfx"
)

// Data — векторный значок. Путь разбирается один раз, при первой отрисовке.
type Data struct {
	d    string
	once sync.Once
	path *gfx.Path
}

// Icon создаёт значок из пути SVG в квадрате 24×24.
func Icon(d string) *Data { return &Data{d: d} }

// Path — разобранный путь значка в координатах 0…24. Ошибка в пути — паника:
// значки вшиты в код, это ошибка разработчика.
func (i *Data) Path() *gfx.Path {
	i.once.Do(func() { i.path = gfx.MustParseSVGPath(i.d) })
	return i.path
}

// Size — сторона квадрата, в котором нарисованы значки.
const Size = 24.0
