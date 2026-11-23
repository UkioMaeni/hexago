//go:build android

package android

/*
#include <jni.h>
#include <stdlib.h>
#include <android/asset_manager.h>
#include <android/asset_manager_jni.h>

static AAssetManager* hx_assets;

static void hx_set_assets(JNIEnv* env, jobject mgr) {
	// Глобальная ссылка держит Java-объект AssetManager (а с ним и AAssetManager) живым весь процесс.
	hx_assets = AAssetManager_fromJava(env, (*env)->NewGlobalRef(env, mgr));
}

static AAsset* hx_open_asset(const char* name) {
	return hx_assets == NULL ? NULL : AAssetManager_open(hx_assets, name, AASSET_MODE_STREAMING);
}
*/
import "C"

import (
	"errors"
	"io"
	"io/fs"
	"path"
	"time"
	"unsafe"
)

func setAssets(env *C.JNIEnv, mgr C.jobject) { C.hx_set_assets(env, mgr) }

// Assets — файлы из папки assets внутри APK (туда Gradle кладёт папку assets/ проекта).
// Открывать можно из любой горутины. Папки не открываются и не перечисляются.
func Assets() fs.FS { return assetFS{} }

type assetFS struct{}

func (assetFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	a := C.hx_open_asset(cs)
	if a == nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &assetFile{a: a, name: name}, nil
}

type assetFile struct {
	a    *C.AAsset
	name string
}

func (f *assetFile) Read(p []byte) (int, error) {
	if f.a == nil {
		return 0, fs.ErrClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	n := C.AAsset_read(f.a, unsafe.Pointer(&p[0]), C.size_t(len(p)))
	switch {
	case n < 0:
		return 0, errors.New("android: не удалось прочитать " + f.name)
	case n == 0:
		return 0, io.EOF
	}
	return int(n), nil
}

func (f *assetFile) Close() error {
	if f.a != nil {
		C.AAsset_close(f.a)
		f.a = nil
	}
	return nil
}

func (f *assetFile) Stat() (fs.FileInfo, error) {
	if f.a == nil {
		return nil, fs.ErrClosed
	}
	return assetInfo{name: path.Base(f.name), size: int64(C.AAsset_getLength64(f.a))}, nil
}

type assetInfo struct {
	name string
	size int64
}

func (i assetInfo) Name() string       { return i.name }
func (i assetInfo) Size() int64        { return i.size }
func (i assetInfo) Mode() fs.FileMode  { return 0o444 }
func (i assetInfo) ModTime() time.Time { return time.Time{} }
func (i assetInfo) IsDir() bool        { return false }
func (i assetInfo) Sys() any           { return nil }
