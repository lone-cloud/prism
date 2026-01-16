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

        when {
            title.startsWith("SUP - ") && !title.contains("(UP)") -> {
                // Direct notification channel
                val topic = title.removePrefix("SUP - ")
                parseAndDisplayNotification(topic, text)
            }
            title.startsWith("SUP - ") && title.contains("(UP)") -> {
                // UnifiedPush notification
                val appName = title.removePrefix("SUP - ").substringBefore(" (UP)")
                parseAndDeliverUnifiedPush(appName, text)
            }
        }
    }

    private fun parseAndDisplayNotification(topic: String, message: String) {
        serviceScope.launch {
            try {
                val subscription = db.subscriptionDao().get(
                    prefs.getString("server_url", "") ?: "",
                    topic
                ) ?: return@launch

                if (subscription.mutedUntil > System.currentTimeMillis() / 1000) {
                    Log.d(TAG, "Subscription $topic is muted")
                    return@launch
                }

                val lines = message.lines()
                val (title, body, priority, clickUrl) = parseNotificationMessage(lines)

                val notif = Notification(
                    id = "${System.currentTimeMillis()}-${Random.nextInt()}",
                    subscriptionId = subscription.id,
                    timestamp = System.currentTimeMillis() / 1000,
                    title = title ?: topic,
                    message = body,
                    notificationId = Random.nextInt(Int.MAX_VALUE),
                    priority = priority,
                    tags = "",
                    deleted = false
                )

                db.notificationDao().add(notif)
                displayNotification(subscription.displayName ?: topic, notif)

                Log.d(TAG, "Displayed notification for topic: $topic")
            } catch (e: Exception) {
                Log.e(TAG, "Failed to display notification", e)
            }
        }
    }

    private fun parseAndDeliverUnifiedPush(appName: String, message: String) {
        try {
            val endpoint = prefs.getString("endpoint_$appName", null)
            val token = prefs.getString("token_$appName", null)

            if (endpoint == null || token == null) {
                Log.w(TAG, "No mapping found for app: $appName")
                return
            }

            val lines = message.lines()
            val body = lines.drop(1).joinToString("\n").trim()

            val intent = Intent("org.unifiedpush.android.connector.MESSAGE").apply {
                putExtra("token", token)
                putExtra("message", body)
                `package` = getAppPackageFromToken(token)
            }
            sendBroadcast(intent)

            Log.d(TAG, "Delivered UnifiedPush notification to $appName")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to parse/deliver UnifiedPush notification", e)
        }
    }

    private fun parseNotificationMessage(lines: List<String>): NotificationData {
        var title: String? = null
        var body = ""
        var priority = 3 // default
        var clickUrl: String? = null

        for (line in lines) {
            when {
                line.startsWith("🚨") || line.startsWith("⚠️") || line.startsWith("🔔") || 
                line.startsWith("🔉") || line.startsWith("🔕") -> {
                    // Parse priority from emoji
                    priority = when {
                        line.startsWith("🚨") -> 5 // urgent
                        line.startsWith("⚠️") -> 4 // high
                        line.startsWith("🔔") -> 3 // default
                        line.startsWith("🔉") -> 2 // low
                        line.startsWith("🔕") -> 1 // min
                        else -> 3
                    }
                    // Extract title (remove emoji and **markdown**)
                    title = line.substring(2).trim()
                        .removePrefix("**").removeSuffix("**").trim()
                }
                line.startsWith("🔗") -> {
                    clickUrl = line.removePrefix("🔗").trim()
                }
                line.startsWith("_Tags:") -> {
                    // Ignore tags line for now
                }
                line.isNotBlank() && title != null -> {
                    // Body content
                    if (body.isNotEmpty()) body += "\n"
                    body += line
                }
            }
        }

        return NotificationData(title, body.ifBlank { lines.joinToString("\n") }, priority, clickUrl)
    }

    private fun displayNotification(topicName: String, notification: Notification) {
        val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

        val intent = Intent(this, MainActivity::class.java)

        val pendingIntent = PendingIntent.getActivity(
            this,
            notification.notificationId,
            intent,
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
        )

        val builder = NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(notification.title)
            .setContentText(notification.message)
            .setStyle(NotificationCompat.BigTextStyle().bigText(notification.message))
            .setPriority(mapPriorityToAndroid(notification.priority))
            .setContentIntent(pendingIntent)
            .setAutoCancel(true)

        notificationManager.notify(notification.notificationId, builder.build())
    }

    private fun mapPriorityToAndroid(priority: Int): Int {
        return when (priority) {
            1 -> NotificationCompat.PRIORITY_MIN
            2 -> NotificationCompat.PRIORITY_LOW
            3 -> NotificationCompat.PRIORITY_DEFAULT
            4 -> NotificationCompat.PRIORITY_HIGH
            5 -> NotificationCompat.PRIORITY_MAX
            else -> NotificationCompat.PRIORITY_DEFAULT
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

    private fun getAppPackageFromToken(token: String): String {
        return token.split(":").firstOrNull() ?: ""
    }

    private data class NotificationData(
        val title: String?,
        val body: String,
        val priority: Int,
        val clickUrl: String?
    )
}
