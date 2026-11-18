package widgets

import (
	"fmt"
	"runtime/debug"

	"hexago/ui/geom"
	"hexago/ui/gestures"
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

	arena    gestures.Arena
	pointers map[int]*pointerTrack // пальцы на экране

	// SetSystemBars вызывается, когда меняется вид системных панелей (виджет SystemBars):
	// lightIcons — светлые значки для тёмного фона.
	SetSystemBars func(lightIcons bool)
	lightIcons    bool
}

// SystemBars задаёт вид системных панелей (строки состояния и навигации), пока он в дереве:
// LightIcons — светлые значки, для тёмного фона; по умолчанию значки тёмные. Действует самый
// внешний SystemBars. Его ставит material.App по яркости темы.
type SystemBars struct {
	Base
	LightIcons bool
	Child      Widget
}

func (w SystemBars) ChildWidget() Widget { return w.Child }

// systemBars — самый внешний SystemBars в дереве.
func systemBars(e *Element) (SystemBars, bool) {
	if s, ok := e.widget.(SystemBars); ok {
		return s, true
	}
	for _, c := range e.children {
		if s, ok := systemBars(c); ok {
			return s, true
		}
	}
	return SystemBars{}, false
}

// pointerTrack — палец на экране: кому доставлять его события и где он был в прошлый раз.
type pointerTrack struct {
	path []rendering.HitEntry
	last geom.Offset
}

// NewBinding готовит дерево приложения. Строится оно в первом кадре, когда уже известен
// размер экрана, — иначе всё дерево строилось бы дважды.
func NewBinding(app Widget) *Binding {
	b := &Binding{app: app, pointers: map[int]*pointerTrack{}}
	b.owner = &BuildOwner{OnNeedsBuild: func() {
		if b.RequestFrame != nil {
			b.RequestFrame()
		}
	}}
	b.owner.gestures = gestures.Env{Arena: &b.arena, Clock: &gestures.FakeClock{}}
	return b
}

// SetClock задаёт часы для таймеров жестов (долгое и двойное нажатие). app.Run даёт настоящие,
// чьи таймеры срабатывают в потоке интерфейса; в тестах — gestures.FakeClock (он же по умолчанию).
func (b *Binding) SetClock(c gestures.Clock) { b.owner.gestures.Clock = c }

// HandlePointer доставляет событие касания (координаты — в dp экрана). Когда палец опускается,
// hit-test находит боксы под ним; все события этого пальца до отпускания получают они же,
// от самого глубокого к корню. Потом арена жестов выбирает, чей жест победил.
func (b *Binding) HandlePointer(e gestures.PointerEvent) {
	// Ошибка в обработчике жеста не роняет приложение — как ошибка в Build.
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Ошибка при обработке касания: %v\n%s\n", r, debug.Stack())
		}
	}()
	p := b.pointers[e.Pointer]
	if e.Kind == gestures.PointerDown {
		p = &pointerTrack{path: b.HitTest(e.Position).Path, last: e.Position}
		b.pointers[e.Pointer] = p
	}
	if p == nil {
		return // палец коснулся раньше, чем появилось приложение
	}
	e.Delta, p.last = e.Position.Sub(p.last), e.Position
	for _, h := range p.path {
		if ph, ok := h.Box.(rendering.PointerHandler); ok {
			ev := e
			ev.Local = h.Transform.Apply(e.Position)
			ph.HandlePointer(ev)
		}
	}
	switch e.Kind {
	case gestures.PointerDown:
		b.arena.Close(e.Pointer)
	case gestures.PointerUp:
		delete(b.pointers, e.Pointer)
		b.arena.Sweep(e.Pointer)
	case gestures.PointerCancel:
		delete(b.pointers, e.Pointer)
		b.arena.Cancel(e.Pointer)
	}
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
	if bars, _ := systemBars(b.root); bars.LightIcons != b.lightIcons {
		b.lightIcons = bars.LightIcons
		if b.SetSystemBars != nil {
			b.SetSystemBars(bars.LightIcons)
		}
	}

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
