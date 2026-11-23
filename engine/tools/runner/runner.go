// Package runner — команда hexago run: собрать приложение и запустить его на устройстве.
package runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
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
  -d  на каком устройстве запускать, если подключено несколько: серийный номер, имя телефона
      или имя эмулятора (списки: hexago devices, hexago emulators). Выключенный эмулятор
      hexago запустит сам и дождётся, пока он загрузится.

Примеры:
  hexago run
  hexago run android -d emulator-5554
  hexago run -d Medium_Phone_API_36`,
	Run: run,
}

func run(ctx *cli.Context, args []string) error {
	fs := cli.NewFlagSet("run")
	serial := fs.String("d", "", "")
	// --events — служебные строки для расширения VS Code: приложение запущено, завершилось.
	events := fs.Bool("events", false, "")
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
	return runAndroid(ctx, *serial, *events)
}

func runAndroid(ctx *cli.Context, serial string, events bool) error {
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
	device, err := pickDevice(ctx, sdkDir, adb, serial)
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
	for _, h := range oldProjectHints(androidDir) {
		ctx.Printf("Внимание: %s\n", h)
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
	if events {
		// В редакторе останавливают кнопкой, а не Ctrl+C.
		ctx.Printf("Приложение запущено. Лог приложения ниже.\n")
		emit(ctx, "started", map[string]string{"app": appID, "pid": pid, "device": device.Serial, "name": device.Name})
		defer emit(ctx, "stopped", nil)
	} else {
		ctx.Printf("Приложение запущено. Лог приложения ниже, Ctrl+C — выход.\n\n")
	}
	return streamLogs(ctx, adb, device.Serial, appID, pid)
}

// pickDevice выбирает устройство: заданное флагом -d или единственное подключённое.
// В -d можно указать серийный номер, имя устройства или имя эмулятора. Выключенный эмулятор
// hexago запускает сам, а загружающееся устройство — ждёт.
func pickDevice(ctx *cli.Context, sdkDir, adb, query string) (sdk.Device, error) {
	tracker := &sdk.DeviceTracker{ADB: adb}
	devices, err := tracker.List()
	if err != nil {
		return sdk.Device{}, err
	}
	if query != "" {
		if d, ok := matchDevice(devices, query); ok {
			return waitReady(ctx, tracker, d.Name, func(x sdk.Device) bool { return x.Serial == d.Serial })
		}
		for _, avd := range sdk.AVDs(sdkDir) {
			if strings.EqualFold(avd, query) {
				ctx.Printf("Запуск эмулятора %s...\n", avd)
				if err := sdk.LaunchEmulator(sdkDir, avd); err != nil {
					return sdk.Device{}, fmt.Errorf("не удалось запустить эмулятор %s: %v", avd, err)
				}
				return waitReady(ctx, tracker, sdk.AVDLabel(avd), func(x sdk.Device) bool { return x.AVD == avd })
			}
		}
		return sdk.Device{}, fmt.Errorf("устройство %q не найдено (список: hexago devices, эмуляторы: hexago emulators)", query)
	}

	var ready, booting []sdk.Device
	for _, d := range devices {
		switch {
		case d.Ready():
			ready = append(ready, d)
		case d.State == sdk.StateBooting:
			booting = append(booting, d)
		}
	}
	switch {
	case len(ready) == 1:
		return ready[0], nil
	case len(ready) > 1:
		var names []string
		for _, d := range ready {
			names = append(names, fmt.Sprintf("%s (%s)", d.Name, d.Serial))
		}
		return sdk.Device{}, fmt.Errorf("подключено несколько устройств: %s — выбери одно: hexago run -d <устройство>",
			strings.Join(names, ", "))
	case len(booting) > 0:
		d := booting[0]
		return waitReady(ctx, tracker, d.Name, func(x sdk.Device) bool { return x.Serial == d.Serial })
	default:
		return sdk.Device{}, errors.New("нет подключённых устройств: запусти эмулятор (hexago emulators) или подключи телефон по USB")
	}
}

// matchDevice ищет устройство по серийному номеру, имени или имени эмулятора, без учёта регистра.
func matchDevice(devices []sdk.Device, query string) (sdk.Device, bool) {
	for _, d := range devices {
		if strings.EqualFold(d.Serial, query) || strings.EqualFold(d.Name, query) ||
			(d.AVD != "" && strings.EqualFold(d.AVD, query)) {
			return d, true
		}
	}
	return sdk.Device{}, false
}

// bootTimeout — сколько ждать устройство: холодный старт эмулятора занимает минуту-две.
const bootTimeout = 3 * time.Minute

// waitReady ждёт, пока устройство появится и будет готово принять приложение.
func waitReady(ctx *cli.Context, tracker *sdk.DeviceTracker, name string, match func(sdk.Device) bool) (sdk.Device, error) {
	said := ""
	for deadline := time.Now().Add(bootTimeout); time.Now().Before(deadline); time.Sleep(2 * time.Second) {
		devices, err := tracker.List()
		if err != nil {
			return sdk.Device{}, err
		}
		i := slices.IndexFunc(devices, match)
		state := "absent"
		if i >= 0 {
			if devices[i].Ready() {
				return devices[i], nil
			}
			state = devices[i].State
		}
		if state != said {
			said = state
			switch state {
			case "unauthorized":
				ctx.Printf("%s не разрешило отладку — подтверди запрос на экране телефона...\n", name)
			default:
				ctx.Printf("Ждём, пока %s загрузится...\n", name)
			}
		}
	}
	return sdk.Device{}, fmt.Errorf("%s так и не стало готово за %v", name, bootTimeout)
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
	exited := make(chan string, 1)
	go func() {
		if reason := watchProcess(c, adb, serial, appID, pid); reason != "" {
			exited <- reason
		}
	}()

	// Если logcat прервётся (сбой adb), сессию не заканчиваем: она идёт, пока живо приложение.
	lines := bufio.NewScanner(pipe)
	go func() {
		for lines.Scan() {
			if line := lines.Text(); !strings.HasPrefix(line, "--------- beginning of") {
				ctx.Printf("%s\n", line)
			}
		}
	}()

	select {
	case <-interrupt:
		ctx.Printf("\nОстановлено. Приложение продолжает работать на устройстве.\n")
	case reason := <-exited:
		time.Sleep(300 * time.Millisecond) // дочитать последние строки лога
		ctx.Printf("\n%s\n", reason)
		// Если Go-код упал, движок сохранил текст паники в кеше приложения.
		// run-as читает файлы отладочного приложения без root.
		crash, err := sdk.Output(adb, "-s", serial, "shell", "run-as", appID, "cat", "cache/"+crashFile)
		if err == nil && strings.TrimSpace(crash) != "" {
			ctx.Printf("\nПаника в Go-коде:\n%s\n", crash)
		}
	}
	cancel()
	logcat.Wait()
	return nil
}

// processState — что ответило устройство на вопрос, жив ли процесс приложения.
type processState int

const (
	processAlive   processState = iota // pidof нашёл наш процесс
	processGone                        // pidof ответил: такого процесса нет
	processUnknown                     // ответа нет: adb занят, устройство отключилось
)

// checkProcess спрашивает у устройства, жив ли процесс pid приложения appID.
// pidof без находок завершается с кодом 1 и ничего не печатает; если же что-то напечатано,
// а нашего pid там нет, — это ошибка adb («error: closed», «device offline»), а не ответ pidof.
func checkProcess(out string, err error, pid string) processState {
	switch {
	case strings.Contains(" "+strings.Join(strings.Fields(out), " ")+" ", " "+pid+" "):
		return processAlive
	case err == nil || strings.TrimSpace(out) == "":
		return processGone // нет вывода — pidof ничего не нашёл; есть другой pid — приложение перезапустили
	default:
		return processUnknown
	}
}

// watchProcess раз в секунду проверяет, жив ли процесс приложения, и возвращает, почему перестал
// следить: приложение завершилось или устройство пропало. Пустая строка — остановили через c.
//
// Одной неудачной проверке не верим: сразу после загрузки эмулятор занят, и adb shell иногда
// отвечает ошибкой, хотя приложение работает. Нужны два ответа «процесса нет» подряд.
func watchProcess(c context.Context, adb, serial, appID, pid string) string {
	gone, unknown := 0, 0
	for {
		select {
		case <-c.Done():
			return ""
		case <-time.After(time.Second):
		}
		out, err := sdk.Output(adb, "-s", serial, "shell", "pidof", appID)
		switch checkProcess(out, err, pid) {
		case processAlive:
			gone, unknown = 0, 0
		case processGone:
			if gone++; gone >= 2 {
				return "Приложение завершилось."
			}
		case processUnknown:
			if unknown++; unknown >= 15 {
				return fmt.Sprintf("Устройство %s не отвечает — похоже, оно отключилось (%s).", serial, strings.TrimSpace(out))
			}
		}
	}
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

// EventPrefix начинает служебные строки hexago run --events. Расширение VS Code вырезает их
// из вывода и по ним показывает запущенное приложение на панели отладки.
const EventPrefix = "hexago-event: "

// emit печатает служебную строку: EventPrefix и JSON {"event": ..., поля}.
func emit(ctx *cli.Context, event string, fields map[string]string) {
	m := map[string]string{"event": event}
	for k, v := range fields {
		m[k] = v
	}
	data, _ := json.Marshal(m)
	ctx.Printf("%s%s\n", EventPrefix, data)
}

// oldProjectHints — чего не хватает проекту, созданному старой версией hexago create:
// сами файлы проекта hexago не правит, только подсказывает.
func oldProjectHints(androidDir string) []string {
	var hints []string
	gradle, _ := os.ReadFile(filepath.Join(androidDir, "app", "build.gradle.kts"))
	if !bytes.Contains(gradle, []byte("assets.srcDir")) {
		hints = append(hints, `файлы из assets/ не попадут в APK. Добавь в android/app/build.gradle.kts, `+
			`в sourceSets { getByName("main") { … } }, строку: assets.srcDir("../../assets")`)
	}
	manifest, _ := os.ReadFile(filepath.Join(androidDir, "app", "src", "main", "AndroidManifest.xml"))
	if !bytes.Contains(manifest, []byte("android.permission.INTERNET")) {
		hints = append(hints, `приложению недоступна сеть (картинки по адресу, запросы). Добавь в `+
			`android/app/src/main/AndroidManifest.xml строку: <uses-permission android:name="android.permission.INTERNET" />`)
	}
	return hints
}
