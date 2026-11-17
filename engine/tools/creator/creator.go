package creator

import (
	"embed"
	"errors"
	"fmt"
	"hexago/tools/sdk"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

//go:embed templates
var templates embed.FS

var segmentRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

var javaKeywords = []string{
	"abstract", "assert", "boolean", "break", "byte", "case", "catch", "char",
	"class", "const", "continue", "default", "do", "double", "else", "enum",
	"extends", "final", "finally", "float", "for", "goto", "if", "implements",
	"import", "instanceof", "int", "interface", "long", "native", "new",
	"package", "private", "protected", "public", "return", "short", "static",
	"strictfp", "super", "switch", "synchronized", "this", "throw", "throws",
	"transient", "try", "void", "volatile", "while",
	"true", "false", "null", // не ключевые слова, но тоже запрещены как имена
}

// moduleNameRe — символы, допустимые в имени папки проекта:
// латинские буквы, цифры и разделители (пробел, _, . и -).
// Дефис стоит последним в [...], поэтому означает сам себя, а не диапазон.
// Точка внутри [...] — тоже просто точка, а не «любой символ».
var moduleNameRe = regexp.MustCompile(`^[a-zA-Z0-9 _.-]+$`)

// CreatorMeta — что нужно знать, чтобы создать проект.
type CreatorMeta struct {
	InitDir string // папка проекта: "." или имя новой папки
	Package string // идентификатор Android-приложения: com.example.newapp
}

// New создаёт проект: Go-часть и Android-хост.
func New(meta CreatorMeta) error {
	// Пакет проверяем до того, как что-то создано, — иначе при ошибке
	// на диске остался бы полусделанный проект без android/.
	if err := validatePackage(meta.Package); err != nil {
		return err
	}
	err := createProjectTemplate(meta.InitDir)
	if err != nil {
		return err // проект не создался — Android-часть не делаем
	}
	if err := createAndroidTemplate(meta.Package, meta.InitDir); err != nil {
		return err
	}
	// Подтянуть модули, которые нужны движку, — иначе первая сборка попросит go mod tidy.
	return sdk.Tidy(meta.InitDir)
}

// validatePackage проверяет идентификатор Android-приложения: минимум две части через точку,
// каждая — латинская буква, затем буквы, цифры и _, и не ключевое слово Java.
func validatePackage(pkg string) error {
	parts := strings.Split(pkg, ".")
	if len(parts) < 2 {
		return fmt.Errorf("пакет %q: нужно минимум две части через точку, например com.example.app", pkg)
	}
	for _, part := range parts {
		if !segmentRe.MatchString(part) || slices.Contains(javaKeywords, part) {
			return fmt.Errorf("пакет %q: неверная часть %q", pkg, part)
		}
	}
	return nil
}

func createProjectTemplate(initDir string) error {

	abs, err := filepath.Abs(initDir)
	if err != nil {
		return err
	}
	name, err := moduleName(filepath.Base(abs))
	if err != nil {
		return err
	}
	engine, err := sdk.EnginePath()
	if err != nil {
		return err
	}
	goVersion, err := sdk.EngineGoVersion(engine)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(initDir), 0o755)
	if err != nil {
		return err
	}
	goModTmpl, err := templates.ReadFile("templates/project/go.mod.tmpl")
	if err != nil {
		return err
	}
	mainGoTmpl, err := templates.ReadFile("templates/project/main.go.tmpl")
	if err != nil {
		return err
	}
	text := strings.ReplaceAll(string(goModTmpl), "{{.Name}}", name)
	text = strings.ReplaceAll(text, "{{.EnginePath}}", engine)
	text = strings.ReplaceAll(text, "{{.GoVersion}}", goVersion)
	err = os.WriteFile(filepath.Join(initDir, "go.mod"), []byte(text), 0o644)
	if err != nil {
		return err
	}
	text = strings.ReplaceAll(string(mainGoTmpl), "{{.Name}}", name)
	err = os.WriteFile(filepath.Join(initDir, "main.go"), []byte(text), 0o644)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(initDir, "src"), 0o755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(initDir, "assets"), 0o755)
	if err != nil {
		return err
	}
	return err
}

func createAndroidTemplate(packageStr string, initDir string) error {

	pkgSlice := strings.Split(packageStr, ".")
	if len(pkgSlice) < 2 {
		return errors.New("ошибочный формат пакета")
	}
	for _, part := range pkgSlice {
		if !segmentRe.MatchString(part) || slices.Contains(javaKeywords, part) {
			return fmt.Errorf("неверная часть пакета %q", part)
		}

	}
	name := pkgSlice[len(pkgSlice)-1]
	// Папка движка: из неё Gradle возьмёт Java-часть движка (HexagoActivity).
	engine, err := sdk.EnginePath()
	if err != nil {
		return err
	}
	sub, _ := fs.Sub(templates, "templates/android")
	err = os.CopyFS(filepath.Join(initDir, "android"), sub)
	if err != nil {
		return err
	}
	//settings.gradle.kts
	settingsGradleKtsPath := filepath.Join(initDir, "android", "settings.gradle.kts")
	data, err := os.ReadFile(settingsGradleKtsPath)
	if err != nil {
		return err
	}
	text := strings.ReplaceAll(string(data), "{{.Name}}", name)
	err = os.WriteFile(settingsGradleKtsPath, []byte(text), 0o644)
	if err != nil {
		return err
	}
	//app/src/main/res/values/strings.xml
	stringsPath := filepath.Join(initDir, "android", "app", "src", "main", "res", "values", "strings.xml")
	data, err = os.ReadFile(stringsPath)
	if err != nil {
		return err
	}
	text = strings.ReplaceAll(string(data), "{{.Name}}", name)
	err = os.WriteFile(stringsPath, []byte(text), 0o644)
	if err != nil {
		return err
	}
	//app/build.gradle.kts
	buildGradleKtsPath := filepath.Join(initDir, "android", "app", "build.gradle.kts")
	data, err = os.ReadFile(buildGradleKtsPath)
	if err != nil {
		return err
	}
	text = strings.ReplaceAll(string(data), "{{.ApplicationID}}", packageStr)
	text = strings.ReplaceAll(text, "{{.EnginePath}}", engine)
	err = os.WriteFile(buildGradleKtsPath, []byte(text), 0o644)
	if err != nil {
		return err
	}
	//app/src/main/java/PACKAGE/MainActivity.java
	mainActivityPath := filepath.Join(initDir, "android", "app", "src", "main", "java", "PACKAGE", "MainActivity.java")
	data, err = os.ReadFile(mainActivityPath)
	if err != nil {
		return err
	}

	newMainActivityFolder := filepath.Join(initDir, "android", "app", "src", "main", "java")
	for i := range pkgSlice {
		newMainActivityFolder = filepath.Join(newMainActivityFolder, pkgSlice[i])
	}

	text = strings.ReplaceAll(string(data), "{{.ApplicationID}}", packageStr)
	text = strings.ReplaceAll(text, "{{.Name}}", name)
	err = os.MkdirAll(newMainActivityFolder, 0o755)
	if err != nil {
		return err
	}
	newMainActivityPath := filepath.Join(newMainActivityFolder, "MainActivity.java")
	err = os.WriteFile(newMainActivityPath, []byte(text), 0o644)
	if err != nil {
		return err
	}
	err = os.RemoveAll(filepath.Join(initDir, "android", "app", "src", "main", "java", "PACKAGE"))
	return err

}

// moduleName превращает имя папки в имя Go-модуля: "My Project" → "my-project".
func moduleName(folder string) (string, error) {
	// 1. Проверка: только латиница, цифры и разделители.
	if !moduleNameRe.MatchString(folder) {
		return "", fmt.Errorf("имя папки %q не подходит: нужны латинские буквы, цифры, пробел, _, . или -", folder)
	}

	// 2–3. Нижний регистр и разбивка на слова. FieldsFunc режет там, где функция
	// вернула true, и сам выбрасывает пустые куски: "my  _project" → [my project].
	words := strings.FieldsFunc(strings.ToLower(folder), func(r rune) bool {
		return r == ' ' || r == '_' || r == '.' || r == '-'
	})

	// Крайний случай: папка из одних разделителей, например "---".
	if len(words) == 0 {
		return "", fmt.Errorf("в имени папки %q нет ни одной буквы или цифры", folder)
	}

	// 4. Склеиваем слова через дефис.
	return strings.Join(words, "-"), nil
}
