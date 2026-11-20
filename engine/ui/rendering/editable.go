package rendering

import (
	"math"
	"strings"
	"unicode/utf8"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/text"
)

// CursorWidth — ширина курсора поля ввода, dp (как в Material 3).
const CursorWidth = 2.0

// EditableBox — текст поля ввода: строки, выделение, подчёркивание набираемого слова и курсор.
// Однострочное поле прокручивается по горизонтали так, чтобы курсор был виден.
type EditableBox struct {
	Node
	Text                     string
	Style                    text.Style
	SelStart, SelEnd         int // выделение в рунах; курсор — в SelEnd
	ComposeStart, ComposeEnd int // слово, которое сейчас набирает клавиатура (подчёркнуто)
	Focused                  bool
	CursorVisible            bool // курсор мигает: сейчас виден
	CursorColor              gfx.Color
	SelectionColor           gfx.Color
	Multiline                bool
	Obscure                  bool // пароль: вместо букв — точки

	layout *text.EditLayout
	scroll float64 // сдвиг однострочного текста влево, dp
}

// display — что показывать: текст или точки на месте букв.
func (b *EditableBox) display() string {
	if b.Obscure {
		return strings.Repeat("•", utf8.RuneCountInString(b.Text))
	}
	return b.Text
}

func (b *EditableBox) PerformLayout(c Constraints) geom.Size {
	wrap := b.Multiline && !math.IsInf(c.MaxWidth, 1)
	b.layout = text.LayoutEditable(b.display(), b.Style, c.MaxWidth-CursorWidth, wrap)
	w := c.MaxWidth
	if math.IsInf(w, 1) {
		w = b.layout.Width + CursorWidth
	}
	size := c.Constrain(geom.Size{Width: w, Height: b.layout.Height})
	if !b.Multiline {
		// Курсор не уходит за края: текст прокручивается вслед за ним.
		scroll := b.scroll
		x, _, _ := b.layout.Caret(b.SelEnd)
		if x-scroll > size.Width-CursorWidth {
			scroll = x - size.Width + CursorWidth
		}
		if x-scroll < 0 {
			scroll = x
		}
		scroll = math.Max(0, math.Min(scroll, b.layout.Width+CursorWidth-size.Width))
		if scroll != b.scroll {
			b.scroll = scroll
			MarkNeedsPaint(b)
		}
	} else {
		b.scroll = 0
	}
	return size
}

func (b *EditableBox) Paint(ctx *PaintContext, offset geom.Offset) {
	if b.layout == nil {
		return
	}
	c := ctx.Canvas
	c.Save()
	defer c.Restore()
	c.ClipRect(geom.RectFromSize(offset, SizeOf(b)))
	at := geom.Offset{X: offset.X - b.scroll, Y: offset.Y}
	if b.Focused && b.SelStart != b.SelEnd {
		for _, r := range b.layout.SelectionRects(b.SelStart, b.SelEnd) {
			c.DrawRect(r.Translate(at), gfx.Paint{Color: b.SelectionColor})
		}
	}
	b.layout.Paint(c, at)
	if b.ComposeEnd > b.ComposeStart {
		under := b.Style.Color
		if under.IsTransparent() {
			under = gfx.Black
		}
		for _, r := range b.layout.SelectionRects(b.ComposeStart, b.ComposeEnd) {
			r = r.Translate(at)
			c.DrawRect(geom.Rect{Left: r.Left, Top: r.Bottom - 1.5, Right: r.Right, Bottom: r.Bottom - 0.5}, gfx.Paint{Color: under})
		}
	}
	if b.Focused && b.CursorVisible && b.SelStart == b.SelEnd {
		x, top, _ := b.layout.Caret(b.SelEnd)
		c.DrawRect(geom.Rect{Left: at.X + x, Top: at.Y + top, Right: at.X + x + CursorWidth,
			Bottom: at.Y + top + b.layout.LineHeight()}, gfx.Paint{Color: b.CursorColor})
	}
}

func (b *EditableBox) HitTestSelf(geom.Offset) bool { return true }

// IndexAt — позиция в тексте под точкой p (в координатах бокса).
func (b *EditableBox) IndexAt(p geom.Offset) int {
	if b.layout == nil {
		return 0
	}
	return b.layout.IndexAt(geom.Offset{X: p.X + b.scroll, Y: p.Y})
}

// CaretRect — где курсор перед позицией i (в координатах бокса): для прокрутки к полю.
func (b *EditableBox) CaretRect(i int) geom.Rect {
	if b.layout == nil {
		return geom.Rect{}
	}
	x, top, _ := b.layout.Caret(i)
	return geom.Rect{Left: x - b.scroll, Top: top, Right: x - b.scroll + CursorWidth, Bottom: top + b.layout.LineHeight()}
}
