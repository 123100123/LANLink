package expo.modules.lanlinkuploader

import android.content.Context
import android.net.Uri
import expo.modules.kotlin.Promise
import expo.modules.kotlin.exception.CodedException
import expo.modules.kotlin.modules.Module
import expo.modules.kotlin.modules.ModuleDefinition
import expo.modules.kotlin.records.Field
import expo.modules.kotlin.records.Record
import okhttp3.Call
import okhttp3.Callback
import okhttp3.MediaType
import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody
import okhttp3.Response
import okio.BufferedSink
import java.io.IOException
import java.io.InputStream
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.TimeUnit

// 1 MiB streaming buffer — large reads keep the okhttp socket saturated and avoid
// the per-small-write overhead that throttled React Native's own uploader.
private const val BUFFER_SIZE = 1 shl 20

class UploadOptions : Record {
  @Field var url: String = ""
  @Field var uri: String = ""
  @Field var filename: String = ""
  @Field var transferId: String = ""
  @Field var authToken: String = ""
  @Field var size: Long? = null
  @Field var mimeType: String? = null
}

class LanlinkUploaderModule : Module() {
  // Our own OkHttp client, completely separate from React Native's NetworkingModule
  // (whose ProgressRequestBody writes upload bodies one byte at a time).
  private val client =
    OkHttpClient.Builder()
      .connectTimeout(30, TimeUnit.SECONDS)
      .writeTimeout(0, TimeUnit.SECONDS) // no write timeout: large files take a while
      .readTimeout(120, TimeUnit.SECONDS)
      .build()

  // transferId -> in-flight Call, so JS can cancel an upload by id.
  private val activeCalls = ConcurrentHashMap<String, Call>()

  private val context: Context
    get() =
      appContext.reactContext
        ?: throw CodedException("ERR_NO_CONTEXT", "Android context unavailable", null)

  override fun definition() = ModuleDefinition {
    Name("LanlinkUploader")

    AsyncFunction("uploadFile") { options: UploadOptions, promise: Promise ->
      try {
        val resolver = context.contentResolver
        val uri = Uri.parse(options.uri)
        val declaredSize = options.size ?: -1L
        val mediaType: MediaType? =
          (options.mimeType ?: "application/octet-stream").toMediaTypeOrNull()

        val body =
          object : RequestBody() {
            override fun contentType(): MediaType? = mediaType

            // A known length avoids chunked transfer-encoding; -1 falls back to it.
            override fun contentLength(): Long = if (declaredSize > 0) declaredSize else -1L

            override fun writeTo(sink: BufferedSink) {
              // openInputStream handles both file:// and content:// URIs and never
              // copies the file into app cache.
              val input: InputStream =
                resolver.openInputStream(uri)
                  ?: throw IOException("Cannot open input stream for ${options.uri}")
              input.use { stream ->
                val buffer = ByteArray(BUFFER_SIZE)
                while (true) {
                  val read = stream.read(buffer)
                  if (read == -1) break
                  sink.write(buffer, 0, read)
                }
              }
            }
          }

        val requestBuilder =
          Request.Builder()
            .url(options.url)
            .post(body)
            .header("Authorization", "Bearer ${options.authToken}")
            .header("X-Filename", options.filename)
            .header("X-Transfer-Id", options.transferId)
            .header("Content-Type", "application/octet-stream")
        if (declaredSize > 0) {
          requestBuilder.header("X-File-Size", declaredSize.toString())
        }

        val call = client.newCall(requestBuilder.build())
        activeCalls[options.transferId] = call

        call.enqueue(
          object : Callback {
            override fun onFailure(call: Call, e: IOException) {
              activeCalls.remove(options.transferId)
              if (call.isCanceled()) {
                promise.reject(CodedException("ERR_UPLOAD_CANCELLED", "Upload cancelled", e))
              } else {
                promise.reject(CodedException("ERR_UPLOAD_FAILED", e.message ?: "Upload failed", e))
              }
            }

            override fun onResponse(call: Call, response: Response) {
              activeCalls.remove(options.transferId)
              response.use { res ->
                val responseBody = res.body?.string() ?: ""
                promise.resolve(mapOf("status" to res.code, "body" to responseBody))
              }
            }
          }
        )
      } catch (e: CodedException) {
        activeCalls.remove(options.transferId)
        promise.reject(e)
      } catch (e: Exception) {
        activeCalls.remove(options.transferId)
        promise.reject(CodedException("ERR_UPLOAD_FAILED", e.message ?: "Upload failed", e))
      }
    }

    AsyncFunction("cancelUpload") { transferId: String ->
      val call = activeCalls.remove(transferId)
      call?.cancel()
      call != null
    }
  }
}
