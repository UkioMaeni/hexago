package gfx

import (
	"image"
	"image/draw"
	"math"

	"hexago/ui/geom"
)

// DrawImage рисует картинку, растянув её на прямоугольник dst (в локальных координатах).
// Пиксели берутся с билинейной интерполяцией — картинка не «рассыпается» на квадраты
// при увеличении. opacity — общая непрозрачность от 0 до 1.
func (c *Canvas) DrawImage(img image.Image, dst geom.Rect, opacity float64) {
	src := AsNRGBA(img)
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	if sw == 0 || sh == 0 || dst.IsEmpty() {
		return
	}
	inv, ok := c.st.m.Invert()
	if !ok {
		return
	}
	d := c.st.m.MapRect(dst)
	area := image.Rect(int(math.Floor(d.Left)), int(math.Floor(d.Top)),
		int(math.Ceil(d.Right)), int(math.Ceil(d.Bottom))).Intersect(c.st.clip)
	alpha := c.st.alpha * clamp01(opacity)
	kx, ky := float64(sw)/dst.Width(), float64(sh)/dst.Height()

	for y := area.Min.Y; y < area.Max.Y; y++ {
		row := c.pix[y*c.stride:]
		for x := area.Min.X; x < area.Max.X; x++ {
			p := inv.Apply(geom.Offset{X: float64(x) + 0.5, Y: float64(y) + 0.5})
			if !dst.Contains(p) {
				continue
			}
			col := sample(src, (p.X-dst.Left)*kx-0.5, (p.Y-dst.Top)*ky-0.5)
			a := float64(col.A) / 255 * alpha * float64(c.maskAt(x, y)) / 255
			if a > 0 {
				row[x] = blend(row[x], col, a)
			}
		}
	}
}

// sample — билинейная выборка из картинки в точке (u, v) в пикселях картинки.
func sample(img *image.NRGBA, u, v float64) Color {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	x0, y0 := int(math.Floor(u)), int(math.Floor(v))
	fx, fy := u-float64(x0), v-float64(y0)
	at := func(x, y int) [4]float64 {
		x, y = min(max(x, 0), w-1), min(max(y, 0), h-1)
		o := y*img.Stride + x*4
		p := img.Pix[o : o+4]
		return [4]float64{float64(p[0]), float64(p[1]), float64(p[2]), float64(p[3])}
	}
	a, b, cc, d := at(x0, y0), at(x0+1, y0), at(x0, y0+1), at(x0+1, y0+1)
	var out [4]uint8
	for i := range out {
		top := a[i] + (b[i]-a[i])*fx
		bottom := cc[i] + (d[i]-cc[i])*fx
		out[i] = uint8(top + (bottom-top)*fy + 0.5)
	}
	return Color{out[0], out[1], out[2], out[3]}
}

// AsNRGBA возвращает картинку в формате NRGBA с началом в (0, 0), преобразуя при необходимости.
// Для частой отрисовки одной картинки лучше преобразовать её один раз заранее.
func AsNRGBA(img image.Image) *image.NRGBA {
	if n, ok := img.(*image.NRGBA); ok && n.Rect.Min == (image.Point{}) {
		return n
	}
	b := img.Bounds()
	n := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(n, n.Rect, img, b.Min, draw.Src)
	return n
}
