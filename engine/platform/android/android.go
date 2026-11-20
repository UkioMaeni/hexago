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
static jmethodID hx_request_frame_id, hx_wake_up_id, hx_system_bars_id, hx_handles_back_id;
static jmethodID hx_show_keyboard_id, hx_update_editing_id, hx_hide_keyboard_id;

static void hx_init(JNIEnv* env, jclass clazz) {
	(*env)->GetJavaVM(env, &hx_vm);
	hx_class = (*env)->NewGlobalRef(env, clazz);
	hx_request_frame_id = (*env)->GetStaticMethodID(env, clazz, "requestFrame", "()V");
	hx_wake_up_id = (*env)->GetStaticMethodID(env, clazz, "wakeUp", "()V");
	hx_system_bars_id = (*env)->GetStaticMethodID(env, clazz, "setSystemBars", "(Z)V");
	hx_handles_back_id = (*env)->GetStaticMethodID(env, clazz, "setHandlesBack", "(Z)V");
	hx_show_keyboard_id = (*env)->GetStaticMethodID(env, clazz, "showKeyboard", "(IIIILjava/lang/String;IIII)V");
	hx_update_editing_id = (*env)->GetStaticMethodID(env, clazz, "updateEditing", "(Ljava/lang/String;IIII)V");
	hx_hide_keyboard_id = (*env)->GetStaticMethodID(env, clazz, "hideKeyboard", "()V");
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
static void hx_handles_back(jboolean handles) { hx_call(hx_handles_back_id, handles); }
static void hx_hide_keyboard(void) { hx_call(hx_hide_keyboard_id, JNI_FALSE); }

// hx_env — JNIEnv текущего потока (attached — поток пришлось подключить, потом отключить).
static JNIEnv* hx_env(int* attached) {
	JNIEnv* env = NULL;
	*attached = 0;
	if (hx_vm == NULL) {
		return NULL;
	}
	if ((*hx_vm)->GetEnv(hx_vm, (void**)&env, JNI_VERSION_1_6) == JNI_EDETACHED) {
		if ((*hx_vm)->AttachCurrentThread(hx_vm, &env, NULL) != 0) {
			return NULL;
		}
		*attached = 1;
	}
	return env;
}

// Текст поля передаётся в Java как UTF-16: NewStringUTF не понимает эмодзи в обычном UTF-8.
static void hx_show_keyboard(jint keyboard, jint action, jint flags, jint caps, const jchar* text, jint n,
		jint s, jint e, jint cs, jint ce) {
	int attached;
	JNIEnv* env = hx_env(&attached);
	if (env == NULL || hx_show_keyboard_id == NULL) {
		return;
	}
	jstring str = (*env)->NewString(env, text, n);
	(*env)->CallStaticVoidMethod(env, hx_class, hx_show_keyboard_id, keyboard, action, flags, caps, str, s, e, cs, ce);
	(*env)->DeleteLocalRef(env, str);
	if (attached) {
		(*hx_vm)->DetachCurrentThread(hx_vm);
	}
}

static void hx_update_editing(const jchar* text, jint n, jint s, jint e, jint cs, jint ce) {
	int attached;
	JNIEnv* env = hx_env(&attached);
	if (env == NULL || hx_update_editing_id == NULL) {
		return;
	}
	jstring str = (*env)->NewString(env, text, n);
	(*env)->CallStaticVoidMethod(env, hx_class, hx_update_editing_id, str, s, e, cs, ce);
	(*env)->DeleteLocalRef(env, str);
	if (attached) {
		(*hx_vm)->DetachCurrentThread(hx_vm);
	}
}

// Строка Java как UTF-16.
static const jchar* hx_chars(JNIEnv* env, jstring s, jsize* n) {
	*n = (*env)->GetStringLength(env, s);
	return (*env)->GetStringChars(env, s, NULL);
}
static void hx_chars_release(JNIEnv* env, jstring s, const jchar* chars) {
	(*env)->ReleaseStringChars(env, s, chars);
}
*/
import "C"

import (
	"image"
	"slices"
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"
)

// FrameInfo — что известно о кадре до рисования.
type FrameInfo struct {
	// Width, Height — размер экрана в физических пикселях.
	Width, Height int

	// Scale — сколько физических пикселей в одном dp (плотность экрана).
	Scale float64

	// Insets — сколько физических пикселей слева, сверху, справа и снизу занимают
	// системные панели и вырез камеры (приложение рисует под ними).
	Insets [4]int

	// Keyboard — высота экранной клавиатуры в физических пикселях (0 — её нет).
	Keyboard int

	// DarkTheme — в настройках телефона включена тёмная тема.
	DarkTheme bool

	// Time — время кадра от включения устройства: по нему идут анимации.
	Time time.Duration

	// Full — кадр нужно нарисовать целиком (новая поверхность, сменилась тема).
	Full bool
}

// Frame — буфер пикселей окна.
type Frame struct {
	// Pixels — все пиксели построчно. Пиксель (x, y) — это Pixels[y*Stride+x].
	// Цвет пикселя — uint32 в порядке 0xAABBGGRR.
	Pixels []uint32

	// Width, Height — размер буфера в физических пикселях.
	Width, Height int

	// Stride — длина строки в памяти, в пикселях. Stride ≥ Width:
	// в конце строки могут быть невидимые пиксели выравнивания.
	Stride int
}

// Renderer рисует кадры в два шага. Prepare — построить и разложить кадр и сказать, какую часть
// экрана (в пикселях) нужно перерисовать; пустой прямоугольник — ничего, кадр не отправляется.
// Paint — нарисовать в буфер окна внутри dirty. Android переносит остальное с прошлого кадра сам
// и иногда просит перерисовать больше (dirty бывает больше того, что вернул Prepare).
type Renderer interface {
	Prepare(f FrameInfo) image.Rectangle
	Paint(f Frame, dirty image.Rectangle)
}

var renderer Renderer

// SetRenderer задаёт, кто рисует кадры. Android просит кадр, когда поверхность готова
// или поменялась и когда Go попросил кадр (RequestFrame).
func SetRenderer(r Renderer) { renderer = r }

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

// SetHandlesBack сообщает Android, есть ли что закрыть кнопкой «Назад»: пока есть, она приходит в
// обработчик SetBack, пока нет — система обрабатывает её сама. Из любого потока.
func SetHandlesBack(handles bool) {
	v := C.jboolean(C.JNI_FALSE)
	if handles {
		v = C.JNI_TRUE
	}
	C.hx_handles_back(v)
}

var back func() bool

// SetBack задаёт обработчик кнопки «Назад» (в потоке интерфейса): true — обработано.
func SetBack(f func() bool) { back = f }

//export Java_dev_hexago_HexagoActivity_nativeOnBack
func Java_dev_hexago_HexagoActivity_nativeOnBack(env *C.JNIEnv, clazz C.jclass) C.jboolean {
	if back != nil && back() {
		return C.JNI_TRUE
	}
	return C.JNI_FALSE
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
	insetLeft, insetTop, insetRight, insetBottom, keyboard C.jint, darkTheme C.jboolean, frameTimeNanos C.jlong, full C.jboolean) {
	if renderer == nil {
		return // приложение ещё не сказало, что рисовать
	}
	win := C.ANativeWindow_fromSurface(env, surface)
	if win == nil {
		return
	}
	defer C.ANativeWindow_release(win)

	info := FrameInfo{
		Width:     int(C.ANativeWindow_getWidth(win)),
		Height:    int(C.ANativeWindow_getHeight(win)),
		Scale:     float64(density),
		Insets:    [4]int{int(insetLeft), int(insetTop), int(insetRight), int(insetBottom)},
		Keyboard:  int(keyboard),
		DarkTheme: darkTheme != C.JNI_FALSE,
		Time:      time.Duration(frameTimeNanos),
		Full:      full != C.JNI_FALSE,
	}
	dirty := renderer.Prepare(info)
	if info.Full {
		dirty = image.Rect(0, 0, info.Width, info.Height)
	}
	if dirty.Empty() {
		return // ничего не изменилось — кадр не отправляем
	}

	// Формат — 4 байта на пиксель RGBA; 0, 0 — размер окна не менять.
	C.ANativeWindow_setBuffersGeometry(win, 0, 0, C.WINDOW_FORMAT_RGBA_8888)

	// Блокируем только изменившуюся часть: Android скопирует остальное из прошлого кадра и вернёт
	// в rect то, что действительно надо перерисовать (иногда больше, например весь экран).
	rect := C.ARect{left: C.int32_t(dirty.Min.X), top: C.int32_t(dirty.Min.Y),
		right: C.int32_t(dirty.Max.X), bottom: C.int32_t(dirty.Max.Y)}
	var buf C.ANativeWindow_Buffer
	if C.ANativeWindow_lock(win, &buf, &rect) != 0 {
		return
	}
	// defer выполняются в обратном порядке: сначала показать кадр, потом отпустить окно.
	defer C.ANativeWindow_unlockAndPost(win)

	stride, height := int(buf.stride), int(buf.height)
	renderer.Paint(Frame{
		Pixels: unsafe.Slice((*uint32)(buf.bits), stride*height),
		Width:  int(buf.width),
		Height: height,
		Stride: stride,
	}, image.Rect(int(rect.left), int(rect.top), int(rect.right), int(rect.bottom)))
}

// EditingState — текст поля ввода, выделение и набираемое слово; позиции — в рунах.
// ComposeEnd <= ComposeStart — слово не набирается.
type EditingState struct {
	Text                     string
	SelStart, SelEnd         int
	ComposeStart, ComposeEnd int
}

// KeyboardConfig — какая клавиатура нужна полю. Keyboard и Action — номера как у
// widgets.KeyboardType и widgets.TextInputAction, Capitalize — как у widgets.Capitalization.
type KeyboardConfig struct {
	Keyboard, Action, Capitalize      int
	Multiline, Obscure, NoAutocorrect bool
}

// utf16Of — текст в UTF-16 (как строки Java) и перевод позиции-руны в позицию UTF-16.
func utf16Of(s string) ([]uint16, func(int) C.jint) {
	runes := []rune(s)
	u := utf16.Encode(runes)
	offs := make([]int, len(runes)+1)
	for i, r := range runes {
		offs[i+1] = offs[i] + utf16.RuneLen(r)
	}
	return u, func(i int) C.jint {
		if i < 0 {
			return -1
		}
		return C.jint(offs[min(i, len(runes))])
	}
}

// jchars — указатель на текст для C (у пустого текста — на ноль: NULL Java не любит).
func jchars(u []uint16) (*C.jchar, C.jint) {
	if len(u) == 0 {
		var zero C.jchar
		return &zero, 0
	}
	return (*C.jchar)(unsafe.Pointer(&u[0])), C.jint(len(u))
}

// composing — набираемое слово для Java: -1, -1, если его нет.
func composing(s EditingState) (int, int) {
	if s.ComposeEnd <= s.ComposeStart {
		return -1, -1
	}
	return s.ComposeStart, s.ComposeEnd
}

// ShowKeyboard подключает клавиатуру к полю с текстом s и показывает её. Из любого потока.
func ShowKeyboard(cfg KeyboardConfig, s EditingState) {
	flags := 0
	if cfg.Multiline {
		flags |= 1
	}
	if cfg.Obscure {
		flags |= 2
	}
	if cfg.NoAutocorrect {
		flags |= 4
	}
	u, at := utf16Of(s.Text)
	p, n := jchars(u)
	cs, ce := composing(s)
	C.hx_show_keyboard(C.jint(cfg.Keyboard), C.jint(cfg.Action), C.jint(flags), C.jint(cfg.Capitalize), p, n,
		at(s.SelStart), at(s.SelEnd), at(cs), at(ce))
}

// UpdateEditing сообщает клавиатуре текст поля, который поменяло приложение. Из любого потока.
func UpdateEditing(s EditingState) {
	u, at := utf16Of(s.Text)
	p, n := jchars(u)
	cs, ce := composing(s)
	C.hx_update_editing(p, n, at(s.SelStart), at(s.SelEnd), at(cs), at(ce))
}

// HideKeyboard прячет клавиатуру. Из любого потока.
func HideKeyboard() { C.hx_hide_keyboard() }

var (
	editing    func(EditingState)
	textAction func(int)
)

// SetEditing задаёт обработчик правок клавиатуры; он вызывается в потоке интерфейса.
func SetEditing(f func(EditingState)) { editing = f }

// SetTextAction задаёт обработчик кнопки ввода клавиатуры (номер — как у widgets.TextInputAction).
func SetTextAction(f func(action int)) { textAction = f }

//export Java_dev_hexago_HexagoActivity_nativeOnEditing
func Java_dev_hexago_HexagoActivity_nativeOnEditing(env *C.JNIEnv, clazz C.jclass, text C.jstring,
	selStart, selEnd, composeStart, composeEnd C.jint) {
	if editing == nil {
		return
	}
	var n C.jsize
	chars := C.hx_chars(env, text, &n)
	u := slices.Clone(unsafe.Slice((*uint16)(unsafe.Pointer(chars)), int(n)))
	C.hx_chars_release(env, text, chars)
	// Позиция UTF-16 → номер руны (середина суррогатной пары — та же руна).
	idx := make([]int, len(u)+1)
	r := 0
	for i := 0; i < len(u); r++ {
		idx[i] = r
		if utf16.IsSurrogate(rune(u[i])) && i+1 < len(u) {
			idx[i+1] = r
			i += 2
		} else {
			i++
		}
	}
	idx[len(u)] = r
	at := func(off C.jint) int {
		if off < 0 {
			return -1
		}
		return idx[min(int(off), len(u))]
	}
	editing(EditingState{Text: string(utf16.Decode(u)), SelStart: at(selStart), SelEnd: at(selEnd),
		ComposeStart: at(composeStart), ComposeEnd: at(composeEnd)})
}

//export Java_dev_hexago_HexagoActivity_nativeOnTextAction
func Java_dev_hexago_HexagoActivity_nativeOnTextAction(env *C.JNIEnv, clazz C.jclass, action C.jint) {
	if textAction != nil {
		textAction(int(action))
	}
}
