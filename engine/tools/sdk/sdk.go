// Package sdk знает, где что лежит: сам движок, Go, Android SDK, NDK, adb, JDK.
//
// Им пользуются все команды hexago: create (путь к движку для replace в go.mod),
// run (компилятор NDK, adb, устройства), test (проверка всего окружения), version.
package sdk

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Version — версия Hexago.
const Version = "0.1.0-dev"

// MinSDK — minSdk Android-приложений Hexago (как в шаблоне app/build.gradle.kts).
// Входит в имя компилятора NDK: x86_64-linux-android26-clang.
const MinSDK = 26

// EnginePath возвращает папку движка — ту, где лежит go.mod модуля hexago.
//
// Пока движок не опубликован, проекты подключают его по пути на диске
// (строка replace в go.mod проекта). Путь ищется так:
//
//  1. Переменная окружения HEXAGO_ROOT, если она задана.
//  2. Иначе — папка, из исходников которой собран сам hexago.
//     runtime.Caller(0) возвращает путь к этому .go-файлу на момент компиляции,
//     например C:/dev/projects/gexago/engine/tools/sdk/sdk.go.
//     Если собирать hexago с флагом -trimpath, путь станет относительным —
//     тогда нужна HEXAGO_ROOT.
func EnginePath() (string, error) {
	dir := os.Getenv("HEXAGO_ROOT")
	if dir == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return "", fmt.Errorf("не удалось определить папку движка: задай переменную HEXAGO_ROOT")
		}
		// file = .../engine/tools/sdk/sdk.go → поднимаемся на три папки вверх
		dir = filepath.Dir(filepath.Dir(filepath.Dir(file)))
	}

	// Проверяем, что это действительно движок: первая строка go.mod — "module hexago".
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("в %s не найден движок hexago: задай переменную HEXAGO_ROOT", dir)
	}
	firstLine, _, _ := strings.Cut(string(data), "\n")
	if strings.TrimSpace(firstLine) != "module hexago" {
		return "", fmt.Errorf("в %s лежит не движок hexago: задай переменную HEXAGO_ROOT", dir)
	}

	// В go.mod и Gradle пути пишутся с прямыми слэшами.
	return filepath.ToSlash(dir), nil
}

// EngineGoVersion возвращает версию Go из go.mod движка (строка "go 1.27.3").
//
// Проект должен требовать версию не ниже, чем его зависимости, иначе Go
// откажется собирать его и попросит go mod tidy. Поэтому новый проект
// получает ту же версию, что и движок.
func EngineGoVersion(engineDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(engineDir, "go.mod"))
	if err != nil {
		return "", err
	}
	for line := range strings.Lines(string(data)) {
		if version, ok := strings.CutPrefix(strings.TrimSpace(line), "go "); ok {
			return strings.TrimSpace(version), nil
		}
	}
	return "", fmt.Errorf("в go.mod движка нет строки go")
}

// exe добавляет .exe на Windows: adb → adb.exe.
func exe(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// isFile сообщает, существует ли обычный файл.
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// isDir сообщает, существует ли папка.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
