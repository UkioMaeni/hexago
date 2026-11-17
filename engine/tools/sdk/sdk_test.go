package sdk

import "testing"

func TestCompareVersions(t *testing.T) {
	if compareVersions("28.2.13676358", "26.3.11579264") <= 0 {
		t.Error("28.2 должна быть новее 26.3")
	}
	if compareVersions("10.0", "9.9") <= 0 {
		t.Error("10.0 должна быть новее 9.9: сравнение по числам, а не по строкам")
	}
	if compareVersions("1.2", "1.2.0") != 0 {
		t.Error("1.2 и 1.2.0 — одна версия")
	}
}

func TestForABI(t *testing.T) {
	cases := map[string]string{"arm64-v8a": "arm64", "x86_64": "amd64", "armeabi-v7a": "arm", "x86": "386"}
	for abi, goarch := range cases {
		if got, ok := ForABI(abi); !ok || got.GOARCH != goarch {
			t.Errorf("ForABI(%q) = %q; ожидали %q", abi, got.GOARCH, goarch)
		}
	}
	if _, ok := ForABI("mips"); ok {
		t.Error("mips не поддерживается")
	}
}

func TestEnginePath(t *testing.T) {
	dir, err := EnginePath()
	if err != nil {
		t.Fatal(err)
	}
	if v, err := EngineGoVersion(dir); err != nil || v == "" {
		t.Errorf("EngineGoVersion(%s) = %q, %v", dir, v, err)
	}
}
