package expo.modules.lanlinkshare

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.provider.OpenableColumns
import expo.modules.kotlin.Promise
import expo.modules.kotlin.modules.Module
import expo.modules.kotlin.modules.ModuleDefinition

// Receives files shared from the Android system share sheet (ACTION_SEND /
// ACTION_SEND_MULTIPLE) and hands their content:// URIs to JS. The URIs are
// streamed straight to the receiver by lanlink-uploader — never copied here.
class LanlinkShareModule : Module() {
  private val context: Context
    get() = appContext.reactContext ?: throw IllegalStateException("Android context unavailable")

  override fun definition() = ModuleDefinition {
    Name("LanlinkShare")

    Events("onShare")

    // Read the intent the activity was (cold-)launched with. Returns [] when the
    // launch wasn't a share, then marks the intent consumed so a JS remount
    // doesn't re-process the same files.
    AsyncFunction("getInitialShareIntent") { promise: Promise ->
      val activity = appContext.currentActivity
      val intent = activity?.intent
      // Relaunching from recents re-delivers the ORIGINAL launch intent (the
      // in-memory action rewrite below doesn't survive process death), which
      // would replay an old share on every reopen. Android marks those intents.
      val fromHistory =
        intent != null && (intent.flags and Intent.FLAG_ACTIVITY_LAUNCHED_FROM_HISTORY) != 0
      val files =
        if (intent != null && !fromHistory) extractSharedFiles(intent) else emptyList()
      if (intent != null && files.isNotEmpty()) {
        // Neutralise the action so the same share isn't picked up twice.
        intent.action = Intent.ACTION_MAIN
      }
      promise.resolve(files)
    }

    // Warm-start: the app is already running when the user shares. MainActivity is
    // singleTask, so the new share arrives via onNewIntent rather than a fresh launch.
    OnNewIntent { intent ->
      val files = extractSharedFiles(intent)
      if (files.isNotEmpty()) {
        sendEvent("onShare", mapOf("files" to files))
      }
    }
  }

  private fun extractSharedFiles(intent: Intent): List<Map<String, Any?>> {
    val uris: List<Uri> =
      when (intent.action) {
        Intent.ACTION_SEND -> listOfNotNull(streamUri(intent))
        Intent.ACTION_SEND_MULTIPLE -> multipleStreamUris(intent)
        else -> emptyList()
      }
    return uris.map { describe(it) }
  }

  @Suppress("DEPRECATION")
  private fun streamUri(intent: Intent): Uri? {
    val extra = intent.getParcelableExtra<Uri>(Intent.EXTRA_STREAM)
    if (extra != null) return extra
    // Some apps attach the payload via clipData instead of EXTRA_STREAM.
    return intent.clipData?.takeIf { it.itemCount > 0 }?.getItemAt(0)?.uri
  }

  @Suppress("DEPRECATION")
  private fun multipleStreamUris(intent: Intent): List<Uri> {
    val extras = intent.getParcelableArrayListExtra<Uri>(Intent.EXTRA_STREAM)
    if (!extras.isNullOrEmpty()) return extras.filterNotNull()
    val clip = intent.clipData ?: return emptyList()
    return (0 until clip.itemCount).mapNotNull { clip.getItemAt(it).uri }
  }

  // Resolve the display name, byte size and MIME type for a content:// (or file://)
  // URI. Missing fields fall back to safe defaults so JS always gets a usable item.
  private fun describe(uri: Uri): Map<String, Any?> {
    var name: String? = null
    var size: Long = 0
    if ("content" == uri.scheme) {
      try {
        context.contentResolver
          .query(uri, arrayOf(OpenableColumns.DISPLAY_NAME, OpenableColumns.SIZE), null, null, null)
          ?.use { cursor ->
            if (cursor.moveToFirst()) {
              val nameIdx = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
              if (nameIdx >= 0 && !cursor.isNull(nameIdx)) name = cursor.getString(nameIdx)
              val sizeIdx = cursor.getColumnIndex(OpenableColumns.SIZE)
              if (sizeIdx >= 0 && !cursor.isNull(sizeIdx)) size = cursor.getLong(sizeIdx)
            }
          }
      } catch (_: Exception) {
        // Best-effort: an unreadable provider still yields a uri JS can try.
      }
    } else {
      name = uri.lastPathSegment
    }
    val mimeType = context.contentResolver.getType(uri)
    return mapOf(
      "uri" to uri.toString(),
      "name" to (name ?: uri.lastPathSegment ?: "shared file"),
      "size" to size,
      "mimeType" to mimeType
    )
  }
}
