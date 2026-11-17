package widgets

type Base struct{}     // общие поля всех виджетов: потом сюда ляжет, например, Key
func (Base) isWidget() {}

type Widget interface {
	isWidget()
}