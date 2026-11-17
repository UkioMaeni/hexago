// Package android — встраивание Hexago в Android (платформенный слой).
//
// Java-часть — класс dev.hexago.HexagoActivity в папке java/ рядом с этим файлом.
// Он загружает libhexago.so и вызывает отсюда JNI-функции: когда Activity создана
// и когда поверхность SurfaceView готова или поменяла размер. Go-часть берёт буфер
// пикселей поверхности (ANativeWindow) и отдаёт его функции рисования из SetDraw.
//
// Код собирается только под Android (GOOS=android, cgo, компилятор из NDK);
// на других платформах в пакете есть только это описание.
package android
