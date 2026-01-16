package com.lonecloud.sup.service

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Intent
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat
import com.lonecloud.sup.R
import com.lonecloud.sup.app.Application
import com.lonecloud.sup.db.Notification
import com.lonecloud.sup.msg.NotificationDispatcher
import com.lonecloud.sup.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

/**
 * Service that listens for Signal notifications and processes them.
 * This replaces ntfy's SubscriberService which polls HTTP/WebSocket.
 * We get notifications pushed via Signal instead.
 */
class SignalListenerService : Service() {
    private val repository by lazy { (application as Application).repository }
    private val dispatcher by lazy { NotificationDispatcher(this, repository) }
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    override fun onCreate() {
        super.onCreate()
        Log.d(TAG, "Service created")
        createForegroundNotification()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        Log.d(TAG, "Service started")
        
        intent?.getStringExtra(EXTRA_NOTIFICATION_DATA)?.let { data ->
            scope.launch {
                processNotification(data)
            }
        }
        
        return START_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun createForegroundNotification() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                NOTIFICATION_CHANNEL_ID,
                getString(R.string.channel_subscriber_service_name),
                NotificationManager.IMPORTANCE_LOW
            )
            val notificationManager = getSystemService(NotificationManager::class.java)
            notificationManager.createNotificationChannel(channel)
        }

        val notification = NotificationCompat.Builder(this, NOTIFICATION_CHANNEL_ID)
            .setContentTitle(getString(R.string.app_name))
            .setContentText("Listening for notifications via Signal")
            .setSmallIcon(R.drawable.ic_notification)
            .build()

        startForeground(NOTIFICATION_ID, notification)
    }

    private suspend fun processNotification(data: String) {
        try {
            Log.d(TAG, "Processing notification: $data")
        } catch (e: Exception) {
            Log.e(TAG, "Error processing notification: ${e.message}", e)
        }
    }

    companion object {
        private const val TAG = "NtfySignalListener"
        private const val NOTIFICATION_CHANNEL_ID = "ntfy-signal"
        private const val NOTIFICATION_ID = 1
        const val EXTRA_NOTIFICATION_DATA = "notification_data"
    }
}
