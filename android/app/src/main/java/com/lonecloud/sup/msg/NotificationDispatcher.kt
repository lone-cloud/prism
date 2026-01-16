package com.lonecloud.sup.msg

import android.content.Context
import com.lonecloud.sup.db.Notification
import com.lonecloud.sup.db.Repository
import com.lonecloud.sup.db.Subscription
import com.lonecloud.sup.util.Log
import com.lonecloud.sup.up.Distributor
import com.lonecloud.sup.util.decodeBytesMessage
import com.lonecloud.sup.util.safeLet

/**
 * The notification dispatcher figures out what to do with a notification.
 * It may display a notification, send out a broadcast, or forward via UnifiedPush.
 */
class NotificationDispatcher(val context: Context, val repository: Repository) {
    private val notifier = NotificationService(context)
    private val broadcaster = BroadcastService(context)
    private val distributor = Distributor(context)

    fun init() {
        notifier.createNotificationChannels()
    }

    fun dispatch(subscription: Subscription, notification: Notification) {
        Log.d(TAG, "Dispatching $notification for subscription $subscription")

        val muted = getMuted(subscription)
        val notify = shouldNotify(subscription, notification, muted)
        val broadcast = shouldBroadcast(subscription)
        val distribute = shouldDistribute(subscription)
        if (notify) {
            notifier.display(subscription, notification)
        }
        if (broadcast) {
            broadcaster.sendMessage(subscription, notification, muted)
        }
        if (distribute) {
            safeLet(subscription.upAppId, subscription.upConnectorToken) { appId, connectorToken ->
                distributor.sendMessage(appId, connectorToken, decodeBytesMessage(notification))
            }
        }
    }

    private fun shouldNotify(subscription: Subscription, notification: Notification, muted: Boolean): Boolean {
        if (subscription.upAppId != null) {
            return false
        }
        val priority = if (notification.priority > 0) notification.priority else 3
        val minPriority = if (subscription.minPriority > 0) subscription.minPriority else repository.getMinPriority()
        if (priority < minPriority) {
            return false
        }
        val detailsVisible = repository.detailViewSubscriptionId.get() == notification.subscriptionId
        return !detailsVisible && !muted
    }

    private fun shouldBroadcast(subscription: Subscription): Boolean {
        if (subscription.upAppId != null) { // Never broadcast for UnifiedPush subscriptions
            return false
        }
        return repository.getBroadcastEnabled()
    }

    private fun shouldDistribute(subscription: Subscription): Boolean {
        return subscription.upAppId != null // Only distribute for UnifiedPush subscriptions
    }

    private fun getMuted(subscription: Subscription): Boolean {
        if (repository.isGlobalMuted()) {
            return true
        }
        return subscription.mutedUntil == 1L || (subscription.mutedUntil > 1L && subscription.mutedUntil > System.currentTimeMillis()/1000)
    }

    companion object {
        private const val TAG = "NtfyNotifDispatch"
    }
}
