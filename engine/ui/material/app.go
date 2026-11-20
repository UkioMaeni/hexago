package material

import (
	"time"

	"hexago/ui/animation"
	"hexago/ui/geom"
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
// фон экрана цвета Surface, стиль текста по умолчанию, значки системных панелей под цвет темы,
// Navigator с переходами экранов Material (PageTransition).
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
	// Тема — над Navigator: её видят все экраны и окна, а не только Home.
	return widgets.SystemBars{
		LightIcons: t.Brightness() == gfx.Dark,
		Child: themed(t, widgets.App{
			Title:          a.Title,
			Color:          t.Colors.Surface,
			Home:           a.Home,
			PageTransition: PageTransition,
		}),
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

// PageTransition — переход экранов Material: новый экран въезжает справа на четверть ширины
// и проявляется, прежний отъезжает влево на четверть; при возврате — наоборот. Его ставит App.
var PageTransition = widgets.Transition{
	Duration:        350 * time.Millisecond,
	ReverseDuration: 300 * time.Millisecond,
	Curve:           animation.Standard,
	Build: func(anim, secondary animation.Animation, child widgets.Widget) widgets.Widget {
		return widgets.SlideTransition{Position: secondary, End: geom.Offset{X: -0.25},
			Child: widgets.SlideTransition{Position: anim, Begin: geom.Offset{X: 0.25},
				Child: widgets.FadeTransition{Opacity: animation.Curved(anim, animation.Interval{End: 0.3}), Child: child}}}
	},
}
