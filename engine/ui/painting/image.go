package painting

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif" // форматы, которые понимает DecodeImage
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"unsafe"

	_ "golang.org/x/image/webp"

	"hexago/ui/assets"
)

// ImageProvider — откуда взять картинку: AssetImage, FileImage, MemoryImage, а из сети —
// netimage.Image (отдельный пакет: HTTP-клиент удваивает размер приложения, и платить за него стоит,
// только если картинки из сети нужны). Свой источник (камера, база данных) — любой тип с этими двумя методами.
type ImageProvider interface {
	// Key — ключ кеша картинок: у одной и той же картинки он один и тот же (сравнивается через ==).
	Key() any
	// Load читает и декодирует картинку. Вызывается не в потоке интерфейса, в отдельной горутине.
	Load() (image.Image, error)
}

// AssetImage — картинка из ассетов приложения (папка assets/ проекта, на Android — внутри APK).
//
//	painting.AssetImage{Name: "images/logo.png"}
type AssetImage struct {
	Name string
}

func (p AssetImage) Key() any { return p }

func (p AssetImage) Load() (image.Image, error) {
	f, err := assets.Open(p.Name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return decodeNamed(f, p.Name)
}

// FileImage — картинка из файла на устройстве (например, снимок камеры).
type FileImage struct {
	Path string
}

func (p FileImage) Key() any { return p }

func (p FileImage) Load() (image.Image, error) {
	f, err := os.Open(p.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return decodeNamed(f, p.Path)
}

// MemoryImage — картинка из байтов файла PNG, JPEG, GIF или WebP в памяти. Один и тот же срез —
// одна картинка в кеше; менять байты после того, как картинку показали, нельзя.
type MemoryImage struct {
	Bytes []byte
}

type memoryKey struct {
	data *byte
	n    int
}

func (p MemoryImage) Key() any { return memoryKey{unsafe.SliceData(p.Bytes), len(p.Bytes)} }

func (p MemoryImage) Load() (image.Image, error) {
	return DecodeImage(bytes.NewReader(p.Bytes))
}

// DecodeImage декодирует картинку PNG, JPEG, GIF (первый кадр) или WebP — для своих ImageProvider.
func DecodeImage(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	if errors.Is(err, image.ErrFormat) {
		return nil, errors.New("неизвестный формат картинки (понимаем PNG, JPEG, GIF и WebP)")
	}
	return img, err
}

func decodeNamed(r io.Reader, name string) (image.Image, error) {
	img, err := DecodeImage(r)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return img, nil
}
