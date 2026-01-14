package com.lonecloud.sup

import android.content.Intent
import android.service.notification.NotificationListenerService
import android.service.notification.StatusBarNotification
import android.util.Log

class SignalNotificationListener : NotificationListenerService() {

    private val prefs by lazy { 
        getSharedPreferences("sup_prefs", MODE_PRIVATE) 
    }

    override fun onNotificationPosted(sbn: StatusBarNotification?) {
        if (sbn?.packageName != "org.thoughtcrime.securesms") return // Signal package
        
        val notification = sbn.notification
        val extras = notification.extras

        val title = extras.getString("android.title") ?: ""
        val text = extras.getCharSequence("android.text")?.toString() ?: ""

        Log.d("SUP", "Signal notification: title=$title, text=$text")

        // Parse SUP message format: **Title**\nbody\nJSON
        if (title.startsWith("SUP - ")) {
            val appName = title.removePrefix("SUP - ")
            parseAndDeliver(appName, text)
        }
    }

    private fun parseAndDeliver(appName: String, message: String) {
        try {
            // Find the endpoint for this app
            val endpoint = prefs.getString("endpoint_$appName", null)
            val token = prefs.getString("token_$appName", null)

            if (endpoint == null || token == null) {
                Log.w("SUP", "No mapping found for app: $appName")
                return
            }

            // Extract message body (skip the formatted parts)
            val lines = message.lines()
            val body = lines.getOrNull(1) ?: message

            // Send to app
            val intent = Intent("org.unifiedpush.android.connector.MESSAGE").apply {
                putExtra("token", token)
                putExtra("message", body)
                `package` = getAppPackageFromToken(token)
            }
            sendBroadcast(intent)

            Log.d("SUP", "Delivered notification to $appName")
        } catch (e: Exception) {
            Log.e("SUP", "Failed to parse/deliver notification", e)
        }
    }

    private fun getAppPackageFromToken(token: String): String {
        return token.split(":").firstOrNull() ?: ""
    }
}
