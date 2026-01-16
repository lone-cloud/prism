package com.lonecloud.sup.msg

import com.google.gson.Gson
import com.lonecloud.sup.db.Notification
import com.lonecloud.sup.util.joinTags
import com.lonecloud.sup.util.toPriority
import java.lang.reflect.Type

class NotificationParser {
    private val gson = Gson()

    fun parse(s: String, subscriptionId: Long = 0, notificationId: Int = 0): Notification? {
        val notificationWithTopic = parseWithTopic(s, subscriptionId = subscriptionId, notificationId = notificationId)
        return notificationWithTopic?.notification
    }

    fun parseWithTopic(s: String, subscriptionId: Long = 0, notificationId: Int = 0): NotificationWithTopic? {
        val message = gson.fromJson(s, Message::class.java)
        if (message.event != ApiService.EVENT_MESSAGE) {
            return null
        }
        val notification = Notification(
            id = message.id,
            subscriptionId = subscriptionId,
            timestamp = message.time,
            title = message.title ?: "",
            message = message.message,
            priority = toPriority(message.priority),
            tags = joinTags(message.tags),
            notificationId = notificationId,
            deleted = false
        )
        return NotificationWithTopic(message.topic, notification)
    }

    data class NotificationWithTopic(val topic: String, val notification: Notification)
}
