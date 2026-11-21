package text

import (
	"image"
	"math"
	"sync"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"

	"hexago/ui/gfx"
)

// Fallback — запасной шрифт: рисует то, чего нет во встроенном шрифте Go (эмодзи, иероглифы,
// редкие символы), системными шрифтами платформы. Его задаёт app.Run (на Android — шрифты Android,
// цветные эмодзи); без него на месте таких символов — пустой квадрат.
type Fallback interface {
	// Measure — ширина строки s при размере шрифта size (в тех же единицах, что size).
	Measure(s string, size float64) float64
	// Render рисует строку s размером size (в пикселях) цветом col (цветные эмодзи — своими цветами).
	// ascent — расстояние от верха картинки до базовой линии, в пикселях.
	Render(s string, size float64, col gfx.Color) (img *image.NRGBA, ascent float64)
}

var fallback Fallback

// SetFallback задаёт запасной шрифт (его вызывает app.Run).
func SetFallback(f Fallback) {
	fallback = f
	measures.reset()
	pictures.reset()
	lines.reset()
}

// covered — есть ли символ во встроенном шрифте Go (у всех насыщенностей набор символов один).
func covered(r rune) bool {
	coverMu.Lock()
	defer coverMu.Unlock()
	if v, ok := coverage[r]; ok {
		return v
	}
	i, err := parsed(Regular).GlyphIndex(&coverBuf, r)
	v := err == nil && i != 0
	coverage[r] = v
	return v
}

var (
	coverMu  sync.Mutex
	coverBuf sfnt.Buffer
	coverage = map[rune]bool{}
)

// ClusterEnd — где кончается видимый символ, который начинается с руны i: эмодзи с оттенком кожи,
// склейки через ZWJ (👨‍👩‍👧), флаги (пары региональных букв), цифра с рамкой, буква с диакритикой.
// Курсор не встаёт внутрь такого символа.
func ClusterEnd(rs []rune, i int) int {
	if i >= len(rs) {
		return len(rs)
	}
	j := i + 1
	if regional(rs[i]) {
		if j < len(rs) && regional(rs[j]) {
			return j + 1
		}
		return j
	}
	for j < len(rs) {
		r := rs[j]
		switch {
		case r == 0xFE0F || r == 0xFE0E || r == 0x20E3, // вид символа, рамка цифры
			r >= 0x1F3FB && r <= 0x1F3FF,                           // оттенок кожи
			r >= 0xE0020 && r <= 0xE007F,                           // теги (флаги регионов)
			unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r): // диакритика
			j++
		case r == 0x200D && j+1 < len(rs): // склейка с следующим
			j += 2
		default:
			return j
		}
	}
	return j
}

func regional(r rune) bool { return r >= 0x1F1E6 && r <= 0x1F1FF }

// needsFallback — символ [i, j) рисует запасной шрифт: его нет в шрифте Go или он просит вид эмодзи.
func needsFallback(rs []rune) bool {
	if fallback == nil {
		return false
	}
	for _, r := range rs {
		if r == 0xFE0F || (!covered(r) && !unicode.IsSpace(r) && r != 0x200D) {
			return true
		}
	}
	return false
}

// textRun — кусок строки: шрифтом Go или запасным.
type textRun struct {
	s        string
	fallback bool
}

// runs делит строку на куски для шрифта Go и для запасного шрифта. Без запасного шрифта — один кусок.
func runs(s string) []textRun {
	if fallback == nil {
		return []textRun{{s: s}}
	}
	rs := []rune(s)
	var out []textRun
	start, fb := 0, false
	for i := 0; i < len(rs); {
		j := ClusterEnd(rs, i)
		f := needsFallback(rs[i:j])
		if i > start && f != fb {
			out = append(out, textRun{s: string(rs[start:i]), fallback: fb})
			start = i
		}
		fb = f
		i = j
	}
	if start < len(rs) || len(out) == 0 {
		out = append(out, textRun{s: string(rs[start:]), fallback: fb})
	}
	return out
}

// measureFallback — ширина куска запасным шрифтом (с кешем: платформу спрашивать дорого).
func measureFallback(s string, size float64) float64 {
	k := measureKey{s: s, size: math.Round(size * 4)}
	if w, ok := measures.get(k); ok {
		return w
	}
	w := fallback.Measure(s, size)
	measures.put(k, w)
	return w
}

type measureKey struct {
	s    string
	size float64
}

type measureCache struct {
	mu sync.Mutex
	m  map[measureKey]float64
}

func (c *measureCache) get(k measureKey) (float64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	w, ok := c.m[k]
	return w, ok
}

func (c *measureCache) put(k measureKey, w float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil || len(c.m) > 4096 {
		c.m = map[measureKey]float64{}
	}
	c.m[k] = w
}

func (c *measureCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = nil
}

var measures measureCache

// picture — готовая картинка куска запасным шрифтом.
type picture struct {
	img    *image.NRGBA
	ascent float64
}

type pictureKey struct {
	s    string
	size float64
	col  gfx.Color
}

type pictureCache struct {
	mu    sync.Mutex
	m     map[pictureKey]*picture
	bytes int
}

func (c *pictureCache) get(k pictureKey) *picture {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.m[k]
}

func (c *pictureCache) put(k pictureKey, p *picture) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	if p.img != nil {
		n = len(p.img.Pix)
	}
	if c.m == nil || c.bytes+n > 8<<20 {
		c.m, c.bytes = map[pictureKey]*picture{}, 0
	}
	c.m[k] = p
	c.bytes += n
}

func (c *pictureCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m, c.bytes = nil, 0
}

var pictures pictureCache

// renderFallback — картинка куска запасным шрифтом размера size пикселей.
func renderFallback(s string, size float64, col gfx.Color) *picture {
	k := pictureKey{s: s, size: math.Round(size * 4), col: col}
	if p := pictures.get(k); p != nil {
		return p
	}
	img, ascent := fallback.Render(s, size, col)
	p := &picture{img: img, ascent: ascent}
	pictures.put(k, p)
	return p
}

// runWidth — ширина куска: шрифтом Go (face) или запасным размера size.
func runWidth(face font.Face, size float64, r textRun, spacing float64) float64 {
	if r.fallback {
		return measureFallback(r.s, size) + spacing*float64(clusters(r.s))
	}
	return advanceGo(face, r.s, spacing)
}

// clusters — сколько видимых символов в строке.
func clusters(s string) int {
	rs := []rune(s)
	n := 0
	for i := 0; i < len(rs); i = ClusterEnd(rs, i) {
		n++
	}
	return n
}
