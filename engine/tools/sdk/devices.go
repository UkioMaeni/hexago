package sdk

import (
	"cmp"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

// Device — подключённый телефон или запущенный эмулятор.
type Device struct {
	Serial  string // emulator-5554, R58M12ABCDE...
	State   string // device — готово; booting — ещё загружается; unauthorized — не разрешена отладка; offline
	Name    string // для людей: модель телефона или имя эмулятора
	AVD     string // имя эмулятора (Medium_Phone_API_36); у телефона пусто
	ABI     string // архитектура процессора: arm64-v8a, x86_64
	Release string // версия Android: 16
}

// Состояния устройства, кроме тех, что сообщает сам adb (device, unauthorized, offline...).
const StateBooting = "booting"

// Ready сообщает, можно ли ставить приложение на устройство.
func (d Device) Ready() bool { return d.State == "device" }

// Emulator сообщает, что это эмулятор, а не телефон.
func (d Device) Emulator() bool { return strings.HasPrefix(d.Serial, "emulator-") }

// Devices возвращает список устройств. Для готовых — с архитектурой, версией Android и именем.
func Devices(adb string) ([]Device, error) { return (&DeviceTracker{ADB: adb}).List() }

// DeviceTracker перечисляет устройства много раз подряд (hexago devices --watch, ожидание
// загрузки эмулятора) и не переспрашивает свойства тех, что уже загрузились.
type DeviceTracker struct {
	ADB   string
	known map[string]Device
}

// List возвращает текущий список устройств.
func (t *DeviceTracker) List() ([]Device, error) {
	out, err := Output(t.ADB, "devices", "-l")
	if err != nil {
		return nil, err
	}
	list := parseDevices(out)
	known := make(map[string]Device, len(list))
	for i, d := range list {
		if d.State == "device" {
			if k, ok := t.known[d.Serial]; ok && k.Ready() {
				list[i] = k
			} else {
				list[i] = t.describe(d)
			}
		}
		known[d.Serial] = list[i]
	}
	t.known = known
	return list, nil
}

// deviceProps — свойства, которые hexago спрашивает у устройства одним вызовом adb shell.
var deviceProps = []string{
	"ro.product.cpu.abi", "ro.build.version.release", "ro.product.model", "sys.boot_completed",
	"ro.boot.qemu.avd_name", "ro.kernel.qemu.avd_name",
}

// describe дополняет устройство свойствами из getprop.
func (t *DeviceTracker) describe(d Device) Device {
	script := "for p in " + strings.Join(deviceProps, " ") + `; do echo "$p=$(getprop $p)"; done`
	out, err := Output(t.ADB, "-s", d.Serial, "shell", script)
	if err != nil {
		d.State = "offline"
		return d
	}
	props := parseProps(out)
	d.ABI = props["ro.product.cpu.abi"]
	d.Release = props["ro.build.version.release"]
	if props["sys.boot_completed"] != "1" {
		d.State = StateBooting
	}
	if model := props["ro.product.model"]; model != "" {
		d.Name = model
	}
	if d.Emulator() {
		d.AVD = cmp.Or(props["ro.boot.qemu.avd_name"], props["ro.kernel.qemu.avd_name"])
		if d.AVD == "" {
			// Старые образы не знают своего имени — спросим консоль эмулятора.
			if out, err := Output(t.ADB, "-s", d.Serial, "emu", "avd", "name"); err == nil {
				d.AVD, _, _ = strings.Cut(strings.TrimSpace(out), "\n")
				d.AVD = strings.TrimSpace(d.AVD)
			}
		}
		if d.AVD != "" {
			d.Name = AVDLabel(d.AVD)
		}
	}
	return d
}

// parseDevices разбирает вывод adb devices -l.
func parseDevices(out string) []Device {
	var list []Device
	for line := range strings.Lines(out) {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.HasPrefix(line, "List of devices") || strings.HasPrefix(line, "*") {
			continue
		}
		d := Device{Serial: fields[0], State: fields[1]}
		for _, f := range fields[2:] {
			if model, ok := strings.CutPrefix(f, "model:"); ok {
				d.Name = strings.ReplaceAll(model, "_", " ")
			}
		}
		if d.Name == "" {
			d.Name = d.Serial
		}
		list = append(list, d)
	}
	return list
}

// parseProps разбирает строки «имя=значение».
func parseProps(out string) map[string]string {
	props := map[string]string{}
	for line := range strings.Lines(out) {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
			props[k] = strings.TrimSpace(v)
		}
	}
	return props
}

// AVDLabel — имя эмулятора для людей: Medium_Phone_API_36 → Medium Phone API 36.
func AVDLabel(avd string) string { return strings.ReplaceAll(avd, "_", " ") }

// EmulatorPath возвращает путь к программе emulator из Android SDK.
func EmulatorPath(sdkDir string) (string, error) {
	path := filepath.Join(sdkDir, "emulator", exe("emulator"))
	if !isFile(path) {
		return "", errors.New("эмулятор не установлен: Android Studio → SDK Manager → SDK Tools → Android Emulator")
	}
	return path, nil
}

// AVDs возвращает имена созданных эмуляторов (Android Studio → Device Manager).
func AVDs(sdkDir string) []string {
	emulator, err := EmulatorPath(sdkDir)
	if err != nil {
		return nil
	}
	out, err := Output(emulator, "-list-avds")
	if err != nil {
		return nil
	}
	var names []string
	for line := range strings.Lines(out) {
		// Кроме имён эмулятор иногда печатает предупреждения — в них есть пробелы.
		if name := strings.TrimSpace(line); name != "" && !strings.ContainsAny(name, " \t") {
			names = append(names, name)
		}
	}
	return names
}

// LaunchEmulator запускает эмулятор и не ждёт его: окно эмулятора живёт само по себе,
// даже когда hexago уже завершился.
func LaunchEmulator(sdkDir, avd string) error {
	emulator, err := EmulatorPath(sdkDir)
	if err != nil {
		return err
	}
	cmd := exec.Command(emulator, "-avd", avd)
	cmd.Dir = filepath.Dir(emulator)
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
