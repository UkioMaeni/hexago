package creator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePackage(t *testing.T) {
	good := []string{"com.example.app", "com.app.simens.group.canon", "dev.my_company.shop2"}
	bad := []string{"", "app", "com..app", "com.my-app", "com.class.app", "com.2app"}
	for _, p := range good {
		if err := validatePackage(p); err != nil {
			t.Errorf("%q должен подходить: %v", p, err)
		}
	}
	for _, p := range bad {
		if err := validatePackage(p); err == nil {
			t.Errorf("%q не должен подходить", p)
		}
	}
}

func TestModuleName(t *testing.T) {
	cases := map[string]string{
		"myapp":        "myapp",
		"My Project":   "my-project",
		"my  _project": "my-project",
		"my.app":       "my-app",
		"Shop-2":       "shop-2",
	}
	for in, want := range cases {
		if got, err := moduleName(in); err != nil || got != want {
			t.Errorf("moduleName(%q) = %q, %v; ожидали %q", in, got, err, want)
		}
	}
	for _, in := range []string{"Мой проект", "---", "a+b"} {
		if _, err := moduleName(in); err == nil {
			t.Errorf("moduleName(%q) должен вернуть ошибку", in)
		}
	}
}

func TestNewCopiesDotFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app")
	if err := createProjectTemplate(dir); err != nil {
		t.Fatal(err)
	}
	if err := createAndroidTemplate("com.example.app", dir); err != nil {
		t.Fatal(err)
	}
	// go:embed без all: молча пропускает файлы с точкой — проверяем, что они на месте.
	for _, f := range []string{".vscode/launch.json", "android/.gitignore", "main.go", "go.mod"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("в новом проекте нет %s", f)
		}
	}
}
