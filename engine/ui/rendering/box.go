package rendering

import (
	"fmt"
	"math"

	"hexago/ui/geom"
	"hexago/ui/gfx"
)

// Box — render-объект прямоугольной формы, как RenderBox во Flutter.
//
// Конкретный бокс встраивает Node (или Proxy, если у него один ребёнок) и реализует
// PerformLayout и Paint. Сами эти методы вызывают не напрямую, а через функции пакета:
// Layout(child, c) и ctx.PaintChild(child, offset).
type Box interface {
	// PerformLayout выбирает размер бокса в пределах c, раскладывает детей через Layout
	// и выставляет им позиции через SetOffset.
	PerformLayout(c Constraints) geom.Size
	// Paint рисует бокс и его детей; offset — где лежит левый верхний угол бокса.
	Paint(ctx *PaintContext, offset geom.Offset)
	node() *Node
}

// Node — общие данные любого бокса: размер, позиция в родителе, дети, данные родителя.
type Node struct {
	size        geom.Size
	offset      geom.Offset
	constraints Constraints
	children    []Box

	// ParentData — что родитель знает о ребёнке: FlexParentData у детей Row/Column,
	// StackParentData у детей Stack. Выставляют виджеты Expanded, Positioned и подобные.
	ParentData any

	damage damageState // для перерисовки только изменившегося (damage.go)
}

func (n *Node) node() *Node { return n }

// SizeOf — размер бокса после лэйаута.
func SizeOf(b Box) geom.Size { return b.node().size }

// OffsetOf — позиция бокса внутри родителя.
func OffsetOf(b Box) geom.Offset { return b.node().offset }

// SetOffset выставляет позицию ребёнка внутри родителя (вызывает родитель в PerformLayout).
func SetOffset(b Box, o geom.Offset) { b.node().offset = o }

// ConstraintsOf — ограничения, с которыми бокс раскладывали последний раз.
func ConstraintsOf(b Box) Constraints { return b.node().constraints }

// Children — дети бокса.
func Children(b Box) []Box { return b.node().children }

// Child — единственный (первый) ребёнок или nil.
func Child(b Box) Box {
	if ch := b.node().children; len(ch) > 0 {
		return ch[0]
	}
	return nil
}

// SetChildren заменяет детей бокса. Вызывает дерево элементов после перестройки.
func SetChildren(b Box, children []Box) { b.node().children = children }

// Layout раскладывает бокс с ограничениями c и возвращает его размер.
func Layout(b Box, c Constraints) geom.Size {
	n := b.node()
	n.constraints = c
	s := c.Constrain(b.PerformLayout(c))
	if math.IsInf(s.Width, 0) || math.IsInf(s.Height, 0) || math.IsNaN(s.Width) || math.IsNaN(s.Height) {
		panic(fmt.Sprintf("rendering: %T получил бесконечный размер %v при ограничениях %v. "+
			"Обычно это значит, что растягивающийся виджет (Expanded, SizedBox.Expand, Container без размера) "+
			"оказался внутри Row/Column/ListView по оси без ограничения.", b, s, c))
	}
	n.size = s
	return s
}

// PaintContext — куда рисовать: холст и текущий кадр.
type PaintContext struct {
	Canvas *gfx.Canvas
}

// PaintChild рисует ребёнка: offset — позиция родителя, к ней прибавляется позиция ребёнка.
// Ребёнок целиком вне области, которую сейчас можно рисовать (перерисовка только изменившегося),
// пропускается — вместе со всем своим поддеревом.
func (ctx *PaintContext) PaintChild(child Box, offset geom.Offset) {
	n := child.node()
	at := offset.Add(n.offset)
	margin := overflowMargin
	if o, ok := child.(PaintOverflower); ok {
		margin += o.PaintOverflow()
	}
	if _, transforms := child.(ChildTransformer); !transforms && // дети с поворотом могут выйти за края
		ctx.Canvas.QuickReject(geom.RectFromSize(at, n.size).Inflate(margin)) {
		return
	}
	child.Paint(ctx, at)
}

// PaintChildren рисует всех детей бокса по порядку.
func PaintChildren(ctx *PaintContext, b Box, offset geom.Offset) {
	for _, ch := range Children(b) {
		ctx.PaintChild(ch, offset)
	}
}

// Proxy — бокс с одним ребёнком, который повторяет его размер и просто рисует его.
// Основа для боксов, которые что-то добавляют к ребёнку: цвет, прозрачность, обрезку.
type Proxy struct {
	Node
}

// PerformLayout передаёт ограничения ребёнку и берёт его размер; без ребёнка — минимальный размер.
func (p *Proxy) PerformLayout(c Constraints) geom.Size {
	if ch := Child(p); ch != nil {
		return Layout(ch, c)
	}
	return c.Smallest()
}

// Paint рисует ребёнка.
func (p *Proxy) Paint(ctx *PaintContext, offset geom.Offset) { PaintChildren(ctx, p, offset) }

// HitEntry — бокс, в который попало касание, и точка касания в его координатах.
type HitEntry struct {
	Box   Box
	Local geom.Offset
	// Transform переводит точку из координат экрана в координаты бокса: по нему
	// считается Local у следующих событий того же пальца.
	Transform geom.Affine
}

// HitTestResult — все боксы под точкой касания, от самого глубокого к корню.
type HitTestResult struct {
	Path       []HitEntry
	transforms []geom.Affine // стек: из координат экрана в координаты текущего бокса
}

// Add записывает бокс b, в который попала точка pos (в координатах b).
func (r *HitTestResult) Add(b Box, pos geom.Offset) {
	r.Path = append(r.Path, HitEntry{Box: b, Local: pos, Transform: r.current()})
}

// PushTransform — проверка уходит к ребёнку: t переводит координаты родителя в координаты ребёнка.
func (r *HitTestResult) PushTransform(t geom.Affine) {
	r.transforms = append(r.transforms, t.Mul(r.current()))
}

// PopTransform — проверка ребёнка закончена.
func (r *HitTestResult) PopTransform() { r.transforms = r.transforms[:len(r.transforms)-1] }

func (r *HitTestResult) current() geom.Affine {
	if len(r.transforms) == 0 {
		return geom.Identity
	}
	return r.transforms[len(r.transforms)-1]
}

// SelfHitTester — бокс, который сам «ловит» касания (например, с цветным фоном).
type SelfHitTester interface {
	HitTestSelf(pos geom.Offset) bool
}

// CustomHitTester — бокс со своей проверкой попадания (преобразования, обрезки).
type CustomHitTester interface {
	HitTest(result *HitTestResult, pos geom.Offset) bool
}

// HitTest проверяет, попадает ли точка pos (в координатах бокса) в бокс или его детей,
// и записывает всех попавших в result.
func HitTest(b Box, result *HitTestResult, pos geom.Offset) bool {
	if h, ok := b.(CustomHitTester); ok {
		return h.HitTest(result, pos)
	}
	return DefaultHitTest(b, result, pos)
}

// DefaultHitTest — обычная проверка: внутри прямоугольника бокса, потом дети, потом сам бокс.
func DefaultHitTest(b Box, result *HitTestResult, pos geom.Offset) bool {
	if !b.node().size.Contains(pos) {
		return false
	}
	hit := HitTestChildren(b, result, pos)
	if s, ok := b.(SelfHitTester); ok && s.HitTestSelf(pos) {
		hit = true
	}
	if hit {
		result.Add(b, pos)
	}
	return hit
}

// HitTestChildren проверяет детей сверху вниз (последний нарисованный — первым).
func HitTestChildren(b Box, result *HitTestResult, pos geom.Offset) bool {
	ch := Children(b)
	for i := len(ch) - 1; i >= 0; i-- {
		off := ch[i].node().offset
		result.PushTransform(geom.Translation(-off.X, -off.Y))
		hit := HitTest(ch[i], result, pos.Sub(off))
		result.PopTransform()
		if hit {
			return true
		}
	}
	return false
}

// ParentDataOf — данные родителя о боксе (FlexParentData, StackParentData или nil).
func ParentDataOf(b Box) any { return b.node().ParentData }

// SetParentData выставляет данные родителя о боксе.
func SetParentData(b Box, pd any) { b.node().ParentData = pd }
