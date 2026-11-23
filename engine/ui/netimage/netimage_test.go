package netimage_test

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"hexago/ui/gfx"
	"hexago/ui/netimage"
	"hexago/ui/painting"
)

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

func TestLoadFromServer(t *testing.T) {
	defer painting.ClearImageCache()
	img := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+3] = 255, 255
	}
	var data bytes.Buffer
	png.Encode(&data, img)

	var auth atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth.Store(r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/red.png":
			w.Write(data.Bytes())
		case "/text":
			w.Write([]byte("не картинка"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	b, err := resolve(t, netimage.Image{URL: srv.URL + "/red.png", Headers: map[string]string{"Authorization": "Bearer x"}})
	if err != nil || b.Width() != 3 || b.Height() != 2 || b.At(1, 1) != (gfx.Color{R: 255, A: 255}) {
		t.Fatalf("картинка: %v, %v", b, err)
	}
	if auth.Load() != "Bearer x" {
		t.Errorf("заголовок запроса: %v", auth.Load())
	}
	if _, err := resolve(t, netimage.Image{URL: srv.URL + "/нет.png"}); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("нет картинки на сервере: %v", err)
	}
	if _, err := resolve(t, netimage.Image{URL: srv.URL + "/text"}); err == nil || !strings.Contains(err.Error(), "формат") {
		t.Errorf("не картинка: %v", err)
	}
}

func TestKeyIncludesHeaders(t *testing.T) {
	a := netimage.Image{URL: "https://x/1.png", Headers: map[string]string{"A": "1", "B": "2"}}
	b := netimage.Image{URL: "https://x/1.png", Headers: map[string]string{"B": "2", "A": "1"}}
	c := netimage.Image{URL: "https://x/1.png"}
	if a.Key() != b.Key() || a.Key() == c.Key() {
		t.Error("одинаковые заголовки — один ключ, разные — разные")
	}
}
