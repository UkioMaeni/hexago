package widgets

// Clipboard — буфер обмена платформы. Его задаёт app.Run (на Android — системный буфер); без него —
// буфер в памяти приложения (так работают тесты).
type Clipboard interface {
	SetText(s string)
	Text() (string, bool)
	HasText() bool // есть ли текст — не читая его (Android сообщает о чтении буфера пользователю)
}

// SetClipboard задаёт буфер обмена (его вызывает app.Run).
func (b *Binding) SetClipboard(c Clipboard) { b.owner.clipboard = c }

// CopyText кладёт текст в буфер обмена.
func CopyText(ctx BuildContext, s string) { clipboardOf(ctx).SetText(s) }

// PasteText — текст из буфера обмена; false — там нет текста.
func PasteText(ctx BuildContext) (string, bool) { return clipboardOf(ctx).Text() }

func clipboardOf(ctx BuildContext) Clipboard {
	o := ctx.element().owner
	if o.clipboard == nil {
		o.clipboard = &memoryClipboard{}
	}
	return o.clipboard
}

// memoryClipboard — буфер обмена в памяти.
type memoryClipboard struct {
	text string
	has  bool
}

func (m *memoryClipboard) SetText(s string)     { m.text, m.has = s, true }
func (m *memoryClipboard) Text() (string, bool) { return m.text, m.has }
func (m *memoryClipboard) HasText() bool        { return m.has && m.text != "" }
