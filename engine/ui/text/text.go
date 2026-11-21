// Package text — вёрстка и отрисовка текста.
//
// Пока это первая, простая версия: шрифт Go (лежит в golang.org/x/image, поддерживает
// латиницу и кириллицу), перенос по словам, выравнивание. Сложные письменности (арабский,
// деванагари), лигатуры и фолбэк шрифтов появятся с переходом на go-text (ADR-0006).
package text

import (
	"fmt"
	"image"
	"math"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"hexago/ui/geom"
	"hexago/ui/gfx"
)

// Weight — насыщенность шрифта. Нулевое значение — «не задана»: берётся у стиля по умолчанию
// (widgets.DefaultTextStyle), а если и там нет — Regular.
type Weight uint8

const (
	Regular Weight = iota + 1 // обычный (400)
	Medium                    // полужирный (500) — в Material 3 им набраны заголовки и подписи кнопок
	Bold                      // жирный (700)
)

// Align — выравнивание строк внутри ширины абзаца.
type Align uint8

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

// Style — как выглядит текст, как TextStyle во Flutter. Нулевое поле — «не задано»: оно берётся
// у стиля по умолчанию (Merge), а если и там нет — значение из комментария.
type Style struct {
	FontSize      float64   // размер в dp; 0 — 14, как во Flutter
	Color         gfx.Color // цвет; прозрачный (нулевой) — чёрный с непрозрачностью 87 %
	FontWeight    Weight    // насыщенность; 0 — Regular
	LineHeight    float64   // высота строки в dp (как line height в Material 3); 0 — как у шрифта
	LetterSpacing float64   // разрядка: добавка после каждой буквы, в dp
}

// Merge дополняет незаданные поля стиля полями base: s.Merge(base).
func (s Style) Merge(base Style) Style {
	if s.FontSize == 0 {
		s.FontSize = base.FontSize
	}
	if s.Color.IsTransparent() {
		s.Color = base.Color
	}
	if s.FontWeight == 0 {
		s.FontWeight = base.FontWeight
	}
	if s.LineHeight == 0 {
		s.LineHeight = base.LineHeight
	}
	if s.LetterSpacing == 0 {
		s.LetterSpacing = base.LetterSpacing
	}
	return s
}

func (s Style) size() float64 {
	if s.FontSize <= 0 {
		return 14
	}
	return s.FontSize
}

func (s Style) color() gfx.Color {
	if s.Color.IsTransparent() {
		return gfx.Black.WithOpacity(0.87)
	}
	return s.Color
}

// Paragraph — свёрстанный текст: строки, их ширины и общий размер (всё в dp).
type Paragraph struct {
	style  Style
	lines  []string
	widths []float64
	Width  float64 // ширина самой длинной строки
	Height float64
	ascent float64
	line   float64 // высота строки
}

// Layout верстает текст в ширину maxWidth: переносит по словам и по символам \n.
func Layout(s string, style Style, maxWidth float64) *Paragraph {
	face := faceFor(style.FontWeight, style.size())
	m := face.Metrics()
	p := &Paragraph{style: style, ascent: fx(m.Ascent), line: fx(m.Height)}
	if style.LineHeight > 0 {
		// Лишнее (или недостающее) место делится поровну над и под строкой, как в Material.
		p.ascent += (style.LineHeight - p.line) / 2
		p.line = style.LineHeight
	}
	measure := func(t string) float64 { return advance(face, style.size(), t, style.LetterSpacing) }

	for _, raw := range strings.Split(s, "\n") {
		words := strings.FieldsFunc(raw, unicode.IsSpace)
		if len(words) == 0 {
			p.add("", 0)
			continue
		}
		cur := words[0]
		for _, w := range words[1:] {
			try := cur + " " + w
			if measure(try) > maxWidth {
				p.add(cur, measure(cur))
				cur = w
				continue
			}
			cur = try
		}
		p.add(cur, measure(cur))
	}
	p.Height = p.line * float64(len(p.lines))
	return p
}

func (p *Paragraph) add(line string, width float64) {
	p.lines = append(p.lines, line)
	p.widths = append(p.widths, width)
	p.Width = max(p.Width, width)
}

// Paint рисует текст: at — левый верхний угол абзаца, boxWidth — ширина, в которой
// выравниваются строки. Текст растеризуется в пикселях экрана (с учётом масштаба холста),
// поэтому он чёткий при любой плотности.
func (p *Paragraph) Paint(c *gfx.Canvas, at geom.Offset, align Align, boxWidth float64) {
	m := c.Matrix()
	k := math.Sqrt(math.Abs(m.A*m.D - m.B*m.C)) // пикселей экрана в одном dp
	face := faceFor(p.style.FontWeight, p.style.size()*k)
	col := p.style.color()

	for i, line := range p.lines {
		if line == "" {
			continue
		}
		x := at.X
		switch align {
		case AlignCenter:
			x += (boxWidth - p.widths[i]) / 2
		case AlignRight:
			x += boxWidth - p.widths[i]
		}
		baseline := m.Apply(geom.Offset{X: x, Y: at.Y + p.ascent + float64(i)*p.line})
		drawLine(c, face, p.style.size()*k, line, baseline, col, p.style.LetterSpacing*k)
	}
}

// advance — ширина строки с разрядкой spacing после каждой буквы: шрифтом face, а символы, которых в нём
// нет, — запасным шрифтом размера size.
func advance(face font.Face, size float64, s string, spacing float64) float64 {
	w := 0.0
	for _, r := range runs(s) {
		w += runWidth(face, size, r, spacing)
	}
	return w
}

// advanceGo — ширина строки шрифтом Go.
func advanceGo(face font.Face, s string, spacing float64) float64 {
	if spacing == 0 {
		return fx(font.MeasureString(face, s))
	}
	var w float64
	for _, r := range s {
		a, _ := face.GlyphAdvance(r)
		w += fx(a) + spacing
	}
	return w
}

// drawLine рисует строку с базовой линией в точке экрана origin (в пикселях): шрифтом face, а символы,
// которых в нём нет, — запасным шрифтом размера size пикселей. spacing — разрядка в пикселях.
func drawLine(c *gfx.Canvas, face font.Face, size float64, s string, origin geom.Offset, col gfx.Color, spacing float64) {
	parts := runs(s)
	if len(parts) == 1 && !parts[0].fallback {
		drawGoRun(c, face, s, origin, col, spacing)
		return
	}
	x := origin.X
	for _, r := range parts {
		if r.fallback {
			if p := renderFallback(r.s, size, col); p.img != nil {
				c.DrawPixels(p.img, image.Point{X: int(math.Round(x)), Y: int(math.Round(origin.Y - p.ascent))})
			}
		} else {
			drawGoRun(c, face, r.s, geom.Offset{X: x, Y: origin.Y}, col, spacing)
		}
		x += runWidth(face, size, r, spacing)
	}
}

// drawGoRun рисует строку шрифтом Go. Маска строки берётся из кеша: текст на экране от кадра к кадру
// обычно тот же, а растеризация букв — дорогая.
func drawGoRun(c *gfx.Canvas, face font.Face, s string, origin geom.Offset, col gfx.Color, spacing float64) {
	ix, iy := int(math.Floor(origin.X)), int(math.Floor(origin.Y))
	// Дробная часть позиции (с шагом 1/4 пикселя) — чтобы буквы стояли точно, а не прыгали по пикселям.
	fx := math.Round((origin.X-float64(ix))*4) / 4
	fy := math.Round((origin.Y-float64(iy))*4) / 4
	key := lineKey{face: face, s: s, spacing: spacing, fx: fx, fy: fy}
	l := lines.get(key)
	if l == nil {
		l = rasterLine(face, s, fx, fy, spacing)
		lines.put(key, l)
	}
	c.DrawMask(l.mask, image.Point{X: ix + l.minX, Y: iy + l.minY}, col)
}

// rasterLine растеризует строку в маску; (minX, minY) — где левый верхний угол маски относительно
// базовой точки строки.
func rasterLine(face font.Face, s string, fx, fy, spacing float64) *lineMask {
	bounds, _ := font.BoundString(face, s)
	extra := int(math.Ceil(math.Abs(spacing) * float64(utf8.RuneCountInString(s))))
	minX, minY := bounds.Min.X.Floor()-1, bounds.Min.Y.Floor()-1
	maxX, maxY := bounds.Max.X.Ceil()+2+extra, bounds.Max.Y.Ceil()+1
	mask := image.NewAlpha(image.Rect(0, 0, maxX-minX, maxY-minY))
	d := font.Drawer{
		Dst:  mask,
		Src:  image.Opaque,
		Face: face,
		Dot: fixed.Point26_6{
			X: fixed.I(-minX) + fixed.Int26_6(fx*64),
			Y: fixed.I(-minY) + fixed.Int26_6(fy*64),
		},
	}
	if spacing == 0 {
		d.DrawString(s)
	} else {
		for _, r := range s {
			d.DrawString(string(r))
			d.Dot.X += fixed.Int26_6(spacing * 64)
		}
	}
	return &lineMask{mask: mask, minX: minX, minY: minY}
}

// lineKey — по чему маски строк одинаковы.
type lineKey struct {
	face    font.Face
	s       string
	spacing float64
	fx, fy  float64
}

type lineMask struct {
	mask       *image.Alpha
	minX, minY int
}

// lineCache — готовые маски строк. Когда они занимают больше lineCacheBytes, кеш очищается целиком:
// просто и не даёт памяти расти при длинной прокрутке.
type lineCache struct {
	mu    sync.Mutex
	m     map[lineKey]*lineMask
	bytes int
}

const lineCacheBytes = 8 << 20

var lines lineCache

func (c *lineCache) get(k lineKey) *lineMask {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.m[k]
}

func (c *lineCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m, c.bytes = nil, 0
}

func (c *lineCache) put(k lineKey, l *lineMask) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil || c.bytes+len(l.mask.Pix) > lineCacheBytes {
		c.m, c.bytes = map[lineKey]*lineMask{}, 0
	}
	c.m[k] = l
	c.bytes += len(l.mask.Pix)
}

// faceFor возвращает шрифт нужной насыщенности и размера (размер в единицах координат).
// Шрифты кешируются: создание face — дорогая операция.
func faceFor(w Weight, size float64) font.Face {
	if w == 0 {
		w = Regular
	}
	key := faceKey{w, math.Round(size*4) / 4}
	facesMu.Lock()
	defer facesMu.Unlock()
	if f, ok := faces[key]; ok {
		return f
	}
	f, err := opentype.NewFace(parsed(w), &opentype.FaceOptions{Size: key.size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		panic(fmt.Sprintf("text: шрифт размера %v: %v", key.size, err))
	}
	faces[key] = f
	return f
}

type faceKey struct {
	w    Weight
	size float64
}

var (
	facesMu sync.Mutex
	faces   = map[faceKey]font.Face{}
	fonts   = map[Weight]*opentype.Font{}
)

func parsed(w Weight) *opentype.Font {
	if f, ok := fonts[w]; ok {
		return f
	}
	src := goregular.TTF
	switch w {
	case Medium:
		src = gomedium.TTF
	case Bold:
		src = gobold.TTF
	}
	f, err := opentype.Parse(src)
	if err != nil {
		panic("text: встроенный шрифт не читается: " + err.Error())
	}
	fonts[w] = f
	return f
}

// fx переводит число 26.6 с фиксированной точкой в float64.
func fx(v fixed.Int26_6) float64 { return float64(v) / 64 }
