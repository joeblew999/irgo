package com.irgo

import android.net.Uri
import android.webkit.WebResourceRequest
import android.webkit.WebResourceResponse
import android.webkit.WebView
import android.webkit.WebViewClient
import java.io.ByteArrayInputStream

/**
 * Custom WebViewClient that intercepts irgo:// resource loads (navigation and
 * static assets such as CSS/JS) and routes them to Go.
 *
 * NOTE: Datastar's fetch() traffic does NOT flow through here. On Android,
 * shouldInterceptRequest cannot access request bodies (POST/PUT/etc.) and
 * cannot stream responses, so irgo-bridge.js patches window.fetch to route
 * those requests through the IrgoJSInterface (window.Irgo) instead. This client
 * only handles plain GET resource loads triggered by the WebView itself.
 */
open class IrgoWebViewClient : WebViewClient() {

    companion object {
        const val SCHEME = "irgo"
        const val HOST = "app"
    }

    override fun shouldInterceptRequest(
        view: WebView?,
        request: WebResourceRequest?
    ): WebResourceResponse? {
        val url = request?.url ?: return null

        // Only intercept irgo:// scheme
        if (url.scheme != SCHEME) {
            return super.shouldInterceptRequest(view, request)
        }

        // Convert irgo://app/path?query -> /path?query
        var path = url.path ?: "/"
        if (path.isEmpty()) {
            path = "/"
        }
        url.query?.let { query ->
            if (query.isNotEmpty()) {
                path += "?$query"
            }
        }

        // Get HTTP method
        val method = request.method ?: "GET"

        // Get headers
        val headers = request.requestHeaders ?: emptyMap()

        // Handle request (this runs on WebView thread, which is fine for our use case)
        val response = IrgoBridge.handleRequest(
            method = method,
            url = path,
            headers = headers,
            body = null // GET resource loads only; body-bearing requests go through IrgoJSInterface
        )

        // Determine MIME type
        val mimeType = response.headers["Content-Type"] ?: "text/html"

        // Create response
        return WebResourceResponse(
            mimeType.split(";").first().trim(),
            "UTF-8",
            response.status,
            if (response.status < 400) "OK" else "Error",
            response.headers,
            ByteArrayInputStream(response.body)
        )
    }

    override fun shouldOverrideUrlLoading(view: WebView?, request: WebResourceRequest?): Boolean {
        val url = request?.url ?: return false

        // Allow irgo:// scheme
        if (url.scheme == SCHEME) {
            return false // Let WebView handle it (will be intercepted)
        }

        // For external URLs, could open in browser
        // For now, allow them
        return false
    }

    override fun onPageFinished(view: WebView?, url: String?) {
        super.onPageFinished(view, url)
        // Page loaded successfully
    }
}
