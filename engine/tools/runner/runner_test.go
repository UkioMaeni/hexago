package runner

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"hexago/tools/sdk"
)

func TestMatchDevice(t *testing.T) {
	devices := []sdk.Device{
		{Serial: "emulator-5554", State: "device", Name: "Medium Phone API 36", AVD: "Medium_Phone_API_36"},
		{Serial: "R58M12ABCDE", State: "device", Name: "SM-G991B"},
	}
	cases := map[string]string{
		"emulator-5554":       "emulator-5554",
		"medium_phone_api_36": "emulator-5554", // имя эмулятора, без учёта регистра
		"Medium Phone API 36": "emulator-5554", // имя для людей
		"sm-g991b":            "R58M12ABCDE",
	}
	for query, serial := range cases {
		if d, ok := matchDevice(devices, query); !ok || d.Serial != serial {
			t.Errorf("matchDevice(%q) = %q, %v; ожидали %q", query, d.Serial, ok, serial)
		}
	}
	if _, ok := matchDevice(devices, "Pixel_7"); ok {
		t.Error("Pixel_7 не подключён — не должен находиться")
	}
}

func TestCheckProcess(t *testing.T) {
	errExit := errors.New("exit status 1")
	cases := []struct {
		out  string
		err  error
		want processState
	}{
		{"12994", nil, processAlive},
		{"12001 12994\r", nil, processAlive},
		{"", errExit, processGone},                 // pidof ничего не нашёл
		{"13175", nil, processGone},                // приложение перезапустили — это уже другой процесс
		{"error: closed", errExit, processUnknown}, // сбой adb — не повод считать, что приложение закрылось
		{"error: device offline", errExit, processUnknown},
	}
	for _, c := range cases {
		if got := checkProcess(c.out, c.err, "12994"); got != c.want {
			t.Errorf("checkProcess(%q, %v) = %v; ожидали %v", c.out, c.err, got, c.want)
		}
	}
}

func TestOldProjectHints(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) {
		p := filepath.Join(dir, "app", name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("build.gradle.kts", `java.srcDir("x")`)
	write(filepath.Join("src", "main", "AndroidManifest.xml"), `<manifest></manifest>`)
	if h := oldProjectHints(dir); len(h) != 2 {
		t.Fatalf("старый проект: %q", h)
	}
	write("build.gradle.kts", `java.srcDir("x")`+"\n"+`assets.srcDir("../../assets")`)
	write(filepath.Join("src", "main", "AndroidManifest.xml"), `<uses-permission android:name="android.permission.INTERNET" />`)
	if h := oldProjectHints(dir); len(h) != 0 {
		t.Errorf("новый проект: %q", h)
	}
}
