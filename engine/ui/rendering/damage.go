package rendering

import (
	"math"
	"reflect"

	"hexago/ui/geom"
)

// Перерисовка только изменившегося. Каждый бокс помнит, где он был на экране в прошлом кадре.
// После раскладки CollectDamage обходит дерево: «повреждённой» становится область бокса, который
// поменял то, как он рисуется (MarkNeedsPaint), сдвинулся или изменил размер, — и где он был,
// и где он теперь. Перерисовывается только повреждённый прямоугольник; остальное остаётся
// с прошлого кадра.

// damageState — то, что бокс помнит для учёта повреждений.
type damageState struct {
	dirty   bool      // поменял то, как рисуется
	prev    geom.Rect // где был на экране в прошлом кадре (dp, с запасом PaintOverflow)
	hasPrev bool
}

// MarkNeedsPaint помечает, что бокс рисуется иначе, чем в прошлом кадре.
func MarkNeedsPaint(b Box) { b.node().damage.dirty = true }

// PaintOverflower — бокс, который рисует за своими границами (тень): насколько, в dp.
type PaintOverflower interface {
	PaintOverflow() float64
}

// ChildTransformer — бокс, который рисует детей с преобразованием (TransformBox).
type ChildTransformer interface {
	ChildTransform() geom.Affine
}

// LayoutOnly — бокс, который сам ничего не рисует, только расставляет детей (Row, Padding…):
// изменение его параметров не повреждает экран само по себе — сдвиги детей заметит сравнение границ.
type LayoutOnly interface {
	layoutOnly()
}

// overflowMargin — запас вокруг любого бокса: сглаженные края и выносные элементы букв.
const overflowMargin = 2.0

// PrevBounds — где бокс был на экране в прошлом кадре (dp).
func PrevBounds(b Box) (geom.Rect, bool) {
	d := b.node().damage
	return d.prev, d.hasPrev
}

// CollectDamage обходит дерево после раскладки и возвращает повреждённую область (в dp, координаты
// экрана). Пустой прямоугольник — ничего не изменилось.
func CollectDamage(root Box) geom.Rect {
	var damage geom.Rect
	add := func(r geom.Rect) {
		if r.IsEmpty() {
			return
		}
		if damage.IsEmpty() {
			damage = r
			return
		}
		damage = geom.Rect{Left: math.Min(damage.Left, r.Left), Top: math.Min(damage.Top, r.Top),
			Right: math.Max(damage.Right, r.Right), Bottom: math.Max(damage.Bottom, r.Bottom)}
	}
	var walk func(b Box, m geom.Affine)
	walk = func(b Box, m geom.Affine) {
		n := b.node()
		margin := overflowMargin
		if o, ok := b.(PaintOverflower); ok {
			margin += o.PaintOverflow()
		}
		cur := m.MapRect(geom.RectFromSize(geom.Zero, n.size)).Inflate(margin)
		d := &n.damage
		if d.dirty || !d.hasPrev || d.prev != cur {
			if d.hasPrev {
				add(d.prev)
			}
			add(cur)
		}
		d.prev, d.hasPrev, d.dirty = cur, true, false
		cm := m
		if t, ok := b.(ChildTransformer); ok {
			cm = m.Mul(t.ChildTransform())
		}
		for _, ch := range n.children {
			o := ch.node().offset
			walk(ch, cm.Mul(geom.Translation(o.X, o.Y)))
		}
	}
	walk(root, geom.Identity)
	return damage
}

// PaintSnapshot — параметры рисования бокса до обновления: по ним SamePaint решает, изменилось ли
// что-то. Сравниваются все поля бокса, кроме встроенных Node и Proxy (размер, дети — это раскладка).
type PaintSnapshot struct{ v reflect.Value }

// Snapshot запоминает параметры бокса.
func Snapshot(b Box) PaintSnapshot {
	if _, ok := b.(LayoutOnly); ok {
		return PaintSnapshot{}
	}
	v := reflect.ValueOf(b).Elem()
	c := reflect.New(v.Type()).Elem()
	c.Set(v)
	return PaintSnapshot{c}
}

// MarkIfChanged помечает бокс для перерисовки, если его параметры отличаются от снимка s.
func MarkIfChanged(b Box, s PaintSnapshot) {
	if !s.v.IsValid() {
		return // LayoutOnly
	}
	v := reflect.ValueOf(b).Elem()
	t := v.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Anonymous && (f.Type == nodeType || f.Type == proxyType) {
			continue
		}
		a, c := s.v.Field(i), v.Field(i)
		if !f.IsExported() {
			// Неэкспортируемые поля (кеши раскладки, как свёрстанный абзац) читаются без Interface.
			if !reflect.DeepEqual(valueOf(a), valueOf(c)) {
				MarkNeedsPaint(b)
				return
			}
			continue
		}
		if !reflect.DeepEqual(a.Interface(), c.Interface()) {
			MarkNeedsPaint(b)
			return
		}
	}
}

var (
	nodeType  = reflect.TypeFor[Node]()
	proxyType = reflect.TypeFor[Proxy]()
)

// valueOf — значение поля для сравнения; у неэкспортируемых полей — только простые виды.
func valueOf(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.String:
		return v.String()
	case reflect.Pointer:
		return v.Pointer()
	}
	return nil // сложные неэкспортируемые поля считаются кешем раскладки и не сравниваются
}

// Боксы, которые только расставляют детей.
func (*FlexBox) layoutOnly()            {}
func (*PaddingBox) layoutOnly()         {}
func (*AlignBox) layoutOnly()           {}
func (*ConstrainedBox) layoutOnly()     {}
func (*LimitedBox) layoutOnly()         {}
func (*AspectRatioBox) layoutOnly()     {}
func (*FractionalBox) layoutOnly()      {}
func (*StackBox) layoutOnly()           {}
func (*WrapBox) layoutOnly()            {}
func (*PointerListenerBox) layoutOnly() {}
func (*IgnorePointerBox) layoutOnly()   {}
func (*AbsorbPointerBox) layoutOnly()   {}

// ChildTransform — дети TransformBox рисуются с его преобразованием.
func (b *TransformBox) ChildTransform() geom.Affine { return b.effective() }

// PaintOverflow — тень выходит за границы бокса на 3σ размытия плюс расширение и сдвиг.
func (b *DecoratedBox) PaintOverflow() float64 {
	out := 0.0
	for _, s := range b.Decoration.BoxShadow {
		e := 3*s.Sigma() + s.SpreadRadius + math.Max(math.Abs(s.Offset.X), math.Abs(s.Offset.Y))
		out = math.Max(out, e)
	}
	return out
}
