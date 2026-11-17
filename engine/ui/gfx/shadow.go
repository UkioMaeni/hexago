package gfx

import (
	"image"
	"math"

	"hexago/ui/geom"
)

// DrawShadow рисует размытую тень прямоугольника со скруглениями.
// sigma — сила размытия (σ гауссова размытия) в тех же единицах, что и rr.
// Во Flutter BoxShadow.blurRadius переводится в sigma как blurRadius·0.57735 + 0.5.
func (c *Canvas) DrawShadow(rr geom.RRect, col Color, sigma float64) {
	s := sigma * c.scaleFactor()
	if s < 0.5 {
		c.DrawRRect(rr, Paint{Color: col})
		return
	}
	pad := int(math.Ceil(3 * s)) // за 3σ гауссиана практически равна нулю
	d := c.st.m.MapRect(rr.Rect)
	b := image.Rect(int(math.Floor(d.Left))-pad, int(math.Floor(d.Top))-pad,
		int(math.Ceil(d.Right))+pad, int(math.Ceil(d.Bottom))+pad)
	if b.Intersect(c.st.clip).Empty() {
		return
	}
	var p Path
	p.AddRRect(rr, false)
	c.rasterizeIn(&p, b) // фигура целиком, без обрезки: размытие берёт края извне
	blurAlpha(c.cover, s)
	c.composite(b, c.cover, b.Min, Paint{Color: col})
}

// blurAlpha размывает маску покрытия, приближая гауссово размытие тремя проходами
// размытия прямоугольным окном — это быстро и почти неотличимо от настоящего.
func blurAlpha(m *image.Alpha, sigma float64) {
	w, h := m.Rect.Dx(), m.Rect.Dy()
	buf := make([]uint8, len(m.Pix))
	for _, size := range boxesForGauss(sigma, 3) {
		r := (size - 1) / 2
		boxBlur(m.Pix, buf, w, h, 1, m.Stride, r) // по строкам
		boxBlur(buf, m.Pix, h, w, m.Stride, 1, r) // по столбцам
	}
}

// boxBlur — среднее по окну 2r+1 вдоль линий. lines линий по n точек;
// step — шаг между точками линии, lineStep — между линиями. Края за пределами — нули.
func boxBlur(src, dst []uint8, n, lines, step, lineStep, r int) {
	div := float64(2*r + 1)
	for l := range lines {
		base := l * lineStep
		sum := 0
		for i := 0; i <= r && i < n; i++ {
			sum += int(src[base+i*step])
		}
		for i := range n {
			dst[base+i*step] = uint8(float64(sum)/div + 0.5)
			if j := i + r + 1; j < n {
				sum += int(src[base+j*step])
			}
			if j := i - r; j >= 0 {
				sum -= int(src[base+j*step])
			}
		}
	}
}

// boxesForGauss подбирает размеры окон для n проходов, дающих размытие с заданной σ.
func boxesForGauss(sigma float64, n int) []int {
	wIdeal := math.Sqrt(12*sigma*sigma/float64(n) + 1)
	wl := int(math.Floor(wIdeal))
	if wl%2 == 0 {
		wl--
	}
	wu := wl + 2
	mIdeal := (12*sigma*sigma - float64(n*wl*wl) - 4*float64(n*wl) - 3*float64(n)) / float64(-4*wl-4)
	m := int(math.Round(mIdeal))
	sizes := make([]int, n)
	for i := range sizes {
		if i < m {
			sizes[i] = wl
		} else {
			sizes[i] = wu
		}
	}
	return sizes
}
