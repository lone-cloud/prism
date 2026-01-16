package com.lonecloud.sup

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.service.notification.NotificationListenerService
import android.service.notification.StatusBarNotification
import android.util.Log
import androidx.core.app.NotificationCompat
import com.lonecloud.sup.db.Database
import com.lonecloud.sup.db.Notification
import com.lonecloud.sup.ui.MainActivity
import kotlinx.coroutines.*
import kotlin.random.Random

class SignalNotificationListener : NotificationListenerService() {

    private val TAG = "SUP_Listener"
    private val prefs by lazy { 
        getSharedPreferences("sup_prefs", MODE_PRIVATE) 
    }
    private val db by lazy { Database.getInstance(this) }
    private val serviceScope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    companion object {
        private const val CHANNEL_ID = "sup_notifications"
        private const val CHANNEL_NAME = "SUP Notifications"
    }

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
    }

    override fun onDestroy() {
        super.onDestroy()
        serviceScope.cancel()
    }

    override fun onNotificationPosted(sbn: StatusBarNotification?) {
        if (sbn?.packageName != "org.thoughtcrime.securesms") return
        
        val notification = sbn.notification
        val extras = notification.extras

        val title = extras.getString("android.title") ?: ""
        val text = extras.getCharSequence("android.text")?.toString() ?: ""

        Log.d(TAG, "Signal notification: title=$title, text=$text")

        if (text.startsWith("[UP:")) {
            parseAndDeliverUnifiedPush(text)
        }
    }

    private fun parseAndDeliverUnifiedPush(message: String) {
        try {
            val endpointMatch = Regex("""\[UP:([^\]]+)\]""").find(message)
            val endpointId = endpointMatch?.groupValues?.get(1) ?: run {
                Log.w(TAG, "No endpoint ID found in message")
                return
            }

            val subscription = runBlocking {
                db.subscriptionDao().getByUpAppId(endpointId)
            } ?: run {
                Log.w(TAG, "No subscription found for upAppId: $endpointId")
                return
            }

            val payload = message.substringAfter("]").trim()

            val intent = Intent("org.unifiedpush.android.connector.MESSAGE").apply {
                putExtra("token", subscription.upConnectorToken)  // UnifiedPush connector token
                putExtra("message", payload)
                `package` = subscription.upAppId  // Target app package
            }
            sendBroadcast(intent)

            Log.d(TAG, "Delivered UnifiedPush notification to app: ${subscription.upAppId}")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to parse/deliver UnifiedPush notification", e)
        }
    }

    private fun createNotificationChannel() {
        val channel = NotificationChannel(
            CHANNEL_ID,
            CHANNEL_NAME,
            NotificationManager.IMPORTANCE_DEFAULT
        ).apply {
            description = "Notifications from SUP topics"
        }

        val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        notificationManager.createNotificationChannel(channel)
    }
}
