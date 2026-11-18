// Package material — Material Design 3 на Hexago: тема (цвета, текст, скругления, высоты)
// и компоненты, которые её используют.
//
// Значения по умолчанию — «родные» Material 3: базовая схема цветов (baseline, фиолетовый #6750A4),
// шкала текста и скругления из спецификации. Свой вариант темы — это та же ThemeData с изменёнными полями:
//
//	t := material.Light()
//	t.Colors.Primary = gfx.ColorFromHex("#006A6A")
//	material.App{Theme: &t, Home: ...}
package material

import "hexago/ui/gfx"

// ColorScheme — цветовые роли Material 3. У каждого цвета фона есть парный цвет содержимого
// (On…): им рисуют текст и значки поверх этого фона.
type ColorScheme struct {
	Brightness gfx.Brightness

	// Главный акцент: заметные кнопки, активные элементы.
	Primary, OnPrimary                   gfx.Color
	PrimaryContainer, OnPrimaryContainer gfx.Color
	InversePrimary                       gfx.Color // акцент на InverseSurface (например, в SnackBar)

	// Второстепенный акцент: менее заметные элементы, фильтры.
	Secondary, OnSecondary                   gfx.Color
	SecondaryContainer, OnSecondaryContainer gfx.Color

	// Третий акцент: противовес первым двум, выделение.
	Tertiary, OnTertiary                   gfx.Color
	TertiaryContainer, OnTertiaryContainer gfx.Color

	// Ошибки.
	Error, OnError                   gfx.Color
	ErrorContainer, OnErrorContainer gfx.Color

	// Поверхности: фон экрана и слои над ним — от самого низкого контейнера до самого высокого.
	Surface, OnSurface, OnSurfaceVariant gfx.Color
	SurfaceVariant                       gfx.Color
	SurfaceDim, SurfaceBright            gfx.Color
	SurfaceContainerLowest               gfx.Color
	SurfaceContainerLow                  gfx.Color
	SurfaceContainer                     gfx.Color
	SurfaceContainerHigh                 gfx.Color
	SurfaceContainerHighest              gfx.Color
	InverseSurface, OnInverseSurface     gfx.Color
	SurfaceTint                          gfx.Color

	// Обводки, тени, затемнение под диалогами.
	Outline, OutlineVariant gfx.Color
	Shadow, Scrim           gfx.Color
}

// Значения базовой схемы — токены md.sys.color из Material Web 2.4.1 (набор v0_192).
var hex = gfx.ColorFromHex

// LightColors — базовая светлая схема Material 3.
func LightColors() ColorScheme {
	return ColorScheme{
		Brightness: gfx.Light,

		Primary: hex("#6750A4"), OnPrimary: hex("#FFFFFF"),
		PrimaryContainer: hex("#EADDFF"), OnPrimaryContainer: hex("#21005D"),
		InversePrimary: hex("#D0BCFF"),

		Secondary: hex("#625B71"), OnSecondary: hex("#FFFFFF"),
		SecondaryContainer: hex("#E8DEF8"), OnSecondaryContainer: hex("#1D192B"),

		Tertiary: hex("#7D5260"), OnTertiary: hex("#FFFFFF"),
		TertiaryContainer: hex("#FFD8E4"), OnTertiaryContainer: hex("#31111D"),

		Error: hex("#B3261E"), OnError: hex("#FFFFFF"),
		ErrorContainer: hex("#F9DEDC"), OnErrorContainer: hex("#410E0B"),

		Surface: hex("#FEF7FF"), OnSurface: hex("#1D1B20"), OnSurfaceVariant: hex("#49454F"),
		SurfaceVariant: hex("#E7E0EC"),
		SurfaceDim:     hex("#DED8E1"), SurfaceBright: hex("#FEF7FF"),
		SurfaceContainerLowest:  hex("#FFFFFF"),
		SurfaceContainerLow:     hex("#F7F2FA"),
		SurfaceContainer:        hex("#F3EDF7"),
		SurfaceContainerHigh:    hex("#ECE6F0"),
		SurfaceContainerHighest: hex("#E6E0E9"),
		InverseSurface:          hex("#322F35"), OnInverseSurface: hex("#F5EFF7"),
		SurfaceTint: hex("#6750A4"),

		Outline: hex("#79747E"), OutlineVariant: hex("#CAC4D0"),
		Shadow: hex("#000000"), Scrim: hex("#000000"),
	}
}

// DarkColors — базовая тёмная схема Material 3.
func DarkColors() ColorScheme {
	return ColorScheme{
		Brightness: gfx.Dark,

		Primary: hex("#D0BCFF"), OnPrimary: hex("#381E72"),
		PrimaryContainer: hex("#4F378B"), OnPrimaryContainer: hex("#EADDFF"),
		InversePrimary: hex("#6750A4"),

		Secondary: hex("#CCC2DC"), OnSecondary: hex("#332D41"),
		SecondaryContainer: hex("#4A4458"), OnSecondaryContainer: hex("#E8DEF8"),

		Tertiary: hex("#EFB8C8"), OnTertiary: hex("#492532"),
		TertiaryContainer: hex("#633B48"), OnTertiaryContainer: hex("#FFD8E4"),

		Error: hex("#F2B8B5"), OnError: hex("#601410"),
		ErrorContainer: hex("#8C1D18"), OnErrorContainer: hex("#F9DEDC"),

		Surface: hex("#141218"), OnSurface: hex("#E6E0E9"), OnSurfaceVariant: hex("#CAC4D0"),
		SurfaceVariant: hex("#49454F"),
		SurfaceDim:     hex("#141218"), SurfaceBright: hex("#3B383E"),
		SurfaceContainerLowest:  hex("#0F0D13"),
		SurfaceContainerLow:     hex("#1D1B20"),
		SurfaceContainer:        hex("#211F26"),
		SurfaceContainerHigh:    hex("#2B2930"),
		SurfaceContainerHighest: hex("#36343B"),
		InverseSurface:          hex("#E6E0E9"), OnInverseSurface: hex("#322F35"),
		SurfaceTint: hex("#D0BCFF"),

		Outline: hex("#938F99"), OutlineVariant: hex("#49454F"),
		Shadow: hex("#000000"), Scrim: hex("#000000"),
	}
}
