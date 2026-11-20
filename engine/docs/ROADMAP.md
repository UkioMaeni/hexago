# Дорожная карта Hexago

Курс — [ADR-0013](adr/0013-flutter-widgets-course.md): виджеты Flutter на нашем движке, с отрисовкой на Android. Каждый этап заканчивается
проверкой: тесты, PNG-эталоны и скриншот эмулятора. После каждого этапа — коммит.

Статусы: ✅ готово · 🔨 в работе · ⏳ впереди.

## Сделано до нового курса ✅

- Архитектура, ADR, дорожная карта.
- `hexago create`: проект (Go + Android-хост с обёрткой Gradle), связь с движком через `replace`.
- `hexago run`: пока `go run .`.
- Android-мост: `HexagoActivity`, `platform/android` (JNI, `ANativeWindow`), `app.Run` вызывает `main`
  приложения, Go заливает экран, плотность экрана приходит в Go.

## P1 — Утилита `hexago` ✅

- `hexago version` — версии hexago, Go и путь к движку.
- `hexago test` — проверка окружения, как `flutter doctor`: Go, JDK, Android SDK, NDK, adb, устройства, движок.
- `hexago run android` — сборка Go в `libhexago.so` под архитектуру устройства, Gradle, установка, запуск,
  вывод лога приложения в терминал (`fmt.Println`, паники со стеком вызовов).

## P2 — Рисование ✅

`ui/geom` (точки, размеры, прямоугольники, отступы, выравнивание), `ui/gfx` (цвет, программный холст:
прямоугольники, скругления со сглаживанием, рамки, тени, градиенты, клипы, прозрачность, сдвиги),
PNG-эталоны для проверки без устройства.

## P3 — Ядро фреймворка ✅

Widget → Element → RenderObject: `StatelessWidget`, состояние в структуре виджета (`widgets.State`, `SetState`, Init, Update, Dispose — ADR-0018), `BuildContext`,
наследуемые данные (`DependOn`), ключи, render-дерево с `BoxConstraints`, лэйаут, отрисовка, hit-test.

## P4 — Виджеты раскладки 🔨

Готово: `Container`, `Padding`, `Center`, `Align`, `SizedBox`, `ConstrainedBox`, `LimitedBox`, `AspectRatio`,
`FractionallySizedBox`, `Row`, `Column`, `Flex`, `Expanded`, `Flexible`, `Spacer`, `Stack`, `Positioned`,
`Wrap`, `ColoredBox`, `DecoratedBox`, `Opacity`, `ClipRect`, `ClipRRect`, `ClipOval`, `Transform`
(`Rotate`, `Scale`, `Translate`), `Offstage`, `Visibility`, `Builder`, `CustomPaint`, `Placeholder`,
`SafeArea`, `MediaQuery`, `Keyed`, `ErrorBox` (паника в Build — красный бокс, ошибка в лог).

Впереди: `LayoutBuilder`, полосы переполнения Row/Column, ошибка лэйаута на месте виджета (сейчас — красный кадр).

## P4.5 — Расширение для VS Code ✅

- Поиск SDK: если `hexago` не найден — спросить папку SDK и либо добавить её в PATH,
  либо записать путь в настройки проекта (`.vscode/settings.json`).
- Действия с виджетами, как во Flutter (`Ctrl+.` / лампочка): «Обернуть в виджет…»,
  «Обернуть в Center / Padding / Column / Row / Container / Expanded / SizedBox», «Удалить виджет»,
  «Поменять местами с ребёнком». Правки вычисляет `hexago` по синтаксическому дереву Go,
  расширение только показывает меню и применяет их.
- Команды: создать проект, запустить на устройстве, проверить окружение.
- Код: `engine/tools/vscode` (TypeScript), правки — `hexago assist` (`engine/tools/assist`).
  Сборка: `npm install`, `npm run compile`, `npm run package` → `hexago.vsix`.
- Выбор устройства, как у Flutter ([ADR-0014](adr/0014-devices-and-emulators.md)): справа в строке
  состояния — где запустится приложение; по щелчку — телефоны, запущенные эмуляторы и «Запустить …» для
  выключенных. Команды `hexago devices [--json] [--watch]`, `hexago emulators [--launch имя]`;
  `hexago run -d` принимает серийный номер, имя устройства или эмулятора, выключенный эмулятор запускает сам.
  Проект замечается и после `hexago create .` в уже открытой папке.
- Run → Start Debugging (F5), как у Flutter ([ADR-0016](adr/0016-run-from-vscode-debugger.md)): сборка
  и запуск на выбранном устройстве, лог в Debug Console, ■ и ⟳ на панели отладки; `hexago create`
  кладёт `.vscode/launch.json`. Точек останова пока нет — появятся с запуском на компьютере.

## P5 — Текст и иконки 🔨

Готово (первая версия): `Text` и `text.Style` (размер, цвет, насыщенность Regular/Medium/Bold, высота строки,
разрядка), `DefaultTextStyle`, перенос по словам, выравнивание строк, шрифт Go из golang.org/x/image
(латиница и кириллица); текст ошибки в `ErrorBox`; шаблон проекта — стартовый экран `HomePage` с надписью «hexago».

Впереди: go-text (шейпинг, сложные письменности, фолбэк шрифтов), `RichText`,
`maxLines` и многоточие, `Icon` (шрифт Material Icons).

## P6 — Ввод, кадры, анимации, скролл ✅

Готово ([ADR-0019](adr/0019-touch-input-and-gestures.md)): касания с Android, кадры по запросу (Choreographer),
`app.Post`, пакет `ui/gestures` (арена, нажатие, двойное и долгое нажатие, перетаскивание по осям и свободное,
масштаб с поворотом, скорость броска), `Listener`, `GestureDetector`, `IgnorePointer`, `AbsorbPointer`.

Готово ([ADR-0022](adr/0022-animations-and-scrolling.md)): пакет `ui/animation` (кривые, Controller, физика),
время кадра от развёртки, `NewAnimation`, неявные анимации (`AnimatedValue`, `AnimatedOpacity`, `AnimatedContainer`…),
`SingleChildScrollView`, `ListView` (ленивый по высоте строки), `GridView`, `PageView`, бросок с трением.

Готово ([ADR-0023](adr/0023-fast-software-rendering.md)): быстрая отрисовка — быстрые пути холста, кеши теней
и текста, перерисовка только изменившейся части экрана. Полный кадр 41 → 8,5 мс, на эмуляторе 57–61 кадр
в секунду вместо 20.

Готово ([ADR-0024](adr/0024-navigator-and-dialogs.md)): навигация — `Navigator` (его ставит `App`), `Push`, `Pop`,
`PopWith`, `PushRoute`, `PushReplacement`, `PopToFirst`, `CanPop`, окна поверх экрана с затемнением, кнопка «Назад»
Android (на Android 13+ — `OnBackInvokedCallback`), `FadeTransition`, `SlideTransition`, переходы экранов снимком
(кадр перехода 16 → 2,5 мс).

Дальше: полоса прокрутки, растяжение у края, ленивые строки разной высоты, `Hero`, запрет «Назад» (`PopScope`),
жест «Назад» с предпросмотром, `Overlay`.

## P7 — Material 🔨

Готово ([ADR-0020](adr/0020-material3-theme.md)): тема Material 3 — `material.ColorScheme` (базовые светлая
и тёмная), `TextTheme`, `Shapes`, `Elevation`, `Theme`, `material.Of`, `material.App` с режимом темы
(как в системе / светлая / тёмная), `SystemBars`, тема системы в `MediaQuery`.

Готово ([ADR-0021](adr/0021-material-components.md)): значки (пути SVG, `ui/icons` — 50 значков Material, `Icon`,
`IconTheme`), `Material`, `InkWell` (подложка нажатия), кнопки `FilledButton`, `FilledTonalButton`, `ElevatedButton`,
`OutlinedButton`, `TextButton`, `IconButton`, `FloatingActionButton`, `Card`, `Divider`, `ListTile`, `Checkbox`,
`Radio`, `Switch`, `Slider`, индикаторы прогресса (с Value), `Scaffold`, `AppBar`, `NavigationBar`.

Волна нажатия, плавный `Switch`, бесконечные индикаторы — готово (ADR-0022).

Готово (ADR-0024): переход экранов Material, `AlertDialog`, `Dialog`, `ShowDialog`, нижняя панель
(`ShowBottomSheet`, тянется пальцем), стрелка «Назад» в `AppBar`.

Готово ([ADR-0025](adr/0025-snackbar-drawer-menus.md)): `SnackBar` (очередь в `App`, плавающая кнопка поднимается над
сообщением), боковая панель `Drawer` и `NavigationDrawer` (кнопка «меню» в `AppBar`, выдвигание от края), меню
`MenuButton`, `ShowMenu`, `MenuItem`; `widgets.AfterFunc`, `widgets.BoundsOf`.

Впереди: схема из одного цвета (fromSeed), чипы, вкладки, смахивание `SnackBar`, `Overlay` и подсказки.

## P8 — Ввод текста и изображения ⏳

`TextField` (клавиатура, IME, выделение), `Image` (из файлов приложения и из сети).

## Дальше

iOS, Windows/Web, GPU-растеризация, доступность, hot restart.
