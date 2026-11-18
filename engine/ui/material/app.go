package material

import (
	"hexago/ui/gfx"
	"hexago/ui/widgets"
)

// ThemeMode — какую тему показывать.
type ThemeMode uint8

const (
	ThemeSystem ThemeMode = iota // как в настройках телефона (по умолчанию)
	ThemeLight                   // всегда светлую
	ThemeDark                    // всегда тёмную
)

// App — корень приложения в стиле Material 3, как MaterialApp во Flutter: тема (светлая и тёмная),
// фон экрана цвета Surface, стиль текста по умолчанию, значки системных панелей под цвет темы.
//
//	material.App{Title: "Заметки", Home: Notes{}}
//
// Theme и DarkTheme — свои варианты темы; nil — стандартные Light() и Dark().
type App struct {
	widgets.Base
	Title     string
	Theme     *ThemeData
	DarkTheme *ThemeData
	Mode      ThemeMode
	Home      widgets.Widget
}

func (a App) Build(ctx widgets.BuildContext) widgets.Widget {
	t := a.theme(widgets.MediaQueryOf(ctx).PlatformBrightness)
	return widgets.App{
		Title: a.Title,
		Color: t.Colors.Surface,
		Home: widgets.SystemBars{
			LightIcons: t.Brightness() == gfx.Dark,
			Child:      themed(t, a.Home),
		},
	}
}

// theme выбирает тему по режиму и теме системы.
func (a App) theme(platform gfx.Brightness) ThemeData {
	dark := a.Mode == ThemeDark || (a.Mode == ThemeSystem && platform == gfx.Dark)
	switch {
	case dark && a.DarkTheme != nil:
		return *a.DarkTheme
	case dark:
		return Dark()
	case a.Theme != nil:
		return *a.Theme
	}
	return Light()
}
