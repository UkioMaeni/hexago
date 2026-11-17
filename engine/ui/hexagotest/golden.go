// Package hexagotest — проверка отрисовки и виджетов без устройства, как flutter_test.
//
// Golden сравнивает картинку с эталоном testdata/<имя>.png. Запуск с флагом -update
// перезаписывает эталоны:
//
//	go test ./ui/... -args -update
package hexagotest

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "перезаписать эталонные PNG в testdata")

// Допуски сравнения: сглаживание может чуть отличаться между версиями растеризатора,
// поэтому разрешаем небольшую разницу каналов и немного отличающихся пикселей.
const (
	channelTolerance = 3     // максимальная разница канала, которая не считается отличием
	pixelTolerance   = 0.002 // доля пикселей, которым разрешено отличаться
)

// Golden сравнивает img с testdata/<name>.png. При расхождении кладёт фактическую картинку
// в testdata/failures/<name>.png, чтобы её можно было открыть и сравнить.
func Golden(t *testing.T, name string, img image.Image) {
	t.Helper()
	path := filepath.Join("testdata", name+".png")
	if *update {
		if err := writePNG(path, img); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := readPNG(path)
	if err != nil {
		t.Fatalf("нет эталона %s (создай: go test -args -update): %v", path, err)
	}
	if msg := diff(want, img); msg != "" {
		failure := filepath.Join("testdata", "failures", name+".png")
		writePNG(failure, img)
		t.Errorf("%s: %s; фактическая картинка: %s", path, msg, failure)
	}
}

func diff(want, got image.Image) string {
	wb, gb := want.Bounds(), got.Bounds()
	if wb.Size() != gb.Size() {
		return fmt.Sprintf("размер %v, ожидали %v", gb.Size(), wb.Size())
	}
	bad, worst := 0, 0
	for y := 0; y < wb.Dy(); y++ {
		for x := 0; x < wb.Dx(); x++ {
			r1, g1, b1, a1 := want.At(wb.Min.X+x, wb.Min.Y+y).RGBA()
			r2, g2, b2, a2 := got.At(gb.Min.X+x, gb.Min.Y+y).RGBA()
			d := max(absDiff(r1, r2), absDiff(g1, g2), absDiff(b1, b2), absDiff(a1, a2)) >> 8
			worst = max(worst, d)
			if d > channelTolerance {
				bad++
			}
		}
	}
	if total := wb.Dx() * wb.Dy(); float64(bad) > pixelTolerance*float64(total) {
		return fmt.Sprintf("отличается %d пикселей из %d (наибольшая разница канала %d)", bad, total, worst)
	}
	return ""
}

func absDiff(a, b uint32) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

func readPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
