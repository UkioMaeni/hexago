package widgets

import (
	"hexago/ui/gestures"
	"hexago/ui/rendering"
)

// Listener — сырые касания, как Listener во Flutter: палец коснулся, двигается, отпущен,
// касание отменено. Без распознавания жестов — для этого GestureDetector.
//
// Behavior — как ловить касания (по умолчанию — только если они попали в ребёнка):
// rendering.Opaque — весь прямоугольник и не пропускать дальше, rendering.Translucent — весь
// прямоугольник, но и то, что под ним, тоже получит касание.
type Listener struct {
	Base
	Behavior        rendering.HitTestBehavior
	OnPointerDown   func(gestures.PointerEvent)
	OnPointerMove   func(gestures.PointerEvent)
	OnPointerUp     func(gestures.PointerEvent)
	OnPointerCancel func(gestures.PointerEvent)
	Child           Widget
}

func (w Listener) ChildWidget() Widget { return w.Child }

func (w Listener) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.PointerListenerBox{Behavior: w.Behavior, OnPointer: w.dispatch}
}

func (w Listener) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	l := b.(*rendering.PointerListenerBox)
	l.Behavior, l.OnPointer = w.Behavior, w.dispatch
}

func (w Listener) dispatch(e gestures.PointerEvent) {
	var f func(gestures.PointerEvent)
	switch e.Kind {
	case gestures.PointerDown:
		f = w.OnPointerDown
	case gestures.PointerMove:
		f = w.OnPointerMove
	case gestures.PointerUp:
		f = w.OnPointerUp
	case gestures.PointerCancel:
		f = w.OnPointerCancel
	}
	if f != nil {
		f(e)
	}
}

// IgnorePointer — поддерево не замечает касаний: они проходят сквозь него к тому, что под ним.
// Disabled: true — выключить IgnorePointer, не убирая его из дерева.
type IgnorePointer struct {
	Base
	Disabled bool
	Child    Widget
}

func (w IgnorePointer) ChildWidget() Widget { return w.Child }

func (w IgnorePointer) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.IgnorePointerBox{Disabled: w.Disabled}
}

func (w IgnorePointer) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	b.(*rendering.IgnorePointerBox).Disabled = w.Disabled
}

// AbsorbPointer — поглощает касания: их не получает ни поддерево, ни то, что под ним
// (например, заблокировать форму, пока она отправляется). Disabled: true — выключить.
type AbsorbPointer struct {
	Base
	Disabled bool
	Child    Widget
}

func (w AbsorbPointer) ChildWidget() Widget { return w.Child }

func (w AbsorbPointer) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.AbsorbPointerBox{Disabled: w.Disabled}
}

func (w AbsorbPointer) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	b.(*rendering.AbsorbPointerBox).Disabled = w.Disabled
}

// GestureDetector распознаёт жесты над ребёнком, как GestureDetector во Flutter: нажатие,
// двойное и долгое нажатие, перетаскивание (вертикальное, горизонтальное, свободное), масштаб.
// Задайте только нужные обработчики — распознаватели создаются по ним. Если на касание претендуют
// несколько жестов (свой и соседних детекторов), победителя выбирает арена жестов:
// при прочих равных — самый глубокий детектор под пальцем.
//
// Без ребёнка детектор занимает всё доступное место и ловит касания на всей площади.
// Свободное перетаскивание (OnPan...) и масштаб (OnScale...) вместе не задают: масштаб одним
// пальцем и есть перетаскивание.
type GestureDetector struct {
	State
	Child    Widget
	Behavior rendering.HitTestBehavior

	OnTapDown   func(gestures.TapDownDetails)
	OnTapUp     func(gestures.TapUpDetails)
	OnTap       func()
	OnTapCancel func()

	OnDoubleTap func()

	OnLongPress           func()
	OnLongPressStart      func(gestures.LongPressStartDetails)
	OnLongPressMoveUpdate func(gestures.LongPressMoveUpdateDetails)
	OnLongPressEnd        func(gestures.LongPressEndDetails)
	OnLongPressUp         func()

	OnVerticalDragStart  func(gestures.DragStartDetails)
	OnVerticalDragUpdate func(gestures.DragUpdateDetails)
	OnVerticalDragEnd    func(gestures.DragEndDetails)

	OnHorizontalDragStart  func(gestures.DragStartDetails)
	OnHorizontalDragUpdate func(gestures.DragUpdateDetails)
	OnHorizontalDragEnd    func(gestures.DragEndDetails)

	OnPanStart  func(gestures.DragStartDetails)
	OnPanUpdate func(gestures.DragUpdateDetails)
	OnPanEnd    func(gestures.DragEndDetails)

	OnScaleStart  func(gestures.ScaleStartDetails)
	OnScaleUpdate func(gestures.ScaleUpdateDetails)
	OnScaleEnd    func(gestures.ScaleEndDetails)

	recognizers [recognizerCount]gestures.Recognizer
}

// Распознаватели детектора — в этом порядке они входят в арену.
const (
	recTap = iota
	recDoubleTap
	recLongPress
	recVertical
	recHorizontal
	recPan
	recScale
	recognizerCount
)

func (g *GestureDetector) Build(ctx BuildContext) Widget {
	g.sync(ctx)
	child, behavior := g.Child, g.Behavior
	if child == nil {
		child = SizedBoxExpand(nil)
		if behavior == rendering.DeferToChild {
			behavior = rendering.Translucent
		}
	}
	return Listener{
		Behavior:        behavior,
		OnPointerDown:   g.addPointer,
		OnPointerMove:   g.handleEvent,
		OnPointerUp:     g.handleEvent,
		OnPointerCancel: g.handleEvent,
		Child:           child,
	}
}

func (g *GestureDetector) Dispose() {
	for i, r := range g.recognizers {
		if r != nil {
			r.Dispose()
			g.recognizers[i] = nil
		}
	}
}

func (g *GestureDetector) addPointer(e gestures.PointerEvent) {
	for _, r := range g.recognizers {
		if r != nil {
			r.AddPointer(e)
		}
	}
}

func (g *GestureDetector) handleEvent(e gestures.PointerEvent) {
	for _, r := range g.recognizers {
		if r != nil {
			r.HandleEvent(e)
		}
	}
}

// sync создаёт распознаватели для заданных обработчиков и убирает ненужные. Обработчики
// распознаватели берут из g в момент вызова — после перестройки родителя действуют новые.
func (g *GestureDetector) sync(ctx BuildContext) {
	pan := g.OnPanStart != nil || g.OnPanUpdate != nil || g.OnPanEnd != nil
	scale := g.OnScaleStart != nil || g.OnScaleUpdate != nil || g.OnScaleEnd != nil
	if pan && scale {
		panic("widgets: у GestureDetector заданы и OnPan..., и OnScale... — масштаб одним пальцем уже " +
			"перетаскивание; оставьте только OnScale...")
	}
	need := [recognizerCount]bool{
		recTap:       g.OnTapDown != nil || g.OnTapUp != nil || g.OnTap != nil || g.OnTapCancel != nil,
		recDoubleTap: g.OnDoubleTap != nil,
		recLongPress: g.OnLongPress != nil || g.OnLongPressStart != nil || g.OnLongPressMoveUpdate != nil ||
			g.OnLongPressEnd != nil || g.OnLongPressUp != nil,
		recVertical:   g.OnVerticalDragStart != nil || g.OnVerticalDragUpdate != nil || g.OnVerticalDragEnd != nil,
		recHorizontal: g.OnHorizontalDragStart != nil || g.OnHorizontalDragUpdate != nil || g.OnHorizontalDragEnd != nil,
		recPan:        pan,
		recScale:      scale,
	}
	env := ctx.element().owner.gestures
	for i, want := range need {
		switch {
		case want && g.recognizers[i] == nil:
			g.recognizers[i] = g.newRecognizer(i, env)
		case !want && g.recognizers[i] != nil:
			g.recognizers[i].Dispose()
			g.recognizers[i] = nil
		}
	}
}

func (g *GestureDetector) newRecognizer(kind int, env gestures.Env) gestures.Recognizer {
	switch kind {
	case recTap:
		return &gestures.TapRecognizer{Env: env,
			OnTapDown:   func(d gestures.TapDownDetails) { call1(g.OnTapDown, d) },
			OnTapUp:     func(d gestures.TapUpDetails) { call1(g.OnTapUp, d) },
			OnTap:       func() { call0(g.OnTap) },
			OnTapCancel: func() { call0(g.OnTapCancel) },
		}
	case recDoubleTap:
		return &gestures.DoubleTapRecognizer{Env: env, OnDoubleTap: func() { call0(g.OnDoubleTap) }}
	case recLongPress:
		return &gestures.LongPressRecognizer{Env: env,
			OnLongPressStart:      func(d gestures.LongPressStartDetails) { call1(g.OnLongPressStart, d) },
			OnLongPress:           func() { call0(g.OnLongPress) },
			OnLongPressMoveUpdate: func(d gestures.LongPressMoveUpdateDetails) { call1(g.OnLongPressMoveUpdate, d) },
			OnLongPressEnd:        func(d gestures.LongPressEndDetails) { call1(g.OnLongPressEnd, d) },
			OnLongPressUp:         func() { call0(g.OnLongPressUp) },
		}
	case recVertical:
		return &gestures.DragRecognizer{Env: env, Axis: gestures.DragVertical,
			OnStart:  func(d gestures.DragStartDetails) { call1(g.OnVerticalDragStart, d) },
			OnUpdate: func(d gestures.DragUpdateDetails) { call1(g.OnVerticalDragUpdate, d) },
			OnEnd:    func(d gestures.DragEndDetails) { call1(g.OnVerticalDragEnd, d) },
		}
	case recHorizontal:
		return &gestures.DragRecognizer{Env: env, Axis: gestures.DragHorizontal,
			OnStart:  func(d gestures.DragStartDetails) { call1(g.OnHorizontalDragStart, d) },
			OnUpdate: func(d gestures.DragUpdateDetails) { call1(g.OnHorizontalDragUpdate, d) },
			OnEnd:    func(d gestures.DragEndDetails) { call1(g.OnHorizontalDragEnd, d) },
		}
	case recPan:
		return &gestures.DragRecognizer{Env: env, Axis: gestures.DragFree,
			OnStart:  func(d gestures.DragStartDetails) { call1(g.OnPanStart, d) },
			OnUpdate: func(d gestures.DragUpdateDetails) { call1(g.OnPanUpdate, d) },
			OnEnd:    func(d gestures.DragEndDetails) { call1(g.OnPanEnd, d) },
		}
	default:
		return &gestures.ScaleRecognizer{Env: env,
			OnStart:  func(d gestures.ScaleStartDetails) { call1(g.OnScaleStart, d) },
			OnUpdate: func(d gestures.ScaleUpdateDetails) { call1(g.OnScaleUpdate, d) },
			OnEnd:    func(d gestures.ScaleEndDetails) { call1(g.OnScaleEnd, d) },
		}
	}
}

func call0(f func()) {
	if f != nil {
		f()
	}
}

func call1[T any](f func(T), v T) {
	if f != nil {
		f(v)
	}
}
