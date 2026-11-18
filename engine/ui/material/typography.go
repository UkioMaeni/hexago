package material

import (
	"hexago/ui/gfx"
	"hexago/ui/text"
)

// TextTheme — шкала текста Material 3: пять ролей по три размера.
// Display — крупные короткие надписи; Headline — заголовки экранов; Title — заголовки блоков;
// Body — основной текст; Label — подписи кнопок, вкладок, полей.
type TextTheme struct {
	DisplayLarge, DisplayMedium, DisplaySmall    text.Style
	HeadlineLarge, HeadlineMedium, HeadlineSmall text.Style
	TitleLarge, TitleMedium, TitleSmall          text.Style
	BodyLarge, BodyMedium, BodySmall             text.Style
	LabelLarge, LabelMedium, LabelSmall          text.Style
}

// NewTextTheme — шкала Material 3 (размер / высота строки / насыщенность / разрядка — из спецификации),
// весь текст цвета color (обычно ColorScheme.OnSurface).
func NewTextTheme(color gfx.Color) TextTheme {
	s := func(size, line float64, w text.Weight, spacing float64) text.Style {
		return text.Style{FontSize: size, LineHeight: line, FontWeight: w, LetterSpacing: spacing, Color: color}
	}
	return TextTheme{
		DisplayLarge:  s(57, 64, text.Regular, -0.25),
		DisplayMedium: s(45, 52, text.Regular, 0),
		DisplaySmall:  s(36, 44, text.Regular, 0),

		HeadlineLarge:  s(32, 40, text.Regular, 0),
		HeadlineMedium: s(28, 36, text.Regular, 0),
		HeadlineSmall:  s(24, 32, text.Regular, 0),

		TitleLarge:  s(22, 28, text.Regular, 0),
		TitleMedium: s(16, 24, text.Medium, 0.15),
		TitleSmall:  s(14, 20, text.Medium, 0.1),

		BodyLarge:  s(16, 24, text.Regular, 0.5),
		BodyMedium: s(14, 20, text.Regular, 0.25),
		BodySmall:  s(12, 16, text.Regular, 0.4),

		LabelLarge:  s(14, 20, text.Medium, 0.1),
		LabelMedium: s(12, 16, text.Medium, 0.5),
		LabelSmall:  s(11, 16, text.Medium, 0.5),
	}
}
