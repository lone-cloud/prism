package com.lonecloud.sup

import android.app.Service
import android.content.Intent
import android.os.IBinder
import android.util.Log
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject

class DistributorService : Service() {
    private val client = OkHttpClient()
    private val prefs by lazy { 
        getSharedPreferences("sup_prefs", MODE_PRIVATE) 
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            "org.unifiedpush.android.distributor.REGISTER" -> handleRegister(intent)
            "org.unifiedpush.android.distributor.UNREGISTER" -> handleUnregister(intent)
        }
        return START_NOT_STICKY
    }

    private fun handleRegister(intent: Intent) {
        val token = intent.getStringExtra("token") ?: return
        val appId = intent.getStringExtra("application") ?: return
        
        Log.d("SUP", "Registering: app=$appId, token=$token")

        val serverUrl = prefs.getString("server_url", null)
        val apiKey = prefs.getString("api_key", null)

        if (serverUrl == null) {
            sendRegistrationRefused(token, "Server not configured")
            return
        }

        Thread {
            try {
                val json = JSONObject().apply {
                    put("appName", appId)
                }

                val request = Request.Builder()
                    .url("$serverUrl/up/$appId")
                    .post(json.toString().toRequestBody("application/json".toMediaType()))
                    .apply {
                        if (apiKey != null) {
                            addHeader("Authorization", "Bearer $apiKey")
                        }
                    }
                    .build()

                val response = client.newCall(request).execute()
                val responseBody = response.body?.string()

                if (response.isSuccessful && responseBody != null) {
                    val jsonResponse = JSONObject(responseBody)
                    val endpoint = jsonResponse.getString("endpoint")

                    prefs.edit()
                        .putString("endpoint_$appId", endpoint)
                        .putString("token_$appId", token)
                        .apply()

                    sendEndpoint(token, endpoint)
                    Log.d("SUP", "Registered successfully: $endpoint")
                } else {
                    sendRegistrationRefused(token, "Server error: ${response.code}")
                }
            } catch (e: Exception) {
                Log.e("SUP", "Registration failed", e)
                sendRegistrationRefused(token, "Failed: ${e.message}")
            }
        }.start()
    }

    private fun handleUnregister(intent: Intent) {
        val token = intent.getStringExtra("token") ?: return
        Log.d("SUP", "Unregistering: token=$token")

        val allPrefs = prefs.all
        for ((key, value) in allPrefs) {
            if (key.startsWith("token_") && value == token) {
                val appId = key.removePrefix("token_")
                prefs.edit()
                    .remove("endpoint_$appId")
                    .remove("token_$appId")
                    .apply()

                sendUnregistered(token)
                Log.d("SUP", "Unregistered: $appId")
                return
            }
        }
    }

    private fun sendEndpoint(token: String, endpoint: String) {
        val intent = Intent("org.unifiedpush.android.connector.MESSAGE").apply {
            putExtra("token", token)
            putExtra("endpoint", endpoint)
            `package` = getAppPackageFromToken(token)
        }
        sendBroadcast(intent)
    }

    private fun sendRegistrationRefused(token: String, message: String) {
        val intent = Intent("org.unifiedpush.android.connector.REGISTRATION_REFUSED").apply {
            putExtra("token", token)
            putExtra("message", message)
            `package` = getAppPackageFromToken(token)
        }
        sendBroadcast(intent)
    }

    private fun sendUnregistered(token: String) {
        val intent = Intent("org.unifiedpush.android.connector.UNREGISTERED").apply {
            putExtra("token", token)
            `package` = getAppPackageFromToken(token)
        }
        sendBroadcast(intent)
    }

    private fun getAppPackageFromToken(token: String): String {
        return token.split(":").firstOrNull() ?: ""
    }
}
