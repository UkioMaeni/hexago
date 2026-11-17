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

## P2 — Рисование ⏳

`ui/geom` (точки, размеры, прямоугольники, отступы, выравнивание), `ui/gfx` (цвет, программный холст:
прямоугольники, скругления со сглаживанием, рамки, тени, градиенты, клипы, прозрачность, сдвиги),
PNG-эталоны для проверки без устройства.

## P3 — Ядро фреймворка ⏳

Widget → Element → RenderObject: `StatelessWidget`, `StatefulWidget` + `SetState`, `BuildContext`,
наследуемые данные (`DependOn`), ключи, render-дерево с `BoxConstraints`, лэйаут, отрисовка, hit-test.

## P4 — Виджеты раскладки ⏳

`Container`, `Padding`, `Center`, `Align`, `SizedBox`, `ConstrainedBox`, `LimitedBox`, `AspectRatio`,
`FractionallySizedBox`, `Row`, `Column`, `Flex`, `Expanded`, `Flexible`, `Spacer`, `Stack`, `Positioned`,
`Wrap`, `ColoredBox`, `DecoratedBox`, `Opacity`, `ClipRect`, `ClipRRect`, `ClipOval`, `Transform`,
`Offstage`, `Visibility`, `Builder`, `LayoutBuilder`, `SafeArea`, `MediaQuery`.

## P5 — Текст и иконки ⏳

Встроенный шрифт, шейпинг и переносы (go-text), `Text`, `RichText`, `DefaultTextStyle`, `TextStyle`,
`Icon` (шрифт Material Icons).

## P6 — Ввод, кадры, анимации, скролл ⏳

Касания с Android, `GestureDetector`, `Listener`, кадры по vsync (Choreographer), `AnimationController`,
`Tween`, `Curves`, неявные анимации (`AnimatedContainer`, `AnimatedOpacity`...),
`SingleChildScrollView`, `ListView`, `GridView`, `PageView`.

## P7 — Material ⏳

`Theme`, `MaterialApp`, `Scaffold`, `AppBar`, кнопки (`ElevatedButton`, `TextButton`, `OutlinedButton`,
`IconButton`, `FloatingActionButton`), `Card`, `ListTile`, `Divider`, `Checkbox`, `Switch`, `Radio`,
`Slider`, индикаторы прогресса, `SnackBar`, диалоги, `BottomNavigationBar`, `Drawer`, `Navigator`.

## P8 — Ввод текста и изображения ⏳

`TextField` (клавиатура, IME, выделение), `Image` (из файлов приложения и из сети).

## Дальше

iOS, Windows/Web, GPU-растеризация, доступность, hot restart.
