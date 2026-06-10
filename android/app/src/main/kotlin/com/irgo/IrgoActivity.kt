package com.irgo

import android.annotation.SuppressLint
import android.os.Bundle
import android.webkit.WebSettings
import android.webkit.WebView
import androidx.appcompat.app.AppCompatActivity

/**
 * Base activity for Irgo apps.
 * Subclass this in your app to customize behavior.
 */
open class IrgoActivity : AppCompatActivity() {

    lateinit var webView: WebView
        private set

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Initialize Go bridge
        IrgoBridge.initialize()

        // Create and configure WebView
        webView = createWebView()
        setContentView(webView)

        // Configure bridge
        IrgoBridge.configure(webView)

        // Load initial page
        loadInitialPage()
    }

    @SuppressLint("SetJavaScriptEnabled")
    protected open fun createWebView(): WebView {
        return WebView(this).apply {
            // Set custom WebViewClient
            webViewClient = IrgoWebViewClient()

            // Configure settings
            settings.apply {
                javaScriptEnabled = true
                domStorageEnabled = true
                databaseEnabled = true
                allowFileAccess = false
                allowContentAccess = false

                // Mobile-friendly settings
                useWideViewPort = true
                loadWithOverviewMode = true

                // Disable zoom for app-like experience
                setSupportZoom(false)
                builtInZoomControls = false
                displayZoomControls = false

                // Cache settings
                cacheMode = WebSettings.LOAD_DEFAULT
            }

            // Expose the unified native bridge to JavaScript as `window.Irgo`.
            // irgo-bridge.js routes Datastar's fetch() (HTTP) and WebSocket
            // traffic through this interface.
            addJavascriptInterface(IrgoJSInterface(this@IrgoActivity), "Irgo")
        }
    }

    protected open fun loadInitialPage() {
        val html = IrgoBridge.renderInitialPage()

        // The bridge script (irgo-bridge.js) is loaded by the page itself via
        // layout.templ and served through shouldInterceptRequest. The base URL
        // uses the irgo:// scheme so relative asset URLs resolve to the native
        // bridge.
        webView.loadDataWithBaseURL(
            "irgo://app/",
            html,
            "text/html",
            "UTF-8",
            null
        )
    }

    /**
     * Navigate to a path within the app
     */
    fun navigate(path: String) {
        var url = path
        if (!url.startsWith("irgo://")) {
            url = if (url.startsWith("/")) {
                "irgo://app$url"
            } else {
                "irgo://app/$url"
            }
        }
        webView.loadUrl(url)
    }

    /**
     * Evaluate JavaScript in the WebView
     */
    fun evaluateJavaScript(script: String, callback: ((String?) -> Unit)? = null) {
        webView.evaluateJavascript(script) { result ->
            callback?.invoke(result)
        }
    }

    override fun onBackPressed() {
        if (webView.canGoBack()) {
            webView.goBack()
        } else {
            super.onBackPressed()
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        IrgoBridge.shutdown()
    }
}
