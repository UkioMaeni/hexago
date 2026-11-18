# ADR-0017: Каждый тип — из своего пакета; в widgets только виджеты

- Статус: принято
- Дата: 2026-11-18

## Контекст

- В `ui/widgets/types.go` были псевдонимы соседних пакетов (`widgets.Color = gfx.Color`,
  `widgets.All = geom.All`, `widgets.TextStyle = text.Style`, `widgets.MainAxisCenter = rendering.MainAxisCenter`…),
  чтобы приложению хватало одного импорта, как во Flutter.
- Автор считает это недопустимым: цвета и размеры сделаны в `gfx` и `geom`, а в коде приложения они выглядели
  как часть виджетов. У одного типа появлялось два имени, и неясно, где он на самом деле живёт.

## Решение

1. Псевдонимов нет. Каждый тип и каждое значение пишется через пакет, где оно определено:
   - `gfx` — цвета: `gfx.Blue`, `gfx.ColorFromHex("#2196F3")`;
   - `geom` — отступы, размеры, скругления, выравнивания: `geom.All(16)`, `geom.RadiusAll(12)`, `geom.Center`;
   - `painting` — украшения: `painting.BoxDecoration`, `painting.BorderAll`, `painting.BoxShadow`, `painting.Circle`;
   - `text` — стиль текста: `text.Style{FontSize: 20, FontWeight: text.Bold}`, `text.AlignCenter`;
   - `rendering` — настройки раскладки: `rendering.MainAxisCenter`, `rendering.CrossAxisStretch`, `rendering.Constraints`;
   - `widgets` — только виджеты и то, что нужно для своих виджетов (`Base`, `BuildContext`…).
2. Шаблон проекта, тестовый проект и тесты переписаны на эти импорты.
3. Действия с виджетами в VS Code («Обернуть в Padding / Align / ClipRRect») вставляют `geom.All(8)`,
   `geom.Center`, `geom.RadiusAll(8)` и сами добавляют импорт `hexago/ui/geom`, если его нет:
   у правки `hexago assist` появилось поле `before` — правка импорта, применяемая раньше основной.

## Альтернативы

- Оставить псевдонимы (как во Flutter, один импорт): короче, но у типа два имени.
- Убрать только цвета и размеры, а украшения, стиль текста и раскладку оставить в `widgets`: получилась бы
  смесь правил. Автор выбрал единое правило.

## Последствия

- У приложения больше импортов (до шести). Взамен по имени всегда видно, откуда тип.
- Уже созданные проекты нужно перевести на новые импорты: `widgets.TextStyle` → `text.Style`,
  `widgets.FontWeightBold` → `text.Bold` и т. п.
