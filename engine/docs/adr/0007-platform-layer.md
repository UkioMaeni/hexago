# ADR-0007: Платформы — минимум cgo, свои события у платформы

- Статус: принято; порядок платформ (п. 6) заменён [ADR-0012](0012-mobile-first-cli-and-layout.md)
- Дата: 2026-11-17

## Контекст

Go вызывает системные API тремя способами: `syscall`/`golang.org/x/sys` (без C-компилятора),
`github.com/ebitengine/purego` (динамическая загрузка C-библиотек без cgo) и cgo (нужен C-тулчейн,
ломает простую кросс-компиляцию, медленнее сборка).

## Решение

1. **cgo только в `platform/...`** и только там, где без него никак (Android, iOS). Проверяется `internal/archtest`.
2. **Windows:** Win32 через `golang.org/x/sys/windows` и `syscall.NewLazyDLL` — cgo не нужен
   (на машине автора сейчас и нет gcc). DPI-awareness per-monitor v2, `WM_POINTER*` для мыши/тача/пера.
3. **Linux, macOS:** purego (`dlopen` libX11 / Wayland, objc-рантайм для Cocoa).
4. **Android, iOS:** `-buildmode=c-shared` / `c-archive` + тонкий нативный хост-проект (Gradle / Xcode).
5. **Платформа говорит своими событиями** — в физических пикселях, со своими кодами клавиш.
   `engine/shell` переводит их в `ui`-события (логические пиксели, нормализованные клавиши, id view).
   Это повторяет границу embedder → engine во Flutter: платформа ничего не знает о движке.
6. Порядок: Windows → headless (сразу, для тестов) → Web → Linux → macOS → Android → iOS.

## Альтернативы

- Общая библиотека окон (GLFW через cgo, SDL): быстрее старт, но cgo везде, нет мобильных платформ
  в нужном виде и нет IME/доступности на нужном уровне.

## Последствия

- `GOOS=windows`, `GOOS=js GOARCH=wasm`, `GOOS=linux`, `GOOS=darwin` собираются с любой машины.
- Дублирование типов событий (platform и ui) — сознательная цена за чистую границу.
