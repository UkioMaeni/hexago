package dev.hexago;

import android.app.Activity;
import android.content.res.Configuration;
import android.graphics.Color;
import android.graphics.Insets;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.view.Choreographer;
import android.view.DisplayCutout;
import android.view.MotionEvent;
import android.view.Surface;
import android.view.SurfaceHolder;
import android.view.View;
import android.view.Window;
import android.view.WindowInsets;
import android.view.WindowInsetsController;
import android.view.WindowManager;
import android.window.OnBackInvokedCallback;
import android.window.OnBackInvokedDispatcher;

/**
 * Activity движка Hexago — как FlutterActivity во Flutter.
 *
 * MainActivity приложения просто наследует этот класс. Здесь — вся связь с Go:
 * загрузка libhexago.so, поверхность для рисования и передача событий в Go.
 * Пакет dev.hexago постоянный, поэтому имена JNI-функций в Go одинаковы для любого
 * приложения: Java_dev_hexago_HexagoActivity_...
 */
public class HexagoActivity extends Activity implements SurfaceHolder.Callback {

    static {
        // libhexago.so — Go-код приложения вместе с движком.
        System.loadLibrary("hexago");
    }

    // Реализации — в Go, пакет hexago/platform/android.
    private static native void nativeOnCreate(String cacheDir);
    private static native void nativeDraw(Surface surface, float density,
            int insetLeft, int insetTop, int insetRight, int insetBottom, int keyboard, boolean darkTheme,
            long frameTimeNanos, boolean full);
    private static native void nativeOnPointer(int kind, int pointer, float x, float y, long timeMillis);
    private static native void nativeRunTasks();
    private static native boolean nativeOnBack();
    private static native void nativeOnEditing(String text, int selStart, int selEnd, int composeStart, int composeEnd);
    private static native void nativeOnTextAction(int action);

    // Виды событий касания — как gestures.PointerKind в Go.
    private static final int POINTER_DOWN = 0, POINTER_MOVE = 1, POINTER_UP = 2, POINTER_CANCEL = 3;

    private static final Handler main = new Handler(Looper.getMainLooper());
    private static HexagoActivity current; // Activity на экране — ей Go заказывает кадры
    private static boolean frameScheduled;
    // Есть ли в Go что закрыть кнопкой «Назад» (Navigator может вернуться). Статическое — переживает
    // пересоздание Activity.
    private static boolean handlesBack;

    private HexagoView view;
    private Object backCallback; // OnBackInvokedCallback (Android 13+), пока Go обрабатывает «Назад»

    /**
     * Go просит новый кадр (что-то изменилось — например, SetState). Кадр рисуется на ближайшей
     * вертикальной развёртке экрана; несколько просьб до неё дают один кадр. Вызывается из любого потока.
     */
    static void requestFrame() {
        main.post(() -> {
            if (frameScheduled || current == null) {
                return;
            }
            frameScheduled = true;
            Choreographer.getInstance().postFrameCallback(frameTimeNanos -> {
                frameScheduled = false;
                if (current != null) {
                    current.redraw(frameTimeNanos, false);
                }
            });
        });
    }

    /**
     * Go задаёт вид системных панелей (виджет SystemBars): светлые значки — для тёмного фона.
     * Вызывается из любого потока.
     */
    static void setSystemBars(boolean lightIcons) {
        main.post(() -> {
            if (current != null) {
                current.setBarIcons(lightIcons);
            }
        });
    }

    /**
     * Go сообщает, есть ли что закрыть кнопкой «Назад». Пока есть — «Назад» уходит в Go (закрывает
     * верхний экран или окно); пока нет — её обрабатывает система (сворачивает приложение, с анимацией
     * «предсказания» на Android 13+). Вызывается из любого потока.
     */
    static void setHandlesBack(boolean handles) {
        main.post(() -> {
            handlesBack = handles;
            if (current != null) {
                current.updateBackCallback();
            }
        });
    }

    /** Go: подключить клавиатуру к полю ввода и показать её (см. HexagoView.show). Из любого потока. */
    static void showKeyboard(int keyboard, int action, int flags, int capitalize, String text, int selStart,
            int selEnd, int composeStart, int composeEnd) {
        main.post(() -> {
            if (current != null) {
                current.view.show(keyboard, action, flags, capitalize, text, selStart, selEnd, composeStart, composeEnd);
            }
        });
    }

    /** Go: текст поля поменяло приложение. Из любого потока. */
    static void updateEditing(String text, int selStart, int selEnd, int composeStart, int composeEnd) {
        main.post(() -> {
            if (current != null) {
                current.view.update(text, selStart, selEnd, composeStart, composeEnd);
            }
        });
    }

    /** Go: спрятать клавиатуру. Из любого потока. */
    static void hideKeyboard() {
        main.post(() -> {
            if (current != null) {
                current.view.hide();
            }
        });
    }

    // Клавиатура изменила текст поля или нажата её кнопка ввода — в Go.
    static void onEditing(String text, int selStart, int selEnd, int composeStart, int composeEnd) {
        nativeOnEditing(text, selStart, selEnd, composeStart, composeEnd);
    }

    static void onTextAction(int action) {
        nativeOnTextAction(action);
    }

    /** Go положил задачи для потока интерфейса (app.Post, таймеры жестов) — выполнить их там. */
    static void wakeUp() {
        main.post(HexagoActivity::nativeRunTasks);
    }

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        current = this;
        // Запускает main приложения (там вызывается app.Run). Папка кеша нужна,
        // чтобы сохранить текст паники, если Go-код упадёт.
        nativeOnCreate(getCacheDir().getAbsolutePath());

        // Поверхность во весь экран. Сама она ничего не рисует — рисует Go.
        view = new HexagoView(this);
        // Высоту экранной клавиатуры окно сообщает отступом снизу (WindowInsets) — поля не прячутся под ней.
        getWindow().setSoftInputMode(WindowManager.LayoutParams.SOFT_INPUT_ADJUST_RESIZE);
        view.getHolder().addCallback(this);
        // Отступы панелей могут поменяться и без смены размера поверхности — перерисовываем.
        view.setOnApplyWindowInsetsListener((v, insets) -> {
            redraw();
            return insets;
        });
        // Касания — в Go: там их разбирают распознаватели жестов.
        view.setOnTouchListener((v, event) -> {
            sendTouch(event);
            return true;
        });
        setContentView(view);
        edgeToEdge();
        updateBackCallback();
    }

    /**
     * С Android 13 «Назад» приходит через OnBackInvokedDispatcher (onBackPressed для приложений под
     * Android 16 больше не вызывается). Обратный вызов зарегистрирован, только пока Go есть что закрыть:
     * иначе система обрабатывает «Назад» сама.
     */
    private void updateBackCallback() {
        if (Build.VERSION.SDK_INT < 33) {
            return;
        }
        OnBackInvokedDispatcher d = getOnBackInvokedDispatcher();
        if (handlesBack && backCallback == null) {
            OnBackInvokedCallback cb = HexagoActivity::nativeOnBack;
            backCallback = cb;
            d.registerOnBackInvokedCallback(OnBackInvokedDispatcher.PRIORITY_DEFAULT, cb);
        } else if (!handlesBack && backCallback != null) {
            d.unregisterOnBackInvokedCallback((OnBackInvokedCallback) backCallback);
            backCallback = null;
        }
    }

    // До Android 13 «Назад» приходит сюда.
    @Override
    @SuppressWarnings("deprecation")
    public void onBackPressed() {
        if (handlesBack && nativeOnBack()) {
            return;
        }
        super.onBackPressed();
    }

    // Тема системы, размер шрифта и т. п. меняются без пересоздания Activity (configChanges) —
    // перерисовать с новыми данными.
    @Override
    public void onConfigurationChanged(Configuration newConfig) {
        super.onConfigurationChanged(newConfig);
        redraw();
    }

    @Override
    protected void onDestroy() {
        if (current == this) {
            current = null;
        }
        super.onDestroy();
    }

    /**
     * Переводит MotionEvent в события отдельных пальцев. Координаты — в физических пикселях
     * поверхности (она во весь экран), время — миллисекунды с включения устройства.
     * Промежуточные точки движения (история) тоже передаются: по ним точнее скорость броска.
     */
    private static void sendTouch(MotionEvent e) {
        switch (e.getActionMasked()) {
            case MotionEvent.ACTION_DOWN:
            case MotionEvent.ACTION_POINTER_DOWN: {
                int i = e.getActionIndex();
                nativeOnPointer(POINTER_DOWN, e.getPointerId(i), e.getX(i), e.getY(i), e.getEventTime());
                break;
            }
            case MotionEvent.ACTION_MOVE:
                for (int h = 0; h < e.getHistorySize(); h++) {
                    for (int i = 0; i < e.getPointerCount(); i++) {
                        nativeOnPointer(POINTER_MOVE, e.getPointerId(i), e.getHistoricalX(i, h),
                                e.getHistoricalY(i, h), e.getHistoricalEventTime(h));
                    }
                }
                for (int i = 0; i < e.getPointerCount(); i++) {
                    nativeOnPointer(POINTER_MOVE, e.getPointerId(i), e.getX(i), e.getY(i), e.getEventTime());
                }
                break;
            case MotionEvent.ACTION_UP:
            case MotionEvent.ACTION_POINTER_UP: {
                int i = e.getActionIndex();
                nativeOnPointer(POINTER_UP, e.getPointerId(i), e.getX(i), e.getY(i), e.getEventTime());
                break;
            }
            case MotionEvent.ACTION_CANCEL:
                for (int i = 0; i < e.getPointerCount(); i++) {
                    nativeOnPointer(POINTER_CANCEL, e.getPointerId(i), e.getX(i), e.getY(i), e.getEventTime());
                }
                break;
        }
    }

    @Override
    public void surfaceCreated(SurfaceHolder holder) {
    }

    // Поверхность готова или поменяла размер (например, при повороте экрана).
    @Override
    public void surfaceChanged(SurfaceHolder holder, int format, int width, int height) {
        redraw();
    }

    @Override
    public void surfaceDestroyed(SurfaceHolder holder) {
    }

    /**
     * Перерисовка не по развёртке (поверхность изменилась, сменилась тема или отступы): время — сейчас,
     * и кадр рисуется целиком — содержимое новой поверхности неизвестно.
     */
    private void redraw() {
        redraw(System.nanoTime(), true);
    }

    /**
     * Передаёт в Go поверхность, плотность, отступы и время кадра — Go рисует кадр.
     * Время — наносекунды от включения устройства (как у Choreographer и System.nanoTime):
     * по нему идут анимации. full — нарисовать всё; иначе Go перерисует только изменившееся.
     */
    private void redraw(long frameTimeNanos, boolean full) {
        Surface surface = view.getHolder().getSurface();
        if (surface == null || !surface.isValid()) {
            return;
        }
        // density — сколько физических пикселей в одном dp (1.0, 2.0, 2.625...).
        // Передаём каждый раз: она может поменяться, например при смене масштаба экрана.
        float density = getResources().getDisplayMetrics().density;
        int[] insets = systemInsets();
        boolean dark = (getResources().getConfiguration().uiMode & Configuration.UI_MODE_NIGHT_MASK)
                == Configuration.UI_MODE_NIGHT_YES;
        nativeDraw(surface, density, insets[0], insets[1], insets[2], insets[3], insets[4], dark, frameTimeNanos, full);
    }

    /**
     * Приложение рисует во весь экран, под строкой состояния и панелью навигации (edge-to-edge),
     * как Flutter. С Android 15 система делает так сама; до неё панели закрашены цветом темы
     * (серым), и окно нужно попросить об этом явно.
     *
     * Значки панелей — тёмные: фон приложения по умолчанию белый, и белые значки на нём не видны.
     */
    private void edgeToEdge() {
        Window w = getWindow();
        w.setStatusBarColor(Color.TRANSPARENT);
        w.setNavigationBarColor(Color.TRANSPARENT);
        if (Build.VERSION.SDK_INT >= 29) {
            // Без этого система подкладывает под кнопки навигации полупрозрачную плашку.
            w.setStatusBarContrastEnforced(false);
            w.setNavigationBarContrastEnforced(false);
        }
        if (Build.VERSION.SDK_INT >= 28) {
            // Рисовать и рядом с вырезом камеры; SafeArea отодвинет содержимое.
            WindowManager.LayoutParams lp = w.getAttributes();
            lp.layoutInDisplayCutoutMode = WindowManager.LayoutParams.LAYOUT_IN_DISPLAY_CUTOUT_MODE_SHORT_EDGES;
            w.setAttributes(lp);
        }
        if (Build.VERSION.SDK_INT >= 30) {
            w.setDecorFitsSystemWindows(false);
        }
        setBarIcons(false);
    }

    /**
     * Цвет значков строки состояния и панели навигации: тёмные — для светлого фона (по умолчанию),
     * светлые — для тёмного.
     */
    private void setBarIcons(boolean lightIcons) {
        Window w = getWindow();
        if (Build.VERSION.SDK_INT >= 30) {
            int mask = WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS
                    | WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS;
            w.getInsetsController().setSystemBarsAppearance(lightIcons ? 0 : mask, mask);
        } else {
            int flags = View.SYSTEM_UI_FLAG_LAYOUT_STABLE
                    | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN
                    | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION;
            if (!lightIcons) {
                flags |= View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR | View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR;
            }
            w.getDecorView().setSystemUiVisibility(flags);
        }
    }

    /**
     * Сколько физических пикселей с каждой стороны занимают системные панели
     * (строка состояния, панель навигации) и вырез камеры. Приложение рисует под ними
     * (edge-to-edge), а виджет SafeArea отодвигает содержимое на эти отступы.
     */
    private int[] systemInsets() {
        WindowInsets wi = view.getRootWindowInsets();
        if (wi == null) {
            return new int[5];
        }
        if (Build.VERSION.SDK_INT >= 30) {
            Insets i = wi.getInsets(WindowInsets.Type.systemBars() | WindowInsets.Type.displayCutout());
            return new int[] {i.left, i.top, i.right, i.bottom, wi.getInsets(WindowInsets.Type.ime()).bottom};
        }
        // До Android 11 клавиатура входит в нижний отступ окна; без неё — «стабильный» отступ.
        int[] r = {wi.getSystemWindowInsetLeft(), wi.getSystemWindowInsetTop(),
                wi.getSystemWindowInsetRight(), wi.getStableInsetBottom(),
                Math.max(0, wi.getSystemWindowInsetBottom() - wi.getStableInsetBottom())};
        if (Build.VERSION.SDK_INT >= 28 && wi.getDisplayCutout() != null) {
            DisplayCutout c = wi.getDisplayCutout();
            r[0] = Math.max(r[0], c.getSafeInsetLeft());
            r[1] = Math.max(r[1], c.getSafeInsetTop());
            r[2] = Math.max(r[2], c.getSafeInsetRight());
            r[3] = Math.max(r[3], c.getSafeInsetBottom());
        }
        return r;
    }
}
