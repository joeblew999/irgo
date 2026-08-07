plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "com.irgo.example"
    compileSdk = 34

    defaultConfig {
        applicationId = "com.irgo.example"
        minSdk = 24
        targetSdk = 34
        // Overridable by `irgo app package android --version 1.2.3` (or the
        // irgo.package.toml [common] version) via -Pirgo.* gradle props.
        versionCode = providers.gradleProperty("irgo.versionCode").map { it.toInt() }.orNull ?: 1
        versionName = providers.gradleProperty("irgo.versionName").orNull ?: "1.0"
    }

    signingConfigs {
        create("irgo") {
            // Populated by `irgo app package android` via -Pirgo.* gradle props
            // (keystore path + passwords, or the debug keystore by default).
            val ks = providers.gradleProperty("irgo.keystore").orNull
            if (ks != null) {
                storeFile = file(ks)
                storePassword = providers.gradleProperty("irgo.keystorePass").orNull ?: "android"
                keyAlias = providers.gradleProperty("irgo.keyAlias").orNull ?: "android"
                keyPassword = providers.gradleProperty("irgo.keyPass").orNull ?: "android"
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
            // Sign only when `irgo app package android` supplied a keystore;
            // plain `gradlew assembleDebug` (dev) is unaffected.
            if (providers.gradleProperty("irgo.keystore").isPresent) {
                signingConfig = signingConfigs.getByName("irgo")
            }
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_1_8
        targetCompatibility = JavaVersion.VERSION_1_8
    }
    kotlinOptions {
        jvmTarget = "1.8"
    }
}

dependencies {
    implementation(files("libs/irgo.aar"))
    implementation("androidx.core:core-ktx:1.12.0")
    implementation("androidx.appcompat:appcompat:1.6.1")
    implementation("com.google.android.material:material:1.11.0")
}
