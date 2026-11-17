# Архитектура Gexago

> Статус: черновик v0.1 (2026-11-17). Документ живой: меняется вместе с решениями в [`docs/adr/`](adr/README.md).
> План работ и кто что пишет — в [ROADMAP.md](ROADMAP.md).
>
> **Раскладка и порядок ([ADR-0012](adr/0012-mobile-first-cli-and-layout.md)):** весь код живёт в папке
> `engine/` (Go-модуль `hexago`), начинаем с Android. Пути вида `engine/…`, `framework/…`, `platform/…`
> ниже — логические слои: их папки появятся внутри `engine/`, когда начнётся их этап.
> Консольная программа `hexago` (`engine/main.go` + `engine/tools`) стоит отдельно от слоёв: движок её не импортирует.

## 1. Что мы строим

Gexago — движок кроссплатформенных приложений на Go по модели Flutter. Приложение описывает интерфейс
декларативно, деревом виджетов, а движок сам рисует каждый пиксель — одинаково на всех платформах.

**Цели**

- Одна кодовая база UI → Windows, Web (wasm), Linux, macOS, Android, iOS (в этом порядке).
- Модель Flutter: Widget → Element → RenderObject, «ограничения вниз, размеры вверх», кадровый конвейер, арена жестов.
- Идиоматичный Go: без эмуляции классов; интерфейсы, встраивание и дженерики — там, где они честно помогают.
- Тестируемость: любой виджет можно отрисовать без окна, с фейковыми часами, и сравнить с эталонной картинкой.
- Учебная ценность: ядро пишется руками; чужие библиотеки — только там, где своя версия заняла бы годы (шейпинг текста).

**Не цели v1**

- hot reload ([ADR-0011](adr/0011-no-hot-reload.md));
- обёртки над нативными контролами ОС;
- система плагинов / platform channels;
- клон Material или Cupertino — своя дизайн-система позже;
- 3D, видео, веб-вью.

## 2. Слои

```
┌───────────────────────────────────────────────────────────────────┐
│  Приложение                          examples/…, твой main.go      │
├───────────────────────────────────────────────────────────────────┤
│  app          app.Run(widget): собирает платформу, движок и фреймворк│
├───────────────────────────────────────────────────────────────────┤
│  FRAMEWORK  (аналог Dart-части Flutter)              framework/…   │
│    widgets      Widget, Element, State, BuildOwner                 │
│    rendering    RenderObject, RenderBox, PipelineOwner             │
│    painting · gestures · animation · semantics                     │
│    scheduler    кадр и его фазы, Ticker                            │
│    foundation   Key, ChangeNotifier, диагностика                   │
├──────────────── граница: engine/ui (аналог dart:ui) ──────────────┤
│  ENGINE  (аналог C++-движка Flutter)                 engine/…      │
│    ui       Dispatcher, события, метрики окна                      │
│    layer    дерево слоёв → Scene                                   │
│    gfx      Canvas, Paint, Path, DisplayList                       │
│    text     шрифты, шейпинг, вёрстка абзаца                        │
│    raster   soft (CPU) · gpu (позже)                               │
│    shell    раннеры, vsync, конвейер кадров                        │
│    geom     Point, Size, Rect, Affine                              │
├───────────────────────────────────────────────────────────────────┤
│  PLATFORM  (аналог embedder)                         platform/…    │
│    win32 · headless · web · (x11, wayland, cocoa, android, ios)    │
└───────────────────────────────────────────────────────────────────┘
```

Три главных правила:

1. **Зависимости идут только вниз.** Framework не знает ни про платформу, ни про растеризатор: он отдаёт
   движку неизменяемую `Scene` и получает события через `engine/ui`. Проверяется тестом `internal/archtest`
   ([ADR-0008](adr/0008-package-layering.md)); тест отложен, пока не появятся слои ([ADR-0012](adr/0012-mobile-first-cli-and-layout.md)).
2. **Платформа ничего не знает о движке.** Она даёт окно, поверхность для пикселей и сырые события
   в физических пикселях. `engine/shell` переводит их в события `ui` в логических пикселях.
3. **Всё, что ушло в `Scene`, неизменяемо.** Поэтому кадры передаются между потоками без мьютексов.

## 3. Жизнь кадра

```
platform          shell                framework (UI-раннер)                raster-раннер
   │  vsync ──────▶ │                          │                                  │
   │                │ OnBeginFrame(t) ───────▶ │ 1. тики анимаций (Ticker)         │
   │                │                          │ 2. задачи из Post                 │
   │                │ OnDrawFrame() ─────────▶ │ 3. build:  перестроить «грязные»  │
   │                │                          │            элементы по глубине    │
   │                │                          │ 4. layout: «грязные» RenderObject  │
   │                │                          │ 5. compositing bits               │
   │                │                          │ 6. paint:  слои + DisplayList     │
   │                │                          │ 7. semantics (позже)              │
   │                │ ◀── Render(view, scene) ─│ 8. post-frame колбэки             │
   │                │ ─────────────── scene ─────────────────────────────────────▶ │ слои → пиксели
   │ ◀──────────────────────────────────────────────────────────────── Present() ─ │
```

- Кадр **запрашивается**, а не крутится постоянно: `ScheduleFrame()` вызывают `SetState`, анимация, ресайз.
  Нет изменений — нет кадров, процессор спит.
- Конвейер глубиной 2: пока raster рисует кадр N, UI может строить N+1. Если растеризатор отстаёт,
  UI ждёт (канал на один элемент) — задержка ввода не накапливается.

## 4. Потоки и раннеры

Подробно — [ADR-0005](adr/0005-threading-and-runners.md). Три роли, как `TaskRunners` во Flutter:

| Роль | Что делает | Десктоп | Web (wasm) | Тесты |
|---|---|---|---|---|
| platform | цикл событий ОС, окна | главный поток ОС (`runtime.LockOSThread` в `init`) | event loop браузера | горутина теста |
| ui | весь framework: build / layout / paint | своя горутина | тот же поток | та же горутина |
| raster | `Scene` → пиксели → `Present` | горутина с `LockOSThread` (GPU-контексты привязаны к потоку) | тот же поток | синхронно |

Раннер — минимальный интерфейс `Post(func())`. Shell собирается из раннеров, поэтому одна и та же логика
работает и в трёх потоках, и в одном — в браузере и в тестах (там это даёт детерминизм).

Правило для пользователя фреймворка: **дерево виджетов трогаем только на UI-раннере.** Из своих горутин
(сеть, файлы) — через `app.Post(func() { … })`. В debug-сборке (тег `gexagodebug`) фреймворк проверяет,
что `SetState` вызывается с UI-горутины.

## 5. Три дерева

Подробно — [ADR-0002](adr/0002-three-trees.md).

```
Widget        описание: дёшево, неизменяемо, создаётся заново при каждом build
   │  создаёт / обновляет
Element       экземпляр в дереве: живёт долго, держит State, решает «обновить или пересоздать»
   │  создаёт / обновляет (только для RenderObjectWidget)
RenderObject  геометрия: лэйаут, отрисовка, hit-test; дорогой, живёт долго
```

- **Reconciliation.** При перестройке Element сравнивает старый и новый виджет: тот же Go-тип и тот же
  ключ → обновить (State сохраняется), иначе → размонтировать старый, смонтировать новый. Для списка
  детей — алгоритм `updateChildren` с ключами: сверка с начала, с конца, середина через map по ключам.
- **Лэйаут.** Constraints идут вниз, Size — вверх, позицию ребёнка выставляет родитель. Один проход, O(n).
  *Relayout boundary*: если размер узла не зависит от его детей (жёсткие ограничения или родитель не
  использует размер), «грязный» лэйаут не всплывает выше этого узла.
- **Отрисовка.** RenderObject рисует в `PaintingContext`. *Repaint boundary* получает свой слой и свою
  `DisplayList`, которая переиспользуется, пока узел не помечен `needsPaint`.

## 6. Go без наследования

Подробно — [ADR-0003](adr/0003-go-idioms-instead-of-inheritance.md). Flutter опирается на классы и
виртуальные методы. В Go встраивание — не наследование: метод встроенной структуры не может вызвать
«переопределённый» метод внешней. Поэтому главный шаблон проекта:

> **Данные — во встроенной базовой структуре. Алгоритм — в функции пакета, принимающей интерфейс.**

```go
package rendering

// Box — общие данные любого бокса. Встраивается в конкретные RenderObject.
type Box struct {
	node                       // parent, depth, owner, флаги needsLayout / needsPaint
	constraints BoxConstraints
	size        geom.Size
}

func (b *Box) box() *Box { return b } // неэкспортируемый «пропуск» в интерфейс

type RenderBox interface {
	// «Виртуальные» методы — их пишет конкретный бокс.
	PerformLayout(c BoxConstraints) geom.Size
	Paint(ctx *PaintingContext, offset geom.Point)
	box() *Box // есть только у тех, кто встроил rendering.Box
}

// Layout — общий алгоритм с кешированием. Живёт в пакете, а не в базе.
func Layout(r RenderBox, c BoxConstraints, parentUsesSize bool) geom.Size {
	b := r.box()
	if !b.needsLayout && b.constraints == c {
		return b.size // ничего не изменилось — пропускаем всё поддерево
	}
	b.constraints = c
	b.size = c.Constrain(r.PerformLayout(c))
	b.needsLayout = false
	return b.size
}
```

API виджетов — struct-литералы вместо именованных параметров Dart:

```go
type Counter struct {
	widgets.Stateful // «вид» виджета; без него тип не станет Widget
	Start int
}

func (c *Counter) CreateState() widgets.State { return &counterState{} }

type counterState struct {
	widgets.StateBase[*Counter] // Widget(), SetState(), Context(), Mounted()
	n int
}

func (s *counterState) InitState() { s.n = s.Widget().Start }

func (s *counterState) Build(ctx widgets.BuildContext) widgets.Widget {
	return &widgets.GestureDetector{
		OnTap: func() { s.SetState(func() { s.n++ }) },
		Child: &widgets.Center{
			Child: &widgets.Text{Data: strconv.Itoa(s.n)},
		},
	}
}

func main() { app.Run(&Counter{Start: 0}) }
```

## 7. Рендеринг

Подробно — [ADR-0004](adr/0004-rendering-pipeline.md).

```
RenderObject.Paint ─▶ gfx.Canvas (Recorder) ─▶ gfx.DisplayList ─┐
                                                                 ├─▶ layer.Scene ─▶ raster.Rasterizer ─▶ Surface
repaint boundary / opacity / clip / transform ─▶ слои layer.* ──┘
```

- `gfx.Canvas` — интерфейс рисования: `Save/Restore`, `Transform`, `ClipRect/ClipPath`,
  `DrawRect/DrawRRect/DrawPath/DrawGlyphs/DrawImage`.
- `gfx.Recorder` записывает вызовы в `DisplayList` — плоский список команд. Фреймворк сам пиксели не рисует никогда.
- `layer.Scene` — дерево слоёв: `Offset`, `Transform`, `ClipRect/ClipRRect/ClipPath`, `Opacity`, `Picture(DisplayList)`.
- `raster.Rasterizer` — интерфейс «нарисуй сцену в цель». Первая реализация — `raster/soft` на CPU:
  кривые Безье → полилинии → покрытие с антиалиасингом → смешивание в premultiplied RGBA.
  Позже `raster/gpu` — без единой правки во framework.

## 8. Текст

Подробно — [ADR-0006](adr/0006-text-stack.md).

```
строка + стили
  ─▶ itemize:  прогоны по скрипту, направлению, шрифту (с фолбэком)
  ─▶ shaping:  go-text/typesetting (HarfBuzz, переписанный на Go)          ← берём готовое
  ─▶ переносы строк (UAX #14) и bidi-перестановка
  ─▶ Paragraph: строки, метрики, выравнивание, многоточие, hit-test, каретка ← пишем сами
  ─▶ GlyphRun в DisplayList
  ─▶ растеризация глифов (контуры → тот же soft-растеризатор) + кеш         ← пишем сами
```

## 9. Ввод

- **Указатели:** платформа → сырое событие (физические пиксели) → shell → `ui.PointerEvent` (логические
  пиксели, тип устройства, id указателя) → `gestures`: hit-test по render-дереву → событие получают все,
  в кого попали → **арена жестов**: Tap, Drag, LongPress, Scale соревнуются, побеждает один.
- **Клавиатура:** `KeyEvent` (физическая + логическая клавиша) → дерево фокуса (`FocusNode`, `FocusScope`)
  → `Shortcuts` / `Actions`.
- **Ввод текста и IME** — отдельный канал: composing-регион, позиция курсора для окна кандидатов.
  Самая платформозависимая часть проекта, этап M9.

## 10. Платформы

Подробно — [ADR-0007](adr/0007-platform-layer.md).

| Платформа | Окно и ввод | Пиксели (soft) | GPU (позже) | cgo | Этап |
|---|---|---|---|---|---|
| Android | своя Activity + SurfaceView, JNI, `c-shared` | `ANativeWindow_lock` | Vulkan / GLES | нужен (NDK) | M1–M3 |
| iOS | UIKit, `c-archive` | `CALayer` | Metal | нужен (Xcode, Mac) | M12 |
| Windows | Win32 через `golang.org/x/sys/windows` | `SetDIBitsToDevice` | D3D11 / DXGI | не нужен | M13 |
| Web | `syscall/js`, canvas | `putImageData` | WebGPU / WebGL2 | — | M13 |
| Linux | X11 (потом Wayland) через purego | `XPutImage` / `wl_shm` | Vulkan / GL | не нужен | после M13 |
| macOS | Cocoa через purego + objc | `CALayer.contents` | Metal | не нужен | после M13 |

Контракт пакета `platform` (набросок, финализируем на M1):

```go
type Host interface {
	Run(h Handler) error                        // цикл событий ОС; возвращается при выходе
	NewWindow(opts WindowOptions) (Window, error)
	Post(fn func())                             // выполнить на платформенном потоке
}

type Window interface {
	Size() (w, h int)  // физические пиксели
	Scale() float32    // device pixel ratio (1.0, 1.25, 1.5, 2.0 …)
	Surface() Surface
	RequestVsync()     // попросить ОДИН сигнал vsync
	SetTitle(string)
	SetCursor(Cursor)
	Close()
}

// SoftwareSurface — кадровый буфер в памяти, в который пишет raster/soft.
type SoftwareSurface interface {
	Lock() (pix []byte, stride, w, h int, format PixelFormat)
	Unlock()
	Present() error
}

type Handler interface{ HandleEvent(e Event) } // зовётся на платформенном потоке — быстро!
```

`platform/headless` — полноценная «платформа» для тестов и CI: окно в памяти, vsync по команде теста.

## 11. Тестирование

Подробно — [ADR-0009](adr/0009-testability-first.md).

- **Модульные тесты:** geom, растеризатор, алгоритмы лэйаута.
- **Golden-тесты:** отрисовать сцену через `raster/soft` в PNG и сравнить с эталоном
  (`go test ./... -args -update` перезаписывает эталоны).
- **Виджет-тесты** (`gexagotest`): headless-платформа, фейковые часы, `Pump`, `Tap`, поиск по тексту / типу / ключу.
- **Бенчмарки** кадра и аллокаций на кадр (`testing.AllocsPerRun`) — с первых этапов, чтобы GC не стал сюрпризом.
- **Архитектурный тест** `internal/archtest`: нарушил слои — красный `go test`. Отложен, пока не появятся слои
  ([ADR-0012](adr/0012-mobile-first-cli-and-layout.md)).

## 12. Правило зависимостей

Это план зависимостей. Тест `internal/archtest`, который его проверял, отложен до появления слоёв
([ADR-0012](adr/0012-mobile-first-cli-and-layout.md)); его код лежит в `C:\dev\projects\gexago_old`.
Пакет всегда может импортировать stdlib и пакеты своего же поддерева.

| Пакет (и подпакеты) | Может импортировать |
|---|---|
| `engine/geom` | — |
| `engine/gfx` | geom |
| `engine/text` | geom, gfx · внешн.: `github.com/go-text/…` |
| `engine/layer` | geom, gfx |
| `engine/raster` | geom, gfx, layer |
| `engine/ui` | geom, gfx, text, layer |
| `engine/shell` | `engine/…`, `platform` (только интерфейсы, не реализации) |
| `platform` | geom · внешн.: `golang.org/x/sys/…`, purego · **cgo разрешён только здесь** |
| `framework/foundation` | — |
| `framework/scheduler` | foundation, engine/ui |
| `framework/animation` | foundation, scheduler, engine/geom, engine/gfx |
| `framework/gestures` | foundation, scheduler, engine/geom, engine/ui |
| `framework/painting` | foundation, engine/{geom, gfx, text, ui} |
| `framework/semantics` | foundation, engine/{geom, ui} |
| `framework/rendering` | foundation, scheduler, animation, gestures, painting, semantics, engine/{geom, gfx, text, layer, ui} |
| `framework/widgets` | `framework/…`, engine/{geom, gfx, text, layer, ui} |
| `app`, `gexagotest` | `framework/…`, `engine/…`, `platform/…` |
| `examples` | app, `framework/…`, engine/{geom, gfx} |
| `tools` (команды hexago) | только stdlib; ни один слой движка не импортирует `tools` |

Ни один пакет `framework/…` не импортирует `platform/…`, `engine/shell` и `engine/raster`.

## 13. Главные риски

| Риск | Почему опасно | Что делаем |
|---|---|---|
| Ввод текста и IME | у каждой ОС свой сложный API | отдельный этап M9, начинаем с Windows (IMM32, потом TSF) |
| Скорость CPU-растеризатора | 4K при 60 fps на CPU — тяжело | repaint boundaries, грязные регионы, потом GPU (M11) |
| Аллокации на кадр → работа GC | Go GC быстрый, но мусор копится | переиспользуемые буферы DisplayList, бенчмарки `AllocsPerRun` |
| Наследование Flutter плохо ложится на Go | соблазн эмулировать классы | шаблон «база + функции пакета» ([ADR-0003](adr/0003-go-idioms-instead-of-inheritance.md)) |
| Мобильные платформы | Go runtime внутри чужого процесса, cgo, тулчейны | в самом конце, когда ядро устоялось |
| Доступность | без неё приложение не для всех | семантическое дерево заложено сразу, реализация позже |
| Бесконечный объём | проект не имеет естественного конца | вертикальные срезы: каждый этап заканчивается работающим демо |

## 14. Словарь Flutter → Gexago

| Flutter | Gexago |
|---|---|
| `dart:ui` | `engine/ui` (+ `gfx`, `text`, `layer`) |
| Embedder | `platform/*` |
| Shell, TaskRunners | `engine/shell`, раннеры |
| Skia / Impeller | `engine/raster/soft`, позже `engine/raster/gpu` |
| `Picture` / `PictureRecorder` | `gfx.DisplayList` / `gfx.Recorder` |
| `SceneBuilder` / `Scene` | `layer.Scene` |
| `SchedulerBinding` | `framework/scheduler` |
| `PipelineOwner` | `rendering.PipelineOwner` |
| `BuildOwner` | `widgets.BuildOwner` |
| `WidgetsFlutterBinding` (mixin-ы) | `app.Binding` — одна структура из полей, без mixin-ов |
| `flutter_test` | `gexagotest` |
