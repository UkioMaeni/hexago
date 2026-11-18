package widgets

import (
	"hexago/ui/gfx"
	"hexago/ui/icons"
	"hexago/ui/rendering"
)

// Icon — векторный значок: widgets.Icon{Icon: icons.Search}. Размер и цвет, если не заданы, берутся
// у ближайшего IconTheme (у кнопок Material — свой), а без него — 24 dp и чёрный с непрозрачностью 87 %.
type Icon struct {
	Base
	Icon  *icons.Data
	Size  float64   // сторона квадрата в dp; 0 — из IconTheme
	Color gfx.Color // прозрачный (нулевой) — из IconTheme
}

func (w Icon) style(ctx BuildContext) (float64, gfx.Color) {
	t := IconThemeOf(ctx)
	size, color := w.Size, w.Color
	if size <= 0 {
		size = t.Size
	}
	if size <= 0 {
		size = 24
	}
	if color.IsTransparent() {
		color = t.Color
	}
	if color.IsTransparent() {
		color = gfx.Black.WithOpacity(0.87)
	}
	return size, color
}

func (w Icon) CreateRenderObject(ctx BuildContext) rendering.Box {
	b := &rendering.IconBox{}
	w.UpdateRenderObject(ctx, b)
	return b
}

func (w Icon) UpdateRenderObject(ctx BuildContext, b rendering.Box) {
	box := b.(*rendering.IconBox)
	box.Size, box.Color = w.style(ctx)
	box.Path = nil
	if w.Icon != nil {
		box.Path = w.Icon.Path()
	}
}

// IconThemeData — размер и цвет значков по умолчанию.
type IconThemeData struct {
	Size  float64
	Color gfx.Color
}

// IconTheme задаёт размер и цвет значков поддереву (как DefaultTextStyle для текста).
type IconTheme struct {
	Base
	Data  IconThemeData
	Child Widget
}

func (w IconTheme) ChildWidget() Widget { return w.Child }

func (w IconTheme) UpdateShouldNotify(old Widget) bool { return old.(IconTheme).Data != w.Data }

// IconThemeOf — значки по умолчанию в этом месте дерева (нулевые, если не заданы).
func IconThemeOf(ctx BuildContext) IconThemeData {
	if t, ok := DependOn[IconTheme](ctx); ok {
		return t.Data
	}
	return IconThemeData{}
}
