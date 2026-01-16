package com.lonecloud.sup.work

import android.content.Context
import androidx.core.content.FileProvider
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.lonecloud.sup.BuildConfig
import com.lonecloud.sup.db.Repository
import com.lonecloud.sup.util.Log
import com.lonecloud.sup.util.maybeFileStat
import com.lonecloud.sup.util.topicShortUrl
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File
import androidx.core.net.toUri

/**
 * Deletes notifications marked for deletion and attachments for deleted notifications.
 */
class DeleteWorker(ctx: Context, params: WorkerParameters) : CoroutineWorker(ctx, params) {
    // IMPORTANT:
    //   Every time the worker is changed, the periodic work has to be REPLACEd.
    //   This is facilitated in the MainActivity using the VERSION below.

    init {
        Log.init(ctx) // Init in all entrypoints
    }

    override suspend fun doWork(): Result {
        return withContext(Dispatchers.IO) {
            try {
                deleteExpiredNotifications()
            } catch (e: Exception) {
                Log.w(TAG, "Failed to delete expired notifications", e)
            }
            return@withContext Result.success()
        }
    }

    private suspend fun deleteExpiredNotifications() {
        Log.d(TAG, "Deleting expired notifications")
        val repository = Repository.getInstance(applicationContext)
        val subscriptions = repository.getSubscriptions()
        subscriptions.forEach { subscription ->
            val logId = topicShortUrl(subscription.baseUrl, subscription.topic)
            val deleteAfterSeconds = if (subscription.autoDelete == Repository.AUTO_DELETE_USE_GLOBAL) {
                repository.getAutoDeleteSeconds()
            } else {
                subscription.autoDelete
            }
            if (deleteAfterSeconds == Repository.AUTO_DELETE_NEVER) {
                Log.d(TAG, "[$logId] Not deleting any notifications; global setting set to NEVER")
                return@forEach
            }

            // Mark as deleted
            val markDeletedOlderThanTimestamp = (System.currentTimeMillis()/1000) - deleteAfterSeconds
            Log.d(TAG, "[$logId] Marking notifications older than $markDeletedOlderThanTimestamp as deleted")
            repository.markAsDeletedIfOlderThan(subscription.id, markDeletedOlderThanTimestamp)

            // Hard delete
            val deleteOlderThanTimestamp = (System.currentTimeMillis()/1000) - HARD_DELETE_AFTER_SECONDS
            Log.d(TAG, "[$logId] Hard deleting notifications older than $markDeletedOlderThanTimestamp")
            repository.removeNotificationsIfOlderThan(subscription.id, deleteOlderThanTimestamp)
        }
    }

    companion object {
        const val VERSION = BuildConfig.VERSION_CODE
        const val TAG = "NtfyDeleteWorker"
        const val WORK_NAME_PERIODIC_ALL = "NtfyDeleteWorkerPeriodic" // Do not change

        private const val ONE_DAY_SECONDS = 24 * 60 * 60L
        const val HARD_DELETE_AFTER_SECONDS = 4 * 30 * ONE_DAY_SECONDS // 4 months
    }
}
