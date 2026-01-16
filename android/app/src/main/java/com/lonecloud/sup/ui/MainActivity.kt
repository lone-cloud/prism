package com.lonecloud.sup.ui

import android.Manifest
import android.animation.Animator
import android.animation.AnimatorListenerAdapter
import android.app.AlarmManager
import android.app.AlertDialog
import android.content.ActivityNotFoundException
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.provider.Settings
import android.provider.Settings.ACTION_REQUEST_SCHEDULE_EXACT_ALARM
import android.text.method.LinkMovementMethod
import android.view.Menu
import android.view.MenuItem
import android.view.View
import android.widget.Button
import android.widget.TextView
import android.widget.Toast
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import androidx.appcompat.app.AppCompatActivity
import androidx.appcompat.app.AppCompatDelegate
import androidx.appcompat.view.ActionMode
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import androidx.core.content.res.ResourcesCompat
import androidx.core.text.HtmlCompat
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import androidx.core.view.updatePadding
import androidx.core.view.isVisible
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.RecyclerView
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout
import com.google.android.material.appbar.AppBarLayout
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.floatingactionbutton.FloatingActionButton
import com.lonecloud.sup.BuildConfig
import com.lonecloud.sup.R
import com.lonecloud.sup.app.Application
import com.lonecloud.sup.db.Repository
import com.lonecloud.sup.db.Subscription
import com.lonecloud.sup.msg.ApiService
import com.lonecloud.sup.msg.NotificationDispatcher
import com.lonecloud.sup.util.Log
import com.lonecloud.sup.util.dangerButton
import com.lonecloud.sup.util.displayName
import com.lonecloud.sup.util.formatDateShort
import com.lonecloud.sup.util.isDarkThemeOn
import com.lonecloud.sup.util.isIgnoringBatteryOptimizations
import com.lonecloud.sup.util.maybeSplitTopicUrl
import com.lonecloud.sup.util.randomSubscriptionId
import com.lonecloud.sup.util.shortUrl
import com.lonecloud.sup.util.topicShortUrl
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.coroutines.isActive
import java.util.Date
import java.util.concurrent.TimeUnit
import kotlin.random.Random
import androidx.core.view.size
import androidx.core.view.get
import androidx.core.net.toUri

class MainActivity : AppCompatActivity(), AddFragment.SubscribeListener {
    private val viewModel by viewModels<SubscriptionsViewModel> {
        SubscriptionsViewModelFactory((application as Application).repository)
    }
    private val repository by lazy { (application as Application).repository }
    private val api by lazy { ApiService(this) }

    // UI elements
    private lateinit var menu: Menu
    private lateinit var mainList: RecyclerView
    private lateinit var adapter: MainAdapter
    private lateinit var fab: FloatingActionButton

    // Other stuff
    private var dispatcher: NotificationDispatcher? = null // Context-dependent
    private var appBaseUrl: String? = null // Context-dependent

    // Action mode stuff
    private var actionMode: ActionMode? = null
    private val actionModeCallback = object : ActionMode.Callback {
        override fun onCreateActionMode(mode: ActionMode?, menu: Menu?): Boolean {
            actionMode = mode
            if (mode != null) {
                mode.menuInflater.inflate(R.menu.menu_main_action_mode, menu)
                mode.title = "1" // One item selected
            }
            return true
        }

        override fun onPrepareActionMode(mode: ActionMode?, menu: Menu?) = false

        override fun onActionItemClicked(mode: ActionMode?, item: MenuItem): Boolean {
            return when (item.itemId) {
                R.id.main_action_mode_delete -> {
                    onMultiDeleteClick()
                    true
                }
                else -> false
            }
        }

        override fun onDestroyActionMode(mode: ActionMode?) {
            endActionModeAndRedraw()
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        Log.init(this) // Init logs in all entry points
        Log.d(TAG, "Create $this")

        // Dependencies that depend on Context
        dispatcher = NotificationDispatcher(this, repository)
        appBaseUrl = getString(R.string.app_base_url)

        // Action bar
        val toolbarLayout = findViewById<AppBarLayout>(R.id.app_bar_drawer)
        val dynamicColors = repository.getDynamicColorsEnabled()
        val darkMode = isDarkThemeOn(this)
        val statusBarColor = Colors.statusBarNormal(this, dynamicColors, darkMode)
        val toolbarTextColor = Colors.toolbarTextColor(this, dynamicColors, darkMode)
        toolbarLayout.setBackgroundColor(statusBarColor)
        
        val toolbar = toolbarLayout.findViewById<com.google.android.material.appbar.MaterialToolbar>(R.id.toolbar)
        toolbar.setTitleTextColor(toolbarTextColor)
        toolbar.setNavigationIconTint(toolbarTextColor)
        toolbar.overflowIcon?.setTint(toolbarTextColor)
        setSupportActionBar(toolbar)
        title = getString(R.string.main_action_bar_title)
        
        // Set system status bar appearance
        WindowInsetsControllerCompat(window, window.decorView).isAppearanceLightStatusBars =
            Colors.shouldUseLightStatusBar(dynamicColors, darkMode)

        // Floating action button ("+")
        fab = findViewById(R.id.fab)
        fab.setOnClickListener {
            onSubscribeButtonClick()
        }
        
        // Add bottom padding to FAB to account for navigation bar
        ViewCompat.setOnApplyWindowInsetsListener(fab) { view, insets ->
            val systemBars = insets.getInsets(WindowInsetsCompat.Type.systemBars())
            val layoutParams = view.layoutParams as androidx.constraintlayout.widget.ConstraintLayout.LayoutParams
            layoutParams.bottomMargin = systemBars.bottom
            view.layoutParams = layoutParams
            insets
        }

        // Update main list based on viewModel (& its datasource/livedata)
        val noEntries: View = findViewById(R.id.main_no_subscriptions)
        val onSubscriptionClick = { s: Subscription -> onSubscriptionItemClick(s) }
        val onSubscriptionLongClick = { s: Subscription -> onSubscriptionItemLongClick(s) }

        mainList = findViewById(R.id.main_subscriptions_list)
        adapter = MainAdapter(
            repository,
            onSubscriptionClick,
            onSubscriptionLongClick,
            ResourcesCompat.getDrawable(resources, R.drawable.ic_circle, theme)!!.apply {
                setTint(Colors.primary(this@MainActivity))
            },
            Colors.onPrimary(this)
        )
        mainList.adapter = adapter
        
        // Apply window insets to ensure content is not covered by navigation bar
        mainList.clipToPadding = false
        ViewCompat.setOnApplyWindowInsetsListener(mainList) { v, insets ->
            val systemBars = insets.getInsets(WindowInsetsCompat.Type.systemBars())
            v.updatePadding(bottom = systemBars.bottom)
            insets
        }

        viewModel.list().observe(this) {
            it?.let { subscriptions ->
                // Update main list
                adapter.submitList(subscriptions as MutableList<Subscription>)
                if (it.isEmpty()) {
                    mainList.visibility = View.GONE
                    noEntries.visibility = View.VISIBLE
                } else {
                    mainList.visibility = View.VISIBLE
                    noEntries.visibility = View.GONE
                }

                // Add scrub terms to log (in case it gets exported)
                subscriptions.forEach { s ->
                    Log.addScrubTerm(shortUrl(s.baseUrl), Log.TermType.Domain)
                    Log.addScrubTerm(s.topic)
                }

                // Update battery banner
                showHideBatteryBanner(subscriptions)
            }
        }


        // Scrub terms for last topics // FIXME this should be in Log.getFormatted
        repository.getLastShareTopics().forEach { topicUrl ->
            maybeSplitTopicUrl(topicUrl)?.let {
                Log.addScrubTerm(shortUrl(it.first), Log.TermType.Domain)
                Log.addScrubTerm(shortUrl(it.second), Log.TermType.Term)
            }
        }

        // React to changes in instant delivery setting
        viewModel.listIdsWithInstantStatus().observe(this) {
            // Signal pushes to us, no service to refresh
        }

        // Observe connection details and update menu item visibility
        repository.getConnectionDetailsLiveData().observe(this) { details ->
            showHideConnectionErrorMenuItem(details)
        }

        // Battery banner
        val batteryBanner = findViewById<View>(R.id.main_banner_battery) // Banner visibility is toggled in onResume()
        val dontAskAgainButton = findViewById<Button>(R.id.main_banner_battery_dontaskagain)
        val askLaterButton = findViewById<Button>(R.id.main_banner_battery_ask_later)
        val fixNowButton = findViewById<Button>(R.id.main_banner_battery_fix_now)
        dontAskAgainButton.setOnClickListener {
            batteryBanner.visibility = View.GONE
            repository.setBatteryOptimizationsRemindTime(Repository.BATTERY_OPTIMIZATIONS_REMIND_TIME_NEVER)
        }
        askLaterButton.setOnClickListener {
            batteryBanner.visibility = View.GONE
            repository.setBatteryOptimizationsRemindTime(System.currentTimeMillis() + ONE_DAY_MILLIS)
        }
        fixNowButton.setOnClickListener {
            try {
                Log.d(TAG, "package:$packageName".toUri().toString())
                startActivity(
                    Intent(
                        Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS,
                        "package:$packageName".toUri()
                    )
                )
            } catch (_: ActivityNotFoundException) {
                try {
                    startActivity(Intent(Settings.ACTION_IGNORE_BATTERY_OPTIMIZATION_SETTINGS))
                } catch (_: ActivityNotFoundException) {
                    startActivity(Intent(Settings.ACTION_SETTINGS))
                }
            }
            // Hide, at least for now
            val batteryBanner = findViewById<View>(R.id.main_banner_battery)
            batteryBanner.visibility = View.GONE
        }



        // Hide links that lead to payments, see https://github.com/binwiederhier/ntfy/issues/1463
        val howToLink = findViewById<TextView>(R.id.main_how_to_link)
        howToLink.isVisible = BuildConfig.PAYMENT_LINKS_AVAILABLE

        // Create notification channels right away, so we can configure them immediately after installing the app
        dispatcher?.init()

        // Signal pushes to us, no Firebase to subscribe to

        // Darrkkkk mode
        AppCompatDelegate.setDefaultNightMode(repository.getDarkMode())

        // Background things
        schedulePeriodicServiceRestartWorker()

        // Permissions
        maybeRequestNotificationPermission()
    }

    private fun maybeRequestNotificationPermission() {
        // Android 13 (SDK 33) requires that we ask for permission to post notifications
        // https://developer.android.com/develop/ui/views/notifications/notification-permission

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU && ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_DENIED) {
            ActivityCompat.requestPermissions(this, arrayOf(Manifest.permission.POST_NOTIFICATIONS), 0)
        }
    }

    override fun onResume() {
        super.onResume()
        showHideNotificationMenuItems()
        showHideConnectionErrorMenuItem(repository.getConnectionDetails())
        redrawList()
    }

    override fun onPause() {
        super.onPause()
    }

    private fun showHideBatteryBanner(subscriptions: List<Subscription>) {
        val batteryRemindTimeReached = repository.getBatteryOptimizationsRemindTime() < System.currentTimeMillis()
        val ignoringOptimizations = isIgnoringBatteryOptimizations(this@MainActivity)
        val showBanner = batteryRemindTimeReached && !ignoringOptimizations
        val batteryBanner = findViewById<View>(R.id.main_banner_battery)
        batteryBanner.visibility = if (showBanner) View.VISIBLE else View.GONE
        Log.d(TAG, "Battery: ignoring optimizations = $ignoringOptimizations (we want this to be true); remind time reached = $batteryRemindTimeReached; banner = $showBanner")
    }

    private fun schedulePeriodicServiceRestartWorker() {
        // Service restart worker not needed for Signal-based implementation
        Log.d(TAG, "ServiceStartWorker: Not scheduling (using Signal push notifications)")
    }

    override fun onCreateOptionsMenu(menu: Menu): Boolean {
        menuInflater.inflate(R.menu.menu_main_action_bar, menu)
        this.menu = menu
        
        // Tint menu icons based on theme
        val toolbarTextColor = Colors.toolbarTextColor(this, repository.getDynamicColorsEnabled(), isDarkThemeOn(this))
        for (i in 0 until menu.size) {
            menu[i].icon?.setTint(toolbarTextColor)
        }
        
        showHideNotificationMenuItems()
        showHideConnectionErrorMenuItem(repository.getConnectionDetails())
        checkSubscriptionsMuted() // This is done here, because then we know that we've initialized the menu
        return true
    }

    private fun checkSubscriptionsMuted(delayMillis: Long = 0L) {
        lifecycleScope.launch(Dispatchers.IO) {
            delay(delayMillis) // Just to be sure we've initialized all the things, we wait a bit ...
            Log.d(TAG, "Checking global and subscription-specific 'muted until' timestamp")

            // Check global
            val changed = repository.checkGlobalMutedUntil()
            if (changed) {
                Log.d(TAG, "Global muted until timestamp expired; updating prefs")
                showHideNotificationMenuItems()
            }

            // Check subscriptions
            var rerenderList = false
            repository.getSubscriptions().forEach { subscription ->
                val mutedUntilExpired = subscription.mutedUntil > 1L && System.currentTimeMillis()/1000 > subscription.mutedUntil
                if (mutedUntilExpired) {
                    Log.d(TAG, "Subscription ${subscription.id}: Muted until timestamp expired, updating subscription")
                    val newSubscription = subscription.copy(mutedUntil = 0L)
                    repository.updateSubscription(newSubscription)
                    rerenderList = true
                }
            }
            if (rerenderList) {
                mainList.post {
                    redrawList()
                }
            }
        }
    }

    private fun showHideNotificationMenuItems() {
        if (!this::menu.isInitialized) {
            return
        }
        val mutedUntilSeconds = repository.getGlobalMutedUntil()
        runOnUiThread {
            // Show/hide menu items based on build config
            val rateAppItem = menu.findItem(R.id.main_menu_rate)
            val docsItem = menu.findItem(R.id.main_menu_docs)
            val reportBugItem = menu.findItem(R.id.main_menu_report_bug)
            rateAppItem.isVisible = BuildConfig.RATE_APP_AVAILABLE
            docsItem.isVisible = BuildConfig.PAYMENT_LINKS_AVAILABLE // Google Payments Policy, see https://github.com/binwiederhier/ntfy/issues/1463
            reportBugItem.isVisible = BuildConfig.PAYMENT_LINKS_AVAILABLE // Google Payments Policy, see https://github.com/binwiederhier/ntfy/issues/1463

            // Pause notification icons
            val notificationsEnabledItem = menu.findItem(R.id.main_menu_notifications_enabled)
            val notificationsDisabledUntilItem = menu.findItem(R.id.main_menu_notifications_disabled_until)
            val notificationsDisabledForeverItem = menu.findItem(R.id.main_menu_notifications_disabled_forever)
            notificationsEnabledItem?.isVisible = mutedUntilSeconds == 0L
            notificationsDisabledForeverItem?.isVisible = mutedUntilSeconds == 1L
            notificationsDisabledUntilItem?.isVisible = mutedUntilSeconds > 1L
            if (mutedUntilSeconds > 1L) {
                val formattedDate = formatDateShort(mutedUntilSeconds)
                notificationsDisabledUntilItem?.title = getString(R.string.main_menu_notifications_disabled_until, formattedDate)
            }
        }
    }

    private fun showHideConnectionErrorMenuItem(details: Map<String, com.lonecloud.sup.db.ConnectionDetails>) {
        if (!this::menu.isInitialized) {
            return
        }
        runOnUiThread {
            val connectionErrorItem = menu.findItem(R.id.main_menu_connection_error)
            val hasErrors = details.values.any { it.hasError() }
            connectionErrorItem?.isVisible = hasErrors
        }
    }

    override fun onOptionsItemSelected(item: MenuItem): Boolean {
        return when (item.itemId) {
            R.id.main_menu_notifications_enabled -> {
                onNotificationSettingsClick(enable = false)
                true
            }
            R.id.main_menu_notifications_disabled_forever -> {
                onNotificationSettingsClick(enable = true)
                true
            }
            R.id.main_menu_notifications_disabled_until -> {
                onNotificationSettingsClick(enable = true)
                true
            }
            R.id.main_menu_connection_error -> {
                onConnectionErrorClick()
                true
            }
            R.id.main_menu_settings -> {
                // Settings activity not implemented
                Log.d(TAG, "Settings not available")
                true
            }
            R.id.main_menu_report_bug -> {
                startActivity(
                    Intent(Intent.ACTION_VIEW, getString(R.string.main_menu_report_bug_url).toUri())
                )
                true
            }
            R.id.main_menu_rate -> {
                try {
                    startActivity(
                        Intent(Intent.ACTION_VIEW, "market://details?id=$packageName".toUri())
                    )
                } catch (_: ActivityNotFoundException) {
                    startActivity(
                        Intent(Intent.ACTION_VIEW, "https://play.google.com/store/apps/details?id=$packageName".toUri())
                    )
                }
                true
            }
            R.id.main_menu_docs -> {
                startActivity(
                    Intent(Intent.ACTION_VIEW, getString(R.string.main_menu_docs_url).toUri())
                )
                true
            }
            else -> super.onOptionsItemSelected(item)
        }
    }

    private fun onNotificationSettingsClick(enable: Boolean) {
        if (!enable) {
            Log.d(TAG, "Notification settings dialog not available")
        } else {
            Log.d(TAG, "Re-enabling global notifications")
            onNotificationMutedUntilChanged(Repository.MUTED_UNTIL_SHOW_ALL)
        }
    }

    private fun onConnectionErrorClick() {
        Log.d(TAG, "Connection error dialog not available")
    }

    fun onNotificationMutedUntilChanged(mutedUntilTimestamp: Long) {
        repository.setGlobalMutedUntil(mutedUntilTimestamp)
        showHideNotificationMenuItems()
        runOnUiThread {
            redrawList() // Update the "muted until" icons
            when (mutedUntilTimestamp) {
                0L -> Toast.makeText(this@MainActivity, getString(R.string.notification_dialog_enabled_toast_message), Toast.LENGTH_LONG).show()
                1L -> Toast.makeText(this@MainActivity, getString(R.string.notification_dialog_muted_forever_toast_message), Toast.LENGTH_LONG).show()
                else -> {
                    val formattedDate = formatDateShort(mutedUntilTimestamp)
                    Toast.makeText(this@MainActivity, getString(R.string.notification_dialog_muted_until_toast_message, formattedDate), Toast.LENGTH_LONG).show()
                }
            }
        }
    }

    private fun onSubscribeButtonClick() {
        val newFragment = AddFragment()
        newFragment.show(supportFragmentManager, AddFragment.TAG)
    }

    override fun onSubscribe(topic: String, baseUrl: String, instant: Boolean) {
        Log.d(TAG, "Adding subscription ${topicShortUrl(baseUrl, topic)} (instant = $instant)")

        // Add subscription to database
        val subscription = Subscription(
            id = randomSubscriptionId(),
            baseUrl = baseUrl,
            topic = topic,
            mutedUntil = 0,
            minPriority = Repository.MIN_PRIORITY_USE_GLOBAL,
            autoDelete = Repository.AUTO_DELETE_USE_GLOBAL,
            insistent = Repository.INSISTENT_MAX_PRIORITY_USE_GLOBAL,
            upAppId = null,
            upConnectorToken = null,
            displayName = null,
            totalCount = 0,
            newCount = 0,
            lastActive = Date().time/1000
        )
        viewModel.add(subscription)

        // Signal pushes to us, no Firebase to subscribe to

        // Fetch cached messages
        lifecycleScope.launch(Dispatchers.IO) {
            try {
                val notifications = api.poll(subscription.id, subscription.baseUrl, subscription.topic)
                notifications.forEach { notification ->
                    repository.addNotification(notification)
                    // Icon download not implemented
                }
            } catch (e: Exception) {
                Log.e(TAG, "Unable to fetch notifications: ${e.message}", e)
            }
        }

        // Switch to detail view after adding it
        onSubscriptionItemClick(subscription)
    }

    private fun onSubscriptionItemClick(subscription: Subscription) {
        if (actionMode != null) {
            handleActionModeClick(subscription)
        } else if (subscription.upAppId != null) {
            startDetailSettingsView(subscription)
        }
    }

    private fun onSubscriptionItemLongClick(subscription: Subscription) {
        if (actionMode == null) {
            beginActionMode(subscription)
        }
    }



    private fun startDetailSettingsView(subscription: Subscription) {
        Log.d(TAG, "Opening subscription settings for ${topicShortUrl(subscription.baseUrl, subscription.topic)}")

        // Detail settings removed
        // val intent = Intent(this, DetailSettingsActivity::class.java)
        // intent.putExtra(DetailActivity.EXTRA_SUBSCRIPTION_ID, subscription.id)
        // intent.putExtra(DetailActivity.EXTRA_SUBSCRIPTION_BASE_URL, subscription.baseUrl)
        // intent.putExtra(DetailActivity.EXTRA_SUBSCRIPTION_TOPIC, subscription.topic)
        // intent.putExtra(DetailActivity.EXTRA_SUBSCRIPTION_DISPLAY_NAME, displayName(appBaseUrl, subscription))
        // startActivity(intent)
    }

    private fun handleActionModeClick(subscription: Subscription) {
        adapter.toggleSelection(subscription.id)
        if (adapter.selected.size == 0) {
            finishActionMode()
        } else {
            actionMode!!.title = adapter.selected.size.toString()
        }
    }

    private fun onMultiDeleteClick() {
        Log.d(TAG, "Showing multi-delete dialog for selected items")

        val dialog = MaterialAlertDialogBuilder(this)
            .setMessage(R.string.main_action_mode_delete_dialog_message)
            .setPositiveButton(R.string.main_action_mode_delete_dialog_permanently_delete) { _, _ ->
                adapter.selected.map { subscriptionId -> viewModel.remove(this, subscriptionId) }
                finishActionMode()
            }
            .setNegativeButton(R.string.main_action_mode_delete_dialog_cancel) { _, _ ->
                finishActionMode()
            }
            .create()
        dialog.setOnShowListener {
            dialog
                .getButton(AlertDialog.BUTTON_POSITIVE)
                .dangerButton()
        }
        dialog.show()
    }

    private fun beginActionMode(subscription: Subscription) {
        actionMode = startSupportActionMode(actionModeCallback)
        adapter.toggleSelection(subscription.id)

            // Fade out FAB
        fab.alpha = 1f
        fab
            .animate()
            .alpha(0f)
            .setDuration(ANIMATION_DURATION)
            .setListener(object : AnimatorListenerAdapter() {
                override fun onAnimationEnd(animation: Animator) {
                    fab.visibility = View.GONE
                }
            })
    }

    private fun finishActionMode() {
        actionMode!!.finish()
        endActionModeAndRedraw()
    }

    private fun endActionModeAndRedraw() {
        actionMode = null
        adapter.selected.clear()
        redrawList()

        // Fade in FAB
        fab.alpha = 0f
        fab.visibility = View.VISIBLE
        fab
            .animate()
            .alpha(1f)
            .setDuration(ANIMATION_DURATION)
            .setListener(object : AnimatorListenerAdapter() {
                override fun onAnimationEnd(animation: Animator) {
                    fab.visibility = View.VISIBLE // Required to replace the old listener
                }
            })
    }

    private fun redrawList() {
        if (!this::mainList.isInitialized) {
            return
        }
        adapter.notifyItemRangeChanged(0, adapter.currentList.size)
    }

    companion object {
        const val TAG = "NtfyMainActivity"
        const val EXTRA_SUBSCRIPTION_ID = "subscriptionId"
        const val EXTRA_SUBSCRIPTION_BASE_URL = "subscriptionBaseUrl"
        const val EXTRA_SUBSCRIPTION_TOPIC = "subscriptionTopic"
        const val EXTRA_SUBSCRIPTION_DISPLAY_NAME = "subscriptionDisplayName"
        const val EXTRA_SUBSCRIPTION_MUTED_UNTIL = "subscriptionMutedUntil"
        const val ANIMATION_DURATION = 80L
        const val ONE_DAY_MILLIS = 86400000L

        // As per documentation: The minimum repeat interval that can be defined is 15 minutes
        // (same as the JobScheduler API), but in practice 15 doesn't work. Using 16 here.
        // Thanks to varunon9 (https://gist.github.com/varunon9/f2beec0a743c96708eb0ef971a9ff9cd) for this!

        const val POLL_WORKER_INTERVAL_MINUTES = 60L
        const val SERVICE_START_WORKER_INTERVAL_MINUTES = 3 * 60L
    }
}
