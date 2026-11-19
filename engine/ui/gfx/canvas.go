package gfx

import (
	"image"
	"image/draw"
	"math"

	"golang.org/x/image/vector"

	"hexago/ui/geom"
)

// Canvas — программный холст: рисует фигуры в буфер пикселей формата 0xAABBGGRR
// (в памяти байты R, G, B, A — так их ждёт Android и так же устроен image.RGBA).
//
// Любая фигура рисуется одинаково: строится её контур, растеризатор считает, какую долю
// каждого пикселя она покрывает (от 0 до 255 — отсюда сглаживание краёв), и цвет
// смешивается с фоном пропорционально этой доле.
//
// Как Canvas во Flutter, холст хранит состояние — преобразование, обрезку, прозрачность —
// и умеет сохранять (Save) и восстанавливать (Restore) его.
type Canvas struct {
	pix    []uint32
	width  int
	height int
	stride int

	st    canvasState
	stack []canvasState

	raster *vector.Rasterizer
	cover  *image.Alpha // черновик: покрытие текущей фигуры
}

type canvasState struct {
	m        geom.Affine     // локальные координаты → пиксели экрана
	clip     image.Rectangle // область, куда можно рисовать (в пикселях)
	clipMask *image.Alpha    // покрытие для скруглённых обрезок; Bounds() == clip; nil — без маски
	alpha    float64         // общая непрозрачность от 0 до 1
}

// NewCanvas создаёт холст поверх готового буфера. stride — длина строки в пикселях (≥ width).
func NewCanvas(pix []uint32, width, height, stride int) *Canvas {
	c := &Canvas{
		pix: pix, width: width, height: height, stride: stride,
		raster: vector.NewRasterizer(1, 1),
		cover:  image.NewAlpha(image.Rect(0, 0, 1, 1)),
	}
	c.st = canvasState{m: geom.Identity, clip: image.Rect(0, 0, width, height), alpha: 1}
	return c
}

// NewImageCanvas создаёт холст со своим буфером — для тестов и картинок.
func NewImageCanvas(width, height int) *Canvas {
	return NewCanvas(make([]uint32, width*height), width, height, width)
}

// Width — ширина холста в пикселях.
func (c *Canvas) Width() int { return c.width }

// Height — высота холста в пикселях.
func (c *Canvas) Height() int { return c.height }

// Save запоминает текущее состояние: преобразование, обрезку, прозрачность.
func (c *Canvas) Save() { c.stack = append(c.stack, c.st) }

// Restore возвращает состояние, запомненное последним Save.
func (c *Canvas) Restore() {
	if len(c.stack) == 0 {
		panic("gfx.Canvas.Restore: вызван без парного Save")
	}
	c.st = c.stack[len(c.stack)-1]
	c.stack = c.stack[:len(c.stack)-1]
}

// Translate сдвигает начало координат.
func (c *Canvas) Translate(dx, dy float64) { c.st.m = c.st.m.Mul(geom.Translation(dx, dy)) }

// Scale масштабирует координаты.
func (c *Canvas) Scale(sx, sy float64) { c.st.m = c.st.m.Mul(geom.Scaling(sx, sy)) }

// Transform применяет произвольное преобразование.
func (c *Canvas) Transform(t geom.Affine) { c.st.m = c.st.m.Mul(t) }

// Matrix — текущее преобразование из локальных координат в пиксели.
func (c *Canvas) Matrix() geom.Affine { return c.st.m }

// MultiplyOpacity уменьшает непрозрачность всего, что рисуется дальше (до Restore).
// Перекрывающиеся фигуры внутри смешиваются друг с другом, а не как единый слой —
// для одиночных фигур разницы нет.
func (c *Canvas) MultiplyOpacity(a float64) { c.st.alpha *= clamp01(a) }

// Clear заливает весь холст цветом, не обращая внимания на обрезку.
func (c *Canvas) Clear(col Color) { c.ClearRect(image.Rect(0, 0, c.width, c.height), col) }

// ClearRect заливает прямоугольник r (в пикселях) цветом, не смешивая с тем, что там было.
func (c *Canvas) ClearRect(r image.Rectangle, col Color) {
	r = r.Intersect(image.Rect(0, 0, c.width, c.height))
	if r.Empty() {
		return
	}
	p := col.Pixel()
	first := c.pix[r.Min.Y*c.stride+r.Min.X : r.Min.Y*c.stride+r.Max.X]
	for x := range first {
		first[x] = p
	}
	for y := r.Min.Y + 1; y < r.Max.Y; y++ {
		copy(c.pix[y*c.stride+r.Min.X:y*c.stride+r.Max.X], first)
	}
}

// QuickReject сообщает, что прямоугольник r (в локальных координатах) целиком вне области,
// куда сейчас можно рисовать: такую фигуру можно не рисовать вовсе.
func (c *Canvas) QuickReject(r geom.Rect) bool {
	d := c.st.m.MapRect(r)
	clip := c.st.clip
	return d.Right <= float64(clip.Min.X) || d.Left >= float64(clip.Max.X) ||
		d.Bottom <= float64(clip.Min.Y) || d.Top >= float64(clip.Max.Y)
}

// ClipRect ограничивает рисование прямоугольником (до Restore).
func (c *Canvas) ClipRect(r geom.Rect) {
	if c.st.m.B == 0 && c.st.m.C == 0 { // без поворота — обрезка целыми пикселями
		d := c.st.m.MapRect(r)
		c.intersectClip(image.Rect(round(d.Left), round(d.Top), round(d.Right), round(d.Bottom)))
		return
	}
	var p Path
	p.AddRect(r, false)
	c.ClipPath(&p)
}

// ClipRRect ограничивает рисование прямоугольником со скруглёнными углами.
func (c *Canvas) ClipRRect(rr geom.RRect) {
	if rr.Radius.IsZero() {
		c.ClipRect(rr.Rect)
		return
	}
	var p Path
	p.AddRRect(rr, false)
	c.ClipPath(&p)
}

// ClipPath ограничивает рисование произвольным контуром (со сглаженными краями).
func (c *Canvas) ClipPath(p *Path) {
	b, ok := c.rasterize(p, c.st.clip)
	if !ok {
		c.st.clip, c.st.clipMask = image.Rectangle{}, nil
		return
	}
	mask := image.NewAlpha(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		src := c.cover.Pix[(y-b.Min.Y)*c.cover.Stride:]
		dst := mask.Pix[(y-b.Min.Y)*mask.Stride:]
		for x := 0; x < b.Dx(); x++ {
			k := uint32(src[x])
			if old := c.st.clipMask; old != nil {
				k = k * uint32(old.Pix[(y-old.Rect.Min.Y)*old.Stride+(b.Min.X+x-old.Rect.Min.X)]) / 255
			}
			dst[x] = uint8(k)
		}
	}
	c.st.clip, c.st.clipMask = b, mask
}

// intersectClip сужает прямоугольную область обрезки; маска, если была, сохраняется.
func (c *Canvas) intersectClip(r image.Rectangle) {
	c.st.clip = c.st.clip.Intersect(r)
	if m := c.st.clipMask; m != nil {
		if c.st.clip.Empty() {
			c.st.clipMask = nil
			return
		}
		c.st.clipMask = m.SubImage(c.st.clip).(*image.Alpha)
	}
}

// DrawRect рисует прямоугольник.
func (c *Canvas) DrawRect(r geom.Rect, paint Paint) {
	c.DrawRRect(geom.RRect{Rect: r}, paint)
}

// DrawRRect рисует прямоугольник со скруглёнными углами.
func (c *Canvas) DrawRRect(rr geom.RRect, paint Paint) {
	var p Path
	if paint.Style == Stroke {
		// Обводка — кольцо между контуром, расширенным и суженным на половину толщины.
		w := paint.StrokeWidth / 2
		p.AddRRect(rr.Inflate(w), false)
		if inner := rr.Inflate(-w); !inner.IsEmpty() {
			p.AddRRect(inner, true)
		}
	} else {
		if paint.Shader == nil && c.fillRRectFast(rr, paint.Color) {
			return
		}
		p.AddRRect(rr, false)
	}
	c.fill(&p, paint)
}

// DrawOval рисует эллипс, вписанный в прямоугольник.
func (c *Canvas) DrawOval(r geom.Rect, paint Paint) {
	rx, ry := r.Width()/2, r.Height()/2
	rr := geom.RRect{Rect: r, Radius: geom.BorderRadius{
		TopLeft: geom.Radius{X: rx, Y: ry}, TopRight: geom.Radius{X: rx, Y: ry},
		BottomRight: geom.Radius{X: rx, Y: ry}, BottomLeft: geom.Radius{X: rx, Y: ry},
	}}
	c.DrawRRect(rr, paint)
}

// DrawCircle рисует круг (или окружность при Style: Stroke).
func (c *Canvas) DrawCircle(center geom.Offset, radius float64, paint Paint) {
	c.DrawOval(geom.Rect{Left: center.X - radius, Top: center.Y - radius, Right: center.X + radius, Bottom: center.Y + radius}, paint)
}

// DrawPath рисует произвольный контур.
func (c *Canvas) DrawPath(p *Path, paint Paint) {
	if paint.Style == Stroke {
		p = c.stroke(p, paint)
	}
	c.fill(p, paint)
}

// DrawLine рисует отрезок линией толщины paint.StrokeWidth.
func (c *Canvas) DrawLine(from, to geom.Offset, paint Paint) {
	var p Path
	p.MoveTo(from.X, from.Y)
	p.LineTo(to.X, to.Y)
	paint.Style = Stroke
	c.DrawPath(&p, paint)
}

// fill заливает контур кистью.
func (c *Canvas) fill(p *Path, paint Paint) {
	if paint.Shader == nil && paint.Color.A == 0 {
		return
	}
	b, ok := c.rasterize(p, c.st.clip)
	if !ok {
		return
	}
	c.composite(b, c.cover, b.Min, paint)
}

// rasterize считает покрытие контура в c.cover для области его пикселей, обрезанной limit.
// Возвращает эту область; c.cover индексируется от её левого верхнего угла.
func (c *Canvas) rasterize(p *Path, limit image.Rectangle) (image.Rectangle, bool) {
	if p.IsEmpty() {
		return image.Rectangle{}, false
	}
	d := c.st.m.MapRect(p.Bounds())
	b := image.Rect(int(math.Floor(d.Left)), int(math.Floor(d.Top)), int(math.Ceil(d.Right)), int(math.Ceil(d.Bottom))).Intersect(limit)
	if b.Empty() {
		return b, false
	}
	c.rasterizeIn(p, b)
	return b, true
}

// rasterizeIn считает покрытие контура в c.cover для заданной области пикселей b.
func (c *Canvas) rasterizeIn(p *Path, b image.Rectangle) {
	w, h := b.Dx(), b.Dy()
	c.raster.Reset(w, h)
	ox, oy := float64(b.Min.X), float64(b.Min.Y)
	pt := func(q geom.Offset) (float32, float32) {
		q = c.st.m.Apply(q)
		return float32(q.X - ox), float32(q.Y - oy)
	}
	open := false
	for _, s := range p.segs {
		switch s.op {
		case opMove:
			if open {
				c.raster.ClosePath()
			}
			c.raster.MoveTo(pt(s.pts[0]))
			open = true
		case opLine:
			c.raster.LineTo(pt(s.pts[0]))
		case opQuad:
			x1, y1 := pt(s.pts[0])
			x2, y2 := pt(s.pts[1])
			c.raster.QuadTo(x1, y1, x2, y2)
		case opCubic:
			x1, y1 := pt(s.pts[0])
			x2, y2 := pt(s.pts[1])
			x3, y3 := pt(s.pts[2])
			c.raster.CubeTo(x1, y1, x2, y2, x3, y3)
		case opClose:
			c.raster.ClosePath()
			open = false
		}
	}
	if open {
		c.raster.ClosePath() // заливка всегда считается по замкнутому контуру
	}
	if cap(c.cover.Pix) < w*h {
		c.cover.Pix = make([]uint8, w*h)
	}
	c.cover.Pix = c.cover.Pix[:w*h]
	c.cover.Stride, c.cover.Rect = w, image.Rect(0, 0, w, h)
	c.raster.DrawOp = draw.Src
	c.raster.Draw(c.cover, c.cover.Rect, image.Opaque, image.Point{})
}

// composite смешивает цвет кисти с пикселями области area по маске покрытия cov.
// origin — где на экране лежит первый байт cov.Pix: покрытие пикселя (x, y) —
// cov.Pix[(y-origin.Y)*cov.Stride + (x-origin.X)]. Область сама обрезается по clip.
func (c *Canvas) composite(area image.Rectangle, cov *image.Alpha, origin image.Point, paint Paint) {
	b := area.Intersect(c.st.clip)
	if b.Empty() {
		return
	}
	var shade func(x, y float64) Color
	if paint.Shader != nil {
		shade = paint.Shader.at(c.st.m)
	}
	alpha := uint32(c.st.alpha*255 + 0.5)
	mask := c.st.clipMask
	base := paint.Color
	// Непрозрачность цвета с учётом общей прозрачности холста (0…255), если цвет один на всю фигуру.
	ca := div255(uint32(base.A) * alpha)
	r, g, bb := uint32(base.R), uint32(base.G), uint32(base.B)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		row := c.pix[y*c.stride:]
		covBase := (y-origin.Y)*cov.Stride - origin.X
		maskBase := 0
		if mask != nil {
			maskBase = (y-mask.Rect.Min.Y)*mask.Stride - mask.Rect.Min.X
		}
		for x := b.Min.X; x < b.Max.X; x++ {
			k := uint32(cov.Pix[covBase+x])
			if k == 0 {
				continue
			}
			if mask != nil {
				if k = div255(k * uint32(mask.Pix[maskBase+x])); k == 0 {
					continue
				}
			}
			if shade != nil {
				col := shade(float64(x)+0.5, float64(y)+0.5)
				if a := div255(div255(uint32(col.A)*alpha) * k); a > 0 {
					row[x] = mix(row[x], uint32(col.R), uint32(col.G), uint32(col.B), a)
				}
				continue
			}
			if a := div255(ca * k); a > 0 {
				row[x] = mix(row[x], r, g, bb, a)
			}
		}
	}
}

// blend накладывает цвет src с непрозрачностью a (0..1) поверх пикселя dst.
func blend(dst uint32, src Color, a float64) uint32 {
	if a >= 1 {
		return uint32(255)<<24 | uint32(src.B)<<16 | uint32(src.G)<<8 | uint32(src.R)
	}
	inv := 1 - a
	r := float64(src.R)*a + float64(uint8(dst))*inv
	g := float64(src.G)*a + float64(uint8(dst>>8))*inv
	b := float64(src.B)*a + float64(uint8(dst>>16))*inv
	al := 255*a + float64(uint8(dst>>24))*inv
	return uint32(al+0.5)<<24 | uint32(b+0.5)<<16 | uint32(g+0.5)<<8 | uint32(r+0.5)
}

// DrawMask рисует цветом по маске покрытия (например, по готовому глифу шрифта).
// at — где на экране (в пикселях) окажется левый верхний угол маски.
func (c *Canvas) DrawMask(mask *image.Alpha, at image.Point, col Color) {
	area := image.Rectangle{Min: at, Max: at.Add(mask.Rect.Size())}
	c.composite(area, mask, at, Paint{Color: col})
}

// maskAt — покрытие обрезки в пикселе (255 — без ограничений).
func (c *Canvas) maskAt(x, y int) uint32 {
	m := c.st.clipMask
	if m == nil {
		return 255
	}
	return uint32(m.Pix[(y-m.Rect.Min.Y)*m.Stride+(x-m.Rect.Min.X)])
}

// scaleFactor — во сколько раз преобразование растягивает длины (для толщины линий, размытия).
func (c *Canvas) scaleFactor() float64 {
	m := c.st.m
	return math.Sqrt(math.Abs(m.A*m.D - m.B*m.C))
}

func round(v float64) int { return int(math.Floor(v + 0.5)) }

// ToNRGBA копирует холст в картинку Go — например, чтобы сохранить в PNG.
func (c *Canvas) ToNRGBA() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, c.width, c.height))
	for y := range c.height {
		for x := range c.width {
			p := c.pix[y*c.stride+x]
			o := y*img.Stride + x*4
			img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = uint8(p), uint8(p>>8), uint8(p>>16), uint8(p>>24)
		}
	}
	return img
}
