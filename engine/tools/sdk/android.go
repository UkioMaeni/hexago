package sdk

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

// AndroidSDK ищет корень Android SDK: сначала переменные ANDROID_HOME и ANDROID_SDK_ROOT,
// потом стандартное место, куда его ставит Android Studio.
func AndroidSDK() (string, error) {
	for _, env := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		if dir := os.Getenv(env); dir != "" {
			if isDir(dir) {
				return dir, nil
			}
			return "", fmt.Errorf("%s=%s, но такой папки нет", env, dir)
		}
	}

	home, _ := os.UserHomeDir()
	var dir string
	switch runtime.GOOS {
	case "windows":
		dir = filepath.Join(os.Getenv("LOCALAPPDATA"), "Android", "Sdk")
	case "darwin":
		dir = filepath.Join(home, "Library", "Android", "sdk")
	default:
		dir = filepath.Join(home, "Android", "Sdk")
	}
	if isDir(dir) {
		return dir, nil
	}
	return "", errors.New("Android SDK не найден: установи Android Studio или задай переменную ANDROID_HOME")
}

// Platforms возвращает установленные платформы SDK: android-35, android-36...
func Platforms(sdkDir string) []string { return subdirs(filepath.Join(sdkDir, "platforms")) }

// BuildTools возвращает установленные версии build-tools.
func BuildTools(sdkDir string) []string { return subdirs(filepath.Join(sdkDir, "build-tools")) }

// NDK — одна установленная версия Android NDK.
type NDK struct {
	Dir      string // папка NDK
	Version  string // например 28.2.13676358
	Complete bool   // есть ли в ней компилятор для этой системы (бывают недокачанные)
}

// NDKs возвращает все NDK из <sdk>/ndk, от новых к старым.
func NDKs(sdkDir string) []NDK {
	var list []NDK
	for _, v := range subdirs(filepath.Join(sdkDir, "ndk")) {
		n := NDK{Dir: filepath.Join(sdkDir, "ndk", v), Version: v}
		_, err := n.Clang("arm64")
		n.Complete = err == nil
		list = append(list, n)
	}
	slices.SortFunc(list, func(a, b NDK) int { return compareVersions(b.Version, a.Version) })
	return list
}

// BestNDK возвращает самый новый NDK, в котором есть компилятор.
func BestNDK(sdkDir string) (NDK, error) {
	for _, n := range NDKs(sdkDir) {
		if n.Complete {
			return n, nil
		}
	}
	return NDK{}, errors.New("Android NDK не найден: установи его в Android Studio (SDK Manager → SDK Tools → NDK)")
}

// Clang возвращает путь к компилятору C из NDK для архитектуры Go (arm64, amd64, arm, 386).
func (n NDK) Clang(goarch string) (string, error) {
	triples := map[string]string{
		"arm64": "aarch64-linux-android",
		"amd64": "x86_64-linux-android",
		"arm":   "armv7a-linux-androideabi",
		"386":   "i686-linux-android",
	}
	triple, ok := triples[goarch]
	if !ok {
		return "", fmt.Errorf("архитектура %s не поддерживается Android", goarch)
	}
	name := fmt.Sprintf("%s%d-clang", triple, MinSDK)
	if runtime.GOOS == "windows" {
		name += ".cmd"
	}
	path := filepath.Join(n.Dir, "toolchains", "llvm", "prebuilt", ndkHostTag(), "bin", name)
	if !isFile(path) {
		return "", fmt.Errorf("в NDK %s нет компилятора %s", n.Version, name)
	}
	return path, nil
}

// ndkHostTag — папка готовых инструментов NDK для этой системы.
// Для macOS NDK поставляет darwin-x86_64 и на процессорах Apple (через Rosetta).
func ndkHostTag() string {
	switch runtime.GOOS {
	case "windows":
		return "windows-x86_64"
	case "darwin":
		return "darwin-x86_64"
	default:
		return "linux-x86_64"
	}
}

// ABIInfo — как архитектура Android-устройства называется в Go и в Android-проекте.
type ABIInfo struct {
	ABI    string // имя Android: arm64-v8a — оно же папка в jniLibs
	GOARCH string // имя в Go: arm64
}

// ForABI находит архитектуру Go по ABI устройства.
func ForABI(abi string) (ABIInfo, bool) {
	goarch, ok := map[string]string{
		"arm64-v8a":   "arm64",
		"x86_64":      "amd64",
		"armeabi-v7a": "arm",
		"x86":         "386",
	}[abi]
	return ABIInfo{ABI: abi, GOARCH: goarch}, ok
}

// ADB возвращает путь к adb из Android SDK.
func ADB(sdkDir string) (string, error) {
	path := filepath.Join(sdkDir, "platform-tools", exe("adb"))
	if !isFile(path) {
		return "", errors.New("adb не найден: установи Android SDK Platform-Tools в Android Studio")
	}
	return path, nil
}

// Output запускает программу и возвращает её вывод без пробелов по краям.
// Ждёт не дольше 30 секунд, чтобы зависший adb не повесил hexago.
func Output(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		return text, fmt.Errorf("%s %s: %v %s", filepath.Base(name), strings.Join(args, " "), err, text)
	}
	return text, nil
}

// subdirs возвращает имена подпапок.
func subdirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

// compareVersions сравнивает версии вида 28.2.13676358 по числам: 10 > 9.
func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < max(len(as), len(bs)); i++ {
		var x, y int
		if i < len(as) {
			x, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(bs[i])
		}
		if x != y {
			return x - y
		}
	}
	return 0
}
