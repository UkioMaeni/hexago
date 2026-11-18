# Архитектурные решения (ADR)

ADR (Architecture Decision Record) — короткая запись одного решения: контекст, что выбрали, какие были
альтернативы и чем за это платим. Решения не переписываются молча: если передумали — новый ADR,
а у старого статус «заменено ADR-XXXX».

| № | Решение | Статус |
|---|---|---|
| [0001](0001-own-rendering.md) | Рисуем каждый пиксель сами | принято |
| [0002](0002-three-trees.md) | Три дерева: Widget → Element → RenderObject | принято |
| [0003](0003-go-idioms-instead-of-inheritance.md) | Go-идиомы вместо наследования; API виджетов | принято, есть открытые вопросы |
| [0004](0004-rendering-pipeline.md) | Canvas → DisplayList → Scene → сменный растеризатор; CPU первым | принято |
| [0005](0005-threading-and-runners.md) | Потоки и раннеры; однопоточный framework | принято |
| [0006](0006-text-stack.md) | Текст: go-text для шейпинга, своя вёрстка абзаца | принято |
| [0007](0007-platform-layer.md) | Платформы: минимум cgo, свои события у платформы | принято; порядок платформ заменён ADR-0012 |
| [0008](0008-package-layering.md) | Слои пакетов, проверка тестом | отложено ADR-0012 |
| [0009](0009-testability-first.md) | Headless, фейковые часы и golden-тесты с первого дня | принято |
| [0010](0010-float32-logical-pixels.md) | float32 и логические пиксели | заменено ADR-0013 |
| [0011](0011-no-hot-reload.md) | Без hot reload в v1 | принято |
| [0012](0012-mobile-first-cli-and-layout.md) | Android первым; консольная программа hexago; весь код в engine/ | принято |
| [0013](0013-flutter-widgets-course.md) | Практичный курс: виджеты как во Flutter; float64; x/image и go-text | принято |
| [0014](0014-devices-and-emulators.md) | Устройства и эмуляторы: hexago devices, emulators, run -d, выбор в VS Code | принято |
| [0015](0015-hexago-rebuilds-itself.md) | hexago пересобирает себя, когда меняются его исходники | принято |
| [0016](0016-run-from-vscode-debugger.md) | Запуск из VS Code через Run → Start Debugging; точки останова — позже | принято |
| [0017](0017-types-from-own-packages.md) | Каждый тип — из своего пакета; в widgets только виджеты | принято |

Шаблон нового ADR:

```markdown
# ADR-NNNN: Название

- Статус: предложено | принято | заменено ADR-XXXX
- Дата: ГГГГ-ММ-ДД

## Контекст
## Решение
## Альтернативы
## Последствия
```
