// Package netimage — картинки по адресу http(s):
//
//	widgets.Image{Source: netimage.Image{URL: "https://example.com/photo.jpg"}, Fit: painting.BoxFitCover}
//
// Пакет отдельный: HTTP-клиент Go заметно увеличивает приложение (на Android — на 3–7 МБ), и платит
// за него только то приложение, которому картинки из сети нужны.
package netimage

import (
	"fmt"
	"image"
	"net/http"
	"slices"
	"strings"
	"time"

	"hexago/ui/painting"
)

// Image — картинка по адресу URL (painting.ImageProvider). Headers — заголовки запроса
// (например, Authorization). Картинка кешируется только в памяти: после перезапуска приложения
// она скачивается снова.
type Image struct {
	URL     string
	Headers map[string]string
}

type key struct{ url, headers string }

func (p Image) Key() any {
	var h []string
	for k, v := range p.Headers {
		h = append(h, k+": "+v)
	}
	slices.Sort(h)
	return key{p.URL, strings.Join(h, "\n")}
}

// client — HTTP-клиент для картинок: зависшая загрузка не держит горутину вечно.
var client = &http.Client{Timeout: 30 * time.Second}

func (p Image) Load() (image.Image, error) {
	req, err := http.NewRequest(http.MethodGet, p.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", p.URL, resp.Status)
	}
	img, err := painting.DecodeImage(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p.URL, err)
	}
	return img, nil
}
