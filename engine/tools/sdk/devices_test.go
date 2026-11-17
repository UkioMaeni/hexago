package sdk

import "testing"

func TestParseDevices(t *testing.T) {
	out := `* daemon started successfully
List of devices attached
emulator-5554          device product:sdk_gphone64_x86_64 model:sdk_gphone64_x86_64 device:emu64xa transport_id:1
R58M12ABCDE            unauthorized usb:1-1 transport_id:2
0123456789             offline

`
	list := parseDevices(out)
	if len(list) != 3 {
		t.Fatalf("устройств %d, ожидали 3: %+v", len(list), list)
	}
	if d := list[0]; d.Serial != "emulator-5554" || !d.Ready() || !d.Emulator() || d.Name != "sdk gphone64 x86 64" {
		t.Errorf("эмулятор разобран неверно: %+v", d)
	}
	if d := list[1]; d.State != "unauthorized" || d.Ready() || d.Emulator() || d.Name != "R58M12ABCDE" {
		t.Errorf("телефон без разрешения разобран неверно: %+v", d)
	}
	if d := list[2]; d.State != "offline" {
		t.Errorf("отключённое устройство разобрано неверно: %+v", d)
	}
}

func TestParseProps(t *testing.T) {
	// Пустые значения (свойство ещё не задано) не должны сдвигать остальные.
	props := parseProps("ro.product.cpu.abi=\r\nro.build.version.release=16\r\nsys.boot_completed=\r\nro.product.model=Pixel 7\n")
	want := map[string]string{
		"ro.product.cpu.abi": "", "ro.build.version.release": "16", "sys.boot_completed": "", "ro.product.model": "Pixel 7",
	}
	for k, v := range want {
		if got, ok := props[k]; !ok || got != v {
			t.Errorf("%s = %q, ожидали %q", k, got, v)
		}
	}
}

func TestAVDLabel(t *testing.T) {
	if got := AVDLabel("Medium_Phone_API_36.1"); got != "Medium Phone API 36.1" {
		t.Errorf("AVDLabel = %q", got)
	}
}
