package sdk

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// JDK — Java, которой будет пользоваться Gradle.
type JDK struct {
	Java    string // путь к программе java
	Version string // 25.0.3
	Major   int    // 25
}

// MinJDK — самая старая Java, с которой работает Android Gradle Plugin 9.
const MinJDK = 17

var javaVersionRe = regexp.MustCompile(`version "([^"]+)"`)

// FindJDK ищет Java так же, как её ищет Gradle: сначала JAVA_HOME, потом java в PATH.
func FindJDK() (JDK, error) {
	java := ""
	if home := os.Getenv("JAVA_HOME"); home != "" {
		java = filepath.Join(home, "bin", exe("java"))
		if !isFile(java) {
			return JDK{}, errors.New("JAVA_HOME=" + home + ", но там нет bin/java")
		}
	} else {
		path, err := exec.LookPath("java")
		if err != nil {
			return JDK{}, errors.New("Java не найдена: установи JDK 17+ или задай JAVA_HOME (подойдёт JDK из Android Studio: <Android Studio>/jbr)")
		}
		java = path
	}

	// java -version пишет в поток ошибок, например: openjdk version "25.0.3" 2026-04-21 LTS
	out, _ := Output(java, "-version")
	m := javaVersionRe.FindStringSubmatch(out)
	if m == nil {
		return JDK{Java: java}, errors.New("не удалось узнать версию Java: " + firstLine(out))
	}
	jdk := JDK{Java: java, Version: m[1]}

	// Старая схема «1.8.0_391» — это Java 8; новая «25.0.3» — Java 25.
	// Главная версия — число в начале строки.
	for _, r := range strings.TrimPrefix(m[1], "1.") {
		if r < '0' || r > '9' {
			break
		}
		jdk.Major = jdk.Major*10 + int(r-'0')
	}
	return jdk, nil
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}
