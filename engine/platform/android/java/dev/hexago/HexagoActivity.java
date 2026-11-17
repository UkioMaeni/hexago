package dev.hexago;

import android.app.Activity;
import android.os.Bundle;
import android.view.Surface;
import android.view.SurfaceHolder;
import android.view.SurfaceView;

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
    private static native void nativeOnSurfaceChanged(Surface surface, float density);

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        // Запускает main приложения (там вызывается app.Run). Папка кеша нужна,
        // чтобы сохранить текст паники, если Go-код упадёт.
        nativeOnCreate(getCacheDir().getAbsolutePath());

        // Поверхность во весь экран. Сама она ничего не рисует — рисует Go.
        SurfaceView view = new SurfaceView(this);
        view.getHolder().addCallback(this);
        setContentView(view);
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
        nativeOnSurfaceChanged(holder.getSurface(), density);
    }

    @Override
    public void surfaceDestroyed(SurfaceHolder holder) {
    }
}
