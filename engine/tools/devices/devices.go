// Package devices — команды hexago devices и hexago emulators: на чём запускать приложение.
package devices

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"hexago/tools/cli"
	"hexago/tools/sdk"
)

// DevicesCommand — hexago devices.
var DevicesCommand = &cli.Command{
	Name:    "devices",
	Usage:   "devices [--json] [--watch]",
	Summary: "показать подключённые телефоны и запущенные эмуляторы",
	Help: `
Показывает устройства, на которых можно запустить приложение (hexago run -d <устройство>).

Флаги:
  --json   вывод в JSON — для редакторов и скриптов
  --watch  не завершаться, а печатать список заново при каждом изменении
           (с --json — одна строка JSON на изменение)`,
	Run: runDevices,
}

// EmulatorsCommand — hexago emulators.
var EmulatorsCommand = &cli.Command{
	Name:    "emulators",
	Usage:   "emulators [--json] [--launch имя]",
	Summary: "показать и запустить эмуляторы Android",
	Help: `
Показывает эмуляторы, созданные в Android Studio (Device Manager), и запускает их.

Флаги:
  --launch  запустить эмулятор с этим именем; hexago не ждёт, пока он загрузится
  --json    вывод в JSON — для редакторов и скриптов

Пример:
  hexago emulators --launch Medium_Phone_API_36`,
	Run: runEmulators,
}

// Device — устройство в выводе --json.
type Device struct {
	ID        string `json:"id"`            // серийный номер: его принимает hexago run -d
	Name      string `json:"name"`          // для людей
	State     string `json:"state"`         // device, booting, unauthorized, offline...
	Emulator  bool   `json:"emulator"`      // эмулятор, а не телефон
	AVD       string `json:"avd,omitempty"` // имя эмулятора
	Android   string `json:"android,omitempty"`
	ABI       string `json:"abi,omitempty"`
	Supported bool   `json:"supported"` // архитектуру поддерживает Hexago
}

// Emulator — эмулятор в выводе --json.
type Emulator struct {
	Name    string `json:"name"`              // имя AVD: его принимают --launch и hexago run -d
	Label   string `json:"label"`             // для людей
	Running string `json:"running,omitempty"` // серийный номер, если эмулятор уже запущен
}

func toJSON(d sdk.Device) Device {
	_, supported := sdk.ForABI(d.ABI)
	return Device{
		ID: d.Serial, Name: d.Name, State: d.State, Emulator: d.Emulator(), AVD: d.AVD,
		Android: d.Release, ABI: d.ABI, Supported: supported || d.ABI == "",
	}
}

func runDevices(ctx *cli.Context, args []string) error {
	fs := cli.NewFlagSet("devices")
	asJSON := fs.Bool("json", false, "")
	watch := fs.Bool("watch", false, "")
	rest, err := cli.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return cli.Usagef("лишние аргументы: %s", strings.Join(rest, " "))
	}
	sdkDir, err := sdk.AndroidSDK()
	if err != nil {
		return err
	}
	adb, err := sdk.ADB(sdkDir)
	if err != nil {
		return err
	}
	tracker := &sdk.DeviceTracker{ADB: adb}
	if !*watch {
		list, err := tracker.List()
		if err != nil {
			return err
		}
		return printDevices(ctx.Stdout, list, *asJSON)
	}

	// Редактор, который нас запустил, закрывает stdin, когда завершается, — тогда выходим и мы,
	// чтобы не опрашивать adb вечно.
	go func() {
		io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}()
	var last string
	for ; ; time.Sleep(2 * time.Second) {
		list, err := tracker.List()
		if err != nil {
			// adb мог перезапускаться — сообщаем и пробуем снова.
			fmt.Fprintln(ctx.Stderr, err)
			continue
		}
		var buf strings.Builder
		printDevices(&buf, list, *asJSON)
		if s := buf.String(); s != last {
			last = s
			if !*asJSON {
				fmt.Fprintf(ctx.Stdout, "— %s —\n", time.Now().Format("15:04:05"))
			}
			io.WriteString(ctx.Stdout, s)
		}
	}
}

func printDevices(w io.Writer, list []sdk.Device, asJSON bool) error {
	if asJSON {
		out := make([]Device, 0, len(list))
		for _, d := range list {
			out = append(out, toJSON(d))
		}
		data, err := json.Marshal(out)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "%s\n", data)
		return err
	}
	if len(list) == 0 {
		fmt.Fprintln(w, "Нет подключённых устройств.")
		fmt.Fprintln(w, "Запусти эмулятор (список: hexago emulators) или подключи телефон по USB с включённой отладкой.")
		return nil
	}
	fmt.Fprintf(w, "Устройства (%d):\n", len(list))
	tw := newTable(w)
	for _, d := range list {
		kind := "телефон"
		if d.Emulator() {
			kind = "эмулятор"
		}
		details := []string{kind}
		if d.Release != "" {
			details = append(details, "Android "+d.Release)
		}
		if d.ABI != "" {
			details = append(details, d.ABI)
		}
		switch d.State {
		case "device":
		case sdk.StateBooting:
			details = append(details, "загружается")
		case "unauthorized":
			details = append(details, "отладка не разрешена — подтверди запрос на телефоне")
		default:
			details = append(details, d.State)
		}
		tw.row(d.Name, d.Serial, strings.Join(details, ", "))
	}
	tw.flush()
	return nil
}

func runEmulators(ctx *cli.Context, args []string) error {
	fs := cli.NewFlagSet("emulators")
	asJSON := fs.Bool("json", false, "")
	launch := fs.String("launch", "", "")
	rest, err := cli.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return cli.Usagef("лишние аргументы: %s", strings.Join(rest, " "))
	}
	sdkDir, err := sdk.AndroidSDK()
	if err != nil {
		return err
	}
	if _, err := sdk.EmulatorPath(sdkDir); err != nil {
		return err
	}
	emulators := List(sdkDir)

	if *launch != "" {
		i := slices.IndexFunc(emulators, func(e Emulator) bool { return strings.EqualFold(e.Name, *launch) })
		if i < 0 {
			return fmt.Errorf("эмулятора %q нет (список: hexago emulators)", *launch)
		}
		e := emulators[i]
		if e.Running != "" {
			ctx.Printf("Эмулятор %s уже запущен: %s\n", e.Name, e.Running)
			return nil
		}
		if err := sdk.LaunchEmulator(sdkDir, e.Name); err != nil {
			return fmt.Errorf("не удалось запустить эмулятор %s: %v", e.Name, err)
		}
		ctx.Printf("Эмулятор %s запускается. Когда загрузится, он появится в hexago devices.\n", e.Name)
		return nil
	}

	if *asJSON {
		data, err := json.Marshal(emulators)
		if err != nil {
			return err
		}
		ctx.Printf("%s\n", data)
		return nil
	}
	if len(emulators) == 0 {
		ctx.Printf("Эмуляторов нет. Создай его в Android Studio: Device Manager → Create Virtual Device.\n")
		return nil
	}
	ctx.Printf("Эмуляторы (%d):\n", len(emulators))
	tw := newTable(ctx.Stdout)
	for _, e := range emulators {
		state := ""
		if e.Running != "" {
			state = "запущен: " + e.Running
		}
		tw.row(e.Name, state)
	}
	tw.flush()
	ctx.Printf("\nЗапуск: hexago emulators --launch <имя>\n")
	return nil
}

// List возвращает эмуляторы и отмечает уже запущенные.
func List(sdkDir string) []Emulator {
	running := map[string]string{}
	if adb, err := sdk.ADB(sdkDir); err == nil {
		if list, err := sdk.Devices(adb); err == nil {
			for _, d := range list {
				if d.AVD != "" {
					running[d.AVD] = d.Serial
				}
			}
		}
	}
	emulators := []Emulator{}
	for _, name := range sdk.AVDs(sdkDir) {
		emulators = append(emulators, Emulator{Name: name, Label: sdk.AVDLabel(name), Running: running[name]})
	}
	return emulators
}

// table печатает колонки, выровненные по ширине. Ширину считаем в символах, а не байтах:
// text/tabwriter сбивается на кириллице.
type table struct {
	w    io.Writer
	rows [][]string
}

func newTable(w io.Writer) *table { return &table{w: w} }

func (t *table) row(cells ...string) { t.rows = append(t.rows, cells) }

func (t *table) flush() {
	var widths []int
	for _, r := range t.rows {
		for i, c := range r {
			if i >= len(widths) {
				widths = append(widths, 0)
			}
			widths[i] = max(widths[i], len([]rune(c)))
		}
	}
	var b strings.Builder
	for _, r := range t.rows {
		var line strings.Builder
		line.WriteString(" ")
		for i, c := range r {
			line.WriteString(" " + c + strings.Repeat(" ", widths[i]-len([]rune(c))+2))
		}
		b.WriteString(strings.TrimRight(line.String(), " ") + "\n")
	}
	io.WriteString(t.w, b.String())
}
