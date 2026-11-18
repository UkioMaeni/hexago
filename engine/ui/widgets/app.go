package widgets

import (
	"fmt"
	"runtime/debug"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/rendering"
)

// App — корень приложения: название, цвет фона и стартовый экран Home.
// Позже сюда добавятся тема и маршруты, как в MaterialApp.
type App struct {
	Base
	Title string
	Color gfx.Color // фон под экраном; по умолчанию белый
	Home  Widget
}

func (a App) Build(BuildContext) Widget {
	bg := a.Color
	if bg.IsTransparent() {
		bg = gfx.White
	}
	return ColoredBox{Color: bg, Child: a.Home}
}

// Binding связывает дерево виджетов с экраном: перестраивает, раскладывает и рисует кадры.
// Его создаёт app.Run для платформы и hexagotest для тестов.
type Binding struct {
	owner *BuildOwner
	root  *Element
	app   Widget
	media MediaQueryData

	// RequestFrame вызывается, когда что-то изменилось (SetState) и нужен новый кадр.
	RequestFrame func()
}

// NewBinding готовит дерево приложения. Строится оно в первом кадре, когда уже известен
// размер экрана, — иначе всё дерево строилось бы дважды.
func NewBinding(app Widget) *Binding {
	b := &Binding{app: app}
	b.owner = &BuildOwner{OnNeedsBuild: func() {
		if b.RequestFrame != nil {
			b.RequestFrame()
		}
	}}
	return b
}

func (b *Binding) rootWidget() Widget {
	return view{Child: MediaQuery{Data: b.media, Child: b.app}}
}

// SetApp заменяет корневой виджет приложения (тип должен остаться тем же) —
// дерево перестроится в следующем кадре с сохранением состояний.
func (b *Binding) SetApp(app Widget) {
	b.app = app
	if b.root != nil {
		b.owner.Update(b.root, b.rootWidget())
	}
}

// Root — корневой элемент дерева (nil до первого кадра).
func (b *Binding) Root() *Element { return b.root }

// DrawFrame рисует кадр: перестраивает изменившиеся виджеты, раскладывает дерево
// под размер экрана media.Size (в dp) и рисует его на холсте в физических пикселях.
func (b *Binding) DrawFrame(c *gfx.Canvas, media MediaQueryData) {
	if media.DevicePixelRatio <= 0 {
		media.DevicePixelRatio = 1
	}
	switch {
	case b.root == nil:
		b.media = media
		b.root = b.owner.Mount(b.rootWidget())
	case media != b.media:
		b.media = media
		b.owner.Update(b.root, b.rootWidget())
	}
	b.owner.BuildScope()
	SyncRenderTree(b.root)

	// Ошибка лэйаута или отрисовки (например, бесконечная ширина внутри Row) не роняет
	// приложение: кадр заливается красным, а текст ошибки уходит в лог.
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Ошибка лэйаута или отрисовки: %v\n%s\n", r, debug.Stack())
			c.Clear(gfx.ARGB(0xFFD32F2F))
		}
	}()
	box := b.root.box
	rendering.Layout(box, rendering.Tight(media.Size))
	c.Save()
	c.Scale(media.DevicePixelRatio, media.DevicePixelRatio)
	box.Paint(&rendering.PaintContext{Canvas: c}, geom.Zero)
	c.Restore()
}

// HitTest находит боксы под точкой pos (в dp) — от самого глубокого к корню.
func (b *Binding) HitTest(pos geom.Offset) rendering.HitTestResult {
	var r rendering.HitTestResult
	if b.root != nil {
		rendering.HitTest(b.root.box, &r, pos)
	}
	return r
}
