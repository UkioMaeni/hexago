package widgets

import (
	"math"

	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/rendering"
	"hexago/ui/text"
)

// Ручки и меню выделения текста, как на Android: долгое нажатие или двойное касание выделяет слово,
// по краям выделения появляются ручки, их можно тянуть; над выделением — меню «Вырезать, Копировать,
// Вставить, Выбрать все». Касание по курсору показывает ручку под ним и меню со «Вставить».

// HandleKind — какая ручка выделения.
type HandleKind uint8

const (
	HandleCollapsed HandleKind = iota // под курсором
	HandleLeft                        // у начала выделения
	HandleRight                       // у конца выделения
)

// SelectionAction — кнопка меню выделения.
type SelectionAction struct {
	Label string
	Do    func()
}

// SelectionControls — как выглядят ручки и меню выделения. material.TextField задаёт свои;
// у EditableText без них ручек и меню нет (выделять можно, копировать — только с физической клавиатуры).
type SelectionControls interface {
	// Handle — ручка вида kind: виджет, его размер и точка в нём, которая встаёт на низ строки
	// у края выделения (под курсором).
	Handle(ctx BuildContext, kind HandleKind) (w Widget, size geom.Size, anchor geom.Offset)
	// Toolbar — меню над выделением с кнопками actions.
	Toolbar(ctx BuildContext, actions []SelectionAction) Widget
}

// showSelection показывает или прячет ручки и меню выделения.
func (e *EditableText) showSelection(handles, toolbar bool) {
	e.showHandles, e.showToolbar = handles, toolbar
	_, hasNav := find[navigatorScope](e.element)
	if e.Controls == nil || !hasNav || !e.focus.HasFocus() || (!handles && !toolbar) {
		e.hideSelection()
		return
	}
	if e.overlay == nil {
		e.overlay = ShowOverlay(e.element, e.buildSelection)
	} else {
		e.overlay.MarkNeedsBuild()
	}
}

func (e *EditableText) hideSelection() {
	e.showHandles, e.showToolbar = false, false
	if e.overlay != nil {
		e.overlay.Remove()
		e.overlay = nil
	}
}

// buildSelection — слой с ручками и меню (ShowOverlay).
func (e *EditableText) buildSelection(ctx BuildContext) Widget {
	b := e.box()
	if b == nil {
		return SizedBox{}
	}
	screen := MediaQueryOf(ctx)
	var kids []Widget
	sel := e.ctrl.Value().Selection
	if e.showHandles {
		if sel.IsCollapsed() {
			kids = append(kids, e.handle(ctx, b, HandleCollapsed))
		} else {
			kids = append(kids, e.handle(ctx, b, HandleLeft), e.handle(ctx, b, HandleRight))
		}
	}
	if acts := e.selectionActions(); e.showToolbar && e.dragging < 0 && len(acts) > 0 {
		kids = append(kids, Follow{Target: e.element, Child: e.Controls.Toolbar(ctx, acts),
			Place: func(r geom.Rect, size geom.Size) geom.Offset {
				sel := e.ctrl.Value().Selection
				s, end := b.CaretRect(sel.Start()), b.CaretRect(sel.End())
				x := r.Left + (s.Left+end.Left)/2 - size.Width/2
				if s.Top != end.Top {
					x = r.Left + r.Width()/2 - size.Width/2
				}
				y := r.Top + s.Top - size.Height - 8 // над выделением
				if y < screen.Padding.Top+8 {
					y = r.Top + end.Bottom + 32 // нет места — под ним (под ручками)
				}
				x = math.Max(8, math.Min(x, screen.Size.Width-size.Width-8))
				return geom.Offset{X: x, Y: y}
			}})
	}
	return Stack{Children: kids}
}

// edgeOf — позиция в тексте у ручки kind.
func edgeOf(sel TextSelection, kind HandleKind) int {
	switch kind {
	case HandleLeft:
		return sel.Start()
	case HandleRight:
		return sel.End()
	}
	return sel.Extent
}

// handle — ручка, которая стоит у края выделения и тянется пальцем.
func (e *EditableText) handle(ctx BuildContext, b *rendering.EditableBox, kind HandleKind) Widget {
	w, _, anchor := e.Controls.Handle(ctx, kind)
	return Follow{Target: e.element,
		Place: func(r geom.Rect, _ geom.Size) geom.Offset {
			c := b.CaretRect(edgeOf(e.ctrl.Value().Selection, kind))
			x := c.Left
			if kind == HandleCollapsed {
				x += rendering.CursorWidth / 2
			}
			return geom.Offset{X: r.Left + x - anchor.X, Y: r.Top + c.Bottom - anchor.Y}
		},
		Child: GestureDetector{
			Behavior:    rendering.Opaque, // вся площадь ручки ловит палец, даже где она прозрачна
			OnPanStart:  func(d gestures.DragStartDetails) { e.dragStart(kind, d.Position, b) },
			OnPanUpdate: func(d gestures.DragUpdateDetails) { e.dragMove(kind, d.Position, b) },
			OnPanEnd:    func(gestures.DragEndDetails) { e.dragEnd() },
			Child:       w,
		}}
}

// dragStart — палец взял ручку: запомнить, насколько он ниже края выделения.
func (e *EditableText) dragStart(kind HandleKind, at geom.Offset, b *rendering.EditableBox) {
	r := BoundsOf(e.element)
	c := b.CaretRect(edgeOf(e.ctrl.Value().Selection, kind))
	e.dragging = int(kind)
	e.grab = geom.Offset{X: r.Left + c.Left, Y: r.Top + (c.Top+c.Bottom)/2}.Sub(at)
	e.overlay.MarkNeedsBuild() // пока тянут, меню не видно
}

// dragMove — ручку тянут: край выделения идёт за пальцем.
func (e *EditableText) dragMove(kind HandleKind, at geom.Offset, b *rendering.EditableBox) {
	r := BoundsOf(e.element)
	i := b.IndexAt(at.Add(e.grab).Sub(r.TopLeft()))
	v := e.ctrl.Value()
	sel := v.Selection
	switch kind {
	case HandleCollapsed:
		sel = Collapsed(i)
	case HandleLeft:
		sel = TextSelection{Base: min(i, sel.End()-1), Extent: sel.End()}
	case HandleRight:
		sel = TextSelection{Base: sel.Start(), Extent: max(i, sel.Start()+1)}
	}
	e.ctrl.SetSelection(sel)
}

func (e *EditableText) dragEnd() {
	e.dragging = -1
	e.showSelection(e.showHandles, !e.ctrl.Value().Selection.IsCollapsed())
}

// selectionActions — что можно сделать с выделением сейчас.
func (e *EditableText) selectionActions() []SelectionAction {
	v := e.ctrl.Value()
	sel := v.Selection
	n := len([]rune(v.Text))
	var a []SelectionAction
	if !sel.IsCollapsed() && !e.Obscure && !e.ReadOnly {
		a = append(a, SelectionAction{Label: "Вырезать", Do: e.cut})
	}
	if !sel.IsCollapsed() && !e.Obscure {
		a = append(a, SelectionAction{Label: "Копировать", Do: e.copySelection})
	}
	if !e.ReadOnly && clipboardOf(e.element).HasText() {
		a = append(a, SelectionAction{Label: "Вставить", Do: e.paste})
	}
	if n > 0 && !(sel.Start() == 0 && sel.End() == n) {
		a = append(a, SelectionAction{Label: "Выбрать все", Do: e.selectAll})
	}
	return a
}

func (e *EditableText) selectedText() string {
	v := e.ctrl.Value()
	rs := []rune(v.Text)
	return string(rs[v.Selection.Start():v.Selection.End()])
}

// replaceSelection ставит s на место выделения (курсор — после s), как правка пользователя.
func (e *EditableText) replaceSelection(s string) {
	v := e.ctrl.Value()
	rs := []rune(v.Text)
	a, b := v.Selection.Start(), v.Selection.End()
	out := string(rs[:a]) + s + string(rs[b:])
	e.ctrl.SetValue(TextEditingValue{Text: out, Selection: Collapsed(a + len([]rune(s)))})
	if e.OnChanged != nil {
		e.OnChanged(out)
	}
}

func (e *EditableText) cut() {
	CopyText(e.element, e.selectedText())
	e.replaceSelection("")
	e.hideSelection()
}

func (e *EditableText) copySelection() {
	CopyText(e.element, e.selectedText())
	e.ctrl.SetSelection(Collapsed(e.ctrl.Value().Selection.End()))
	e.hideSelection()
}

func (e *EditableText) paste() {
	if s, ok := PasteText(e.element); ok {
		e.replaceSelection(s)
	}
	e.hideSelection()
}

func (e *EditableText) selectAll() {
	e.ctrl.SetSelection(TextSelection{Base: 0, Extent: len([]rune(e.ctrl.Text()))})
	e.showSelection(true, true)
}

// selectWordAt выделяет слово (у пароля — весь текст) и показывает ручки и меню.
func (e *EditableText) selectWordAt(i int) {
	v := e.ctrl.Value()
	rs := []rune(v.Text)
	s, end := text.WordAt(rs, i)
	if e.Obscure {
		s, end = 0, len(rs)
	}
	e.ctrl.SetValue(TextEditingValue{Text: v.Text, Selection: TextSelection{Base: s, Extent: end}})
	e.focus.RequestFocus() // после выделения: клавиатура подключится уже с ним (см. tap)
	e.showSelection(len(rs) > 0, true)
}
