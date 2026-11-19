package gfx

import (
	"image"
	"math"
	"sync"

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
	// Размытая маска зависит только от размера, скруглений, размытия и дробного сдвига фигуры
	// внутри пикселя — её берём из кеша: тени кнопок и карточек одинаковы из кадра в кадр.
	m := c.st.m
	q := func(v float64) float64 { return math.Round(v*4) / 4 }
	key := shadowKey{
		w: q(d.Width()), h: q(d.Height()), sigma: q(s),
		fx: q(d.Left - math.Floor(d.Left)), fy: q(d.Top - math.Floor(d.Top)),
		r: [4]float64{q(rr.Radius.TopLeft.X * m.A), q(rr.Radius.TopRight.X * m.A),
			q(rr.Radius.BottomRight.X * m.A), q(rr.Radius.BottomLeft.X * m.A)},
		axis: m.B == 0 && m.C == 0,
	}
	if key.axis {
		if mask := shadows.get(key); mask != nil {
			c.composite(b, mask, b.Min, Paint{Color: col})
			return
		}
	}
	var p Path
	p.AddRRect(rr, false)
	c.rasterizeIn(&p, b) // фигура целиком, без обрезки: размытие берёт края извне
	blurAlpha(c.cover, s)
	if key.axis {
		mask := image.NewAlpha(c.cover.Rect)
		copy(mask.Pix, c.cover.Pix)
		shadows.put(key, mask)
	}
	c.composite(b, c.cover, b.Min, Paint{Color: col})
}

// shadowKey — по чему тени одинаковы (всё — в пикселях экрана, с шагом 1/4 пикселя).
type shadowKey struct {
	w, h, sigma, fx, fy float64
	r                   [4]float64
	axis                bool // без поворота (с поворотом тени не кешируются)
}

// shadowCache — готовые размытые маски теней. Когда их становится много, кеш очищается целиком:
// на экране обычно десятки разных теней, а не тысячи.
type shadowCache struct {
	mu sync.Mutex
	m  map[shadowKey]*image.Alpha
}

const shadowCacheSize = 128

var shadows shadowCache

func (c *shadowCache) get(k shadowKey) *image.Alpha {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.m[k]
}

func (c *shadowCache) put(k shadowKey, m *image.Alpha) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil || len(c.m) >= shadowCacheSize {
		c.m = map[shadowKey]*image.Alpha{}
	}
	c.m[k] = m
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
