package painting_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"hexago/ui/assets"
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/painting"
)

func TestFitSizes(t *testing.T) {
	img := geom.Size{Width: 200, Height: 100} // картинка 2:1
	box := geom.Size{Width: 100, Height: 100} // квадрат
	cases := []struct {
		fit          painting.BoxFit
		source, dest geom.Size
	}{
		{painting.BoxFitFill, img, box},
		{painting.BoxFitContain, img, geom.Size{Width: 100, Height: 50}},
		{painting.BoxFitCover, geom.Size{Width: 100, Height: 100}, box},
		{painting.BoxFitWidth, img, geom.Size{Width: 100, Height: 50}},
		{painting.BoxFitHeight, geom.Size{Width: 100, Height: 100}, box},
		{painting.BoxFitNone, geom.Size{Width: 100, Height: 100}, geom.Size{Width: 100, Height: 100}},
		{painting.BoxFitScaleDown, img, geom.Size{Width: 100, Height: 50}},
	}
	for _, c := range cases {
		if s, d := painting.FitSizes(c.fit, img, box); s != c.source || d != c.dest {
			t.Errorf("fit %d: source %v dest %v, ожидали %v %v", c.fit, s, d, c.source, c.dest)
		}
	}
	// Маленькая картинка: ScaleDown не увеличивает, Contain — увеличивает.
	small := geom.Size{Width: 20, Height: 10}
	if _, d := painting.FitSizes(painting.BoxFitScaleDown, small, box); d != small {
		t.Errorf("ScaleDown увеличил: %v", d)
	}
	if _, d := painting.FitSizes(painting.BoxFitContain, small, box); d != (geom.Size{Width: 100, Height: 50}) {
		t.Errorf("Contain: %v", d)
	}
	r := painting.Inscribe(geom.BottomRight, geom.Size{Width: 10, Height: 20}, geom.RectXYWH(0, 0, 100, 100))
	if r != geom.RectXYWH(90, 80, 10, 20) {
		t.Errorf("Inscribe: %v", r)
	}
}

// pngBytes — PNG w×h одного цвета.
func pngBytes(t *testing.T, w, h int, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// resolve ждёт картинку.
func resolve(t *testing.T, p painting.ImageProvider) (*gfx.Bitmap, error) {
	t.Helper()
	type result struct {
		b   *gfx.Bitmap
		err error
	}
	ch := make(chan result, 1)
	painting.ResolveImage(p, func(b *gfx.Bitmap, err error) { ch <- result{b, err} })
	select {
	case r := <-ch:
		return r.b, r.err
	case <-time.After(5 * time.Second):
		t.Fatal("картинка не загрузилась за 5 с")
		return nil, nil
	}
}

func TestProviders(t *testing.T) {
	defer painting.ClearImageCache()
	red := color.NRGBA{R: 255, A: 255}
	data := pngBytes(t, 3, 2, red)

	old := assets.FS()
	defer assets.SetFS(old)
	assets.SetFS(fstest.MapFS{"img/red.png": {Data: data}})

	file := filepath.Join(t.TempDir(), "red.png")
	if err := os.WriteFile(file, data, 0o644); err != nil {
		t.Fatal(err)
	}

	for _, p := range []painting.ImageProvider{
		painting.AssetImage{Name: "img/red.png"},
		painting.FileImage{Path: file},
		painting.MemoryImage{Bytes: data},
	} {
		b, err := resolve(t, p)
		if err != nil || b.Width() != 3 || b.Height() != 2 || b.At(1, 1) != (gfx.Color{R: 255, A: 255}) {
			t.Errorf("%T: %v, %v", p, b, err)
		}
	}
	if _, err := resolve(t, painting.MemoryImage{Bytes: []byte("не картинка")}); err == nil || !strings.Contains(err.Error(), "формат") {
		t.Errorf("не картинка: %v", err)
	}
	if _, err := resolve(t, painting.AssetImage{Name: "нет.png"}); err == nil {
		t.Error("нет ассета — нет ошибки")
	}
}

// counting — источник, который считает загрузки и может ждать сигнала или падать.
type counting struct {
	name  string
	loads *atomic.Int32
	gate  chan struct{}
	fail  *atomic.Bool
	size  int
}

func (p counting) Key() any { return p.name }

func (p counting) Load() (image.Image, error) {
	p.loads.Add(1)
	if p.gate != nil {
		<-p.gate
	}
	if p.fail != nil && p.fail.Load() {
		return nil, errors.New("сбой")
	}
	return image.NewNRGBA(image.Rect(0, 0, p.size, p.size)), nil
}

func TestCacheLoadsOnceForManyWaiters(t *testing.T) {
	defer painting.ClearImageCache()
	p := counting{name: "общая", loads: new(atomic.Int32), gate: make(chan struct{}), size: 4}
	var wg sync.WaitGroup
	got := make([]*gfx.Bitmap, 3)
	for i := range got {
		wg.Add(1)
		painting.ResolveImage(p, func(b *gfx.Bitmap, err error) { got[i] = b; wg.Done() })
	}
	close(p.gate)
	wg.Wait()
	if p.loads.Load() != 1 || got[0] == nil || got[0] != got[1] || got[1] != got[2] {
		t.Fatalf("загрузок %d, картинки %v", p.loads.Load(), got)
	}
	if b, ok := painting.CachedImage(p); !ok || b != got[0] {
		t.Error("картинки нет в кеше")
	}
	painting.EvictImage(p)
	if _, ok := painting.CachedImage(p); ok {
		t.Error("EvictImage не убрал картинку")
	}
}

func TestCacheDoesNotKeepErrors(t *testing.T) {
	defer painting.ClearImageCache()
	fail := new(atomic.Bool)
	fail.Store(true)
	p := counting{name: "сбойная", loads: new(atomic.Int32), fail: fail, size: 1}
	if _, err := resolve(t, p); err == nil {
		t.Fatal("нет ошибки")
	}
	fail.Store(false)
	if b, err := resolve(t, p); err != nil || b == nil || p.loads.Load() != 2 {
		t.Errorf("повтор после ошибки: %v, %v, загрузок %d", b, err, p.loads.Load())
	}
}

type panicking struct{}

func (panicking) Key() any                   { return "паника" }
func (panicking) Load() (image.Image, error) { panic("сломался") }

func TestCachePanicBecomesError(t *testing.T) {
	if _, err := resolve(t, panicking{}); err == nil || !strings.Contains(err.Error(), "сломался") {
		t.Errorf("паника в источнике: %v", err)
	}
}

func TestCacheEvictsOldest(t *testing.T) {
	defer painting.ClearImageCache()
	defer painting.SetImageCacheLimit(100 << 20)
	painting.SetImageCacheLimit(2 * 10 * 10 * 4) // две картинки 10×10
	loads := new(atomic.Int32)
	a, b, c := counting{"a", loads, nil, nil, 10}, counting{"b", loads, nil, nil, 10}, counting{"c", loads, nil, nil, 10}
	resolve(t, a)
	resolve(t, b)
	painting.CachedImage(a) // a нужна — b теперь самая старая
	resolve(t, c)
	if _, ok := painting.CachedImage(b); ok {
		t.Error("самая старая картинка не вытеснена")
	}
	if _, ok := painting.CachedImage(a); !ok {
		t.Error("недавно нужная картинка вытеснена")
	}
	big := counting{"большая", loads, nil, nil, 100}
	if bm, err := resolve(t, big); err != nil || bm == nil {
		t.Fatal(err)
	}
	if _, ok := painting.CachedImage(big); ok {
		t.Error("картинка больше всего кеша в нём осталась")
	}
}
