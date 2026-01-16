package com.lonecloud.sup.msg

import android.app.*
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.media.AudioAttributes
import android.media.AudioManager
import android.media.RingtoneManager
import android.net.Uri
import androidx.core.app.NotificationCompat
import com.lonecloud.sup.R
import com.lonecloud.sup.db.*
import com.lonecloud.sup.db.Notification
import com.lonecloud.sup.ui.Colors
import com.lonecloud.sup.ui.DetailActivity
import com.lonecloud.sup.ui.MainActivity
import com.lonecloud.sup.util.*
import java.util.*

class NotificationService(val context: Context) {
    private val notificationManager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
    private val repository = Repository.getInstance(context)
    private val appBaseUrl = context.getString(R.string.app_base_url)

    fun display(subscription: Subscription, notification: Notification) {
        displayInternal(subscription, notification, update = false)
    }

    fun update(subscription: Subscription, notification: Notification, isNew: Boolean) {
        displayInternal(subscription, notification, update = !isNew)
    }

    fun cancel(notificationId: Int) {
        notificationManager.cancel(notificationId)
    }

    fun cancel(subscription: Subscription, notification: Notification) {
        notificationManager.cancel(notification.notificationId)
    }

    fun createNotificationChannels() {
        val groupId = DEFAULT_GROUP
        val groupName = context.getString(R.string.channel_notifications_group_default_name)
        maybeCreateNotificationGroup(groupId, groupName)
        (PRIORITY_MIN..PRIORITY_MAX).forEach { priority ->
            maybeCreateNotificationChannel(groupId, priority)
        }
    }

    private fun displayInternal(subscription: Subscription, notification: Notification, update: Boolean = false) {
        val title = formatTitle(appBaseUrl, subscription, notification)
        val groupId = DEFAULT_GROUP
        val channelId = toChannelId(groupId, notification.priority)
        val insistent = notification.priority == PRIORITY_MAX &&
                (repository.getInsistentMaxPriorityEnabled() || subscription.insistent == Repository.INSISTENT_MAX_PRIORITY_ENABLED)
        val builder = NotificationCompat.Builder(context, channelId)
            .setSmallIcon(R.drawable.ic_notification)
            .setColor(Colors.notificationIcon(context))
            .setContentTitle(title)
            .setOnlyAlertOnce(true)
            .setAutoCancel(true)
        setStyleAndText(builder, notification)
        setClickAction(builder, subscription)
        maybeSetDeleteIntent(builder, insistent)
        maybeSetSound(builder, insistent, update)

        maybeCreateNotificationGroup(groupId, subscriptionGroupName(subscription))
        maybeCreateNotificationChannel(groupId, notification.priority)
        maybePlayInsistentSound(groupId, insistent)

        notificationManager.notify(notification.notificationId, builder.build())
    }

    private fun maybeSetDeleteIntent(builder: NotificationCompat.Builder, insistent: Boolean) {
        if (!insistent) {
            return
        }
        val intent = Intent(context, DeleteBroadcastReceiver::class.java)
        val pendingIntent = PendingIntent.getBroadcast(context, Random().nextInt(), intent, PendingIntent.FLAG_IMMUTABLE)
        builder.setDeleteIntent(pendingIntent)
    }

    private fun maybeSetSound(builder: NotificationCompat.Builder, insistent: Boolean, update: Boolean) {
        val hasSound = !update && !insistent
        if (hasSound) {
            val defaultSoundUri = RingtoneManager.getDefaultUri(RingtoneManager.TYPE_NOTIFICATION)
            builder.setSound(defaultSoundUri)
        } else {
            builder.setSound(null)
        }
    }

    private fun setStyleAndText(builder: NotificationCompat.Builder, notification: Notification) {
        val message = formatMessage(notification)
        builder
            .setContentText(message)
            .setStyle(NotificationCompat.BigTextStyle().bigText(message))
    }

    private fun setClickAction(builder: NotificationCompat.Builder, subscription: Subscription) {
        builder.setContentIntent(detailActivityIntent(subscription))
    }

    private fun subscriptionGroupName(subscription: Subscription): String {
        return displayName(appBaseUrl, subscription)
    }

    private fun displayName(appBaseUrl: String?, subscription: Subscription): String {
        return subscription.displayName ?: subscriptionTopicShortUrl(subscription)
    }

    class DeleteBroadcastReceiver : BroadcastReceiver() {
        override fun onReceive(context: Context, intent: Intent) {
            Log.d(TAG, "Media player: Stopping insistent ring")
            val mediaPlayer = Repository.getInstance(context).mediaPlayer
            mediaPlayer.stop()
        }
    }

    private fun detailActivityIntent(subscription: Subscription): PendingIntent? {
        val intent = Intent(context, DetailActivity::class.java).apply {
            putExtra(MainActivity.EXTRA_SUBSCRIPTION_ID, subscription.id)
            putExtra(MainActivity.EXTRA_SUBSCRIPTION_BASE_URL, subscription.baseUrl)
            putExtra(MainActivity.EXTRA_SUBSCRIPTION_TOPIC, subscription.topic)
            putExtra(MainActivity.EXTRA_SUBSCRIPTION_DISPLAY_NAME, displayName(appBaseUrl, subscription))
            putExtra(MainActivity.EXTRA_SUBSCRIPTION_MUTED_UNTIL, subscription.mutedUntil)
        }
        return TaskStackBuilder.create(context).run {
            addNextIntentWithParentStack(intent)
            getPendingIntent(Random().nextInt(), PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
        }
    }

    private fun maybeCreateNotificationChannel(group: String, priority: Int) {
        val channelId = toChannelId(group, priority)
        val pause = 300L
        val channel = when (priority) {
            PRIORITY_MIN -> NotificationChannel(channelId, context.getString(R.string.common_priority_min_name), NotificationManager.IMPORTANCE_MIN)
            PRIORITY_LOW -> NotificationChannel(channelId, context.getString(R.string.common_priority_low_name), NotificationManager.IMPORTANCE_LOW)
            PRIORITY_HIGH -> {
                val channel = NotificationChannel(channelId, context.getString(R.string.common_priority_high_name), NotificationManager.IMPORTANCE_HIGH)
                channel.enableVibration(true)
                channel.vibrationPattern = longArrayOf(
                    pause, 100, pause, 100, pause, 100,
                    pause, 2000
                )
                channel
            }
            PRIORITY_MAX -> {
                val channel = NotificationChannel(channelId, context.getString(R.string.common_priority_max_name), NotificationManager.IMPORTANCE_HIGH)
                channel.enableLights(true)
                channel.enableVibration(true)
                channel.setBypassDnd(true)
                channel.vibrationPattern = longArrayOf(
                    pause, 100, pause, 100, pause, 100,
                    pause, 2000,
                    pause, 100, pause, 100, pause, 100,
                    pause, 2000,
                    pause, 100, pause, 100, pause, 100,
                    pause, 2000
                )
                channel
            }
            else -> NotificationChannel(channelId, context.getString(R.string.common_priority_default_name), NotificationManager.IMPORTANCE_DEFAULT)
        }
        channel.group = group
        notificationManager.createNotificationChannel(channel)
    }

    private fun maybeDeleteNotificationChannel(group: String, priority: Int) {
        notificationManager.deleteNotificationChannel(toChannelId(group, priority))
    }

    private fun maybeCreateNotificationGroup(id: String, name: String) {
        notificationManager.createNotificationChannelGroup(NotificationChannelGroup(id, name))
    }

    private fun maybeDeleteNotificationGroup(id: String) {
        notificationManager.deleteNotificationChannelGroup(id)
    }

    private fun toChannelId(groupId: String, priority: Int): String {
        return when (priority) {
            PRIORITY_MIN -> groupId + GROUP_SUFFIX_PRIORITY_MIN
            PRIORITY_LOW -> groupId + GROUP_SUFFIX_PRIORITY_LOW
            PRIORITY_HIGH -> groupId + GROUP_SUFFIX_PRIORITY_HIGH
            PRIORITY_MAX -> groupId + GROUP_SUFFIX_PRIORITY_MAX
            else -> groupId + GROUP_SUFFIX_PRIORITY_DEFAULT
        }
    }

    private fun maybePlayInsistentSound(groupId: String, insistent: Boolean) {
        if (!insistent) {
            return
        }
        try {
            val mediaPlayer = repository.mediaPlayer
            val audioManager = context.getSystemService(Context.AUDIO_SERVICE) as AudioManager
            if (audioManager.getStreamVolume(AudioManager.STREAM_ALARM) != 0) {
                Log.d(TAG, "Media player: Playing insistent alarm on alarm channel")
                mediaPlayer.reset()
                mediaPlayer.setDataSource(context, getInsistentSound(groupId))
                mediaPlayer.setAudioAttributes(AudioAttributes.Builder().setUsage(AudioAttributes.USAGE_ALARM).build())
                mediaPlayer.isLooping = true
                mediaPlayer.prepare()
                mediaPlayer.start()
            } else {
                Log.d(TAG, "Media player: Alarm volume is 0; not playing insistent alarm")
            }
        } catch (e: Exception) {
            Log.w(TAG, "Media player: Failed to play insistent alarm", e)
        }
    }

    private fun getInsistentSound(groupId: String): Uri {
        val channelId = toChannelId(groupId, PRIORITY_MAX)
        val channel = notificationManager.getNotificationChannel(channelId)
        return channel.sound
    }

    companion object {
        private const val TAG = "NtfyNotifService"
        private const val DEFAULT_GROUP = "ntfy"
        private const val SUBSCRIPTION_GROUP_PREFIX = "ntfy-subscription-"
        private const val GROUP_SUFFIX_PRIORITY_MIN = "-min"
        private const val GROUP_SUFFIX_PRIORITY_LOW = "-low"
        private const val GROUP_SUFFIX_PRIORITY_DEFAULT = ""
        private const val GROUP_SUFFIX_PRIORITY_HIGH = "-high"
        private const val GROUP_SUFFIX_PRIORITY_MAX = "-max"
    }
}
