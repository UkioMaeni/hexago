package text

import (
	"math"
	"testing"

	"hexago/ui/geom"
)

func TestEditLayoutKeepsSpacesAndFindsCarets(t *testing.T) {
	st := Style{FontSize: 16}
	l := LayoutEditable("ab  c", st, math.Inf(1), false)
	if len(l.lines) != 1 {
		t.Fatalf("строк %d", len(l.lines))
	}
	// Два пробела не схлопываются: каждый двигает курсор вправо.
	x2, _, _ := l.Caret(2)
	x3, _, _ := l.Caret(3)
	x4, _, _ := l.Caret(4)
	if !(x2 < x3 && x3 < x4) {
		t.Errorf("курсоры у пробелов: %v %v %v", x2, x3, x4)
	}
	// Точка между символами находит ближайшую границу.
	for i := 0; i <= 5; i++ {
		x, _, _ := l.Caret(i)
		if got := l.IndexAt(geom.Offset{X: x + 0.5, Y: 5}); got != i {
			t.Errorf("IndexAt у позиции %d вернул %d", i, got)
		}
	}
	// Пустой текст — одна строка.
	if e := LayoutEditable("", st, 100, true); len(e.lines) != 1 || e.Height <= 0 {
		t.Errorf("пустой текст: %d строк, высота %v", len(e.lines), e.Height)
	}
}

func TestEditLayoutWrapsByWords(t *testing.T) {
	st := Style{FontSize: 16}
	full := LayoutEditable("привет мир", st, math.Inf(1), false).Width
	l := LayoutEditable("привет мир", st, full-10, true)
	if len(l.lines) != 2 {
		t.Fatalf("строк %d, ждали 2", len(l.lines))
	}
	if l.lines[0].start != 0 || l.lines[0].end != 7 || l.lines[1].start != 7 {
		t.Errorf("перенос не по слову: %+v %+v", l.lines[0], l.lines[1])
	}
	// Позиция 7 (начало «мир») — на второй строке.
	if _, top, n := l.Caret(7); n != 1 || top != l.LineHeight() {
		t.Errorf("курсор перед «мир»: строка %d, верх %v", n, top)
	}
	// Перевод строки: \n делит строки.
	if m := LayoutEditable("а\nб", st, 500, true); len(m.lines) != 2 || m.lines[1].start != 2 {
		t.Errorf("\n: %+v", m.lines)
	}
	// Выделение через перенос — по прямоугольнику на строку.
	if r := l.SelectionRects(3, 9); len(r) != 2 {
		t.Errorf("прямоугольников выделения %d", len(r))
	}
	if s, e := WordAt([]rune("привет мир"), 8); s != 7 || e != 10 {
		t.Errorf("слово вокруг 8: [%d, %d)", s, e)
	}
}
