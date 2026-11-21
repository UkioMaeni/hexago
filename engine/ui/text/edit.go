package text

import (
	"math"
	"unicode"

	"golang.org/x/image/font"

	"hexago/ui/geom"
	"hexago/ui/gfx"
)

// EditLayout — вёрстка редактируемого текста. В отличие от Layout, все символы стоят на своих местах:
// пробелы не схлопываются, и известно, где начинается каждый символ, — так ставятся курсор и
// выделение и находится символ под пальцем. Позиции в тексте — номера рун (0…len), как у
// widgets.TextSelection.
type EditLayout struct {
	style  Style
	runes  []rune
	lines  []editLine
	Width  float64 // ширина самой длинной строки, dp
	Height float64 // высота всех строк, dp (у пустого текста — одна строка)
	ascent float64
	line   float64 // высота строки
}

// editLine — строка: руны [start, end) без перевода строки; x[i] — где начинается руна start+i,
// x[end-start] — где строка кончается. stop[i] — может ли там стоять курсор (не внутри эмодзи из
// нескольких рун и не между буквой и её диакритикой).
type editLine struct {
	start, end int
	x          []float64
	stop       []bool
}

// LayoutEditable верстает текст s. wrap — переносить по словам в ширину maxWidth (многострочное поле);
// иначе строки делятся только символами \n.
func LayoutEditable(s string, style Style, maxWidth float64, wrap bool) *EditLayout {
	face := faceFor(style.FontWeight, style.size())
	m := face.Metrics()
	l := &EditLayout{style: style, runes: []rune(s), ascent: fx(m.Ascent), line: fx(m.Height)}
	if style.LineHeight > 0 {
		l.ascent += (style.LineHeight - l.line) / 2
		l.line = style.LineHeight
	}
	start := 0
	for i := 0; i <= len(l.runes); i++ {
		if i == len(l.runes) || l.runes[i] == '\n' {
			l.addHardLine(face, start, i, maxWidth, wrap)
			start = i + 1
		}
	}
	l.Height = l.line * float64(len(l.lines))
	return l
}

// addHardLine добавляет строку [start, end) — с переносами по словам, если wrap.
func (l *EditLayout) addHardLine(face font.Face, start, end int, maxWidth float64, wrap bool) {
	spacing := l.style.LetterSpacing
	size := l.style.size()
	for {
		x, stop := []float64{0}, []bool{true}
		prev := rune(-1)
		breakAt := -1 // после последнего пробела: здесь можно перенести
		i := start
		for i < end {
			j := min(ClusterEnd(l.runes, i), end) // видимый символ [i, j)
			cl := l.runes[i:j]
			before := x[len(x)-1]
			w := before
			if needsFallback(cl) {
				w += measureFallback(string(cl), size) + spacing // эмодзи и то, чего нет в шрифте Go
				prev = -1
			} else {
				for _, r := range cl {
					if prev >= 0 {
						w += fx(face.Kern(prev, r))
					}
					a, _ := face.GlyphAdvance(r)
					w += fx(a) + spacing
					prev = r
				}
			}
			// Не помещается — перенести после последнего пробела (пробелы остаются в конце строки)
			// или, если пробела не было, прямо здесь.
			if wrap && w > maxWidth && !unicode.IsSpace(l.runes[i]) && i > start {
				if breakAt > start {
					i = breakAt
					x, stop = x[:breakAt-start+1], stop[:breakAt-start+1]
				}
				break
			}
			for k := i + 1; k < j; k++ { // внутри символа курсор не стоит
				x, stop = append(x, before), append(stop, false)
			}
			x, stop = append(x, w), append(stop, true)
			if unicode.IsSpace(l.runes[i]) {
				breakAt = j
			}
			i = j
		}
		l.lines = append(l.lines, editLine{start: start, end: i, x: x, stop: stop})

		l.Width = math.Max(l.Width, x[len(x)-1])
		if i >= end {
			return
		}
		start = i
	}
}

// LineHeight — высота одной строки, dp.
func (l *EditLayout) LineHeight() float64 { return l.line }

// lineOf — строка, в которой стоит позиция i. Позиция на мягком переносе принадлежит следующей строке.
func (l *EditLayout) lineOf(i int) int {
	for n := len(l.lines) - 1; n >= 0; n-- {
		if i >= l.lines[n].start {
			return n
		}
	}
	return 0
}

// Caret — где курсор перед руной i: x, верх строки и её номер.
func (l *EditLayout) Caret(i int) (x, top float64, line int) {
	i = max(0, min(i, len(l.runes)))
	n := l.lineOf(i)
	ln := l.lines[n]
	j := min(i, ln.end) - ln.start
	return ln.x[j], float64(n) * l.line, n
}

// IndexAt — позиция в тексте, ближайшая к точке p (dp от левого верхнего угла текста).
func (l *EditLayout) IndexAt(p geom.Offset) int {
	n := int(math.Floor(p.Y / l.line))
	n = max(0, min(n, len(l.lines)-1))
	ln := l.lines[n]
	best, prevX := ln.start, ln.x[0]
	for j := 1; j < len(ln.x); j++ {
		if !ln.stop[j] {
			continue
		}
		if p.X < (prevX+ln.x[j])/2 {
			break
		}
		best, prevX = ln.start+j, ln.x[j]
	}
	// Мягкий перенос: пробел в конце строки, конец строки — уже начало следующей.
	if best == ln.end && n+1 < len(l.lines) && l.lines[n+1].start == ln.end && best > ln.start {
		for best--; best > ln.start && !ln.stop[best-ln.start]; best-- {
		}
	}
	return best
}

// SelectionRects — прямоугольники выделения [a, b) по строкам (dp).
func (l *EditLayout) SelectionRects(a, b int) []geom.Rect {
	if a > b {
		a, b = b, a
	}
	var out []geom.Rect
	for n, ln := range l.lines {
		s, e := max(a, ln.start), min(b, ln.end)
		lineBreak := b > ln.end && a <= ln.end && n+1 < len(l.lines) // выделение уходит на следующую строку
		if s > e || (s == e && !lineBreak) {
			continue
		}
		x0, x1 := ln.x[s-ln.start], ln.x[e-ln.start]
		if lineBreak {
			x1 += l.line / 4 // видно, что выделен и перевод строки
		}
		top := float64(n) * l.line
		out = append(out, geom.Rect{Left: x0, Top: top, Right: x1, Bottom: top + l.line})
	}
	return out
}

// WordAt — слово вокруг позиции i: [start, end). Слово — буквы и цифры подряд; на пробеле или знаке —
// сам этот символ.
func (l *EditLayout) WordAt(i int) (start, end int) {
	return WordAt(l.runes, i)
}

// WordAt — слово в runes вокруг позиции i: [start, end).
func WordAt(runes []rune, i int) (start, end int) {
	if len(runes) == 0 {
		return 0, 0
	}
	i = max(0, min(i, len(runes)-1))
	isWord := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
	if !isWord(runes[i]) {
		if i > 0 && isWord(runes[i-1]) {
			i--
		} else {
			for c := 0; c < len(runes); c = ClusterEnd(runes, c) { // эмодзи и знаки — целым символом
				if e := ClusterEnd(runes, c); i < e {
					return c, e
				}
			}
			return i, i + 1
		}
	}
	start, end = i, i
	for start > 0 && isWord(runes[start-1]) {
		start--
	}
	for end < len(runes) && isWord(runes[end]) {
		end++
	}
	return start, end
}

// Paint рисует текст: at — левый верхний угол (dp), цвет — из стиля.
func (l *EditLayout) Paint(c *gfx.Canvas, at geom.Offset) {
	m := c.Matrix()
	k := math.Sqrt(math.Abs(m.A*m.D - m.B*m.C))
	face := faceFor(l.style.FontWeight, l.style.size()*k)
	col := l.style.color()
	for n, ln := range l.lines {
		if ln.end == ln.start {
			continue
		}
		baseline := m.Apply(geom.Offset{X: at.X, Y: at.Y + l.ascent + float64(n)*l.line})
		drawLine(c, face, l.style.size()*k, string(l.runes[ln.start:ln.end]), baseline, col, l.style.LetterSpacing*k)
	}
}
