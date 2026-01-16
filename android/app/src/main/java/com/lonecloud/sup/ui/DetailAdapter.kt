package com.lonecloud.sup.ui

import android.Manifest
import android.app.Activity
import android.content.*
import android.content.pm.PackageManager
import android.graphics.Bitmap
import android.os.Build
import android.os.Environment
import android.provider.MediaStore
import android.text.util.Linkify
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.*
import androidx.cardview.widget.CardView
import androidx.constraintlayout.helper.widget.Flow
import androidx.constraintlayout.widget.ConstraintLayout
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import androidx.core.content.FileProvider
import androidx.core.view.allViews
import androidx.recyclerview.widget.DiffUtil
import androidx.recyclerview.widget.ListAdapter
import androidx.recyclerview.widget.RecyclerView
import com.bumptech.glide.Glide
import com.google.android.material.button.MaterialButton
import com.stfalcon.imageviewer.StfalconImageViewer
import com.lonecloud.sup.R
import com.lonecloud.sup.db.*
import com.lonecloud.sup.msg.NotificationService
import com.lonecloud.sup.util.*
import io.noties.markwon.Markwon
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.launch
import me.saket.bettermovementmethod.BetterLinkMovementMethod
import androidx.core.net.toUri

class DetailAdapter(private val activity: Activity, private val lifecycleScope: CoroutineScope, private val repository: Repository, private val onClick: (Notification) -> Unit, private val onLongClick: (Notification) -> Unit) :
    ListAdapter<Notification, DetailAdapter.DetailViewHolder>(TopicDiffCallback) {
    private val markwon: Markwon = MarkwonFactory.createForMessage(activity)
    val selected = mutableSetOf<String>() // Notification IDs

    /* Creates and inflates view and return TopicViewHolder. */
    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): DetailViewHolder {
        val view = LayoutInflater.from(parent.context)
            .inflate(R.layout.fragment_detail_item, parent, false)
        return DetailViewHolder(activity, lifecycleScope, repository, markwon, view, selected, onClick, onLongClick)
    }

    /* Gets current topic and uses it to bind view. */
    override fun onBindViewHolder(holder: DetailViewHolder, position: Int) {
        holder.bind(getItem(position))
    }

    fun get(position: Int): Notification {
        return getItem(position)
    }

    fun toggleSelection(notificationId: String) {
        if (selected.contains(notificationId)) {
            selected.remove(notificationId)
        } else {
            selected.add(notificationId)
        }

        if (selected.isNotEmpty()) {
            val listIds = currentList.map { notification -> notification.id }
            val notificationPosition = listIds.indexOf(notificationId)
            notifyItemChanged(notificationPosition)
        }
    }

    /* ViewHolder for Topic, takes in the inflated view and the onClick behavior. */
    class DetailViewHolder(
        private val activity: Activity,
        private val lifecycleScope: CoroutineScope,
        private val repository: Repository,
        private val markwon: Markwon,
        itemView: View,
        private val selected: Set<String>,
        val onClick: (Notification) -> Unit,
        val onLongClick: (Notification) -> Unit
    ) :
        RecyclerView.ViewHolder(itemView) {
        private var notification: Notification? = null
        private val layout: View = itemView.findViewById(R.id.detail_item_layout)
        private val cardView: CardView = itemView.findViewById(R.id.detail_item_card)
        private val priorityImageView: ImageView = itemView.findViewById(R.id.detail_item_priority_image)
        private val dateView: TextView = itemView.findViewById(R.id.detail_item_date_text)
        private val titleView: TextView = itemView.findViewById(R.id.detail_item_title_text)
        private val messageView: TextView = itemView.findViewById(R.id.detail_item_message_text)
        private val newDotImageView: View = itemView.findViewById(R.id.detail_item_new_dot)
        private val tagsView: TextView = itemView.findViewById(R.id.detail_item_tags_text)
        private val menuButton: ImageButton = itemView.findViewById(R.id.detail_item_menu_button)
        private val actionsWrapperView: ConstraintLayout = itemView.findViewById(R.id.detail_item_actions_wrapper)
        private val actionsFlow: Flow = itemView.findViewById(R.id.detail_item_actions_flow)

        fun bind(notification: Notification) {
            this.notification = notification

            val context = itemView.context
            val unmatchedTags = unmatchedTags(splitTags(notification.tags))
            val message = formatMessage(notification)

            dateView.text = formatDateShort(notification.timestamp)
            messageView.autoLinkMask = Linkify.WEB_URLS or Linkify.EMAIL_ADDRESSES or Linkify.PHONE_NUMBERS
            messageView.text = message
            messageView.movementMethod = BetterLinkMovementMethod.getInstance()
            messageView.setOnClickListener {
                // Click & Long-click listeners on the text as well, because "autoLink=web" makes them
                // clickable, and so we cannot rely on the underlying card to perform the action.
                // It's weird because "layout" is the ripple-able, but the card is clickable.
                // See https://github.com/binwiederhier/ntfy/issues/226
                layout.ripple(lifecycleScope)
                onClick(notification)
            }
            messageView.setOnLongClickListener {
                onLongClick(notification); true
            }
            newDotImageView.visibility = if (notification.notificationId == 0) View.GONE else View.VISIBLE
            cardView.setOnClickListener { onClick(notification) }
            cardView.setOnLongClickListener { onLongClick(notification); true }
            if (notification.title != "") {
                titleView.visibility = View.VISIBLE
                titleView.text = formatTitle(notification)
            } else {
                titleView.visibility = View.GONE
            }
            if (unmatchedTags.isNotEmpty()) {
                tagsView.visibility = View.VISIBLE
                tagsView.text = context.getString(R.string.detail_item_tags, unmatchedTags.joinToString(", "))
            } else {
                tagsView.visibility = View.GONE
            }
            if (selected.contains(notification.id)) {
                cardView.setCardBackgroundColor(Colors.cardSelectedBackgroundColor(context))
            } else {
                cardView.setCardBackgroundColor(Colors.cardBackgroundColor(context))
            }
            renderPriority(context, notification)
            resetCardButtons()
            maybeRenderMenu(context, notification)
        }

        private fun renderPriority(context: Context, notification: Notification) {
            when (notification.priority) {
                PRIORITY_MIN -> {
                    priorityImageView.visibility = View.VISIBLE
                    priorityImageView.setImageDrawable(ContextCompat.getDrawable(context, R.drawable.ic_priority_1_24dp))
                }
                PRIORITY_LOW -> {
                    priorityImageView.visibility = View.VISIBLE
                    priorityImageView.setImageDrawable(ContextCompat.getDrawable(context, R.drawable.ic_priority_2_24dp))
                }
                PRIORITY_DEFAULT -> {
                    priorityImageView.visibility = View.GONE
                }
                PRIORITY_HIGH -> {
                    priorityImageView.visibility = View.VISIBLE
                    priorityImageView.setImageDrawable(ContextCompat.getDrawable(context, R.drawable.ic_priority_4_24dp))
                }
                PRIORITY_MAX -> {
                    priorityImageView.visibility = View.VISIBLE
                    priorityImageView.setImageDrawable(ContextCompat.getDrawable(context, R.drawable.ic_priority_5_24dp))
                }
            }
        }

        private fun maybeRenderMenu(context: Context, notification: Notification) {
            val menuButtonPopupMenu = maybeCreateMenuPopup(context, menuButton, notification) // Heavy lifting not during on-click
            if (menuButtonPopupMenu != null) {
                menuButton.setOnClickListener { menuButtonPopupMenu.show() }
                menuButton.visibility = View.VISIBLE
            } else {
                menuButton.visibility = View.GONE
            }
        }

        private fun resetCardButtons() {
            // clear any previously created dynamic buttons
            actionsFlow.allViews.forEach { actionsFlow.removeView(it) }
            actionsWrapperView.removeAllViews()
            actionsWrapperView.addView(actionsFlow)
        }

        private fun addButtonToCard(button: View) {
            actionsWrapperView.addView(button)
            actionsFlow.addView(button)
        }

        private fun createCardButton(context: Context, label: String, onClick: () -> Boolean): View {
            // See https://stackoverflow.com/a/41139179/1440785
            val button = LayoutInflater.from(context).inflate(R.layout.button_action, null) as MaterialButton
            button.id = View.generateViewId()
            button.text = label
            button.setOnClickListener { onClick() }
            return button
        }

        private fun maybeCreateMenuPopup(context: Context, anchor: View?, notification: Notification): PopupMenu? {
            val popup = PopupMenu(context, anchor)
            popup.menuInflater.inflate(R.menu.menu_detail_attachment, popup.menu)
            val downloadItem = popup.menu.findItem(R.id.detail_item_menu_download)
            val cancelItem = popup.menu.findItem(R.id.detail_item_menu_cancel)
            val openItem = popup.menu.findItem(R.id.detail_item_menu_open)
            val deleteItem = popup.menu.findItem(R.id.detail_item_menu_delete)
            val saveFileItem = popup.menu.findItem(R.id.detail_item_menu_save_file)
            val copyUrlItem = popup.menu.findItem(R.id.detail_item_menu_copy_url)
            val copyContentsItem = popup.menu.findItem(R.id.detail_item_menu_copy_contents)
            
            copyContentsItem.setOnMenuItemClickListener {
                copyToClipboard(context, "notification", decodeMessage(notification)); true
            }
            
            openItem.isVisible = false
            downloadItem.isVisible = false
            deleteItem.isVisible = false
            saveFileItem.isVisible = false
            copyUrlItem.isVisible = false
            cancelItem.isVisible = false
            copyContentsItem.isVisible = true
            
            return popup
        }
    }

    object TopicDiffCallback : DiffUtil.ItemCallback<Notification>() {
        override fun areItemsTheSame(oldItem: Notification, newItem: Notification): Boolean {
            return oldItem.id == newItem.id
        }

        override fun areContentsTheSame(oldItem: Notification, newItem: Notification): Boolean {
            return oldItem == newItem
        }
    }

    companion object {
        const val TAG = "NtfyDetailAdapter"
        const val REQUEST_CODE_WRITE_STORAGE_PERMISSION_FOR_DOWNLOAD = 9876
        const val IMAGE_PREVIEW_MAX_BYTES = 5 * 1024 * 1024 // Too large images crash the app with "Canvas: trying to draw too large(233280000bytes) bitmap."
    }
}
