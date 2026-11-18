# ADR-0020: Тема Material 3 — пакет ui/material; стиль текста по умолчанию; тёмная тема системы

- Статус: принято
- Дата: 2026-11-18

## Контекст

- Автор хочет стандартные темы Material 3 для всех компонентов: «родные» значения, а поверх них — свои варианты.
- Шкала текста Material 3 использует насыщенность 500, высоту строки и разрядку; в `text.Style` их не было.
- Тёмная тема бессмысленна без знания темы системы и без светлых значков строки состояния.

## Решение

1. **Пакет `ui/material`** над `widgets`: `geom → gfx → … → widgets → material`.
   - `ColorScheme` — все роли Material 3 (Primary/OnPrimary, …Container, Secondary, Tertiary, Error, Surface и
     пять SurfaceContainer, Inverse…, Outline, Shadow, Scrim, SurfaceTint). `LightColors()` и `DarkColors()` —
     базовая схема (#6750A4) из токенов Material Web 2.4.1, набор v0_192.
   - `TextTheme` — 15 ролей (Display, Headline, Title, Body, Label × Large/Medium/Small) с размерами, высотой строки,
     насыщенностью и разрядкой из спецификации.
   - `Shapes` (0, 4, 8, 12, 16, 28, Full) и `Elevation` (0, 1, 3, 6, 8, 12 dp).
   - `ThemeData{Colors, Text, Shapes, Elevation}`, `Light()`, `Dark()`, `FromColors(scheme)`; свой вариант —
     та же структура с изменёнными полями.
   - `Theme` — наследуемая тема для поддерева, `material.Of(ctx)`; без неё — `Light()`.
   - `material.App{Title, Theme, DarkTheme, Mode, Home}`: режим `ThemeSystem` (по умолчанию), `ThemeLight`,
     `ThemeDark`; фон — `Surface`, стиль текста по умолчанию — `BodyMedium` цвета `OnSurface`, значки панелей —
     под яркость темы.
2. **`text.Style`:** насыщенность `Medium` (шрифт Go Medium из того же golang.org/x/image), `LineHeight` в dp
   (лишнее место делится поровну над и под строкой), `LetterSpacing` в dp. Нулевое поле — «не задано»:
   `Weight` теперь начинается с 1 (`Regular`), а `Style.Merge(base)` дополняет незаданные поля.
3. **`widgets.DefaultTextStyle`** — стиль текста поддереву; `Text` дополняет им свой стиль. Когда наследуемые
   данные меняются, боксы зависящих от них виджетов обновляются (`UpdateRenderObject`).
4. **Тема системы:** `MediaQueryData.PlatformBrightness` (`gfx.Brightness`: Light, Dark). Android передаёт её
   при каждом кадре, а при смене темы в настройках перерисовывает экран (Activity не пересоздаётся).
5. **`widgets.SystemBars{LightIcons}`** — декларативно задаёт цвет значков строки состояния и панели навигации
   (как AnnotatedRegion во Flutter); действует самый внешний. `Binding.SetSystemBars` передаёт его платформе.

## Альтернативы

- Генерировать схему из одного цвета (ColorScheme.fromSeed, алгоритм HCT из material-color-utilities):
  удобно для своих вариантов, но это порт отдельной библиотеки. Отложено — пока свой вариант задают полями.
- Шрифт Roboto, как в Android: точнее совпадает с Material, но это новый файл шрифта в движке (отдельное решение,
  вместе с go-text, ADR-0006). Пока — шрифты Go с тремя насыщенностями.

## Последствия

- Компоненты Material (кнопки, карточки, поля) берут цвета, текст и скругления из `material.Of(ctx)`.
- Уже написанный код с `text.Style{FontWeight: text.Regular}` работает как раньше; нулевая насыщенность тоже
  означает Regular, если стиль по умолчанию её не задал.
