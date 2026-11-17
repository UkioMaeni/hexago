package gfx

import (
	"fmt"
	"strconv"
	"strings"
)

// Color — цвет RGBA, каждый канал от 0 до 255. A — непрозрачность: 0 — прозрачный, 255 — сплошной.
// Нулевое значение Color{} — полностью прозрачный цвет, то есть «цвета нет».
type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

// RGB — непрозрачный цвет.
func RGB(r, g, b uint8) Color { return Color{r, g, b, 255} }

// RGBA — цвет с прозрачностью.
func RGBA(r, g, b, a uint8) Color { return Color{r, g, b, a} }

// ARGB — цвет из числа 0xAARRGGBB, как Color(0xFF2196F3) во Flutter.
func ARGB(v uint32) Color {
	return Color{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: uint8(v >> 24)}
}

// ColorFromHex разбирает цвет из строки: "#RGB", "#RRGGBB" или "#AARRGGBB" (решётка не обязательна).
// Неверная строка — ошибка программиста, поэтому паника с понятным текстом.
func ColorFromHex(hex string) Color {
	s := strings.TrimPrefix(hex, "#")
	if len(s) == 3 { // #RGB → #RRGGBB
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 && len(s) != 8 {
		panic(fmt.Sprintf("gfx.ColorFromHex(%q): нужно 3, 6 или 8 шестнадцатеричных цифр, например \"#FF8800\"", hex))
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		panic(fmt.Sprintf("gfx.ColorFromHex(%q): это не шестнадцатеричное число", hex))
	}
	if len(s) == 6 {
		v |= 0xFF000000
	}
	return ARGB(uint32(v))
}

// WithAlpha возвращает тот же цвет с другой непрозрачностью (0–255).
func (c Color) WithAlpha(a uint8) Color { c.A = a; return c }

// WithOpacity возвращает тот же цвет с непрозрачностью от 0 до 1.
func (c Color) WithOpacity(o float64) Color { return c.WithAlpha(uint8(clamp01(o)*255 + 0.5)) }

// Opacity — непрозрачность от 0 до 1.
func (c Color) Opacity() float64 { return float64(c.A) / 255 }

// IsTransparent сообщает, что цвет полностью прозрачный.
func (c Color) IsTransparent() bool { return c.A == 0 }

// Pixel — цвет в формате пикселя буфера: uint32 0xAABBGGRR (в памяти байты R, G, B, A).
func (c Color) Pixel() uint32 {
	return uint32(c.A)<<24 | uint32(c.B)<<16 | uint32(c.G)<<8 | uint32(c.R)
}

// FromPixel — обратное к Pixel.
func FromPixel(p uint32) Color {
	return Color{R: uint8(p), G: uint8(p >> 8), B: uint8(p >> 16), A: uint8(p >> 24)}
}

// LerpColor — смешивание цветов: t=0 → a, t=1 → b.
func LerpColor(a, b Color, t float64) Color {
	t = clamp01(t)
	mix := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return Color{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), mix(a.A, b.A)}
}

func (c Color) String() string { return fmt.Sprintf("#%02X%02X%02X%02X", c.A, c.R, c.G, c.B) }

// Основные цвета.
var (
	Transparent = Color{}
	Black       = RGB(0, 0, 0)
	White       = RGB(255, 255, 255)
	Red         = ARGB(0xFFF44336)
	Pink        = ARGB(0xFFE91E63)
	Purple      = ARGB(0xFF9C27B0)
	Indigo      = ARGB(0xFF3F51B5)
	Blue        = ARGB(0xFF2196F3)
	Cyan        = ARGB(0xFF00BCD4)
	Teal        = ARGB(0xFF009688)
	Green       = ARGB(0xFF4CAF50)
	Lime        = ARGB(0xFFCDDC39)
	Yellow      = ARGB(0xFFFFEB3B)
	Amber       = ARGB(0xFFFFC107)
	Orange      = ARGB(0xFFFF9800)
	Brown       = ARGB(0xFF795548)
	Grey        = ARGB(0xFF9E9E9E)
)

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	}
	return v
}
