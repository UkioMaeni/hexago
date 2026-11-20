package widgets

import (
	"slices"
	"unicode/utf8"
)

// Ввод текста: значение поля (текст, выделение, набираемое слово), контроллер, через который
// приложение читает и задаёт текст, и связь с клавиатурой платформы (TextInput). Сами поля —
// EditableText и material.TextField.

// TextSelection — выделение в тексте: от Base до Extent (номера рун, 0…длина). Base == Extent —
// просто курсор. Extent — тот край, где курсор (его двигают стрелки и Shift).
type TextSelection struct{ Base, Extent int }

// Collapsed — курсор в позиции i без выделения.
func Collapsed(i int) TextSelection { return TextSelection{Base: i, Extent: i} }

// Start — левый край выделения.
func (s TextSelection) Start() int { return min(s.Base, s.Extent) }

// End — правый край выделения.
func (s TextSelection) End() int { return max(s.Base, s.Extent) }

// IsCollapsed — выделения нет, только курсор.
func (s TextSelection) IsCollapsed() bool { return s.Base == s.Extent }

// TextRange — часть текста [Start, End) в рунах; Start == End — пусто.
type TextRange struct{ Start, End int }

// TextEditingValue — текст поля, выделение в нём и слово, которое сейчас набирает клавиатура
// (Composing: подчёркнуто, клавиатура может его заменить — подсказкой или исправлением).
type TextEditingValue struct {
	Text      string
	Selection TextSelection
	Composing TextRange
}

// clamped — значение с выделением и набираемым словом в пределах текста.
func (v TextEditingValue) clamped() TextEditingValue {
	n := utf8.RuneCountInString(v.Text)
	c := func(i int) int { return max(0, min(i, n)) }
	v.Selection = TextSelection{Base: c(v.Selection.Base), Extent: c(v.Selection.Extent)}
	v.Composing = TextRange{Start: c(v.Composing.Start), End: c(v.Composing.End)}
	if v.Composing.End < v.Composing.Start {
		v.Composing = TextRange{}
	}
	return v
}

// TextEditingController — текст поля ввода в руках приложения: прочитать, задать, очистить,
// выделить. Поле с этим контроллером перестраивается само, когда текст меняется.
// Нулевое значение готово к работе: &widgets.TextEditingController{}.
//
//	c.name = widgets.NewTextController("Анна")
//	material.TextField{Controller: c.name, Label: "Имя"}
//	...
//	fmt.Println(c.name.Text())
type TextEditingController struct {
	value     TextEditingValue
	listeners []*func()
}

// NewTextController — контроллер с текстом s и курсором в конце.
func NewTextController(s string) *TextEditingController {
	return &TextEditingController{value: TextEditingValue{Text: s, Selection: Collapsed(utf8.RuneCountInString(s))}}
}

// Text — текст поля.
func (c *TextEditingController) Text() string { return c.value.Text }

// SetText заменяет текст; курсор — в конец.
func (c *TextEditingController) SetText(s string) {
	c.SetValue(TextEditingValue{Text: s, Selection: Collapsed(utf8.RuneCountInString(s))})
}

// Clear очищает поле.
func (c *TextEditingController) Clear() { c.SetText("") }

// Value — текст с выделением и набираемым словом.
func (c *TextEditingController) Value() TextEditingValue { return c.value }

// SetValue заменяет всё сразу (выделение обрезается по длине текста).
func (c *TextEditingController) SetValue(v TextEditingValue) {
	v = v.clamped()
	if v == c.value {
		return
	}
	c.value = v
	for _, f := range slices.Clone(c.listeners) {
		(*f)()
	}
}

// Selection — выделение (или позиция курсора).
func (c *TextEditingController) Selection() TextSelection { return c.value.Selection }

// SetSelection ставит курсор или выделение.
func (c *TextEditingController) SetSelection(s TextSelection) {
	v := c.value
	v.Selection = s
	c.SetValue(v)
}

// AddListener — f вызывается при каждом изменении текста или выделения. Возвращает отписку.
func (c *TextEditingController) AddListener(f func()) (remove func()) {
	p := &f
	c.listeners = append(c.listeners, p)
	return func() {
		if i := slices.Index(c.listeners, p); i >= 0 {
			c.listeners = slices.Delete(c.listeners, i, i+1)
		}
	}
}

// KeyboardType — какая клавиатура нужна полю.
type KeyboardType uint8

const (
	KeyboardText    KeyboardType = iota // обычный текст
	KeyboardNumber                      // цифры
	KeyboardDecimal                     // число с дробной частью
	KeyboardPhone                       // телефон
	KeyboardEmail                       // адрес почты
	KeyboardURL                         // адрес сайта
)

// TextInputAction — что делает кнопка ввода на клавиатуре.
type TextInputAction uint8

const (
	ActionDefault TextInputAction = iota // однострочное поле — Done, многострочное — новая строка
	ActionDone                           // «Готово»: поле теряет фокус, клавиатура прячется
	ActionGo                             // «Перейти»
	ActionNext                           // «Далее»: фокус — следующему полю на экране
	ActionSearch                         // «Найти»
	ActionSend                           // «Отправить»
	ActionNewline                        // новая строка (многострочное поле)
)

// Capitalization — какие буквы клавиатура предлагает заглавными.
type Capitalization uint8

const (
	CapitalizeNone       Capitalization = iota
	CapitalizeSentences                 // первую букву предложения
	CapitalizeWords                     // первую букву слова
	CapitalizeCharacters                // все
)

// TextInputConfig — какая клавиатура нужна полю.
type TextInputConfig struct {
	Keyboard      KeyboardType
	Action        TextInputAction
	Multiline     bool
	Obscure       bool // пароль: клавиатура не запоминает и не подсказывает
	NoAutocorrect bool
	Capitalize    Capitalization
}

// TextInput — связь с клавиатурой платформы. Её задаёт app.Run (на Android — экранная клавиатура
// и физическая); в тестах — заглушка, через которую «печатают» Binding.HandleTextEditing.
type TextInput interface {
	// Show подключает клавиатуру к полю и показывает её (снова — если её спрятали).
	Show(cfg TextInputConfig, v TextEditingValue)
	// Update сообщает клавиатуре текст поля, когда его поменяли не с неё (контроллером).
	Update(v TextEditingValue)
	// Hide прячет клавиатуру: ни одно поле не в фокусе.
	Hide()
}

// textInputClient — поле, к которому сейчас подключена клавиатура.
type textInputClient interface {
	editingFromPlatform(v TextEditingValue)
	actionFromPlatform(a TextInputAction)
}

// SetTextInput задаёт связь с клавиатурой (её вызывает app.Run).
func (b *Binding) SetTextInput(t TextInput) { b.owner.textInput = t }

// HandleTextEditing — клавиатура изменила текст поля (набор, подсказка, удаление, выделение).
func (b *Binding) HandleTextEditing(v TextEditingValue) {
	defer b.recoverEvent("ввода текста")
	if c := b.owner.textClient; c != nil {
		c.editingFromPlatform(v.clamped())
	}
}

// HandleTextAction — нажата кнопка ввода клавиатуры («Готово», «Далее», «Отправить»…).
func (b *Binding) HandleTextAction(a TextInputAction) {
	defer b.recoverEvent("кнопки клавиатуры")
	if c := b.owner.textClient; c != nil {
		c.actionFromPlatform(a)
	}
}
