package com.lonecloud.sup

import android.content.Intent
import android.os.Bundle
import android.provider.Settings
import android.widget.Button
import android.widget.EditText
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity

class MainActivity : AppCompatActivity() {

    private lateinit var serverUrlInput: EditText
    private lateinit var apiKeyInput: EditText
    private lateinit var saveButton: Button
    private lateinit var enableListenerButton: Button

    private val prefs by lazy { 
        getSharedPreferences("sup_prefs", MODE_PRIVATE) 
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        serverUrlInput = findViewById(R.id.server_url)
        apiKeyInput = findViewById(R.id.api_key)
        saveButton = findViewById(R.id.save_button)
        enableListenerButton = findViewById(R.id.enable_listener_button)

        // Load saved settings
        serverUrlInput.setText(prefs.getString("server_url", ""))
        apiKeyInput.setText(prefs.getString("api_key", ""))

        saveButton.setOnClickListener {
            val serverUrl = serverUrlInput.text.toString().trim()
            val apiKey = apiKeyInput.text.toString().trim()

            prefs.edit()
                .putString("server_url", serverUrl)
                .putString("api_key", apiKey)
                .apply()

            Toast.makeText(this, "Settings saved", Toast.LENGTH_SHORT).show()
        }

        enableListenerButton.setOnClickListener {
            val intent = Intent(Settings.ACTION_NOTIFICATION_LISTENER_SETTINGS)
            startActivity(intent)
        }
    }
}
