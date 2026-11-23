plugins {
    id("com.android.application")
}

android {
    namespace = "com.example.newapp"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.example.newapp"
        minSdk = 26 // Android 8.0: адаптивные иконки и современный NDK
        targetSdk = 36
        versionCode = 1
        versionName = "1.0"
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    // Java-часть движка (HexagoActivity) берётся прямо из папки движка —
    // так же, как Go-часть через replace в go.mod.
    // Папка assets/ проекта (картинки и другие файлы приложения) попадает в APK как есть.
    sourceSets {
        getByName("main") {
            java.srcDir("C:/dev/projects/gexago/engine/platform/android/java")
            assets.srcDir("../../assets")
        }
    }

    buildTypes {
        release {
            // Пока своего ключа нет, release подписывается отладочным —
            // так release-сборку можно поставить на телефон.
            // Перед публикацией в магазин замени на свой ключ.
            signingConfig = signingConfigs.getByName("debug")
        }
    }
}
