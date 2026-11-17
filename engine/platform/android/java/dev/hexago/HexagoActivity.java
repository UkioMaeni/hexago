package dev.hexago;

import android.app.Activity;
import android.graphics.Insets;
import android.os.Build;
import android.os.Bundle;
import android.view.Surface;
import android.view.SurfaceHolder;
import android.view.SurfaceView;
import android.view.View;
import android.view.WindowInsets;
import android.view.WindowInsetsController;

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
    private static native void nativeOnSurfaceChanged(Surface surface, float density,
            int insetLeft, int insetTop, int insetRight, int insetBottom);

    private SurfaceView view;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        // Запускает main приложения (там вызывается app.Run). Папка кеша нужна,
        // чтобы сохранить текст паники, если Go-код упадёт.
        nativeOnCreate(getCacheDir().getAbsolutePath());

        // Поверхность во весь экран. Сама она ничего не рисует — рисует Go.
        view = new SurfaceView(this);
        view.getHolder().addCallback(this);
        setContentView(view);
        useDarkSystemBarIcons();
    }

    @Override
    public void surfaceCreated(SurfaceHolder holder) {
    }

    // Поверхность готова или поменяла размер (например, при повороте экрана).
    @Override
    public void surfaceChanged(SurfaceHolder holder, int format, int width, int height) {
        // density — сколько физических пикселей в одном dp (1.0, 2.0, 2.625...).
        // Передаём каждый раз: она может поменяться, например при смене масштаба экрана.
        float density = getResources().getDisplayMetrics().density;
        int[] insets = systemInsets();
        nativeOnSurfaceChanged(holder.getSurface(), density, insets[0], insets[1], insets[2], insets[3]);
    }

    @Override
    public void surfaceDestroyed(SurfaceHolder holder) {
    }

    /**
     * Тёмные значки строки состояния и панели навигации — фон приложения по умолчанию белый,
     * и белые значки на нём не видны.
     */
    private void useDarkSystemBarIcons() {
        int light = WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS
                | WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS;
        if (Build.VERSION.SDK_INT >= 30) {
            getWindow().getInsetsController().setSystemBarsAppearance(light, light);
        } else {
            getWindow().getDecorView().setSystemUiVisibility(View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR);
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
        return new int[] {wi.getSystemWindowInsetLeft(), wi.getSystemWindowInsetTop(),
                wi.getSystemWindowInsetRight(), wi.getSystemWindowInsetBottom()};
    }
}
