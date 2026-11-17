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

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"hexago/ui/geom"
	"hexago/ui/gfx"
)

// Weight — насыщенность шрифта.
type Weight uint8

const (
	Regular Weight = iota
	Bold
)

// Align — выравнивание строк внутри ширины абзаца.
type Align uint8

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

// Style — как выглядит текст, как TextStyle во Flutter.
type Style struct {
	FontSize   float64   // размер в dp; 0 — 14, как во Flutter
	Color      gfx.Color // цвет; прозрачный (нулевой) — чёрный с непрозрачностью 87 %
	FontWeight Weight
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
	measure := func(t string) float64 { return fx(font.MeasureString(face, t)) }

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
		drawLine(c, face, line, baseline, col)
	}
}

// drawLine рисует строку с базовой линией в точке экрана origin (в пикселях).
func drawLine(c *gfx.Canvas, face font.Face, s string, origin geom.Offset, col gfx.Color) {
	bounds, _ := font.BoundString(face, s)
	minX, minY := bounds.Min.X.Floor()-1, bounds.Min.Y.Floor()-1
	maxX, maxY := bounds.Max.X.Ceil()+2, bounds.Max.Y.Ceil()+1
	ix, iy := int(math.Floor(origin.X)), int(math.Floor(origin.Y))
	mask := image.NewAlpha(image.Rect(0, 0, maxX-minX, maxY-minY))
	d := font.Drawer{
		Dst:  mask,
		Src:  image.Opaque,
		Face: face,
		// Дробная часть позиции — чтобы буквы стояли точно, а не прыгали по целым пикселям.
		Dot: fixed.Point26_6{
			X: fixed.I(-minX) + fixed.Int26_6((origin.X-float64(ix))*64),
			Y: fixed.I(-minY) + fixed.Int26_6((origin.Y-float64(iy))*64),
		},
	}
	d.DrawString(s)
	c.DrawMask(mask, image.Point{X: ix + minX, Y: iy + minY}, col)
}

// faceFor возвращает шрифт нужной насыщенности и размера (размер в единицах координат).
// Шрифты кешируются: создание face — дорогая операция.
func faceFor(w Weight, size float64) font.Face {
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
	if w == Bold {
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
