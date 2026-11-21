package widgets

import (
	"fmt"
	"math"
	"time"

	"hexago/ui/animation"
	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/rendering"
)

// Прокрутка: SingleChildScrollView, ListView, GridView, PageView. Палец тянет содержимое, отпущенное
// с разгону — оно летит дальше и замедляется трением, у краёв останавливается. Касание останавливает
// полёт. Прокрутка соревнуется с другими жестами через арену: нажатие на строку списка срабатывает,
// если палец не ушёл в прокрутку.

// ScrollController управляет прокруткой снаружи: узнать, где она (Offset), и прокрутить (JumpTo,
// AnimateTo). Один контроллер — одной прокрутке.
//
//	c.scroll = &widgets.ScrollController{}
//	widgets.ListView{Controller: c.scroll, ...}
//	c.scroll.AnimateTo(0, 300*time.Millisecond, nil) // наверх
type ScrollController struct {
	InitialOffset float64
	s             *scrollable
}

// Offset — насколько прокручено, в dp.
func (c *ScrollController) Offset() float64 {
	if c.s == nil {
		return c.InitialOffset
	}
	return c.s.offset
}

// MaxOffset — насколько можно прокрутить (0, пока прокрутка не на экране).
func (c *ScrollController) MaxOffset() float64 {
	if c.s == nil {
		return 0
	}
	return c.s.metrics.MaxOffset()
}

// JumpTo прокручивает к offset сразу.
func (c *ScrollController) JumpTo(offset float64) {
	if c.s == nil {
		c.InitialOffset = offset
		return
	}
	c.s.jumpTo(offset)
}

// AnimateTo плавно прокручивает к offset за d по кривой curve (nil — animation.Standard).
func (c *ScrollController) AnimateTo(offset float64, d time.Duration, curve animation.Curve) {
	if c.s == nil {
		c.InitialOffset = offset
		return
	}
	c.s.animateTo(offset, d, curve)
}

// scrollable — общая часть прокрутки: смещение, перетаскивание, полёт. Содержимое строит Content
// по смещению и длине видимой области (ленивый список строит только видимые строки).
type scrollable struct {
	State
	Horizontal bool
	Controller *ScrollController
	Content    func(ctx BuildContext, offset, viewport float64) Widget

	offset  float64
	metrics rendering.ScrollMetrics
	motion  *animation.Controller
}

func (s *scrollable) Init(ctx BuildContext) {
	s.motion = NewAnimation(ctx, 0)
	s.motion.AddListener(func() { s.offset = s.motion.Value() })
	if s.Controller != nil {
		s.offset = s.Controller.InitialOffset
		s.Controller.s = s
	}
}

func (s *scrollable) Update(_ BuildContext, old scrollable) {
	if old.Controller != s.Controller {
		if old.Controller != nil && old.Controller.s == s {
			old.Controller.s = nil
		}
		if s.Controller != nil {
			s.Controller.s = s
		}
	}
}

func (s *scrollable) Dispose() {
	if s.Controller != nil && s.Controller.s == s {
		s.Controller.s = nil
	}
}

// clamp ограничивает смещение краями (когда размеры уже известны).
func (s *scrollable) clamp(v float64) float64 {
	if s.metrics.Viewport <= 0 {
		return math.Max(0, v)
	}
	return math.Max(0, math.Min(v, s.metrics.MaxOffset()))
}

func (s *scrollable) jumpTo(v float64) {
	s.motion.Stop()
	s.SetState(func() { s.offset = s.clamp(v) })
}

func (s *scrollable) animateTo(v float64, d time.Duration, curve animation.Curve) {
	if d <= 0 {
		d = defaultDuration
	}
	if curve == nil {
		curve = animation.Standard
	}
	s.motion.AnimateWith(animation.TweenSimulation(s.offset, s.clamp(v), d, curve))
}

func (s *scrollable) Build(ctx BuildContext) Widget {
	viewport := s.metrics.Viewport
	if viewport <= 0 { // первый кадр: размер ещё неизвестен — берём экран
		size := MediaQueryOf(ctx).Size
		viewport = size.Height
		if s.Horizontal {
			viewport = size.Width
		}
	} else if !s.motion.IsAnimating() {
		s.offset = s.clamp(s.offset) // содержимое могло уменьшиться
	}
	axis := rendering.Vertical
	if s.Horizontal {
		axis = rendering.Horizontal
	}
	view := scrollViewport{Axis: axis, Offset: s.offset, Metrics: &s.metrics, Child: s.Content(ctx, s.offset, viewport)}

	drag := func(delta float64) {
		s.motion.Stop()
		if v := s.clamp(s.offset - delta); v != s.offset {
			s.SetState(func() { s.offset = v })
		}
	}
	fling := func(velocity float64) {
		if velocity == 0 {
			return
		}
		s.motion.AnimateWith(animation.Clamped{
			Sim: animation.Friction{Drag: animation.ScrollFriction, Position: s.offset, Velocity: -velocity},
			Min: 0, Max: s.metrics.MaxOffset(),
		})
	}
	g := GestureDetector{Behavior: rendering.Opaque, Child: view}
	if s.Horizontal {
		g.OnHorizontalDragUpdate = func(d gestures.DragUpdateDetails) { drag(d.Delta.X) }
		g.OnHorizontalDragEnd = func(d gestures.DragEndDetails) { fling(d.Velocity.X) }
	} else {
		g.OnVerticalDragUpdate = func(d gestures.DragUpdateDetails) { drag(d.Delta.Y) }
		g.OnVerticalDragEnd = func(d gestures.DragEndDetails) { fling(d.Velocity.Y) }
	}
	// Касание останавливает летящий список.
	return Listener{OnPointerDown: func(gestures.PointerEvent) { s.motion.Stop() }, Child: g}
}

// scrollViewport — видимая область прокрутки (rendering.ScrollViewportBox).
type scrollViewport struct {
	Base
	Axis    rendering.Axis
	Offset  float64
	Metrics *rendering.ScrollMetrics
	Child   Widget
}

func (w scrollViewport) ChildWidget() Widget { return w.Child }

func (w scrollViewport) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.ScrollViewportBox{Axis: w.Axis, Offset: w.Offset, Metrics: w.Metrics}
}

func (w scrollViewport) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	v := b.(*rendering.ScrollViewportBox)
	v.Axis, v.Offset, v.Metrics = w.Axis, w.Offset, w.Metrics
}

// SingleChildScrollView прокручивает одного ребёнка, который не помещается на экран:
// длинную форму, текст. По умолчанию — вертикально.
type SingleChildScrollView struct {
	Base
	Horizontal bool
	Controller *ScrollController
	Padding    geom.EdgeInsets
	Child      Widget
}

func (w SingleChildScrollView) Build(BuildContext) Widget {
	return scrollable{Horizontal: w.Horizontal, Controller: w.Controller,
		Content: func(BuildContext, float64, float64) Widget { return Padding{Padding: w.Padding, Child: w.Child} }}
}

// ListView — прокручиваемый список. Два вида:
//
//   - Children — все строки сразу (для коротких списков);
//
//   - ItemCount, ItemExtent и ItemBuilder — ленивый: строятся только строки на экране и рядом,
//     поэтому список может быть сколь угодно длинным. Высота строки (ItemExtent) пока обязательна.
//
//     widgets.ListView{ItemCount: 10000, ItemExtent: 56, ItemBuilder: func(ctx widgets.BuildContext, i int) widgets.Widget {
//     return material.ListTile{Title: widgets.Text{Data: fmt.Sprint("Строка ", i)}}
//     }}
type ListView struct {
	Base
	Horizontal bool
	Controller *ScrollController
	Padding    geom.EdgeInsets
	Spacing    float64 // зазор между строками (для Children)
	Children   []Widget

	ItemCount   int
	ItemExtent  float64
	ItemBuilder func(ctx BuildContext, i int) Widget
}

// listCache — сколько dp за краями экрана строить заранее, чтобы при прокрутке строки были готовы.
const listCache = 250.0

func (w ListView) Build(BuildContext) Widget {
	content := func(BuildContext, float64, float64) Widget {
		var list Widget = Column{MainAxisSize: rendering.MainAxisMin, CrossAxisAlignment: rendering.CrossAxisStretch,
			Spacing: w.Spacing, Children: w.Children}
		if w.Horizontal {
			list = Row{MainAxisSize: rendering.MainAxisMin, CrossAxisAlignment: rendering.CrossAxisStretch,
				Spacing: w.Spacing, Children: w.Children}
		}
		return Padding{Padding: w.Padding, Child: list}
	}
	if w.ItemBuilder != nil {
		if w.ItemExtent <= 0 {
			panic(fmt.Sprintf("widgets: ленивому ListView (ItemBuilder) пока нужна высота строки ItemExtent, а она %v", w.ItemExtent))
		}
		content = w.lazy
	}
	return scrollable{Horizontal: w.Horizontal, Controller: w.Controller, Content: content}
}

// lazy строит только строки в видимой области (плюс запас listCache с каждой стороны).
func (w ListView) lazy(ctx BuildContext, offset, viewport float64) Widget {
	start, end := w.Padding.Top, w.Padding.Bottom
	if w.Horizontal {
		start, end = w.Padding.Left, w.Padding.Right
	}
	first := int(math.Floor((offset - listCache - start) / w.ItemExtent))
	last := int(math.Ceil((offset + viewport + listCache - start) / w.ItemExtent))
	first, last = max(first, 0), min(last, w.ItemCount)
	items := make([]Widget, 0, max(last-first, 0))
	for i := first; i < last; i++ {
		at := start + float64(i)*w.ItemExtent
		p := Positioned{Top: new(at), Left: new(w.Padding.Left), Right: new(w.Padding.Right), Height: new(w.ItemExtent)}
		if w.Horizontal {
			p = Positioned{Left: new(at), Top: new(w.Padding.Top), Bottom: new(w.Padding.Bottom), Width: new(w.ItemExtent)}
		}
		p.Child = w.ItemBuilder(ctx, i)
		items = append(items, Keyed{Key: i, Child: p})
	}
	total := start + float64(w.ItemCount)*w.ItemExtent + end
	if w.Horizontal {
		return SizedBox{Width: new(total), Child: Stack{Children: items}}
	}
	return SizedBox{Height: new(total), Child: Stack{Children: items}}
}

// GridView — прокручиваемая сетка: CrossAxisCount колонок, ячейки с соотношением сторон
// ChildAspectRatio (0 — квадратные).
type GridView struct {
	Base
	Controller       *ScrollController
	Padding          geom.EdgeInsets
	CrossAxisCount   int
	Spacing          float64 // между колонками
	RunSpacing       float64 // между рядами
	ChildAspectRatio float64
	Children         []Widget
}

func (w GridView) Build(BuildContext) Widget {
	n := max(w.CrossAxisCount, 1)
	ratio := w.ChildAspectRatio
	if ratio <= 0 {
		ratio = 1
	}
	var rows []Widget
	for i := 0; i < len(w.Children); i += n {
		cells := make([]Widget, 0, n)
		for j := i; j < i+n; j++ {
			var c Widget = SizedBox{}
			if j < len(w.Children) {
				c = w.Children[j]
			}
			cells = append(cells, Expanded{Child: AspectRatio{Ratio: ratio, Child: c}})
		}
		rows = append(rows, Row{Spacing: w.Spacing, Children: cells})
	}
	return SingleChildScrollView{Controller: w.Controller, Padding: w.Padding, Child: Column{
		MainAxisSize: rendering.MainAxisMin, CrossAxisAlignment: rendering.CrossAxisStretch, Spacing: w.RunSpacing, Children: rows,
	}}
}

// PageController управляет PageView снаружи: какая страница (Page) и перелистнуть (JumpTo, AnimateTo).
type PageController struct {
	p *PageView
}

// Page — текущая страница (во время перелистывания — ближайшая).
func (c *PageController) Page() int {
	if c.p == nil {
		return 0
	}
	return int(math.Round(c.p.page))
}

// JumpTo открывает страницу сразу.
func (c *PageController) JumpTo(page int) {
	if c.p != nil {
		c.p.motion.Stop()
		c.p.SetState(func() { c.p.page = c.p.clampPage(float64(page)) })
		c.p.report()
	}
}

// AnimateTo плавно перелистывает к странице.
func (c *PageController) AnimateTo(page int) {
	if c.p != nil {
		c.p.settle(float64(page))
	}
}

// PageView — страницы, которые листают пальцем: каждая во весь размер PageView, отпущенная страница
// доезжает до ближайшей (или до следующей, если её бросили). По умолчанию листается по горизонтали.
type PageView struct {
	State
	Vertical      bool
	Pages         []Widget
	InitialPage   int
	OnPageChanged func(int)
	Controller    *PageController

	page     float64 // текущее положение в страницах (дробное во время перелистывания)
	reported int
	metrics  rendering.ScrollMetrics // Viewport — размер страницы
	motion   *animation.Controller
}

func (p *PageView) Init(ctx BuildContext) {
	p.page = p.clampPage(float64(p.InitialPage))
	p.reported = int(p.page)
	p.motion = NewAnimation(ctx, 0)
	p.motion.AddListener(func() { p.page = p.motion.Value() })
	if p.Controller != nil {
		p.Controller.p = p
	}
}

func (p *PageView) Update(_ BuildContext, old PageView) {
	if old.Controller != p.Controller {
		if old.Controller != nil && old.Controller.p == p {
			old.Controller.p = nil
		}
		if p.Controller != nil {
			p.Controller.p = p
		}
	}
}

func (p *PageView) Dispose() {
	if p.Controller != nil && p.Controller.p == p {
		p.Controller.p = nil
	}
}

func (p *PageView) clampPage(v float64) float64 {
	return math.Max(0, math.Min(v, float64(max(len(p.Pages)-1, 0))))
}

// settle — доехать до страницы target.
func (p *PageView) settle(target float64) {
	target = p.clampPage(math.Round(target))
	p.motion.AnimateWith(animation.TweenSimulation(p.page, target, defaultDuration, animation.Standard))
	p.reportPage(int(target))
}

func (p *PageView) report() { p.reportPage(int(math.Round(p.page))) }

func (p *PageView) reportPage(i int) {
	if i != p.reported {
		p.reported = i
		if p.OnPageChanged != nil {
			p.OnPageChanged(i)
		}
	}
}

// pageFling — быстрее этого (dp/с) отпущенная страница перелистывается дальше, а не возвращается.
const pageFling = 300.0

func (p *PageView) Build(BuildContext) Widget {
	if !p.motion.IsAnimating() {
		p.page = p.clampPage(p.page)
	}
	drag := func(delta float64) {
		if p.metrics.Viewport <= 0 {
			return
		}
		p.motion.Stop()
		p.SetState(func() { p.page = p.clampPage(p.page - delta/p.metrics.Viewport) })
	}
	end := func(velocity float64) {
		target := math.Round(p.page)
		switch {
		case velocity < -pageFling:
			target = math.Floor(p.page) + 1
		case velocity > pageFling:
			target = math.Ceil(p.page) - 1
		}
		p.settle(target)
	}
	axis := rendering.Horizontal
	if p.Vertical {
		axis = rendering.Vertical
	}
	g := GestureDetector{Behavior: rendering.Opaque,
		Child: pageViewport{Axis: axis, Page: p.page, Metrics: &p.metrics, Pages: p.Pages}}
	if p.Vertical {
		g.OnVerticalDragUpdate = func(d gestures.DragUpdateDetails) { drag(d.Delta.Y) }
		g.OnVerticalDragEnd = func(d gestures.DragEndDetails) { end(d.Velocity.Y) }
	} else {
		g.OnHorizontalDragUpdate = func(d gestures.DragUpdateDetails) { drag(d.Delta.X) }
		g.OnHorizontalDragEnd = func(d gestures.DragEndDetails) { end(d.Velocity.X) }
	}
	return Listener{OnPointerDown: func(gestures.PointerEvent) { p.motion.Stop() }, Child: g}
}

// pageViewport — страницы рядом друг с другом (rendering.PageViewportBox).
type pageViewport struct {
	Base
	Axis    rendering.Axis
	Page    float64
	Metrics *rendering.ScrollMetrics
	Pages   []Widget
}

func (w pageViewport) ChildWidgets() []Widget { return w.Pages }

func (w pageViewport) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.PageViewportBox{Axis: w.Axis, Page: w.Page, Metrics: w.Metrics}
}

func (w pageViewport) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	v := b.(*rendering.PageViewportBox)
	v.Axis, v.Page, v.Metrics = w.Axis, w.Page, w.Metrics
}

// EnsureVisible плавно прокручивает ближайшую прокрутку над ctx так, чтобы виджет был виден целиком
// вместе с запасом padding (сверху — над ним, снизу — под ним). Так поле ввода выезжает из-под
// клавиатуры. Положения берутся из прошлого кадра.
func EnsureVisible(ctx BuildContext, padding geom.EdgeInsets) {
	e := ctx.element()
	target := BoundsOf(e)
	if target.IsEmpty() {
		return
	}
	for p := e.parent; p != nil; p = p.parent {
		s, ok := p.State().(*scrollable)
		if !ok {
			continue
		}
		view := BoundsOf(p)
		var delta float64
		if s.Horizontal {
			lo, hi := target.Left-padding.Left, target.Right+padding.Right
			switch {
			case hi > view.Right && lo-(hi-view.Right) >= view.Left:
				delta = hi - view.Right
			case lo < view.Left:
				delta = lo - view.Left
			}
		} else {
			lo, hi := target.Top-padding.Top, target.Bottom+padding.Bottom
			switch {
			case hi > view.Bottom:
				delta = math.Min(hi-view.Bottom, lo-view.Top) // не уводить верх за край
			case lo < view.Top:
				delta = lo - view.Top
			}
		}
		if delta != 0 {
			s.animateTo(s.offset+delta, 200*time.Millisecond, animation.Standard)
		}
		return
	}
}
