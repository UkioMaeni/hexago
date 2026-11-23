package widgets

import (
	"fmt"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/painting"
	"hexago/ui/rendering"
)

// Image — картинка из ассетов, файла, сети или памяти, как Image во Flutter:
//
//	widgets.Image{Source: painting.AssetImage{Name: "images/logo.png"}, Width: new(120.0)}
//	widgets.Image{Source: netimage.Image{URL: url}, Height: new(200.0), Fit: painting.BoxFitCover}
//
// Картинка грузится в отдельной горутине и попадает в кеш (painting.ResolveImage); уже загруженная
// показывается сразу, без пустого кадра. Один пиксель картинки — один dp, если Width и Height
// не задают другой размер.
//
// Пока картинка грузится, на её месте Placeholder (или пустое место размером Width×Height). Если она
// не загрузилась — Error (или пустое место, а ошибка — в лог).
type Image struct {
	State
	Source        painting.ImageProvider
	Width, Height *float64
	Fit           painting.BoxFit // по умолчанию ScaleDown: как есть, а большую — уменьшить целиком
	Alignment     geom.Alignment  // где картинка в своём прямоугольнике; по умолчанию — посередине
	Placeholder   Widget
	Error         func(ctx BuildContext, err error) Widget

	key    any // какую картинку показываем или ждём
	bitmap *gfx.Bitmap
	err    error
}

func (w *Image) Init(BuildContext) { w.resolve() }

func (w *Image) Update(_ BuildContext, old Image) {
	if imageKey(w.Source) != imageKey(old.Source) {
		w.resolve()
	}
}

func imageKey(p painting.ImageProvider) any {
	if p == nil {
		return nil
	}
	return p.Key()
}

// resolve берёт картинку из кеша или запускает загрузку.
func (w *Image) resolve() {
	w.bitmap, w.err = nil, nil
	w.key = imageKey(w.Source)
	if w.Source == nil {
		return
	}
	if b, ok := painting.CachedImage(w.Source); ok {
		w.bitmap = b
		return
	}
	key, src, post := w.key, w.Source, w.element.owner.post
	painting.ResolveImage(src, func(b *gfx.Bitmap, err error) {
		post(func() {
			if !w.Mounted() || w.key != key {
				return // виджет убран или ждёт уже другую картинку
			}
			if err != nil && w.Error == nil {
				fmt.Printf("Картинка не загрузилась: %v\n", err)
			}
			w.SetState(func() { w.bitmap, w.err = b, err })
		})
	})
}

func (w *Image) Build(ctx BuildContext) Widget {
	switch {
	case w.err != nil && w.Error != nil:
		return w.Error(ctx, w.err)
	case w.bitmap == nil && w.err == nil && w.Placeholder != nil:
		return w.Placeholder
	}
	return RawImage{Image: w.bitmap, Width: w.Width, Height: w.Height, Fit: w.Fit, Alignment: w.Alignment}
}

// RawImage рисует уже готовую картинку (gfx.Bitmap) — например, сделанную своим кодом.
// Обычно нужен Image, который сам загружает картинку.
type RawImage struct {
	Base
	Image         *gfx.Bitmap
	Width, Height *float64
	Fit           painting.BoxFit
	Alignment     geom.Alignment
}

func (w RawImage) CreateRenderObject(BuildContext) rendering.Box {
	return &rendering.ImageBox{Image: w.Image, Width: w.Width, Height: w.Height, Fit: w.Fit, Alignment: w.Alignment}
}

func (w RawImage) UpdateRenderObject(_ BuildContext, b rendering.Box) {
	ib := b.(*rendering.ImageBox)
	ib.Image, ib.Width, ib.Height, ib.Fit, ib.Alignment = w.Image, w.Width, w.Height, w.Fit, w.Alignment
}
