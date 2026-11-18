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

Готово (первая версия): `Text` и `TextStyle` (размер, цвет, жирность), перенос по словам, выравнивание строк,
шрифт Go из golang.org/x/image (латиница и кириллица); текст ошибки в `ErrorBox`; шаблон проекта —
стартовый экран `HomePage` с надписью «hexago».

Впереди: go-text (шейпинг, сложные письменности, фолбэк шрифтов), `RichText`, `DefaultTextStyle`,
`maxLines` и многоточие, `Icon` (шрифт Material Icons).

## P6 — Ввод, кадры, анимации, скролл 🔨

Готово ([ADR-0019](adr/0019-touch-input-and-gestures.md)): касания с Android, кадры по запросу (Choreographer),
`app.Post`, пакет `ui/gestures` (арена, нажатие, двойное и долгое нажатие, перетаскивание по осям и свободное,
масштаб с поворотом, скорость броска), `Listener`, `GestureDetector`, `IgnorePointer`, `AbsorbPointer`.

Впереди: `AnimationController`, `Tween`, `Curves`, неявные анимации (`AnimatedContainer`, `AnimatedOpacity`...),
`SingleChildScrollView`, `ListView`, `GridView`, `PageView`.

## P7 — Material ⏳

`Theme`, `MaterialApp`, `Scaffold`, `AppBar`, кнопки (`ElevatedButton`, `TextButton`, `OutlinedButton`,
`IconButton`, `FloatingActionButton`), `Card`, `ListTile`, `Divider`, `Checkbox`, `Switch`, `Radio`,
`Slider`, индикаторы прогресса, `SnackBar`, диалоги, `BottomNavigationBar`, `Drawer`, `Navigator`.

## P8 — Ввод текста и изображения ⏳

`TextField` (клавиатура, IME, выделение), `Image` (из файлов приложения и из сети).

## Дальше

iOS, Windows/Web, GPU-растеризация, доступность, hot restart.
