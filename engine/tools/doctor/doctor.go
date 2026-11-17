// Package doctor — команды hexago test (проверка окружения, как flutter doctor) и hexago version.
package doctor

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"

	"hexago/tools/cli"
	"hexago/tools/sdk"
)

// VersionCommand — hexago version.
var VersionCommand = &cli.Command{
	Name:    "version",
	Usage:   "version",
	Summary: "показать версию Hexago",
	Help:    "Показывает версию Hexago, версию Go, которой собран hexago, и папку движка.",
	Run: func(ctx *cli.Context, args []string) error {
		if len(args) > 0 {
			if args[0] == "-h" || args[0] == "--help" {
				return flag.ErrHelp
			}
			return cli.Usagef("лишние аргументы: %s", strings.Join(args, " "))
		}
		ctx.Printf("Hexago %s\n", sdk.Version)
		ctx.Printf("Go:     %s %s/%s (сборка hexago)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		if engine, err := sdk.EnginePath(); err == nil {
			ctx.Printf("Движок: %s\n", engine)
		} else {
			ctx.Printf("Движок: не найден (%v)\n", err)
		}
		return nil
	},
}

// TestCommand — hexago test: проверяет всё, что нужно для сборки и запуска приложений.
var TestCommand = &cli.Command{
	Name:    "test",
	Usage:   "test",
	Summary: "проверить окружение: Go, Android SDK, NDK, Java, устройства",
	Help: `
Проверяет всё, что нужно, чтобы собирать и запускать приложения Hexago (как flutter doctor):
Go, движок, Android SDK, NDK, Java для Gradle, adb и подключённые устройства.

[✓] — в порядке, [!] — предупреждение (работать можно), [✗] — проблема, её нужно исправить.
Код выхода 1, если есть хотя бы одна проблема.`,
	Run: runTest,
}

// report печатает строки проверки и считает проблемы.
type report struct {
	ctx      *cli.Context
	problems int
	warnings int
}

func (r *report) ok(format string, args ...any) { r.line("✓", format, args...) }

func (r *report) warn(format string, args ...any) {
	r.warnings++
	r.line("!", format, args...)
}

func (r *report) fail(format string, args ...any) {
	r.problems++
	r.line("✗", format, args...)
}

// note — пояснение под строкой проверки, с отступом.
func (r *report) note(format string, args ...any) {
	r.ctx.Printf("    "+format+"\n", args...)
}

func (r *report) line(mark, format string, args ...any) {
	r.ctx.Printf("[%s] %s\n", mark, fmt.Sprintf(format, args...))
}

func runTest(ctx *cli.Context, args []string) error {
	if len(args) > 0 {
		if args[0] == "-h" || args[0] == "--help" {
			return flag.ErrHelp
		}
		return cli.Usagef("лишние аргументы: %s", strings.Join(args, " "))
	}

	r := &report{ctx: ctx}
	ctx.Printf("Hexago %s — проверка окружения\n\n", sdk.Version)

	checkGo(r)
	checkEngine(r)
	sdkDir := checkAndroidSDK(r)
	if sdkDir != "" {
		checkNDK(r, sdkDir)
	}
	checkJava(r)
	if sdkDir != "" {
		checkDevices(r, sdkDir)
	}

	ctx.Printf("\n")
	switch {
	case r.problems > 0:
		ctx.Printf("Найдено проблем: %d. Исправь строки с [✗] и запусти hexago test ещё раз.\n", r.problems)
		return cli.SilentError
	case r.warnings > 0:
		ctx.Printf("Можно работать. Предупреждений: %d.\n", r.warnings)
	default:
		ctx.Printf("Всё готово.\n")
	}
	return nil
}

func checkGo(r *report) {
	out, err := sdk.Output("go", "version")
	if err != nil {
		r.fail("Go не найден в PATH: установи Go с https://go.dev/dl")
		return
	}
	// go version go1.27.3 windows/amd64
	r.ok("%s", strings.TrimPrefix(out, "go version "))
}

func checkEngine(r *report) {
	engine, err := sdk.EnginePath()
	if err != nil {
		r.fail("Движок: %v", err)
		return
	}
	version, err := sdk.EngineGoVersion(engine)
	if err != nil {
		r.fail("Движок %s: %v", engine, err)
		return
	}
	r.ok("Движок: %s (go %s)", engine, version)
}

func checkAndroidSDK(r *report) string {
	dir, err := sdk.AndroidSDK()
	if err != nil {
		r.fail("%v", err)
		return ""
	}
	platforms := sdk.Platforms(dir)
	buildTools := sdk.BuildTools(dir)
	if len(platforms) == 0 {
		r.warn("Android SDK: %s — нет ни одной платформы (Gradle попробует скачать сам)", dir)
		return dir
	}
	r.ok("Android SDK: %s", dir)
	r.note("платформы: %s", strings.Join(platforms, ", "))
	if len(buildTools) > 0 {
		r.note("build-tools: %s", strings.Join(buildTools, ", "))
	}
	if !slices.Contains(platforms, "android-36") {
		r.warn("Нет платформы android-36, на неё собираются приложения (Gradle попробует скачать сам)")
	}
	return dir
}

func checkNDK(r *report, sdkDir string) {
	all := sdk.NDKs(sdkDir)
	best, err := sdk.BestNDK(sdkDir)
	if err != nil {
		r.fail("%v", err)
	} else {
		var arches []string
		for _, goarch := range []string{"arm64", "amd64"} {
			if _, err := best.Clang(goarch); err == nil {
				arches = append(arches, goarch)
			}
		}
		r.ok("Android NDK %s (компилятор для %s)", best.Version, strings.Join(arches, " и "))
	}
	for _, n := range all {
		if !n.Complete {
			r.warn("NDK %s установлен не полностью (нет компилятора) — не используется", n.Version)
		}
	}
}

func checkJava(r *report) {
	jdk, err := sdk.FindJDK()
	if err != nil {
		r.fail("%v", err)
		return
	}
	source := "из PATH"
	if os.Getenv("JAVA_HOME") != "" {
		source = "из JAVA_HOME"
	}
	if jdk.Major < sdk.MinJDK {
		r.fail("Java %s (%s) слишком старая: Gradle нужна Java %d или новее", jdk.Version, source, sdk.MinJDK)
		return
	}
	r.ok("Java %s (%s)", jdk.Version, source)
}

func checkDevices(r *report, sdkDir string) {
	adb, err := sdk.ADB(sdkDir)
	if err != nil {
		r.fail("%v", err)
		return
	}
	r.ok("adb: %s", adb)

	devices, err := sdk.Devices(adb)
	if err != nil {
		r.warn("Не удалось получить список устройств: %v", err)
		return
	}
	ready := 0
	for _, d := range devices {
		switch {
		case d.Ready():
			ready++
			if _, ok := sdk.ForABI(d.ABI); ok {
				r.ok("Устройство %s — %s, Android %s, %s", d.Serial, d.Name, d.Release, d.ABI)
			} else {
				r.warn("Устройство %s — архитектура %q не поддерживается", d.Serial, d.ABI)
			}
		case d.State == sdk.StateBooting:
			r.warn("Устройство %s (%s) ещё загружается", d.Serial, d.Name)
		case d.State == "unauthorized":
			r.warn("Устройство %s не разрешило отладку: подтверди запрос на экране телефона", d.Serial)
		default:
			r.warn("Устройство %s в состоянии %q", d.Serial, d.State)
		}
	}
	if ready == 0 {
		r.warn("Нет подключённых устройств — для hexago run запусти эмулятор или подключи телефон")
		if avds := sdk.AVDs(sdkDir); len(avds) > 0 {
			r.note("эмуляторы: %s", strings.Join(avds, ", "))
			r.note("запуск: hexago emulators --launch %s", avds[0])
		}
	}
}
