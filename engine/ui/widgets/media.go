package widgets

import (
	"hexago/ui/geom"
)

// MediaQueryData — сведения об экране: размер в dp, плотность, отступы системных панелей.
type MediaQueryData struct {
	Size             geom.Size       // размер экрана в логических пикселях (dp)
	DevicePixelRatio float64         // физических пикселей в одном dp
	Padding          geom.EdgeInsets // области, занятые строкой состояния, вырезом, навигацией
}

// MediaQuery передаёт сведения об экране всему поддереву. Его ставит сам движок в корень.
type MediaQuery struct {
	Base
	Data  MediaQueryData
	Child Widget
}

func (w MediaQuery) ChildWidget() Widget { return w.Child }
func (w MediaQuery) UpdateShouldNotify(old Widget) bool {
	return old.(MediaQuery).Data != w.Data
}

// MediaQueryOf возвращает сведения об экране и подписывает виджет на их изменения
// (поворот, смена размера окна).
func MediaQueryOf(ctx BuildContext) MediaQueryData {
	if m, ok := DependOn[MediaQuery](ctx); ok {
		return m.Data
	}
	return MediaQueryData{DevicePixelRatio: 1}
}

// SafeArea отодвигает ребёнка от системных панелей: строки состояния, выреза камеры,
// панели навигации. Левая/верхняя/правая/нижняя стороны включаются по отдельности
// (по умолчанию — все: нулевое значение NoLeft и т.п. — false).
type SafeArea struct {
	Base
	NoLeft, NoTop, NoRight, NoBottom bool
	Minimum                          geom.EdgeInsets // минимальный отступ
	Child                            Widget
}

func (w SafeArea) Build(ctx BuildContext) Widget {
	p := MediaQueryOf(ctx).Padding
	if w.NoLeft {
		p.Left = 0
	}
	if w.NoTop {
		p.Top = 0
	}
	if w.NoRight {
		p.Right = 0
	}
	if w.NoBottom {
		p.Bottom = 0
	}
	p = geom.EdgeInsets{
		Left: max(p.Left, w.Minimum.Left), Top: max(p.Top, w.Minimum.Top),
		Right: max(p.Right, w.Minimum.Right), Bottom: max(p.Bottom, w.Minimum.Bottom),
	}
	return Padding{Padding: p, Child: w.Child}
}
