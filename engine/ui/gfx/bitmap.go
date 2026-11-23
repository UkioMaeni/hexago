package gfx

import (
	"image"
	"image/draw"
	"math"
	"runtime"
	"sync"
	"sync/atomic"

	"hexago/ui/geom"
)

// Bitmap — картинка, готовая к рисованию: пиксели в формате холста (0xAABBGGRR, без предумножения).
// Её делают один раз из декодированной картинки (NewBitmap — можно не в потоке интерфейса),
// а рисуют много раз (DrawBitmap — в потоке интерфейса).
//
// Копии картинки под размер, в котором её рисуют на экране, Bitmap запоминает сам: уменьшить
// фотографию дорого, а скопировать готовую копию — дёшево.
type Bitmap struct {
	id     uint64 // первым полем: сравнение параметров боксов различает картинки, не сравнивая пиксели
	pix    []uint32
	w, h   int
	opaque bool
	scaled []rendition // последние копии под размер на экране, свежая — в конце
}

// rendition — часть src картинки, пересчитанная в w×h пикселей экрана.
type rendition struct {
	src  geom.Rect
	w, h int
	pix  []uint32
}

const maxRenditions = 2 // одну картинку редко рисуют сразу в трёх размерах

var bitmapIDs atomic.Uint64

// NewBitmap переводит картинку в формат холста. Работает для любой image.Image; быстрее всего —
// для NRGBA, RGBA и YCbCr (так их отдают декодеры PNG и JPEG).
func NewBitmap(img image.Image) *Bitmap {
	r := img.Bounds()
	w, h := r.Dx(), r.Dy()
	b := &Bitmap{id: bitmapIDs.Add(1), pix: make([]uint32, w*h), w: w, h: h, opaque: true}
	switch src := img.(type) {
	case *image.NRGBA:
		b.fromStraight(src.Pix[src.PixOffset(r.Min.X, r.Min.Y):], src.Stride)
	case *image.RGBA:
		b.fromPremultiplied(src.Pix[src.PixOffset(r.Min.X, r.Min.Y):], src.Stride)
	default:
		// Остальное (YCbCr из JPEG, палитры, серое) — через RGBA: у draw.Draw для него быстрые пути.
		tmp := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(tmp, tmp.Rect, img, r.Min, draw.Src)
		b.fromPremultiplied(tmp.Pix, tmp.Stride)
	}
	return b
}

func (b *Bitmap) fromStraight(pix []uint8, stride int) {
	for y := range b.h {
		row, out := pix[y*stride:], b.pix[y*b.w:(y+1)*b.w]
		for x := range out {
			p := row[x*4 : x*4+4 : x*4+4]
			if p[3] != 255 {
				b.opaque = false
			}
			out[x] = uint32(p[3])<<24 | uint32(p[2])<<16 | uint32(p[1])<<8 | uint32(p[0])
		}
	}
}

func (b *Bitmap) fromPremultiplied(pix []uint8, stride int) {
	for y := range b.h {
		row, out := pix[y*stride:], b.pix[y*b.w:(y+1)*b.w]
		for x := range out {
			p := row[x*4 : x*4+4 : x*4+4]
			r, g, bl, a := uint32(p[0]), uint32(p[1]), uint32(p[2]), uint32(p[3])
			if a != 255 {
				b.opaque = false
				r, g, bl = unpremultiply(r, a), unpremultiply(g, a), unpremultiply(bl, a)
			}
			out[x] = a<<24 | bl<<16 | g<<8 | r
		}
	}
}

func unpremultiply(c, a uint32) uint32 {
	if a == 0 {
		return 0
	}
	return min(255, (c*255+a/2)/a)
}

// Width и Height — размер картинки в пикселях.
func (b *Bitmap) Width() int  { return b.w }
func (b *Bitmap) Height() int { return b.h }

// Opaque — в картинке нет прозрачных пикселей (так у фотографий): её можно просто копировать.
func (b *Bitmap) Opaque() bool { return b.opaque }

// Bytes — сколько памяти занимают пиксели (для кеша картинок).
func (b *Bitmap) Bytes() int { return len(b.pix) * 4 }

// At — пиксель (x, y) картинки.
func (b *Bitmap) At(x, y int) Color {
	p := b.pix[y*b.w+x]
	return Color{R: uint8(p), G: uint8(p >> 8), B: uint8(p >> 16), A: uint8(p >> 24)}
}

// DrawBitmap рисует часть src картинки (в её пикселях; пустой прямоугольник — вся картинка),
// растянув её на прямоугольник dst (в локальных координатах). opacity — от 0 до 1.
//
// Без поворота картинка встаёт ровно по пикселям экрана: она один раз пересчитывается под этот
// размер (уменьшение — усреднением по площади, без «ряби»; увеличение — билинейно), а дальше
// только копируется. С поворотом каждый пиксель берётся билинейно прямо из картинки.
func (c *Canvas) DrawBitmap(b *Bitmap, src, dst geom.Rect, opacity float64) {
	if b == nil || b.w == 0 || b.h == 0 || dst.IsEmpty() {
		return
	}
	full := geom.Rect{Right: float64(b.w), Bottom: float64(b.h)}
	if src.IsEmpty() {
		src = full
	}
	alpha := uint32(c.st.alpha*clamp01(opacity)*255 + 0.5)
	if alpha == 0 {
		return
	}
	m := c.st.m
	if m.B != 0 || m.C != 0 || m.A <= 0 || m.D <= 0 {
		c.sampleBitmap(b, src, dst, alpha)
		return
	}
	d := m.MapRect(dst)
	area := image.Rect(int(math.Round(d.Left)), int(math.Round(d.Top)), int(math.Round(d.Right)), int(math.Round(d.Bottom)))
	w, h := area.Dx(), area.Dy()
	if w <= 0 || h <= 0 || area.Intersect(c.st.clip).Empty() {
		return
	}
	var pix []uint32
	switch {
	case src == full && w == b.w && h == b.h:
		pix = b.pix
	case w*h <= max(4*c.width*c.height, 1<<20):
		pix = b.rendition(src, w, h)
	default:
		// Копия была бы больше нескольких экранов (сильное увеличение): берём только видимые пиксели.
		c.sampleBitmap(b, src, dst, alpha)
		return
	}
	c.blit(pix, w, area.Min, b.opaque, alpha)
}

// blit накладывает пиксели (w в строке) так, что их левый верхний угол встаёт в at.
func (c *Canvas) blit(pix []uint32, w int, at image.Point, opaque bool, alpha uint32) {
	h := len(pix) / w
	r := image.Rect(at.X, at.Y, at.X+w, at.Y+h).Intersect(c.st.clip)
	if r.Empty() {
		return
	}
	mask := c.st.clipMask
	if opaque && alpha == 255 && mask == nil {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			o := (y-at.Y)*w - at.X
			copy(c.pix[y*c.stride+r.Min.X:y*c.stride+r.Max.X], pix[o+r.Min.X:o+r.Max.X])
		}
		return
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		row := c.pix[y*c.stride:]
		base := (y-at.Y)*w - at.X // пиксель (x, y) экрана — pix[base+x]
		for x := r.Min.X; x < r.Max.X; x++ {
			s := pix[base+x]
			k := div255(s >> 24 * alpha)
			if mask != nil {
				k = div255(k * uint32(mask.Pix[(y-mask.Rect.Min.Y)*mask.Stride+x-mask.Rect.Min.X]))
			}
			if k > 0 {
				row[x] = mix(row[x], s&0xFF, s>>8&0xFF, s>>16&0xFF, k)
			}
		}
	}
}

// sampleBitmap — общий путь (поворот, сильное увеличение): для каждого пикселя экрана билинейно
// берётся точка картинки. Если картинка при этом сильно уменьшается, точки берутся из копии
// примерно нужного размера — иначе мелкие детали дают «рябь».
func (c *Canvas) sampleBitmap(b *Bitmap, src, dst geom.Rect, alpha uint32) {
	inv, ok := c.st.m.Invert()
	if !ok {
		return
	}
	d := c.st.m.MapRect(dst)
	area := image.Rect(int(math.Floor(d.Left)), int(math.Floor(d.Top)),
		int(math.Ceil(d.Right)), int(math.Ceil(d.Bottom))).Intersect(c.st.clip)
	if area.Empty() {
		return
	}
	pix, pw, ph := b.pix, b.w, b.h
	s := c.scaleFactor()
	if tw, th := int(math.Ceil(dst.Width()*s)), int(math.Ceil(dst.Height()*s)); tw > 0 && th > 0 &&
		float64(tw)*1.5 < src.Width() && float64(th)*1.5 < src.Height() {
		pix, pw, ph = b.rendition(src, tw, th), tw, th
		src = geom.Rect{Right: float64(tw), Bottom: float64(th)}
	}
	// Как и у копий, точки берутся только из самого куска картинки.
	lo := image.Pt(max(0, int(math.Floor(src.Left))), max(0, int(math.Floor(src.Top))))
	hi := image.Pt(min(pw-1, max(lo.X, int(math.Ceil(src.Right))-1)), min(ph-1, max(lo.Y, int(math.Ceil(src.Bottom))-1)))
	kx, ky := src.Width()/dst.Width(), src.Height()/dst.Height()
	for y := area.Min.Y; y < area.Max.Y; y++ {
		row := c.pix[y*c.stride:]
		for x := area.Min.X; x < area.Max.X; x++ {
			p := inv.Apply(geom.Offset{X: float64(x) + 0.5, Y: float64(y) + 0.5})
			if !dst.Contains(p) {
				continue
			}
			px := bilinear(pix, pw, lo, hi, src.Left+(p.X-dst.Left)*kx-0.5, src.Top+(p.Y-dst.Top)*ky-0.5)
			k := div255(div255(px>>24*alpha) * c.maskAt(x, y))
			if k > 0 {
				row[x] = mix(row[x], px&0xFF, px>>8&0xFF, px>>16&0xFF, k)
			}
		}
	}
}

// bilinear — пиксель в точке (u, v) картинки pix шириной w; соседи берутся в пределах lo…hi.
// Каналы смешиваются с предумножением на прозрачность.
func bilinear(pix []uint32, w int, lo, hi image.Point, u, v float64) uint32 {
	x0, y0 := int(math.Floor(u)), int(math.Floor(v))
	fx, fy := uint32((u-float64(x0))*256), uint32((v-float64(y0))*256)
	at := func(x, y int) uint32 { return pix[min(max(y, lo.Y), hi.Y)*w+min(max(x, lo.X), hi.X)] }
	var acc [4]uint32
	add := func(p, wgt uint32) {
		a := p >> 24
		acc[3] += a * wgt
		acc[0] += premul(p&0xFF, a) * wgt
		acc[1] += premul(p>>8&0xFF, a) * wgt
		acc[2] += premul(p>>16&0xFF, a) * wgt
	}
	add(at(x0, y0), (256-fx)*(256-fy))
	add(at(x0+1, y0), fx*(256-fy))
	add(at(x0, y0+1), (256-fx)*fy)
	add(at(x0+1, y0+1), fx*fy)
	return packPremultiplied(acc, 16)
}

func premul(c, a uint32) uint32 {
	if a == 255 {
		return c
	}
	return div255(c * a)
}

// packPremultiplied — пиксель холста из сумм предумноженных каналов, сдвинутых на shift бит.
func packPremultiplied(acc [4]uint32, shift uint) uint32 {
	half := uint32(1) << (shift - 1)
	a := min(255, (acc[3]+half)>>shift)
	r, g, bl := min(a, (acc[0]+half)>>shift), min(a, (acc[1]+half)>>shift), min(a, (acc[2]+half)>>shift)
	if a != 255 {
		r, g, bl = unpremultiply(r, a), unpremultiply(g, a), unpremultiply(bl, a)
	}
	return a<<24 | bl<<16 | g<<8 | r
}

// rendition — часть src картинки в w×h пикселей: готовая копия или новая.
func (b *Bitmap) rendition(src geom.Rect, w, h int) []uint32 {
	for i, r := range b.scaled {
		if r.src == src && r.w == w && r.h == h {
			if i != len(b.scaled)-1 { // свежую — в конец
				b.scaled = append(append(b.scaled[:i:i], b.scaled[i+1:]...), r)
			}
			return r.pix
		}
	}
	r := rendition{src: src, w: w, h: h, pix: b.resample(src, w, h)}
	if len(b.scaled) >= maxRenditions {
		b.scaled = append(b.scaled[:0:0], b.scaled[1:]...)
	}
	b.scaled = append(b.scaled, r)
	return r.pix
}

// taps — из каких пикселей картинки и с какими весами (сумма — 1<<16) собирается пиксель копии.
type taps struct {
	start   []int32 // первый пиксель картинки для каждого пикселя копии
	weights [][]uint32
}

// axisTaps — веса по одной оси: из отрезка картинки [from, from+length) в n пикселей.
// Уменьшение — среднее по площади (каждый пиксель картинки входит с долей, которую он покрывает),
// увеличение — билинейно между двумя соседями.
func axisTaps(from, length float64, n, size int) taps {
	t := taps{start: make([]int32, n), weights: make([][]uint32, n)}
	k := length / float64(n)
	// Пиксели берутся только из самого отрезка: у куска картинки (лист спрайтов) не подтекают соседи.
	first := max(0, int(math.Floor(from)))
	last := min(size-1, max(first, int(math.Ceil(from+length))-1))
	clamp := func(j int) int { return min(max(j, first), last) }
	for i := range n {
		var idx []int
		var wf []float64
		if k >= 1 {
			a, bnd := from+float64(i)*k, from+float64(i+1)*k
			for j := int(math.Floor(a)); float64(j) < bnd; j++ {
				if ov := math.Min(bnd, float64(j+1)) - math.Max(a, float64(j)); ov > 0 {
					idx, wf = append(idx, clamp(j)), append(wf, ov/k)
				}
			}
		} else {
			cpos := from + (float64(i)+0.5)*k - 0.5
			j := int(math.Floor(cpos))
			f := cpos - float64(j)
			idx, wf = []int{clamp(j), clamp(j + 1)}, []float64{1 - f, f}
		}
		// Индексы идут подряд (после clamp могут повторяться на краях) — веса раскладываем по ним.
		lo, hi := idx[0], idx[len(idx)-1]
		ws := make([]uint32, hi-lo+1)
		total, big := uint32(0), 0
		for m, j := range idx {
			v := uint32(wf[m]*65536 + 0.5)
			ws[j-lo] += v
		}
		for m, v := range ws {
			total += v
			if v > ws[big] {
				big = m
			}
		}
		ws[big] += 65536 - total // сумма ровно 1<<16
		t.start[i], t.weights[i] = int32(lo), ws
	}
	return t
}

// resample пересчитывает часть src картинки в w×h пикселей: сначала по горизонтали (строка
// картинки → строка шириной w), потом по вертикали. Каналы — с предумножением на прозрачность,
// чтобы прозрачные пиксели не давали тёмную кайму. Строки делятся между ядрами процессора.
func (b *Bitmap) resample(src geom.Rect, w, h int) []uint32 {
	out := make([]uint32, w*h)
	hx := axisTaps(src.Left, src.Width(), w, b.w)
	vy := axisTaps(src.Top, src.Height(), h, b.h)
	workers := 1
	if w*h >= 64*1024 || b.w*b.h >= 256*1024 {
		workers = min(runtime.GOMAXPROCS(0), h, 8)
	}
	var wg sync.WaitGroup
	for part := range workers {
		y0, y1 := h*part/workers, h*(part+1)/workers
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.resampleRows(out, w, hx, vy, y0, y1)
		}()
	}
	wg.Wait()
	return out
}

// resampleRows считает строки копии y0…y1. Строка картинки после горизонтального прохода хранится
// в каналах 8.8 (до 255·256), так что вертикальная сумма с весами 1<<16 помещается в uint32.
func (b *Bitmap) resampleRows(out []uint32, w int, hx, vy taps, y0, y1 int) {
	type row struct {
		y int
		c []uint32 // 4 канала на пиксель: R, G, B, A
	}
	cache := [2]row{{y: -1, c: make([]uint32, w*4)}, {y: -1, c: make([]uint32, w*4)}}
	next := 0
	horizontal := func(y int) []uint32 {
		for _, r := range cache {
			if r.y == y {
				return r.c
			}
		}
		r := &cache[next]
		next ^= 1
		r.y = y
		line := b.pix[y*b.w : (y+1)*b.w]
		for x := range w {
			var acc [4]uint32
			start := int(hx.start[x])
			for m, wgt := range hx.weights[x] {
				p := line[start+m]
				a := p >> 24
				acc[3] += a * wgt
				if a != 0 {
					acc[0] += premul(p&0xFF, a) * wgt
					acc[1] += premul(p>>8&0xFF, a) * wgt
					acc[2] += premul(p>>16&0xFF, a) * wgt
				}
			}
			c := r.c[x*4 : x*4+4 : x*4+4]
			c[0], c[1], c[2], c[3] = (acc[0]+128)>>8, (acc[1]+128)>>8, (acc[2]+128)>>8, (acc[3]+128)>>8
		}
		return r.c
	}
	acc := make([]uint32, w*4)
	for y := y0; y < y1; y++ {
		clear(acc)
		start := int(vy.start[y])
		for m, wgt := range vy.weights[y] {
			line := horizontal(start + m)
			for i, v := range line {
				acc[i] += v * wgt
			}
		}
		dst := out[y*w : (y+1)*w]
		for x := range dst {
			dst[x] = packPremultiplied([4]uint32{acc[x*4], acc[x*4+1], acc[x*4+2], acc[x*4+3]}, 24)
		}
	}
}
