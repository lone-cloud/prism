package com.lonecloud.sup.db

import android.content.Context
import androidx.room.ColumnInfo
import androidx.room.Dao
import androidx.room.Entity
import androidx.room.Ignore
import androidx.room.Index
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.PrimaryKey
import androidx.room.Query
import androidx.room.Room
import androidx.room.RoomDatabase
import androidx.room.Update
import androidx.room.migration.Migration
import androidx.sqlite.db.SupportSQLiteDatabase
import com.lonecloud.sup.service.NotAuthorizedException
import com.lonecloud.sup.service.hasCause
import kotlinx.coroutines.flow.Flow
import java.net.ConnectException

@Entity(indices = [Index(value = ["baseUrl", "topic"], unique = true), Index(value = ["upConnectorToken"], unique = true)])
data class Subscription(
    @PrimaryKey val id: Long,
    @ColumnInfo(name = "baseUrl") val baseUrl: String,
    @ColumnInfo(name = "topic") val topic: String,
    @ColumnInfo(name = "mutedUntil") val mutedUntil: Long,
    @ColumnInfo(name = "minPriority") val minPriority: Int,
    @ColumnInfo(name = "autoDelete") val autoDelete: Long, // Seconds
    @ColumnInfo(name = "insistent") val insistent: Int, // Ring constantly for max priority notifications (-1 = use global, 0 = off, 1 = on)
    @ColumnInfo(name = "upAppId") val upAppId: String?, // UnifiedPush application package name
    @ColumnInfo(name = "upConnectorToken") val upConnectorToken: String?, // UnifiedPush connector token
    @ColumnInfo(name = "displayName") val displayName: String?,
    @Ignore val totalCount: Int = 0, // Total notifications
    @Ignore val newCount: Int = 0, // New notifications
    @Ignore val lastActive: Long = 0, // Unix timestamp
    @Ignore val connectionDetails: ConnectionDetails = ConnectionDetails()
) {
    constructor(
        id: Long,
        baseUrl: String,
        topic: String,
        mutedUntil: Long,
        minPriority: Int,
        autoDelete: Long,
        insistent: Int,
        upAppId: String?,
        upConnectorToken: String?,
        displayName: String?
    ) :
            this(
                id,
                baseUrl,
                topic,
                mutedUntil,
                minPriority,
                autoDelete,
                insistent,
                upAppId,
                upConnectorToken,
                displayName,
                totalCount = 0,
                newCount = 0,
                lastActive = 0,
                connectionDetails = ConnectionDetails()
            )
}

enum class ConnectionState {
    NOT_APPLICABLE, CONNECTING, CONNECTED
}

data class ConnectionDetails(
    val state: ConnectionState = ConnectionState.NOT_APPLICABLE,
    val error: Throwable? = null,
    val nextRetryTime: Long = 0L
) {
    fun getStackTraceString(): String {
        return error?.stackTraceToString() ?: ""
    }
    
    fun hasError(): Boolean {
        return error != null
    }
    
    fun isConnectionRefused(): Boolean {
        return error?.hasCause(ConnectException::class.java) ?: false
    }

    fun isNotAuthorized(): Boolean {
        return error?.hasCause(NotAuthorizedException::class.java) ?: false
    }
}

data class SubscriptionWithMetadata(
    val id: Long,
    val baseUrl: String,
    val topic: String,
    val mutedUntil: Long,
    val autoDelete: Long,
    val minPriority: Int,
    val insistent: Int,
    val upAppId: String?,
    val upConnectorToken: String?,
    val displayName: String?,
    val totalCount: Int,
    val newCount: Int,
    val lastActive: Long
)

@Entity(primaryKeys = ["id", "subscriptionId"])
data class Notification(
    @ColumnInfo(name = "id") val id: String,
    @ColumnInfo(name = "subscriptionId") val subscriptionId: Long,
    @ColumnInfo(name = "timestamp") val timestamp: Long, // Unix timestamp
    @ColumnInfo(name = "title") val title: String,
    @ColumnInfo(name = "message") val message: String,
    @ColumnInfo(name = "notificationId") val notificationId: Int, // Android notification popup ID
    @ColumnInfo(name = "priority", defaultValue = "3") val priority: Int, // 1=min, 3=default, 5=max
    @ColumnInfo(name = "tags") val tags: String,
    @ColumnInfo(name = "deleted") val deleted: Boolean,
)

@Entity(tableName = "Log")
data class LogEntry(
    @PrimaryKey(autoGenerate = true) val id: Long,
    @ColumnInfo(name = "timestamp") val timestamp: Long,
    @ColumnInfo(name = "tag") val tag: String,
    @ColumnInfo(name = "level") val level: Int,
    @ColumnInfo(name = "message") val message: String,
    @ColumnInfo(name = "exception") val exception: String?
) {
    @Ignore constructor(timestamp: Long, tag: String, level: Int, message: String, exception: String?) :
            this(0, timestamp, tag, level, message, exception)
}

@androidx.room.Database(
    version = 17,
    entities = [
        Subscription::class,
        Notification::class,
        LogEntry::class
   ]
)
abstract class Database : RoomDatabase() {
    abstract fun subscriptionDao(): SubscriptionDao
    abstract fun notificationDao(): NotificationDao
    abstract fun logDao(): LogDao

    companion object {
        @Volatile
        private var instance: Database? = null

        fun getInstance(context: Context): Database {
            return instance ?: synchronized(this) {
                val instance = Room
                    .databaseBuilder(context.applicationContext, Database::class.java, "AppDatabase")
                    .addMigrations(MIGRATION_1_2)
                    .addMigrations(MIGRATION_2_3)
                    .addMigrations(MIGRATION_3_4)
                    .addMigrations(MIGRATION_4_5)
                    .addMigrations(MIGRATION_5_6)
                    .addMigrations(MIGRATION_6_7)
                    .addMigrations(MIGRATION_7_8)
                    .addMigrations(MIGRATION_8_9)
                    .addMigrations(MIGRATION_9_10)
                    .addMigrations(MIGRATION_10_11)
                    .addMigrations(MIGRATION_11_12)
                    .addMigrations(MIGRATION_12_13)
                    .addMigrations(MIGRATION_13_14)
                    .addMigrations(MIGRATION_14_15)
                    .addMigrations(MIGRATION_15_16)
                    .addMigrations(MIGRATION_16_17)
                    .fallbackToDestructiveMigration(true)
                    .build()
                this.instance = instance
                instance
            }
        }

        private val MIGRATION_1_2 = object : Migration(1, 2) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("CREATE TABLE Subscription_New (id INTEGER NOT NULL, baseUrl TEXT NOT NULL, topic TEXT NOT NULL, instant INTEGER NOT NULL DEFAULT('0'), PRIMARY KEY(id))")
                db.execSQL("INSERT INTO Subscription_New SELECT id, baseUrl, topic, 0 FROM Subscription")
                db.execSQL("DROP TABLE Subscription")
                db.execSQL("ALTER TABLE Subscription_New RENAME TO Subscription")
                db.execSQL("CREATE UNIQUE INDEX index_Subscription_baseUrl_topic ON Subscription (baseUrl, topic)")

                db.execSQL("ALTER TABLE Notification ADD COLUMN notificationId INTEGER NOT NULL DEFAULT('0')")
                db.execSQL("ALTER TABLE Notification ADD COLUMN deleted INTEGER NOT NULL DEFAULT('0')")
            }
        }

        private val MIGRATION_2_3 = object : Migration(2, 3) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("ALTER TABLE Subscription ADD COLUMN mutedUntil INTEGER NOT NULL DEFAULT('0')")
            }
        }

        private val MIGRATION_3_4 = object : Migration(3, 4) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("CREATE TABLE Notification_New (id TEXT NOT NULL, subscriptionId INTEGER NOT NULL, timestamp INTEGER NOT NULL, title TEXT NOT NULL, message TEXT NOT NULL, notificationId INTEGER NOT NULL, priority INTEGER NOT NULL DEFAULT(3), tags TEXT NOT NULL, deleted INTEGER NOT NULL, PRIMARY KEY(id, subscriptionId))")
                db.execSQL("INSERT INTO Notification_New SELECT id, subscriptionId, timestamp, '', message, notificationId, 3, '', deleted FROM Notification")
                db.execSQL("DROP TABLE Notification")
                db.execSQL("ALTER TABLE Notification_New RENAME TO Notification")
            }
        }

        private val MIGRATION_4_5 = object : Migration(4, 5) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("ALTER TABLE Subscription ADD COLUMN upAppId TEXT")
                db.execSQL("ALTER TABLE Subscription ADD COLUMN upConnectorToken TEXT")
                db.execSQL("CREATE UNIQUE INDEX index_Subscription_upConnectorToken ON Subscription (upConnectorToken)")
            }
        }

        private val MIGRATION_5_6 = object : Migration(5, 6) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("ALTER TABLE Notification ADD COLUMN click TEXT NOT NULL DEFAULT('')")
                db.execSQL("ALTER TABLE Notification ADD COLUMN attachment_name TEXT")
                db.execSQL("ALTER TABLE Notification ADD COLUMN attachment_type TEXT")
                db.execSQL("ALTER TABLE Notification ADD COLUMN attachment_size INT")
                db.execSQL("ALTER TABLE Notification ADD COLUMN attachment_expires INT")
                db.execSQL("ALTER TABLE Notification ADD COLUMN attachment_url TEXT")
                db.execSQL("ALTER TABLE Notification ADD COLUMN attachment_contentUri TEXT")
                db.execSQL("ALTER TABLE Notification ADD COLUMN attachment_progress INT")
            }
        }

        private val MIGRATION_6_7 = object : Migration(6, 7) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("CREATE TABLE Log (id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, timestamp INT NOT NULL, tag TEXT NOT NULL, level INT NOT NULL, message TEXT NOT NULL, exception TEXT)")
            }
        }

        private val MIGRATION_7_8 = object : Migration(7, 8) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("CREATE TABLE User (baseUrl TEXT NOT NULL, username TEXT NOT NULL, password TEXT NOT NULL, PRIMARY KEY(baseUrl))")
            }
        }

        private val MIGRATION_8_9 = object : Migration(8, 9) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("ALTER TABLE Notification ADD COLUMN encoding TEXT NOT NULL DEFAULT('')")
            }
        }

        private val MIGRATION_9_10 = object : Migration(9, 10) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("ALTER TABLE Notification ADD COLUMN actions TEXT")
            }
        }

        private val MIGRATION_10_11 = object : Migration(10, 11) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("ALTER TABLE Subscription ADD COLUMN minPriority INT NOT NULL DEFAULT (0)")
                db.execSQL("ALTER TABLE Subscription ADD COLUMN autoDelete INT NOT NULL DEFAULT (-1)")
                db.execSQL("ALTER TABLE Subscription ADD COLUMN icon TEXT")
            }
        }

        private val MIGRATION_11_12 = object : Migration(11, 12) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("ALTER TABLE Subscription ADD COLUMN lastNotificationId TEXT")
                db.execSQL("ALTER TABLE Subscription ADD COLUMN displayName TEXT")
                db.execSQL("ALTER TABLE Notification ADD COLUMN icon_url TEXT")
                db.execSQL("ALTER TABLE Notification ADD COLUMN icon_contentUri TEXT")
            }
        }

        private val MIGRATION_12_13 = object : Migration(12, 13) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("ALTER TABLE Subscription ADD COLUMN insistent INTEGER NOT NULL DEFAULT (-1)")
                db.execSQL("ALTER TABLE Subscription ADD COLUMN dedicatedChannels INTEGER NOT NULL DEFAULT (0)")
            }
        }

        private val MIGRATION_13_14 = object : Migration(13, 14) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("ALTER TABLE Notification ADD COLUMN contentType TEXT NOT NULL DEFAULT ('')")
            }
        }

        private val MIGRATION_14_15 = object : Migration(14, 15) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("CREATE TABLE CustomHeader (baseUrl TEXT NOT NULL, name TEXT NOT NULL, value TEXT NOT NULL, PRIMARY KEY(baseUrl, name))")
            }
        }

        private val MIGRATION_15_16 = object : Migration(15, 16) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("CREATE TABLE TrustedCertificate (baseUrl TEXT NOT NULL, pem TEXT NOT NULL, PRIMARY KEY(baseUrl))")
                db.execSQL("CREATE TABLE ClientCertificate (baseUrl TEXT NOT NULL, p12Base64 TEXT NOT NULL, password TEXT NOT NULL, PRIMARY KEY(baseUrl))")
            }
        }

        private val MIGRATION_16_17 = object : Migration(16, 17) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL("UPDATE Notification SET icon_contentUri = NULL WHERE icon_url IS NULL AND icon_contentUri IS NOT NULL")
            }
        }


    }
}

@Dao
interface SubscriptionDao {
    @Query("""
        SELECT 
          s.id, s.baseUrl, s.topic, s.mutedUntil, s.minPriority, s.autoDelete, s.insistent, s.upAppId, s.upConnectorToken, s.displayName,
          COUNT(n.id) totalCount, 
          COUNT(CASE n.notificationId WHEN 0 THEN NULL ELSE n.id END) newCount, 
          IFNULL(MAX(n.timestamp),0) AS lastActive
        FROM Subscription AS s
        LEFT JOIN Notification AS n ON s.id=n.subscriptionId AND n.deleted != 1
        GROUP BY s.id
        ORDER BY s.upAppId ASC, MAX(n.timestamp) DESC
    """)
    fun listFlow(): Flow<List<SubscriptionWithMetadata>>

    @Query("""
        SELECT 
          s.id, s.baseUrl, s.topic, s.mutedUntil, s.minPriority, s.autoDelete, s.insistent, s.upAppId, s.upConnectorToken, s.displayName,
          COUNT(n.id) totalCount, 
          COUNT(CASE n.notificationId WHEN 0 THEN NULL ELSE n.id END) newCount, 
          IFNULL(MAX(n.timestamp),0) AS lastActive
        FROM Subscription AS s
        LEFT JOIN Notification AS n ON s.id=n.subscriptionId AND n.deleted != 1
        GROUP BY s.id
        ORDER BY s.upAppId ASC, MAX(n.timestamp) DESC
    """)
    suspend fun list(): List<SubscriptionWithMetadata>

    @Query("""
        SELECT 
          s.id, s.baseUrl, s.topic, s.mutedUntil, s.minPriority, s.autoDelete, s.insistent, s.upAppId, s.upConnectorToken, s.displayName,
          COUNT(n.id) totalCount, 
          COUNT(CASE n.notificationId WHEN 0 THEN NULL ELSE n.id END) newCount, 
          IFNULL(MAX(n.timestamp),0) AS lastActive
        FROM Subscription AS s
        LEFT JOIN Notification AS n ON s.id=n.subscriptionId AND n.deleted != 1
        WHERE s.baseUrl = :baseUrl AND s.topic = :topic
        GROUP BY s.id
    """)
    fun get(baseUrl: String, topic: String): SubscriptionWithMetadata?

    @Query("""
        SELECT 
          s.id, s.baseUrl, s.topic, s.mutedUntil, s.minPriority, s.autoDelete, s.insistent, s.upAppId, s.upConnectorToken, s.displayName,
          COUNT(n.id) totalCount, 
          COUNT(CASE n.notificationId WHEN 0 THEN NULL ELSE n.id END) newCount, 
          IFNULL(MAX(n.timestamp),0) AS lastActive
        FROM Subscription AS s
        LEFT JOIN Notification AS n ON s.id=n.subscriptionId AND n.deleted != 1
        WHERE s.id = :subscriptionId
        GROUP BY s.id
    """)
    fun get(subscriptionId: Long): SubscriptionWithMetadata?

    @Query("""
        SELECT 
          s.id, s.baseUrl, s.topic, s.mutedUntil, s.minPriority, s.autoDelete, s.insistent, s.upAppId, s.upConnectorToken, s.displayName,
          COUNT(n.id) totalCount, 
          COUNT(CASE n.notificationId WHEN 0 THEN NULL ELSE n.id END) newCount, 
          IFNULL(MAX(n.timestamp),0) AS lastActive
        FROM Subscription AS s
        LEFT JOIN Notification AS n ON s.id=n.subscriptionId AND n.deleted != 1
        WHERE s.upConnectorToken = :connectorToken
        GROUP BY s.id
    """)
    fun getByConnectorToken(connectorToken: String): SubscriptionWithMetadata?

    @Query("""
        SELECT 
          s.id, s.baseUrl, s.topic, s.mutedUntil, s.minPriority, s.autoDelete, s.insistent, s.upAppId, s.upConnectorToken, s.displayName,
          COUNT(n.id) totalCount, 
          COUNT(CASE n.notificationId WHEN 0 THEN NULL ELSE n.id END) newCount, 
          IFNULL(MAX(n.timestamp),0) AS lastActive
        FROM Subscription AS s
        LEFT JOIN Notification AS n ON s.id=n.subscriptionId AND n.deleted != 1
        WHERE s.upAppId = :upAppId
        GROUP BY s.id
    """)
    fun getByUpAppId(upAppId: String): SubscriptionWithMetadata?

    @Insert
    fun add(subscription: Subscription)

    @Update
    fun update(subscription: Subscription)

    @Query("DELETE FROM subscription WHERE id = :subscriptionId")
    fun remove(subscriptionId: Long)
}

@Dao
interface NotificationDao {
    @Query("SELECT * FROM notification")
    suspend fun list(): List<Notification>

    @Query("SELECT * FROM notification WHERE subscriptionId = :subscriptionId AND deleted != 1 ORDER BY timestamp DESC")
    fun listFlow(subscriptionId: Long): Flow<List<Notification>>

    @Query("SELECT id FROM notification WHERE subscriptionId = :subscriptionId")
    fun listIds(subscriptionId: Long): List<String>

    @Insert(onConflict = OnConflictStrategy.IGNORE)
    fun add(notification: Notification)

    @Update(onConflict = OnConflictStrategy.IGNORE)
    fun update(notification: Notification)

    @Query("SELECT * FROM notification WHERE id = :notificationId")
    fun get(notificationId: String): Notification?

    @Query("UPDATE notification SET notificationId = 0 WHERE subscriptionId = :subscriptionId")
    fun clearAllNotificationIds(subscriptionId: Long)

    @Query("UPDATE notification SET deleted = 1 WHERE id = :notificationId")
    fun markAsDeleted(notificationId: String)

    @Query("UPDATE notification SET deleted = 1 WHERE subscriptionId = :subscriptionId")
    fun markAllAsDeleted(subscriptionId: Long)

    @Query("UPDATE notification SET deleted = 1 WHERE subscriptionId = :subscriptionId AND timestamp < :olderThanTimestamp")
    fun markAsDeletedIfOlderThan(subscriptionId: Long, olderThanTimestamp: Long)

    @Query("UPDATE notification SET deleted = 0 WHERE id = :notificationId")
    fun undelete(notificationId: String)

    @Query("DELETE FROM notification WHERE subscriptionId = :subscriptionId AND timestamp < :olderThanTimestamp")
    fun removeIfOlderThan(subscriptionId: Long, olderThanTimestamp: Long)

    @Query("DELETE FROM notification WHERE subscriptionId = :subscriptionId")
    fun removeAll(subscriptionId: Long)
}

@Dao
interface LogDao {
    @Insert
    suspend fun insert(entry: LogEntry)

    @Query("DELETE FROM log WHERE id NOT IN (SELECT id FROM log ORDER BY timestamp DESC, id DESC LIMIT :keepCount)")
    suspend fun prune(keepCount: Int)

    @Query("SELECT * FROM log ORDER BY timestamp ASC, id ASC")
    fun getAll(): List<LogEntry>

    @Query("DELETE FROM log")
    fun deleteAll()
}
