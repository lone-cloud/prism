plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("com.google.devtools.ksp") version "2.1.0-1.0.29"
}

android {
    namespace = "com.lonecloud.sup"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.lonecloud.sup"
        minSdk = 26
        targetSdk = 36
        versionCode = 1
        versionName = "0.1.0"
    }

    signingConfigs {
        create("release") {
            storeFile = file(System.getenv("KEYSTORE_FILE") ?: "../release.keystore")
            storePassword = System.getenv("KEYSTORE_PASSWORD")
            keyAlias = System.getenv("KEY_ALIAS") ?: "sup-release"
            keyPassword = System.getenv("KEYSTORE_PASSWORD") // PKCS12 uses same password
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
            signingConfig = signingConfigs.getByName("release")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlin {
        compilerOptions {
            jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17)
        }
    }

    buildFeatures {
        viewBinding = true
        buildConfig = true
    }

    lint {
        checkReleaseBuilds = false
        abortOnError = false
    }

    ksp {
        arg("room.schemaLocation", "$projectDir/schemas")
    }

    defaultConfig {
        // Build config fields
        buildConfigField("boolean", "FIREBASE_AVAILABLE", "false")
        buildConfigField("boolean", "RATE_APP_AVAILABLE", "false")
        buildConfigField("boolean", "PAYMENT_LINKS_AVAILABLE", "false")
        buildConfigField("String", "FLAVOR", "\"sup\"")
    }
}

dependencies {
    // AndroidX Core
    implementation("androidx.appcompat:appcompat:1.7.1")
    implementation("androidx.core:core-ktx:1.17.0")
    implementation("androidx.constraintlayout:constraintlayout:2.2.1")
    implementation("androidx.activity:activity-ktx:1.12.2")
    implementation("androidx.fragment:fragment-ktx:1.8.9")
    implementation("androidx.work:work-runtime-ktx:2.11.0")
    implementation("androidx.preference:preference-ktx:1.2.1")

    // JSON (Gson)
    implementation("com.google.code.gson:gson:2.13.2")

    // Room (SQLite)
    val roomVersion = "2.6.1"
    implementation("androidx.room:room-runtime:$roomVersion")
    ksp("androidx.room:room-compiler:$roomVersion")
    implementation("androidx.room:room-ktx:$roomVersion")

    // OkHttp
    implementation("com.squareup.okhttp3:okhttp:5.3.2")

    // RecyclerView
    implementation("androidx.recyclerview:recyclerview:1.4.0")

    // Swipe to refresh
    implementation("androidx.swiperefreshlayout:swiperefreshlayout:1.2.0")

    // Material Design
    implementation("com.google.android.material:material:1.13.0")

    // LiveData
    implementation("androidx.lifecycle:lifecycle-livedata-ktx:2.10.0")
    implementation("androidx.legacy:legacy-support-v4:1.0.0")

    // Image viewer
    implementation("com.github.stfalcon-studio:StfalconImageViewer:1.0.1")

    // Glide (GIF support)
    val glideVersion = "5.0.5"
    implementation("com.github.bumptech.glide:glide:$glideVersion")
    ksp("com.github.bumptech.glide:ksp:$glideVersion")

    // Better click handling for links
    implementation("me.saket:better-link-movement-method:2.2.0")

    // Markdown
    implementation("io.noties.markwon:core:4.6.2")
    implementation("io.noties.markwon:image-picasso:4.6.2")
    implementation("io.noties.markwon:image:4.6.2")
    implementation("io.noties.markwon:linkify:4.6.2")
    implementation("io.noties.markwon:ext-tables:4.6.2")
    implementation("io.noties.markwon:ext-strikethrough:4.6.2")

    // Markdown dependencies (R8 requirements)
    implementation("pl.droidsonroids.gif:android-gif-drawable:1.2.29")
    implementation("com.caverock:androidsvg:1.4")

    // UnifiedPush
    implementation("com.github.UnifiedPush:android-connector:3.0.10")
}

