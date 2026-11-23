package painting

import (
	"container/list"
	"fmt"
	"image"
	"sync"

	"hexago/ui/gfx"
)

// Кеш картинок, как ImageCache во Flutter: загруженные картинки держатся в памяти, пока их пиксели
// умещаются в предел (по умолчанию 100 МБ); давно не нужные вытесняются первыми. Одну картинку,
// которую ждут несколько виджетов, грузят один раз. Ошибки не кешируются: следующая попытка
// загрузит картинку заново.

const defaultCacheLimit = 100 << 20

// maxParallelLoads — сколько картинок грузится и декодируется одновременно: длинный список картинок
// из сети не должен разом занять всю память и все ядра.
const maxParallelLoads = 6

type cacheEntry struct {
	key    any
	bitmap *gfx.Bitmap
}

var cache = struct {
	sync.Mutex
	limit   int
	bytes   int
	lru     *list.List // *cacheEntry; спереди — недавно нужные
	entries map[any]*list.Element
	waiting map[any][]func(*gfx.Bitmap, error)
}{
	limit:   defaultCacheLimit,
	lru:     list.New(),
	entries: map[any]*list.Element{},
	waiting: map[any][]func(*gfx.Bitmap, error){},
}

var loadSlots = make(chan struct{}, maxParallelLoads)

// CachedImage — картинка из кеша, если она уже загружена.
func CachedImage(p ImageProvider) (*gfx.Bitmap, bool) {
	cache.Lock()
	defer cache.Unlock()
	if el, ok := cache.entries[p.Key()]; ok {
		cache.lru.MoveToFront(el)
		return el.Value.(*cacheEntry).bitmap, true
	}
	return nil, false
}

// ResolveImage даёт картинку p: из кеша или загрузив её в отдельной горутине. done вызывается
// из любой горутины (бывает, что сразу); в поток интерфейса результат передают через Post.
func ResolveImage(p ImageProvider, done func(*gfx.Bitmap, error)) {
	key := p.Key()
	cache.Lock()
	if el, ok := cache.entries[key]; ok {
		cache.lru.MoveToFront(el)
		b := el.Value.(*cacheEntry).bitmap
		cache.Unlock()
		done(b, nil)
		return
	}
	if w, ok := cache.waiting[key]; ok {
		cache.waiting[key] = append(w, done) // уже грузится — ждём ту же загрузку
		cache.Unlock()
		return
	}
	cache.waiting[key] = []func(*gfx.Bitmap, error){done}
	cache.Unlock()

	go func() {
		loadSlots <- struct{}{}
		b, err := load(p)
		<-loadSlots

		cache.Lock()
		waiters := cache.waiting[key]
		delete(cache.waiting, key)
		if err == nil {
			put(key, b)
		}
		cache.Unlock()
		for _, f := range waiters {
			f(b, err)
		}
	}()
}

// load загружает картинку; паника в чужом ImageProvider становится ошибкой, а не падением приложения.
func load(p ImageProvider) (b *gfx.Bitmap, err error) {
	defer func() {
		if r := recover(); r != nil {
			b, err = nil, fmt.Errorf("загрузка картинки упала: %v", r)
		}
	}()
	var img image.Image
	if img, err = p.Load(); err != nil {
		return nil, err
	}
	return gfx.NewBitmap(img), nil
}

// put кладёт картинку в кеш и вытесняет старые сверх предела. Картинку больше предела не кладёт.
func put(key any, b *gfx.Bitmap) {
	if b.Bytes() > cache.limit {
		return
	}
	cache.entries[key] = cache.lru.PushFront(&cacheEntry{key, b})
	cache.bytes += b.Bytes()
	trim()
}

func trim() {
	for cache.bytes > cache.limit {
		el := cache.lru.Back()
		remove(el)
	}
}

func remove(el *list.Element) {
	e := el.Value.(*cacheEntry)
	cache.lru.Remove(el)
	delete(cache.entries, e.key)
	cache.bytes -= e.bitmap.Bytes()
}

// EvictImage убирает картинку из кеша — например, чтобы скачать её заново. Виджеты, которые её уже
// показывают, продолжают показывать.
func EvictImage(p ImageProvider) {
	cache.Lock()
	defer cache.Unlock()
	if el, ok := cache.entries[p.Key()]; ok {
		remove(el)
	}
}

// ClearImageCache очищает кеш картинок.
func ClearImageCache() {
	cache.Lock()
	defer cache.Unlock()
	cache.lru.Init()
	clear(cache.entries)
	cache.bytes = 0
}

// SetImageCacheLimit задаёт, сколько байт пикселей держит кеш картинок (картинка w×h — это w·h·4 байт).
func SetImageCacheLimit(bytes int) {
	cache.Lock()
	defer cache.Unlock()
	cache.limit = bytes
	trim()
}
