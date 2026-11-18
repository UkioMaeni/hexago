package rendering

import (
	"hexago/ui/geom"
	"hexago/ui/gestures"
)

// PointerHandler — бокс, которому доставляются касания. Если касание попало в бокс (hit-test),
// он получает все события этого пальца до его отпускания — даже когда палец ушёл за его края.
type PointerHandler interface {
	HandlePointer(e gestures.PointerEvent)
}

// HitTestBehavior — как бокс с обработчиком касаний участвует в hit-test.
type HitTestBehavior uint8

const (
	// DeferToChild — касание засчитывается, только если попало в ребёнка (по умолчанию).
	DeferToChild HitTestBehavior = iota
	// Opaque — весь прямоугольник бокса ловит касания, и то, что под ним, их не получает.
	Opaque
	// Translucent — весь прямоугольник ловит касания, но и то, что под ним, тоже их получает.
	Translucent
)

// PointerListenerBox передаёт касания в OnPointer — основа Listener и GestureDetector.
type PointerListenerBox struct {
	Proxy
	Behavior  HitTestBehavior
	OnPointer func(e gestures.PointerEvent)
}

func (b *PointerListenerBox) HandlePointer(e gestures.PointerEvent) {
	if b.OnPointer != nil {
		b.OnPointer(e)
	}
}

func (b *PointerListenerBox) HitTest(result *HitTestResult, pos geom.Offset) bool {
	if !SizeOf(b).Contains(pos) {
		return false
	}
	hit := HitTestChildren(b, result, pos) || b.Behavior == Opaque
	if hit || b.Behavior == Translucent {
		result.Add(b, pos)
	}
	return hit
}

// IgnorePointerBox — поддерево не замечает касаний: они проходят к тому, что под ним.
type IgnorePointerBox struct {
	Proxy
	Disabled bool // true — IgnorePointer выключен, касания доходят до поддерева как обычно
}

func (b *IgnorePointerBox) HitTest(result *HitTestResult, pos geom.Offset) bool {
	if !b.Disabled {
		return false
	}
	return DefaultHitTest(b, result, pos)
}

// AbsorbPointerBox — поглощает касания: ни поддерево, ни то, что под ним, их не получает.
type AbsorbPointerBox struct {
	Proxy
	Disabled bool // true — AbsorbPointer выключен, касания доходят до поддерева как обычно
}

func (b *AbsorbPointerBox) HitTest(result *HitTestResult, pos geom.Offset) bool {
	if !b.Disabled {
		return SizeOf(b).Contains(pos)
	}
	return DefaultHitTest(b, result, pos)
}
