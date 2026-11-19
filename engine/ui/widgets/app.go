package widgets

import (
	"fmt"
	"image"
	"math"
	"runtime/debug"
	"time"

	"hexago/ui/animation"
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
	tickers  []animation.Ticker    // идущие анимации

	// SetSystemBars вызывается, когда меняется вид системных панелей (виджет SystemBars):
	// lightIcons — светлые значки для тёмного фона.
	SetSystemBars func(lightIcons bool)
	lightIcons    bool

	failed    any  // ошибка раскладки этого кадра (кадр заливается красным)
	wasFailed bool // прошлый кадр был с ошибкой: следующий рисуется целиком
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
	b.owner.scheduler = b
	return b
}

// Animate — для animation.Scheduler: t получает время каждого кадра, пока не закончит.
func (b *Binding) Animate(t animation.Ticker) {
	b.tickers = append(b.tickers, t)
	if b.RequestFrame != nil {
		b.RequestFrame()
	}
}

// Tick — начало кадра: идущие анимации получают время кадра now (отсчёт от включения устройства).
// Платформа вызывает его перед DrawFrame; пока анимации идут, кадры запрашиваются снова и снова.
func (b *Binding) Tick(now time.Duration) {
	active := b.tickers
	b.tickers = nil
	for _, t := range active {
		if t.Tick(now) {
			b.tickers = append(b.tickers, t)
		}
	}
	if len(b.tickers) > 0 && b.RequestFrame != nil {
		b.RequestFrame()
	}
}

// Animating сообщает, что идут анимации.
func (b *Binding) Animating() bool { return len(b.tickers) > 0 }

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

// DrawFrame рисует кадр целиком: Layout, затем Paint всего холста. Так рисуют тесты; платформа
// вызывает Layout и Paint по отдельности, чтобы перерисовывать только изменившееся.
func (b *Binding) DrawFrame(c *gfx.Canvas, media MediaQueryData) {
	b.Layout(media)
	b.Paint(c, image.Rect(0, 0, c.Width(), c.Height()))
}

// errorColor — цвет кадра, если раскладка или отрисовка упали.
var errorColor = gfx.ARGB(0xFFD32F2F)

// Layout строит изменившиеся виджеты и раскладывает дерево под размер экрана media.Size (dp).
// Возвращает, какую часть экрана (в физических пикселях) нужно перерисовать: пустой прямоугольник —
// ничего не изменилось, кадр можно не рисовать.
func (b *Binding) Layout(media MediaQueryData) image.Rectangle {
	if media.DevicePixelRatio <= 0 {
		media.DevicePixelRatio = 1
	}
	full := false
	switch {
	case b.root == nil:
		b.media, full = media, true
		b.root = b.owner.Mount(b.rootWidget())
	case media != b.media:
		b.media, full = media, true
		b.owner.Update(b.root, b.rootWidget())
	}
	b.owner.BuildScope()
	var removed geom.Rect
	syncRenderTree(b.root, func(old rendering.Box) {
		if r, ok := rendering.PrevBounds(old); ok {
			removed = union(removed, r)
		}
	})
	if bars, _ := systemBars(b.root); bars.LightIcons != b.lightIcons {
		b.lightIcons = bars.LightIcons
		if b.SetSystemBars != nil {
			b.SetSystemBars(bars.LightIcons)
		}
	}

	screen := image.Rect(0, 0, int(math.Ceil(media.Size.Width*media.DevicePixelRatio)),
		int(math.Ceil(media.Size.Height*media.DevicePixelRatio)))
	b.failed = nil
	// Ошибка раскладки (например, бесконечная ширина внутри Row) не роняет приложение:
	// кадр заливается красным, а текст ошибки уходит в лог.
	func() {
		defer func() {
			if r := recover(); r != nil {
				b.failed = r
				fmt.Printf("Ошибка лэйаута: %v\n%s\n", r, debug.Stack())
			}
		}()
		rendering.Layout(b.root.box, rendering.Tight(media.Size))
	}()
	if b.failed != nil || full || b.wasFailed {
		b.wasFailed = b.failed != nil
		rendering.CollectDamage(b.root.box) // запомнить границы на будущее
		return screen
	}
	damage := union(rendering.CollectDamage(b.root.box), removed)
	if damage.IsEmpty() {
		return image.Rectangle{}
	}
	k := media.DevicePixelRatio
	return image.Rect(int(math.Floor(damage.Left*k)), int(math.Floor(damage.Top*k)),
		int(math.Ceil(damage.Right*k)), int(math.Ceil(damage.Bottom*k))).Intersect(screen)
}

// Paint рисует кадр на холсте внутри area (физические пиксели); за её пределами холст не меняется.
// area может быть больше того, что вернул Layout (платформа иногда просит перерисовать больше).
func (b *Binding) Paint(c *gfx.Canvas, area image.Rectangle) {
	if area.Empty() || b.root == nil {
		return
	}
	if b.failed != nil {
		c.ClearRect(area, errorColor)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Ошибка отрисовки: %v\n%s\n", r, debug.Stack())
			c.ClearRect(area, errorColor)
		}
	}()
	c.Save()
	defer c.Restore()
	c.ClipRect(geom.Rect{Left: float64(area.Min.X), Top: float64(area.Min.Y), Right: float64(area.Max.X), Bottom: float64(area.Max.Y)})
	c.ClearRect(area, gfx.White)
	k := b.media.DevicePixelRatio
	c.Scale(k, k)
	b.root.box.Paint(&rendering.PaintContext{Canvas: c}, geom.Zero)
}

// union — наименьший прямоугольник, содержащий a и b (пустые не считаются).
func union(a, b geom.Rect) geom.Rect {
	switch {
	case a.IsEmpty():
		return b
	case b.IsEmpty():
		return a
	}
	return geom.Rect{Left: math.Min(a.Left, b.Left), Top: math.Min(a.Top, b.Top),
		Right: math.Max(a.Right, b.Right), Bottom: math.Max(a.Bottom, b.Bottom)}
}

// HitTest находит боксы под точкой pos (в dp) — от самого глубокого к корню.
func (b *Binding) HitTest(pos geom.Offset) rendering.HitTestResult {
	var r rendering.HitTestResult
	if b.root != nil {
		rendering.HitTest(b.root.box, &r, pos)
	}
	return r
}
