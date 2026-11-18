//go:build android

package android

/*
#cgo LDFLAGS: -landroid -llog
#include <jni.h>
#include <stdlib.h>
#include <android/log.h>
#include <android/native_window_jni.h>

// Функции JNI вызываются через таблицу в JNIEnv; из Go к ней удобнее
// обращаться через маленькие обёртки на C.
static const char* hx_string_chars(JNIEnv* env, jstring s) {
	return (*env)->GetStringUTFChars(env, s, NULL);
}
static void hx_string_release(JNIEnv* env, jstring s, const char* chars) {
	(*env)->ReleaseStringUTFChars(env, s, chars);
}

// Статические методы HexagoActivity, которые Go вызывает сам: requestFrame и wakeUp.
static JavaVM* hx_vm;
static jclass hx_class;
static jmethodID hx_request_frame_id, hx_wake_up_id, hx_system_bars_id;

static void hx_init(JNIEnv* env, jclass clazz) {
	(*env)->GetJavaVM(env, &hx_vm);
	hx_class = (*env)->NewGlobalRef(env, clazz);
	hx_request_frame_id = (*env)->GetStaticMethodID(env, clazz, "requestFrame", "()V");
	hx_wake_up_id = (*env)->GetStaticMethodID(env, clazz, "wakeUp", "()V");
	hx_system_bars_id = (*env)->GetStaticMethodID(env, clazz, "setSystemBars", "(Z)V");
}

// hx_call вызывает статический метод (без аргументов или с одним boolean) из любого потока:
// чужой поток на время вызова подключается к Java-машине.
static void hx_call(jmethodID m, jboolean arg) {
	if (hx_vm == NULL || m == NULL) {
		return;
	}
	JNIEnv* env;
	int attached = 0;
	if ((*hx_vm)->GetEnv(hx_vm, (void**)&env, JNI_VERSION_1_6) == JNI_EDETACHED) {
		if ((*hx_vm)->AttachCurrentThread(hx_vm, &env, NULL) != 0) {
			return;
		}
		attached = 1;
	}
	(*env)->CallStaticVoidMethod(env, hx_class, m, arg);
	if (attached) {
		(*hx_vm)->DetachCurrentThread(hx_vm);
	}
}
static void hx_request_frame(void) { hx_call(hx_request_frame_id, JNI_FALSE); }
static void hx_wake_up(void) { hx_call(hx_wake_up_id, JNI_FALSE); }
static void hx_system_bars(jboolean light) { hx_call(hx_system_bars_id, light); }
*/
import "C"

import (
	"sync"
	"time"
	"unsafe"
)

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

	// Insets — сколько физических пикселей слева, сверху, справа и снизу занимают
	// системные панели и вырез камеры (приложение рисует под ними).
	Insets [4]int

	// DarkTheme — в настройках телефона включена тёмная тема.
	DarkTheme bool

	// Time — время кадра от включения устройства: по нему идут анимации.
	Time time.Duration
}

// DrawFunc рисует кадр.
type DrawFunc func(f Frame)

var draw DrawFunc

// SetDraw задаёт функцию рисования кадра. Android вызывает её, когда поверхность готова
// или поменяла размер и когда Go попросил кадр (RequestFrame).
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
func Java_dev_hexago_HexagoActivity_nativeOnCreate(env *C.JNIEnv, clazz C.jclass, cacheDir C.jstring) {
	// Activity может создаваться заново (например, после возврата в приложение),
	// а main должна выполниться один раз за всё время жизни процесса.
	if started {
		return
	}
	started = true
	C.hx_init(env, clazz)
	setupCrashOutput(goString(env, cacheDir))
	mainMain()
}

// SetSystemBars задаёт цвет значков строки состояния и панели навигации:
// светлые (lightIcons) — для тёмного фона, иначе тёмные. Из любого потока.
func SetSystemBars(lightIcons bool) {
	v := C.jboolean(C.JNI_FALSE)
	if lightIcons {
		v = C.JNI_TRUE
	}
	C.hx_system_bars(v)
}

// RequestFrame просит Android нарисовать новый кадр на ближайшей развёртке экрана.
// Можно вызывать из любого потока; несколько просьб до кадра дают один кадр.
func RequestFrame() { C.hx_request_frame() }

var (
	tasksMu sync.Mutex
	tasks   []func()
)

// Post выполняет f в потоке интерфейса — там же, где рисуются кадры и приходят касания.
// Можно вызывать из любой горутины.
func Post(f func()) {
	tasksMu.Lock()
	tasks = append(tasks, f)
	first := len(tasks) == 1
	tasksMu.Unlock()
	if first {
		C.hx_wake_up() // задачи уже в очереди — значит, поток интерфейса уже разбужен
	}
}

//export Java_dev_hexago_HexagoActivity_nativeRunTasks
func Java_dev_hexago_HexagoActivity_nativeRunTasks(env *C.JNIEnv, clazz C.jclass) {
	tasksMu.Lock()
	run := tasks
	tasks = nil
	tasksMu.Unlock()
	for _, f := range run {
		f()
	}
}

// PointerEvent — событие одного пальца, как его прислал Android.
type PointerEvent struct {
	Kind    int           // 0 — коснулся, 1 — двигается, 2 — отпущен, 3 — касание отменено
	Pointer int           // номер пальца
	X, Y    float64       // в физических пикселях экрана
	Time    time.Duration // от включения устройства
}

var pointer func(PointerEvent)

// SetPointer задаёт обработчик касаний; он вызывается в потоке интерфейса.
func SetPointer(f func(PointerEvent)) { pointer = f }

//export Java_dev_hexago_HexagoActivity_nativeOnPointer
func Java_dev_hexago_HexagoActivity_nativeOnPointer(env *C.JNIEnv, clazz C.jclass, kind, id C.jint, x, y C.jfloat, timeMillis C.jlong) {
	if pointer != nil {
		pointer(PointerEvent{Kind: int(kind), Pointer: int(id), X: float64(x), Y: float64(y),
			Time: time.Duration(timeMillis) * time.Millisecond})
	}
}

// goString превращает строку Java в строку Go.
func goString(env *C.JNIEnv, s C.jstring) string {
	chars := C.hx_string_chars(env, s)
	defer C.hx_string_release(env, s, chars)
	return C.GoString(chars)
}

//export Java_dev_hexago_HexagoActivity_nativeDraw
func Java_dev_hexago_HexagoActivity_nativeDraw(env *C.JNIEnv, clazz C.jclass, surface C.jobject, density C.jfloat,
	insetLeft, insetTop, insetRight, insetBottom C.jint, darkTheme C.jboolean, frameTimeNanos C.jlong) {
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
		Pixels:    unsafe.Slice((*uint32)(buf.bits), stride*height),
		Width:     int(buf.width),
		Height:    height,
		Stride:    stride,
		Scale:     float64(density),
		Insets:    [4]int{int(insetLeft), int(insetTop), int(insetRight), int(insetBottom)},
		DarkTheme: darkTheme != C.JNI_FALSE,
		Time:      time.Duration(frameTimeNanos),
	})
}
