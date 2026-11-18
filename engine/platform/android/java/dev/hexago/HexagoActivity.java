package dev.hexago;

import android.app.Activity;
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
import android.view.SurfaceView;
import android.view.View;
import android.view.Window;
import android.view.WindowInsets;
import android.view.WindowInsetsController;
import android.view.WindowManager;

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
            int insetLeft, int insetTop, int insetRight, int insetBottom);
    private static native void nativeOnPointer(int kind, int pointer, float x, float y, long timeMillis);
    private static native void nativeRunTasks();

    // Виды событий касания — как gestures.PointerKind в Go.
    private static final int POINTER_DOWN = 0, POINTER_MOVE = 1, POINTER_UP = 2, POINTER_CANCEL = 3;

    private static final Handler main = new Handler(Looper.getMainLooper());
    private static HexagoActivity current; // Activity на экране — ей Go заказывает кадры
    private static boolean frameScheduled;

    private SurfaceView view;

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
                    current.redraw();
                }
            });
        });
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
        view = new SurfaceView(this);
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

    /** Передаёт в Go поверхность, плотность и отступы — Go рисует кадр. */
    private void redraw() {
        Surface surface = view.getHolder().getSurface();
        if (surface == null || !surface.isValid()) {
            return;
        }
        // density — сколько физических пикселей в одном dp (1.0, 2.0, 2.625...).
        // Передаём каждый раз: она может поменяться, например при смене масштаба экрана.
        float density = getResources().getDisplayMetrics().density;
        int[] insets = systemInsets();
        nativeDraw(surface, density, insets[0], insets[1], insets[2], insets[3]);
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
            int light = WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS
                    | WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS;
            w.getInsetsController().setSystemBarsAppearance(light, light);
        } else {
            w.getDecorView().setSystemUiVisibility(View.SYSTEM_UI_FLAG_LAYOUT_STABLE
                    | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN
                    | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION
                    | View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR
                    | View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR);
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
            return new int[4];
        }
        if (Build.VERSION.SDK_INT >= 30) {
            Insets i = wi.getInsets(WindowInsets.Type.systemBars() | WindowInsets.Type.displayCutout());
            return new int[] {i.left, i.top, i.right, i.bottom};
        }
        int[] r = {wi.getSystemWindowInsetLeft(), wi.getSystemWindowInsetTop(),
                wi.getSystemWindowInsetRight(), wi.getSystemWindowInsetBottom()};
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
