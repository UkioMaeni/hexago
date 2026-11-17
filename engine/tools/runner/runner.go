// Package runner — команда hexago run: собрать приложение и запустить его на устройстве.
package runner

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"hexago/tools/cli"
	"hexago/tools/sdk"
)

// Command — hexago run.
var Command = &cli.Command{
	Name:    "run",
	Usage:   "run [android] [-d устройство]",
	Summary: "собрать и запустить приложение на телефоне или эмуляторе",
	Help: `
Собирает приложение из текущей папки и запускает его — как flutter run:

  1. Go-код компилируется в libhexago.so под архитектуру устройства (компилятор из NDK).
  2. Gradle собирает APK и устанавливает его на устройство.
  3. Приложение запускается, а его лог (fmt.Println, паники) печатается здесь же.
     Ctrl+C — выйти.

Флаги:
  -d  серийный номер устройства, если подключено несколько (список: hexago test)

Примеры:
  hexago run
  hexago run android -d emulator-5554`,
	Run: run,
}

func run(ctx *cli.Context, args []string) error {
	fs := cli.NewFlagSet("run")
	serial := fs.String("d", "", "")
	positional, err := cli.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	platform := "android"
	switch len(positional) {
	case 0:
	case 1:
		platform = positional[0]
	default:
		return cli.Usagef("лишние аргументы: %s", strings.Join(positional[1:], " "))
	}
	if platform != "android" {
		return cli.Usagef("платформа %q пока не поддерживается, есть только android", platform)
	}
	return runAndroid(ctx, *serial)
}

func runAndroid(ctx *cli.Context, serial string) error {
	project := ctx.Dir
	androidDir := filepath.Join(project, "android")
	if _, err := os.Stat(filepath.Join(project, "go.mod")); err != nil {
		return errors.New("здесь нет проекта Hexago (нет go.mod) — перейди в папку проекта")
	}
	if _, err := os.Stat(androidDir); err != nil {
		return errors.New("в проекте нет папки android/ — он создан не hexago create?")
	}

	// Окружение: SDK, adb, устройство, NDK.
	sdkDir, err := sdk.AndroidSDK()
	if err != nil {
		return err
	}
	adb, err := sdk.ADB(sdkDir)
	if err != nil {
		return err
	}
	device, err := pickDevice(adb, serial)
	if err != nil {
		return err
	}
	abi, ok := sdk.ForABI(device.ABI)
	if !ok {
		return fmt.Errorf("архитектура устройства %q не поддерживается", device.ABI)
	}
	ndk, err := sdk.BestNDK(sdkDir)
	if err != nil {
		return err
	}
	clang, err := ndk.Clang(abi.GOARCH)
	if err != nil {
		return err
	}
	appID, err := applicationID(androidDir)
	if err != nil {
		return err
	}

	// 1. Go → libhexago.so. Сначала зависимости: движок может требовать новые модули.
	if err := sdk.Tidy(project); err != nil {
		return err
	}
	ctx.Printf("Сборка Go для %s (%s)...\n", device.Serial, abi.ABI)
	libDir := filepath.Join(androidDir, "app", "src", "main", "jniLibs", abi.ABI)
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		return err
	}
	lib := filepath.Join(libDir, "libhexago.so")
	build := exec.Command("go", "build", "-buildmode=c-shared", "-o", lib, ".")
	build.Dir = project
	build.Env = append(os.Environ(),
		"CGO_ENABLED=1", "GOOS=android", "GOARCH="+abi.GOARCH, "CC="+quoteIfSpaces(clang))
	build.Stdout, build.Stderr = ctx.Stdout, ctx.Stderr
	if err := build.Run(); err != nil {
		return errors.New("сборка Go не удалась (ошибки выше)")
	}
	os.Remove(strings.TrimSuffix(lib, ".so") + ".h") // заголовок для C нам не нужен

	// 2. Gradle: APK и установка.
	if err := ensureLocalProperties(androidDir, sdkDir); err != nil {
		return err
	}
	ctx.Printf("Сборка APK и установка (Gradle)...\n")
	if err := gradle(androidDir, sdkDir, device.Serial, "installDebug"); err != nil {
		return err
	}

	// 3. Запуск и лог.
	ctx.Printf("Запуск %s...\n", appID)
	sdk.Output(adb, "-s", device.Serial, "logcat", "-c") // старый лог не показываем
	if out, err := sdk.Output(adb, "-s", device.Serial, "shell", "am", "start", "-S", "-n", appID+"/.MainActivity"); err != nil || strings.Contains(out, "Error") {
		return fmt.Errorf("не удалось запустить приложение: %v %s", err, out)
	}
	pid, err := waitPID(adb, device.Serial, appID)
	if err != nil {
		return err
	}
	ctx.Printf("Приложение запущено. Лог приложения ниже, Ctrl+C — выход.\n\n")
	return streamLogs(ctx, adb, device.Serial, appID, pid)
}

// pickDevice выбирает устройство: заданное флагом -d или единственное подключённое.
func pickDevice(adb, serial string) (sdk.Device, error) {
	devices, err := sdk.Devices(adb)
	if err != nil {
		return sdk.Device{}, err
	}
	var ready []sdk.Device
	for _, d := range devices {
		if d.Ready() {
			ready = append(ready, d)
		}
	}
	if serial != "" {
		for _, d := range ready {
			if d.Serial == serial {
				return d, nil
			}
		}
		return sdk.Device{}, fmt.Errorf("устройство %s не подключено (список: hexago test)", serial)
	}
	switch len(ready) {
	case 0:
		return sdk.Device{}, errors.New("нет подключённых устройств: запусти эмулятор или подключи телефон (hexago test подскажет, как)")
	case 1:
		return ready[0], nil
	default:
		var names []string
		for _, d := range ready {
			names = append(names, d.Serial)
		}
		return sdk.Device{}, fmt.Errorf("подключено несколько устройств (%s): выбери одно флагом -d", strings.Join(names, ", "))
	}
}

var applicationIDRe = regexp.MustCompile(`applicationId\s*=\s*"([^"]+)"`)

// applicationID читает идентификатор приложения из android/app/build.gradle.kts.
func applicationID(androidDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(androidDir, "app", "build.gradle.kts"))
	if err != nil {
		return "", err
	}
	m := applicationIDRe.FindSubmatch(data)
	if m == nil {
		return "", errors.New("в android/app/build.gradle.kts нет applicationId")
	}
	return string(m[1]), nil
}

// ensureLocalProperties записывает путь к Android SDK в android/local.properties,
// если файла ещё нет. Так Gradle и Android Studio находят SDK без ANDROID_HOME.
func ensureLocalProperties(androidDir, sdkDir string) error {
	path := filepath.Join(androidDir, "local.properties")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte("sdk.dir="+filepath.ToSlash(sdkDir)+"\n"), 0o644)
}

// gradle запускает обёртку Gradle проекта. Вывод показываем только при ошибке:
// при успехе Gradle печатает предупреждения, которые пользователю не нужны.
func gradle(androidDir, sdkDir, serial string, tasks ...string) error {
	wrapper := filepath.Join(androidDir, "gradlew")
	if runtime.GOOS == "windows" {
		wrapper += ".bat"
	}
	cmd := exec.Command(wrapper, append(tasks, "-q", "--console=plain")...)
	cmd.Dir = androidDir
	// ANDROID_SERIAL — на какое устройство ставить, если их несколько.
	cmd.Env = append(os.Environ(), "ANDROID_HOME="+sdkDir, "ANDROID_SERIAL="+serial)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Gradle: %v\n%s", err, out.String())
	}
	return nil
}

// waitPID ждёт, пока процесс приложения появится, и возвращает его номер.
func waitPID(adb, serial, appID string) (string, error) {
	for range 50 {
		if pid, err := sdk.Output(adb, "-s", serial, "shell", "pidof", appID); err == nil && pid != "" {
			return strings.Fields(pid)[0], nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return "", errors.New("приложение запустилось, но процесс не найден — посмотри лог: adb logcat")
}

// streamLogs печатает лог приложения, пока оно работает или пока не нажали Ctrl+C.
// Показываем только свой тег hexago (туда движок пишет fmt.Println и паники)
// и падения Java (AndroidRuntime).
func streamLogs(ctx *cli.Context, adb, serial, appID, pid string) error {
	c, cancel := context.WithCancel(context.Background())
	defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)

	logcat := exec.CommandContext(c, adb, "-s", serial, "logcat", "-v", "raw",
		"--pid="+pid, "-s", "hexago:V", "AndroidRuntime:E")
	pipe, err := logcat.StdoutPipe()
	if err != nil {
		return err
	}
	if err := logcat.Start(); err != nil {
		return err
	}

	// Приложение закрыли или оно упало — logcat сам не завершится, следим за процессом.
	exited := make(chan struct{})
	go func() {
		for {
			select {
			case <-c.Done():
				return
			case <-time.After(time.Second):
			}
			if now, err := sdk.Output(adb, "-s", serial, "shell", "pidof", appID); err != nil || !strings.Contains(" "+now+" ", " "+pid+" ") {
				close(exited)
				return
			}
		}
	}()

	lines := bufio.NewScanner(pipe)
	done := make(chan struct{})
	go func() {
		for lines.Scan() {
			if line := lines.Text(); !strings.HasPrefix(line, "--------- beginning of") {
				ctx.Printf("%s\n", line)
			}
		}
		close(done)
	}()

	select {
	case <-interrupt:
		ctx.Printf("\nОстановлено. Приложение продолжает работать на устройстве.\n")
	case <-exited:
		time.Sleep(300 * time.Millisecond) // дочитать последние строки лога
		ctx.Printf("\nПриложение завершилось.\n")
		// Если Go-код упал, движок сохранил текст паники в кеше приложения.
		// run-as читает файлы отладочного приложения без root.
		crash, err := sdk.Output(adb, "-s", serial, "shell", "run-as", appID, "cat", "cache/"+crashFile)
		if err == nil && strings.TrimSpace(crash) != "" {
			ctx.Printf("\nПаника в Go-коде:\n%s\n", crash)
		}
	case <-done:
	}
	cancel()
	logcat.Wait()
	return nil
}

// crashFile — файл с текстом паники в кеше приложения.
// Совпадает с android.CrashFile в движке (утилиты не импортируют движок).
const crashFile = "hexago-crash.txt"

// quoteIfSpaces берёт путь в кавычки, если в нём есть пробелы:
// cgo делит значение CC по пробелам, и путь вида C:\Program Files\... иначе сломается.
func quoteIfSpaces(path string) string {
	if strings.ContainsAny(path, " \t") {
		return `"` + path + `"`
	}
	return path
}
