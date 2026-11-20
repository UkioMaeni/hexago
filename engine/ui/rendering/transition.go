package rendering

import (
	"image"
	"math"
	"slices"

	"hexago/ui/animation"
	"hexago/ui/geom"
	"hexago/ui/gfx"
)

// Боксы переходов: берут значение анимации сами, в каждом кадре, — дерево виджетов при этом
// не перестраивается. Так экран въезжает и проявляется, а сам экран не строится заново 60 раз в секунду.

// FadeBox рисует ребёнка с прозрачностью из анимации Opacity (0 — невидим, 1 — как есть).
type FadeBox struct {
	Proxy
	Opacity animation.Animation

	last float64 // прозрачность, которую видел прошлый CollectDamage
	seen bool
}

func (b *FadeBox) value() float64 {
	if b.Opacity == nil {
		return 1
	}
	return min(1, max(0, b.Opacity.Value()))
}

func (b *FadeBox) Paint(ctx *PaintContext, offset geom.Offset) {
	v := b.value()
	if v <= 0 {
		return
	}
	ctx.Canvas.Save()
	ctx.Canvas.MultiplyOpacity(v)
	PaintChildren(ctx, b, offset)
	ctx.Canvas.Restore()
}

// PaintChanged — прозрачность изменилась с прошлого кадра.
func (b *FadeBox) PaintChanged() bool {
	v := b.value()
	changed := !b.seen || v != b.last
	b.last, b.seen = v, true
	return changed
}

// SlideBox сдвигает ребёнка на долю его размера: от Begin (значение анимации 0) до End (значение 1).
// Offset{X: 1} — на всю ширину вправо. Раскладку ребёнка сдвиг не меняет.
type SlideBox struct {
	Proxy
	Position   animation.Animation
	Begin, End geom.Offset
}

// shift — сдвиг ребёнка сейчас, в dp.
func (b *SlideBox) shift() geom.Offset {
	t := 0.0
	if b.Position != nil {
		t = b.Position.Value()
	}
	s := SizeOf(b)
	return geom.Offset{
		X: geom.Lerp(b.Begin.X, b.End.X, t) * s.Width,
		Y: geom.Lerp(b.Begin.Y, b.End.Y, t) * s.Height,
	}
}

// ChildTransform — для учёта повреждений: дети нарисованы со сдвигом.
func (b *SlideBox) ChildTransform() geom.Affine {
	o := b.shift()
	return geom.Translation(o.X, o.Y)
}

func (b *SlideBox) Paint(ctx *PaintContext, offset geom.Offset) {
	PaintChildren(ctx, b, offset.Add(b.snappedShift(ctx.Canvas.Matrix(), offset)))
}

// snappedShift — сдвиг, округлённый так, чтобы ребёнок лёг ровно на пиксели экрана (m — преобразование
// холста, offset — где лежит бокс). Едущий экран остаётся чётким, а маски текста и теней берутся
// из кеша: их положение внутри пикселя то же, что у неподвижного экрана.
func (b *SlideBox) snappedShift(m geom.Affine, offset geom.Offset) geom.Offset {
	o := b.shift()
	if m.B != 0 || m.C != 0 || m.A <= 0 || m.D <= 0 {
		return o
	}
	x := (offset.X+o.X)*m.A + m.E
	y := (offset.Y+o.Y)*m.D + m.F
	o.X += (math.Round(x) - x) / m.A
	o.Y += (math.Round(y) - y) / m.D
	return o
}

func (b *SlideBox) HitTest(result *HitTestResult, pos geom.Offset) bool {
	o := b.shift()
	result.PushTransform(geom.Translation(-o.X, -o.Y))
	hit := HitTestChildren(b, result, pos.Sub(o))
	result.PopTransform()
	if hit {
		result.Add(b, pos)
	}
	return hit
}

// TheatreBox — стопка слоёв во весь свой размер, как экраны в Navigator. Слой, который сейчас
// непрозрачен (TheatreParentData.Opaque), закрывает всё под собой: нижние слои не раскладываются,
// не рисуются и не получают касаний. Сами слои остаются в дереве — их состояние живёт.
type TheatreBox struct {
	Node
	visible []Box // слои, которые видны сейчас (снизу вверх)
}

// TheatreParentData — данные слоя TheatreBox: Opaque сообщает, закрывает ли он сейчас всё под собой
// (экран, который доехал до места). Спрашивается при каждой раскладке.
type TheatreParentData struct {
	Opaque func() bool
	// Page — слой-экран. Если он сейчас не полупрозрачен (FadeBox = 1), а его содержимое закрашивает
	// всё само (CoversOpaque, как Scaffold), нижние слои рисуются только там, куда он не дотягивается:
	// во время перехода закрытая часть нижнего экрана не рисуется зря.
	Page bool
}

func (b *TheatreBox) PerformLayout(c Constraints) geom.Size {
	size := c.Biggest()
	ch := Children(b)
	first := 0
	for i := len(ch) - 1; i >= 0; i-- {
		if pd, _ := ch[i].node().ParentData.(*TheatreParentData); pd != nil && pd.Opaque != nil && pd.Opaque() {
			first = i
			break
		}
	}
	visible := ch[first:]
	if !slices.Equal(visible, b.visible) {
		// Открылся слой, который был закрыт (или закрылся видимый): то, что он нарисовал когда-то,
		// давно затёрто — перерисовать всю стопку.
		MarkNeedsPaint(b)
		b.visible = slices.Clone(visible)
		for _, hidden := range ch[:first] {
			releaseSnapshot(hidden)
		}
	}
	for _, v := range visible {
		Layout(v, Tight(size))
		SetOffset(v, geom.Zero)
	}
	return size
}

func (b *TheatreBox) Paint(ctx *PaintContext, offset geom.Offset) {
	m := ctx.Canvas.Matrix()
	full := geom.RectFromSize(offset, SizeOf(b))
	for i, v := range b.visible {
		// Что из слоя видно: всё, кроме того, что закрыли непрозрачные экраны над ним.
		area, hidden := full, false
		for _, above := range b.visible[i+1:] {
			if r, ok := coverRect(above, m, offset); ok {
				if area, hidden = subtractRect(area, r); hidden {
					break
				}
			}
		}
		switch {
		case hidden:
		case area == full:
			ctx.PaintChild(v, offset)
		default:
			ctx.Canvas.Save()
			ctx.Canvas.ClipRect(area)
			ctx.PaintChild(v, offset)
			ctx.Canvas.Restore()
		}
	}
}

// coverRect — где слой-экран сейчас непрозрачен (в координатах до преобразования холста m): его
// прямоугольник после сдвигов переходов. false — слой не экран или сейчас полупрозрачен.
func coverRect(layer Box, m geom.Affine, offset geom.Offset) (geom.Rect, bool) {
	if pd, _ := layer.node().ParentData.(*TheatreParentData); pd == nil || !pd.Page {
		return geom.Rect{}, false
	}
	at := offset.Add(layer.node().offset)
	b := layer
	for {
		switch t := b.(type) {
		case *FadeBox:
			if t.value() < 1 {
				return geom.Rect{}, false
			}
		case *SlideBox:
			at = at.Add(t.snappedShift(m, at))
		case *IgnorePointerBox, *SnapshotBox:
		default:
			// Содержимое экрана: непрозрачно, только если само закрашивает всё (Scaffold — да,
			// голая колонка кнопок на фоне App — нет).
			return geom.RectFromSize(at, SizeOf(b)), CoversOpaque(b, m, at)
		}
		ch := Child(b)
		if ch == nil {
			return geom.Rect{}, false
		}
		at, b = at.Add(ch.node().offset), ch
	}
}

// subtractRect — что останется от area без r, если остаток — прямоугольник (r закрывает area
// по всей высоте или ширине с одного края). hidden — r закрывает area целиком.
func subtractRect(area, r geom.Rect) (rest geom.Rect, hidden bool) {
	tall := r.Top <= area.Top && r.Bottom >= area.Bottom
	wide := r.Left <= area.Left && r.Right >= area.Right
	switch {
	case tall && wide:
		return area, true
	case tall && r.Left <= area.Left && r.Right > area.Left:
		area.Left = r.Right
	case tall && r.Right >= area.Right && r.Left < area.Right:
		area.Right = r.Left
	case wide && r.Top <= area.Top && r.Bottom > area.Top:
		area.Top = r.Bottom
	case wide && r.Bottom >= area.Bottom && r.Top < area.Bottom:
		area.Bottom = r.Top
	}
	return area, area.IsEmpty()
}

// PaintedChildren — для учёта повреждений: закрытые слои не рисуются.
func (b *TheatreBox) PaintedChildren() []Box { return b.visible }

func (b *TheatreBox) HitTest(result *HitTestResult, pos geom.Offset) bool {
	if !b.node().size.Contains(pos) {
		return false
	}
	for i := len(b.visible) - 1; i >= 0; i-- {
		ch := b.visible[i]
		off := ch.node().offset
		result.PushTransform(geom.Translation(-off.X, -off.Y))
		hit := HitTest(ch, result, pos.Sub(off))
		result.PopTransform()
		if hit {
			result.Add(b, pos)
			return true
		}
	}
	return false
}

// OpaqueCoverer — бокс, который сейчас закрашивает весь свой прямоугольник непрозрачно (стопка
// экранов, где экран на месте): то, что под ним, можно не рисовать. m — преобразование холста,
// offset — где лежит бокс.
type OpaqueCoverer interface {
	CoversOpaque(m geom.Affine, offset geom.Offset) bool
}

// CoversOpaque сообщает, закрашивает ли бокс весь свой прямоугольник непрозрачно: тогда фон под ним
// (заливка ColoredBox, очистка кадра) не нужен — на телефоне это экономит по заливке экрана на кадр.
func CoversOpaque(b Box, m geom.Affine, offset geom.Offset) bool {
	switch t := b.(type) {
	case OpaqueCoverer:
		return t.CoversOpaque(m, offset)
	case *Proxy:
		return childCoversOpaque(t, m, offset)
	}
	return false
}

// childCoversOpaque — единственный ребёнок бокса закрашивает весь бокс непрозрачно.
func childCoversOpaque(b Box, m geom.Affine, offset geom.Offset) bool {
	ch := Child(b)
	return ch != nil && ch.node().offset == geom.Zero && SizeOf(ch) == SizeOf(b) && CoversOpaque(ch, m, offset)
}

// CoversOpaque — заливка непрозрачным цветом или ребёнок, который закрашивает всё сам.
func (b *ColoredBox) CoversOpaque(m geom.Affine, offset geom.Offset) bool {
	return b.Color.A == 255 || childCoversOpaque(b, m, offset)
}

// CoversOpaque — экраны на месте закрывают всю стопку (вместе с полосками экранов под ними).
func (b *TheatreBox) CoversOpaque(m geom.Affine, offset geom.Offset) bool {
	area := geom.RectFromSize(offset, SizeOf(b))
	for i := len(b.visible) - 1; i >= 0; i-- {
		if r, ok := coverRect(b.visible[i], m, offset); ok {
			var hidden bool
			if area, hidden = subtractRect(area, r); hidden {
				return true
			}
		}
	}
	return false
}

// SnapshotBox — снимок содержимого: пока Active() (обычно — пока идёт переход), ребёнок рисуется один
// раз в отдельную картинку, а в кадрах она только переносится на экран — со сдвигом и прозрачностью
// переходов вокруг. Если внутри что-то изменилось, снимок рисуется заново. Без Active — обычный бокс.
type SnapshotBox struct {
	Proxy
	Active func() bool

	layer  *gfx.Canvas
	scale  [2]float64 // масштаб, с которым нарисован снимок
	valid  bool       // снимок совпадает с тем, что нарисовал бы ребёнок
	opaque bool       // в снимке нет полупрозрачных пикселей
	walked uint64     // в каком обходе CollectDamage бокс видели последний раз
}

func (b *SnapshotBox) Paint(ctx *PaintContext, offset geom.Offset) {
	c := ctx.Canvas
	m := c.Matrix()
	s := SizeOf(b)
	w, h := int(math.Ceil(s.Width*m.A)), int(math.Ceil(s.Height*m.D))
	if b.Active == nil || !b.Active() || m.B != 0 || m.C != 0 || w <= 0 || h <= 0 {
		b.layer = nil // память снимка (весь экран) не держим, пока он не нужен
		PaintChildren(ctx, b, offset)
		return
	}
	scale := [2]float64{m.A, m.D}
	if b.layer == nil || b.layer.Width() != w || b.layer.Height() != h || b.scale != scale {
		b.layer, b.scale, b.valid = gfx.NewImageCanvas(w, h), scale, false
	}
	if !b.valid {
		l := b.layer
		l.ClearRect(image.Rect(0, 0, w, h), gfx.Transparent)
		l.Save()
		l.Scale(m.A, m.D)
		l.Translate(-offset.X, -offset.Y)
		b.opaque = childCoversOpaque(b, l.Matrix(), offset)
		PaintChildren(&PaintContext{Canvas: l}, b, offset)
		l.Restore()
		b.valid = true
	}
	at := image.Pt(int(math.Round(offset.X*m.A+m.E)), int(math.Round(offset.Y*m.D+m.F)))
	c.DrawLayer(b.layer, at, b.opaque)
}

// releaseSnapshot отдаёт память снимка слоя, который закрыли: пока он не виден, снимок не нужен
// (и устарел бы — закрытый слой не проверяется на перемены).
func releaseSnapshot(layer Box) {
	for b := layer; b != nil; b = Child(b) {
		switch t := b.(type) {
		case *SnapshotBox:
			t.layer, t.valid = nil, false
			return
		case *IgnorePointerBox, *SlideBox, *FadeBox:
		default:
			return
		}
	}
}

// CoversOpaque — снимок закрашивает всё, если это делает его содержимое.
func (b *SnapshotBox) CoversOpaque(m geom.Affine, offset geom.Offset) bool {
	return childCoversOpaque(b, m, offset)
}
