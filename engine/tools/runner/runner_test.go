package runner

import (
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
