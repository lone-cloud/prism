-dontwarn okhttp3.**
-dontwarn okio.**
-keepnames class okhttp3.internal.publicsuffix.PublicSuffixDatabase

-keepattributes *Annotation*, InnerClasses
-dontnote kotlinx.serialization.AnnotationsKt
-keepclassmembers class kotlinx.serialization.json.** {
    *** Companion;
}
-keepclasseswithmembers class kotlinx.serialization.json.** {
    kotlinx.serialization.KSerializer serializer(...);
}
-keep,includedescriptorclasses class com.lonecloud.sup.**$$serializer { *; }
-keepclassmembers class com.lonecloud.sup.** {
    *** Companion;
}
-keepclasseswithmembers class com.lonecloud.sup.** {
    kotlinx.serialization.KSerializer serializer(...);
}

-keep class org.unifiedpush.android.connector.** { *; }

