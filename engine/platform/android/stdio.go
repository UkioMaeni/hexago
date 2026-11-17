//go:build android

package android

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
)

// У Android-приложений стандартный вывод и поток ошибок никуда не выводятся.
// Перенаправляем вывод Go-кода (fmt.Println, log, os.Stderr) в Logcat с тегом hexago —
// оттуда его показывает hexago run, как flutter run показывает print.
//
// Подменяем именно переменные os.Stdout и os.Stderr, а не дескрипторы 1 и 2:
// в те же дескрипторы пишут системные библиотеки Android, и их служебные строки
// засоряли бы лог приложения.
func init() {
	r, w, err := os.Pipe()
	if err != nil {
		return
	}
	os.Stdout = w
	os.Stderr = w
	log.SetOutput(w)

	go func() {
		lines := bufio.NewScanner(r)
		lines.Buffer(make([]byte, 64*1024), 1024*1024)
		for lines.Scan() {
			Log(lines.Text())
		}
	}()
}

// CrashFile — имя файла в папке кеша приложения, куда Go пишет текст паники.
// hexago run читает его после падения приложения.
const CrashFile = "hexago-crash.txt"

// setupCrashOutput направляет текст необработанных паник в файл.
//
// Паника сразу завершает процесс, и строки, отправленные в Logcat через трубу,
// не успевают дойти. Запись в файл синхронная — текст сохранится.
// При каждом запуске файл очищается: в нём только паника текущего запуска.
func setupCrashOutput(cacheDir string) {
	path := filepath.Join(cacheDir, CrashFile)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	debug.SetCrashOutput(f, debug.CrashOptions{})
	f.Close() // SetCrashOutput держит свою копию дескриптора

	// В режиме библиотеки Go после паники посылает процессу SIGABRT, его перехватывает
	// обработчик падений Android и завершает процесс раньше, чем Go допишет стек вызовов.
	// "single" — напечатать стек упавшей горутины (как в обычной Go-программе) и выйти.
	debug.SetTraceback("single")
}
