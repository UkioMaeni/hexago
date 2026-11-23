// Package assets — файлы приложения из папки assets/ проекта: картинки, данные, звуки.
//
// На Android они лежат внутри APK, и app.Run подключает их сам. В тестах и на компьютере это папка
// assets в текущей папке. Имена пишутся через «/», от корня папки assets:
//
//	data, err := assets.ReadFile("data/cities.json")
//	widgets.Image{Source: painting.AssetImage{Name: "images/logo.png"}}
package assets

import (
	"io/fs"
	"os"
	"sync"
)

var (
	mu      sync.RWMutex
	current fs.FS = os.DirFS("assets")
)

// SetFS задаёт, откуда брать файлы (app.Run на Android — из APK; тесты — свои файлы).
func SetFS(fsys fs.FS) {
	mu.Lock()
	defer mu.Unlock()
	current = fsys
}

// FS — файлы приложения как fs.FS: их можно отдать fs.WalkDir, template.ParseFS и другим.
func FS() fs.FS {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// Open открывает файл приложения. Можно из любой горутины.
func Open(name string) (fs.File, error) { return FS().Open(name) }

// ReadFile читает файл приложения целиком. Можно из любой горутины.
func ReadFile(name string) ([]byte, error) { return fs.ReadFile(FS(), name) }
