# Hexago для VS Code

- **SDK.** Если команда `hexago` не найдена, расширение спрашивает папку движка Hexago и запоминает её —
  для всех проектов или только для текущего (`.vscode/settings.json`, настройка `hexago.sdkPath`).
  По желанию добавляет `hexago` в PATH.
- **Действия с виджетами** — лампочка или `Ctrl+.`, а также пункт контекстного меню
  «Действия с виджетом…» (`Ctrl+Alt+W`): обернуть в виджет, в Center, Padding, Container, SizedBox,
  Align, Expanded, Column, Row, Stack, Opacity, ClipRRect, SafeArea, Builder; удалить виджет,
  оставив содержимое; поменять местами с ребёнком.
- **Команды** (палитра, `Hexago: …`): создать проект, запустить на устройстве (кнопка ▶ Hexago
  в строке состояния), проверить окружение, указать путь к SDK.

## Сборка

```bash
npm install
npm run compile
npm run package          # → hexago.vsix
code --install-extension hexago.vsix
```
