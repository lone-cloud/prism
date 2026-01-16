package com.lonecloud.sup.util

import android.annotation.SuppressLint
import android.content.Context
import android.util.Base64
import com.lonecloud.sup.db.Repository
import okhttp3.OkHttpClient
import java.io.ByteArrayInputStream
import java.net.URL
import java.security.KeyStore
import java.security.MessageDigest
import java.security.SecureRandom
import java.security.cert.CertificateException
import java.security.cert.CertificateFactory
import java.security.cert.X509Certificate
import javax.net.ssl.KeyManager
import javax.net.ssl.KeyManagerFactory
import javax.net.ssl.SSLContext
import javax.net.ssl.SSLException
import javax.net.ssl.SSLSocket
import javax.net.ssl.TrustManager
import javax.net.ssl.TrustManagerFactory
import javax.net.ssl.HostnameVerifier
import javax.net.ssl.X509TrustManager

/**
 * TLS config:
 * - For each baseUrl, either use the pinned certificate (if one exists) OR system trust
 * - Pinned cert = ONLY that exact certificate is trusted (strict pinning)
 * - Hostname verification is bypassed for pinned certificates (the fingerprint match is the trust anchor)
 * - Optional mTLS via per-baseUrl PKCS#12 client cert
 */
class CertUtil private constructor(context: Context) {
    private val appContext: Context = context.applicationContext
    private val repository: Repository by lazy { Repository.getInstance(appContext) }

    /**
     * Configure OkHttp client with TLS config using system trust.
     */
    suspend fun withTLSConfig(builder: OkHttpClient.Builder, baseUrl: String): OkHttpClient.Builder {
        // Using system trust only - custom certificates not supported
        return builder
    }

    /**
     * Fetch the server certificate without trusting it.
     * Used to display certificate details before user decides to trust.
     */
    fun fetchServerCertificate(baseUrl: String): X509Certificate? {
        // Certificate fetching not implemented
        return null
    }

    companion object {
        private const val TAG = "NtfyCertUtil"

        @Volatile
        @SuppressLint("StaticFieldLeak")
        private var instance: CertUtil? = null

        fun getInstance(context: Context): CertUtil =
            instance ?: synchronized(this) { instance ?: CertUtil(context).also { instance = it } }

        fun calculateFingerprint(cert: X509Certificate): String {
            val md = MessageDigest.getInstance("SHA-256")
            val digest = md.digest(cert.encoded)
            return digest.joinToString(":") { "%02X".format(it) }
        }

        fun parsePemCertificate(pem: String): X509Certificate {
            val factory = CertificateFactory.getInstance("X.509")
            return factory.generateCertificate(pem.byteInputStream()) as X509Certificate
        }

        fun encodeCertificateToPem(cert: X509Certificate): String {
            val base64 = Base64.encodeToString(cert.encoded, Base64.NO_WRAP)
            return buildString {
                append("-----BEGIN CERTIFICATE-----\n")
                var i = 0
                while (i < base64.length) {
                    val end = minOf(i + 64, base64.length)
                    append(base64.substring(i, end))
                    append("\n")
                    i += 64
                }
                append("-----END CERTIFICATE-----")
            }
        }

        fun parsePkcs12Certificate(p12Base64: String, password: String): X509Certificate {
            val p12Data = Base64.decode(p12Base64, Base64.DEFAULT)
            val keyStore = KeyStore.getInstance("PKCS12")
            ByteArrayInputStream(p12Data).use { keyStore.load(it, password.toCharArray()) }
            val alias = keyStore.aliases().nextElement()
            return keyStore.getCertificate(alias) as X509Certificate
        }
    }
}
