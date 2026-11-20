package widgets

import (
	"time"
	"unicode/utf8"

	"hexago/ui/geom"
	"hexago/ui/gestures"
	"hexago/ui/gfx"
	"hexago/ui/rendering"
	"hexago/ui/text"
)

// EditableText — поле ввода без оформления, как EditableText во Flutter: текст, курсор, выделение,
// клавиатура. Касание ставит курсор (и открывает клавиатуру), двойное касание и долгое нажатие
// выделяют слово. Оформление Material (подпись, рамка, подсказка) — material.TextField.
//
// Без Controller поле хранит текст само и сообщает о нём через OnChanged; без Focus — свой узел фокуса.
type EditableText struct {
	State
	Controller *TextEditingController
	Focus      *FocusNode

	Style          text.Style // незаданное — из DefaultTextStyle
	CursorColor    gfx.Color  // прозрачный — чёрный
	SelectionColor gfx.Color  // прозрачный — голубой

	Multiline     bool // переносить строки и вводить Enter как новую строку
	Obscure       bool // пароль: точки вместо букв
	Keyboard      KeyboardType
	Action        TextInputAction
	NoAutocorrect bool
	Capitalize    Capitalization
	AutoFocus     bool // получить фокус сразу, как поле появится
	ReadOnly      bool // текст можно выделить, но не изменить

	OnChanged   func(text string) // текст изменён с клавиатуры
	OnSubmitted func(text string) // нажата кнопка ввода («Готово», «Отправить»…)

	ctrl       *TextEditingController
	ownCtrl    bool
	stopCtrl   func()
	focus      *FocusNode
	ownFocus   bool
	stopFocus  func()
	fromInput  bool // значение меняет клавиатура: не отправлять его ей обратно
	cursorOn   bool
	stopBlink  func()
	downAt     time.Duration // когда палец коснулся поля в последний раз
	lastTap    time.Duration // когда было прошлое касание — для двойного
	lastTapPos geom.Offset
}

// Курсор мигает с таким полупериодом; два касания ближе doubleTapTime — двойное касание.
const (
	cursorBlink   = 500 * time.Millisecond
	doubleTapTime = 300 * time.Millisecond
)

func (e *EditableText) Init(ctx BuildContext) {
	e.useController(e.Controller)
	e.useFocus(e.Focus)
	if e.AutoFocus {
		e.focus.RequestFocus()
	}
}

func (e *EditableText) Update(_ BuildContext, old EditableText) {
	if e.Controller != old.Controller && !(e.Controller == nil && e.ownCtrl) {
		e.useController(e.Controller)
	}
	if e.Focus != old.Focus && !(e.Focus == nil && e.ownFocus) {
		e.useFocus(e.Focus)
	}
	if o := e.owner(); e.focus.HasFocus() && e.connected() && o.textInput != nil && e.config() != old.config() {
		o.textInput.Show(e.config(), e.ctrl.Value()) // сменились настройки клавиатуры
	}
}

func (e *EditableText) Dispose() {
	e.stopCtrl()
	e.stopFocus()
	e.disconnect() // сначала отключить клавиатуру: тогда снятие фокуса её спрячет
	e.releaseFocus()
	if e.stopBlink != nil {
		e.stopBlink()
	}
}

func (e *EditableText) owner() *BuildOwner { return e.element.owner }

func (e *EditableText) useController(c *TextEditingController) {
	if e.stopCtrl != nil {
		e.stopCtrl()
	}
	e.ctrl, e.ownCtrl = c, c == nil
	if c == nil {
		e.ctrl = &TextEditingController{}
	}
	e.stopCtrl = e.ctrl.AddListener(e.valueChanged)
}

func (e *EditableText) useFocus(f *FocusNode) {
	if e.stopFocus != nil {
		e.stopFocus()
		e.releaseFocus()
	}
	e.focus, e.ownFocus = f, f == nil
	if f == nil {
		e.focus = &FocusNode{}
	}
	if e.focus.owner == nil {
		e.focus.attach(e.element)
	} else if e.focus.element == nil {
		e.focus.element = e.element
	}
	e.stopFocus = e.focus.AddListener(e.focusChanged)
	if e.focus.HasFocus() {
		e.focusChanged()
	}
}

// config — какая клавиатура нужна полю.
func (e *EditableText) config() TextInputConfig {
	a := e.Action
	if a == ActionDefault {
		a = ActionDone
		if e.Multiline {
			a = ActionNewline
		}
	}
	return TextInputConfig{Keyboard: e.Keyboard, Action: a, Multiline: e.Multiline, Obscure: e.Obscure,
		NoAutocorrect: e.NoAutocorrect || e.Obscure, Capitalize: e.Capitalize}
}

func (e *EditableText) connected() bool { return e.owner().textClient == textInputClient(e) }

// connect подключает клавиатуру к полю и показывает её.
func (e *EditableText) connect() {
	o := e.owner()
	o.textClient = e
	if o.textInput != nil {
		o.keyboardShown = true
		o.textInput.Show(e.config(), e.ctrl.Value())
	}
}

// disconnect отключает клавиатуру от поля (спрячет её setFocus, если фокус никуда не перешёл).
func (e *EditableText) disconnect() {
	if o := e.owner(); o.textClient == textInputClient(e) {
		o.textClient = nil
	}
}

func (e *EditableText) focusChanged() {
	if !e.Mounted() {
		return
	}
	if e.focus.HasFocus() {
		e.connect()
		e.restartBlink()
	} else {
		e.disconnect()
		if e.stopBlink != nil {
			e.stopBlink()
		}
		if !e.ctrl.Value().Selection.IsCollapsed() { // без фокуса выделения не видно
			e.ctrl.SetSelection(Collapsed(e.ctrl.Value().Selection.Extent))
		}
	}
	e.SetState(nil)
}

// valueChanged — контроллер изменился: с клавиатуры или из кода приложения.
func (e *EditableText) valueChanged() {
	if !e.Mounted() {
		return
	}
	if !e.fromInput && e.connected() && e.owner().textInput != nil {
		e.owner().textInput.Update(e.ctrl.Value())
	}
	if e.focus.HasFocus() {
		e.restartBlink()
	}
	e.SetState(nil)
}

// restartBlink показывает курсор и запускает мигание заново (пока печатают, курсор не мигает).
func (e *EditableText) restartBlink() {
	if e.stopBlink != nil {
		e.stopBlink()
	}
	e.cursorOn = true
	var tick func()
	tick = func() {
		e.SetState(func() { e.cursorOn = !e.cursorOn })
		e.stopBlink = AfterFunc(e.element, cursorBlink, tick)
	}
	e.stopBlink = AfterFunc(e.element, cursorBlink, tick)
}

func (e *EditableText) editingFromPlatform(v TextEditingValue) {
	if e.ReadOnly {
		v.Text, v.Composing = e.ctrl.Value().Text, TextRange{}
	}
	old := e.ctrl.Value().Text
	e.fromInput = true
	e.ctrl.SetValue(v)
	e.fromInput = false
	if e.ctrl.Text() != old && e.OnChanged != nil {
		e.OnChanged(e.ctrl.Text())
	}
}

func (e *EditableText) actionFromPlatform(a TextInputAction) {
	switch a {
	case ActionNewline:
		return // новую строку клавиатура вставляет сама
	case ActionNext:
		if next := e.owner().nextFocus(e.focus); next != nil {
			next.RequestFocus()
		} else {
			e.focus.Unfocus()
		}
	case ActionDone, ActionGo, ActionSearch, ActionSend:
		e.focus.Unfocus()
	}
	if e.OnSubmitted != nil {
		e.OnSubmitted(e.ctrl.Text())
	}
}

// box — бокс текста поля (для поиска позиции под пальцем).
func (e *EditableText) box() *rendering.EditableBox {
	for b := e.element.FindBox(); b != nil; b = rendering.Child(b) {
		if eb, ok := b.(*rendering.EditableBox); ok {
			return eb
		}
	}
	return nil
}

func (e *EditableText) pointerDown(p gestures.PointerEvent) { e.downAt = p.Time }

// tap — касание: фокус и клавиатура, курсор под пальцем; второе быстрое касание выделяет слово.
func (e *EditableText) tap(d gestures.TapUpDetails) {
	if !e.focus.HasFocus() {
		e.focus.RequestFocus()
	} else if o := e.owner(); o.textInput != nil {
		o.keyboardShown = true
		o.textInput.Show(e.config(), e.ctrl.Value()) // клавиатуру могли спрятать кнопкой «Назад»
	}
	b := e.box()
	if b == nil {
		return
	}
	i := b.IndexAt(d.Local)
	double := e.lastTap > 0 && e.downAt-e.lastTap < doubleTapTime && d.Local.Sub(e.lastTapPos).Distance() < 24
	e.lastTap, e.lastTapPos = e.downAt, d.Local
	if double {
		e.lastTap = 0
		e.selectWord(i)
		return
	}
	e.ctrl.SetValue(TextEditingValue{Text: e.ctrl.Text(), Selection: Collapsed(i)})
}

// longPress — долгое нажатие выделяет слово под пальцем.
func (e *EditableText) longPress(d gestures.LongPressStartDetails) {
	if !e.focus.HasFocus() {
		e.focus.RequestFocus()
	}
	if b := e.box(); b != nil {
		e.selectWord(b.IndexAt(d.Local))
	}
}

func (e *EditableText) selectWord(i int) {
	if e.Obscure { // в пароле слов не видно — выделяется всё
		n := utf8.RuneCountInString(e.ctrl.Text())
		e.ctrl.SetValue(TextEditingValue{Text: e.ctrl.Text(), Selection: TextSelection{Base: 0, Extent: n}})
		return
	}
	s, end := text.WordAt([]rune(e.ctrl.Text()), i)
	e.ctrl.SetValue(TextEditingValue{Text: e.ctrl.Text(), Selection: TextSelection{Base: s, Extent: end}})
}

func (e *EditableText) Build(ctx BuildContext) Widget {
	v := e.ctrl.Value()
	cursor, sel := e.CursorColor, e.SelectionColor
	if cursor.IsTransparent() {
		cursor = gfx.Black
	}
	if sel.IsTransparent() {
		sel = gfx.ARGB(0x662196F3)
	}
	return Listener{OnPointerDown: e.pointerDown, Child: GestureDetector{
		OnTapUp:          e.tap,
		OnLongPressStart: e.longPress,
		Child: editableBox{
			text: v.Text, style: e.Style.Merge(DefaultTextStyleOf(ctx)),
			sel: v.Selection, composing: v.Composing,
			focused: e.focus.HasFocus(), cursorOn: e.cursorOn, cursor: cursor, selection: sel,
			multiline: e.Multiline, obscure: e.Obscure,
		},
	}}
}

// editableBox — бокс текста поля.
type editableBox struct {
	Base
	text              string
	style             text.Style
	sel               TextSelection
	composing         TextRange
	focused, cursorOn bool
	cursor, selection gfx.Color
	multiline         bool
	obscure           bool
}

func (w editableBox) CreateRenderObject(ctx BuildContext) rendering.Box {
	b := &rendering.EditableBox{}
	w.UpdateRenderObject(ctx, b)
	return b
}

func (w editableBox) UpdateRenderObject(_ BuildContext, rb rendering.Box) {
	b := rb.(*rendering.EditableBox)
	b.Text, b.Style = w.text, w.style
	b.SelStart, b.SelEnd = w.sel.Base, w.sel.Extent
	b.ComposeStart, b.ComposeEnd = w.composing.Start, w.composing.End
	b.Focused, b.CursorVisible = w.focused, w.cursorOn
	b.CursorColor, b.SelectionColor = w.cursor, w.selection
	b.Multiline, b.Obscure = w.multiline, w.obscure
}

// releaseFocus отвязывает узел фокуса от поля, которое уходит (или меняет узел).
func (e *EditableText) releaseFocus() {
	if e.focus.element == e.element {
		e.focus.detach()
	} else if e.focus.HasFocus() {
		e.focus.Unfocus()
	}
}
