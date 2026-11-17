package sdk

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"
)

// noRebuildEnv — защита от зацикливания: пересобранный hexago себя уже не проверяет.
const noRebuildEnv = "HEXAGO_NO_REBUILD"

// RestartIfStale пересобирает hexago, если исходники утилиты в движке новее программы, и
// запускает новую сборку с теми же аргументами — как Flutter пересобирает flutter_tools.
// Так `go install .` нужен один раз: дальше hexago сам следит, чтобы соответствовать движку.
//
// Возвращает, только если пересборка не нужна или не удалась (тогда работает старая сборка);
// иначе завершает процесс с кодом новой сборки.
func RestartIfStale(stderr io.Writer) {
	if os.Getenv(noRebuildEnv) != "" {
		return
	}
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	exePath, _ = filepath.EvalSymlinks(exePath)
	removeOld(exePath)
	engine, err := EnginePath()
	if err != nil {
		return
	}
	if fresh(exePath, engine) {
		return
	}
	fmt.Fprintln(stderr, "Утилита hexago изменилась в движке — пересборка...")
	if err := rebuild(exePath, engine); err != nil && !fresh(exePath, engine) {
		fmt.Fprintf(stderr, "Не удалось пересобрать hexago, работает прежняя сборка: %v\n", err)
		return
	}
	os.Exit(reexec(exePath))
}

// fresh сообщает, что программа собрана позже, чем менялись исходники утилиты.
func fresh(exePath, engine string) bool {
	info, err := os.Stat(exePath)
	if err != nil {
		return false
	}
	newest, ok := newestSource(engine)
	return !ok || !newest.After(info.ModTime())
}

// newestSource — время последнего изменения исходников hexago: main.go, go.mod, go.sum и tools/
// (вместе с шаблонами, которые встраиваются в программу). Расширение VS Code — не часть hexago.
// ok = false, если исходников нет (hexago собран не из папки движка).
func newestSource(engine string) (newest time.Time, ok bool) {
	consider := func(path string) {
		if info, err := os.Stat(path); err == nil && info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}
	consider(filepath.Join(engine, "main.go"))
	consider(filepath.Join(engine, "go.mod"))
	consider(filepath.Join(engine, "go.sum"))
	tools := filepath.Join(engine, "tools")
	if !isDir(tools) {
		return time.Time{}, false
	}
	filepath.WalkDir(tools, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == "vscode" || d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		consider(path)
		return nil
	})
	return newest, !newest.IsZero()
}

// rebuild собирает новую программу рядом со старой и подменяет её. Запущенную программу
// Windows удалить не даёт, но переименовать — даёт: старая сборка уходит в .old-<pid>.
func rebuild(exePath, engine string) error {
	suffix := strconv.Itoa(os.Getpid())
	tmp := exePath + ".new-" + suffix
	cmd := exec.Command("go", "build", "-o", tmp, ".")
	cmd.Dir = engine
	if out, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("go build: %v\n%s", err, out)
	}
	old := exePath + ".old-" + suffix
	if err := os.Rename(exePath, old); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, exePath); err != nil {
		os.Rename(old, exePath)
		os.Remove(tmp)
		return err
	}
	return nil
}

// reexec запускает новую сборку с теми же аргументами и возвращает её код выхода.
func reexec(exePath string) int {
	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append(os.Environ(), noRebuildEnv+"=1")
	// Ctrl+C получат оба процесса; выйти должна новая сборка, а мы — дождаться её.
	signal.Ignore(os.Interrupt)
	err := cmd.Run()
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return exitErr.ExitCode()
	}
	if err != nil {
		return 1
	}
	return 0
}

// removeOld удаляет старые сборки, оставшиеся после прошлых пересборок.
// Ещё запущенные Windows удалить не даст — они уйдут в следующий раз.
func removeOld(exePath string) {
	olds, _ := filepath.Glob(exePath + ".old-*")
	for _, f := range olds {
		os.Remove(f)
	}
}
