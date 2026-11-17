//go:build android

package android

/*
#cgo LDFLAGS: -landroid -llog
#include <jni.h>
#include <stdlib.h>
#include <android/log.h>
#include <android/native_window_jni.h>
*/
import "C"

import "unsafe"

// Frame — буфер пикселей одного кадра.
type Frame struct {
	// Pixels — все пиксели построчно. Пиксель (x, y) — это Pixels[y*Stride+x].
	// Цвет пикселя — uint32 в порядке 0xAABBGGRR.
	Pixels []uint32

	// Width, Height — размер экрана в физических пикселях.
	Width, Height int

	// Stride — длина строки в памяти, в пикселях. Stride ≥ Width:
	// в конце строки могут быть невидимые пиксели выравнивания.
	Stride int

	// Scale — сколько физических пикселей в одном dp (плотность экрана).
	// Размер в dp × Scale = размер в пикселях.
	Scale float64
}

// DrawFunc рисует кадр.
type DrawFunc func(f Frame)

var draw DrawFunc

// SetDraw задаёт функцию рисования кадра. Android вызывает её,
// когда поверхность готова или поменяла размер.
func SetDraw(f DrawFunc) { draw = f }

// Log пишет строку в Logcat с тегом hexago.
func Log(msg string) {
	tag := C.CString("hexago")
	text := C.CString(msg)
	defer C.free(unsafe.Pointer(tag))
	defer C.free(unsafe.Pointer(text))
	C.__android_log_write(C.ANDROID_LOG_INFO, tag, text)
}

// mainMain — функция main приложения пользователя.
//
// В библиотеке (-buildmode=c-shared) Go сам не вызывает main: библиотеку загружает
// Java. Поэтому вызываем её отсюда, когда Activity создана. В main пользователь
// вызывает app.Run, и так движок узнаёт его приложение.
// go:linkname связывает это объявление без тела с настоящей функцией main.main.
//
//go:linkname mainMain main.main
func mainMain()

var started bool

//export Java_dev_hexago_HexagoActivity_nativeOnCreate
func Java_dev_hexago_HexagoActivity_nativeOnCreate(env *C.JNIEnv, clazz C.jclass) {
	// Activity может создаваться заново (например, после возврата в приложение),
	// а main должна выполниться один раз за всё время жизни процесса.
	if started {
		return
	}
	started = true
	mainMain()
}

//export Java_dev_hexago_HexagoActivity_nativeOnSurfaceChanged
func Java_dev_hexago_HexagoActivity_nativeOnSurfaceChanged(env *C.JNIEnv, clazz C.jclass, surface C.jobject, density C.jfloat) {
	win := C.ANativeWindow_fromSurface(env, surface)
	if win == nil {
		return
	}
	defer C.ANativeWindow_release(win)

	// Формат — 4 байта на пиксель RGBA; 0, 0 — размер окна не менять.
	C.ANativeWindow_setBuffersGeometry(win, 0, 0, C.WINDOW_FORMAT_RGBA_8888)

	var buf C.ANativeWindow_Buffer
	if C.ANativeWindow_lock(win, &buf, nil) != 0 {
		return
	}
	// defer выполняются в обратном порядке: сначала показать кадр, потом отпустить окно.
	defer C.ANativeWindow_unlockAndPost(win)

	if draw == nil {
		return // приложение ещё не сказало, что рисовать
	}
	stride, height := int(buf.stride), int(buf.height)
	draw(Frame{
		Pixels: unsafe.Slice((*uint32)(buf.bits), stride*height),
		Width:  int(buf.width),
		Height: height,
		Stride: stride,
		Scale:  float64(density),
	})
}
