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
        private const val CHANNEL_ID_PROTON = "sup_email"
        private const val CHANNEL_NAME_PROTON = "Email Notifications"
        private const val SUP_ENDPOINT_PREFIX = "[SUP:"
        private const val LAUNCH_PREFIX = "[LAUNCH:"
        
        private val SIGNAL_PACKAGES = setOf(
            "org.thoughtcrime.securesms",  // Signal
            "im.molly.app"                 // Molly
        )
    }

    override fun onCreate() {
        super.onCreate()
        createNotificationChannels()
    }

    override fun onDestroy() {
        super.onDestroy()
        serviceScope.cancel()
    }

    override fun onNotificationPosted(sbn: StatusBarNotification?) {
        if (sbn?.packageName !in SIGNAL_PACKAGES) return
        
        val notification = sbn?.notification ?: return
        val extras = notification.extras

        val title = extras.getString("android.title") ?: ""
        val text = extras.getCharSequence("android.text")?.toString() ?: ""

        Log.d(TAG, "Signal notification: title=$title, text=$text")

        when {
            text.startsWith(SUP_ENDPOINT_PREFIX) -> {
                parseAndDeliverUnifiedPush(text)
                cancelNotification(sbn?.key ?: return)
            }
            text.startsWith(LAUNCH_PREFIX) -> {
                parseAndShowLaunchNotification(text)
                cancelNotification(sbn?.key ?: return)
            }
        }
    }

    private fun parseAndDeliverUnifiedPush(message: String) {
        try {
            val endpointMatch = Regex("""\[SUP:([^\]]+)\]""").find(message)
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
                putExtra("token", subscription.upConnectorToken)
                putExtra("message", payload)
                `package` = subscription.upAppId
            }
            sendBroadcast(intent)

            Log.d(TAG, "Delivered UnifiedPush notification to app: ${subscription.upAppId}")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to parse/deliver UnifiedPush notification", e)
        }
    }

    private fun parseAndShowLaunchNotification(message: String) {
        try {
            val packageMatch = Regex("""\[LAUNCH:([^\]]+)\]""").find(message)
            val packageName = packageMatch?.groupValues?.get(1) ?: run {
                Log.w(TAG, "No package name found in LAUNCH message")
                return
            }

            val content = message.substringAfter("]").trim()
            
            // Parse title and body (format: **Title**\nBody)
            val titleMatch = Regex("""\*\*([^*]+)\*\*""").find(content)
            val title = titleMatch?.groupValues?.get(1) ?: "Email"
            val body = content.substringAfter("**", "").substringAfter("**", "").trim()

            // Check if target app is installed
            val isInstalled = try {
                packageManager.getPackageInfo(packageName, 0)
                true
            } catch (e: Exception) {
                false
            }

            val clickIntent = if (isInstalled) {
                packageManager.getLaunchIntentForPackage(packageName)?.let { intent ->
                    PendingIntent.getActivity(
                        this,
                        Random.nextInt(),
                        intent.apply {
                            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                            addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP)
                        },
                        PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
                    )
                }
            } else {
                null
            }

            val notification = NotificationCompat.Builder(this, CHANNEL_ID_PROTON)
                .setContentTitle(title)
                .setContentText(body)
                .setSmallIcon(android.R.drawable.ic_dialog_email)
                .setPriority(NotificationCompat.PRIORITY_DEFAULT)
                .setAutoCancel(true)
                .apply {
                    if (clickIntent != null) {
                        setContentIntent(clickIntent)
                    }
                }
                .build()

            val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            notificationManager.notify(Random.nextInt(), notification)

            Log.d(TAG, "Showed app launch notification")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to parse/show app launch notification", e)
        }
    }

    private fun createNotificationChannels() {
        val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        
        val channel = NotificationChannel(
            CHANNEL_ID,
            CHANNEL_NAME,
            NotificationManager.IMPORTANCE_DEFAULT
        ).apply {
            description = "Notifications from SUP topics"
        }
        notificationManager.createNotificationChannel(channel)

        val protonChannel = NotificationChannel(
            CHANNEL_ID_PROTON,
            CHANNEL_NAME_PROTON,
            NotificationManager.IMPORTANCE_DEFAULT
        ).apply {
            description = "Email notifications"
        }
        notificationManager.createNotificationChannel(protonChannel)
    }
}
