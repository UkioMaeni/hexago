# Hexago

Движок кроссплатформенных приложений на Go по модели Flutter: декларативные виджеты,
собственная отрисовка каждого пикселя. Начинаем с Android.

## Установка

```bash
cd engine
go install .
```

`hexago.exe` ставится в `%USERPROFILE%\go\bin` (эта папка должна быть в PATH). Это нужно один раз:
дальше `hexago` сам пересобирается, когда его исходники в движке меняются.

## Команды

```bash
hexago test                     # проверить окружение: Go, Android SDK, NDK, Java, устройства
hexago create myapp             # новое приложение в папке myapp
hexago create --package dev.mycompany.shop shop
cd myapp
hexago run android              # собрать, установить и запустить; лог приложения — в терминале
hexago devices                  # подключённые телефоны и запущенные эмуляторы
hexago emulators                # эмуляторы из Android Studio; --launch <имя> — запустить
hexago run -d Pixel_7           # запустить на нужном устройстве (выключенный эмулятор hexago запустит сам)
hexago version
```

## Структура

```
engine/
  main.go     консольная программа hexago
  tools/      её команды: create, run, devices, emulators, test, version; расширение VS Code
  ui/         то, что подключают приложения: виджеты, отрисовка, запуск
  platform/   встраивание в ОС (Android: JNI, NDK, HexagoActivity)
  docs/       архитектура, решения (ADR), дорожная карта
```

Документы: [архитектура](engine/docs/ARCHITECTURE.md) · [решения](engine/docs/adr/README.md) · [дорожная карта](engine/docs/ROADMAP.md).
