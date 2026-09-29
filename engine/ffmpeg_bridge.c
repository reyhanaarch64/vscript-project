#include "ffmpeg_bridge.h"

#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <dirent.h>

#define AV_CODEC_ID_H264 27
#define AV_CODEC_ID_MPEG4 12
#define AV_PIX_FMT_YUV420P 0
#define AVMEDIA_TYPE_VIDEO 0
#define AVFMT_NOFILE 0x0001
#define AV_CODEC_FLAG_GLOBAL_HEADER (1 << 22)

#define AVERROR_EOF (-541478725)
#define AVERROR_EAGAIN (-11)

typedef struct AVRational {
    int num;
    int den;
} AVRational;

typedef struct AVFramePrefix {
    uint8_t *data[8];
    int linesize[8];
    uint8_t **extended_data;
    int width;
    int height;
    int nb_samples;
    int format;
    int pict_type;
    AVRational sample_aspect_ratio;
    int64_t pts;
    int64_t pkt_dts;
    AVRational time_base;
} AVFramePrefix;

typedef struct AVPacketPrefix {
    void *buf;
    int64_t pts;
    int64_t dts;
    uint8_t *data;
    int size;
    int stream_index;
    int flags;
} AVPacketPrefix;

typedef struct AVCodecParametersPrefix {
    int codec_type;
    int codec_id;
    uint32_t codec_tag;
    uint8_t *extradata;
    int extradata_size;
} AVCodecParametersPrefix;

typedef struct AVStreamPrefix {
    void *av_class;
    int index;
    int id;
    void *codecpar;
    void *priv_data;
    AVRational time_base;
} AVStreamPrefix;

typedef struct AVFormatContextPrefix {
    const void *av_class;
    void *iformat;
    void *oformat;
    void *priv_data;
    void *pb;
} AVFormatContextPrefix;

typedef struct AVCodecContextPrefix {
    const void *av_class;
} AVCodecContextPrefix;

typedef struct AVOutputFormatPrefix {
    const char *name;
    const char *long_name;
    const char *mime_type;
    const char *extensions;
    int audio_codec;
    int video_codec;
    int subtitle_codec;
    int flags;
} AVOutputFormatPrefix;

typedef struct VSFrame {
    void *ptr;
} VSFrame;

typedef struct VSEncoder {
    void *lib_codec;
    void *lib_format;
    void *lib_util;

    const void *codec;
    void *codec_ctx;
    void *fmt_ctx;
    void *stream;
    void *frame;
    void *packet;

    int width;
    int height;
    int fps;
    int64_t bitrate;
    int ready;
    int64_t next_pts;
    AVRational stream_time_base;

    int (*av_opt_set_int)(void *, const char *, int64_t, int);
    int (*av_opt_set_q)(void *, const char *, AVRational, int);
    int (*av_opt_set)(void *, const char *, const char *, int);
    int (*av_opt_set_image_size)(void *, const char *, int, int, int);
    int (*av_opt_set_pixel_fmt)(void *, const char *, int, int);

    const void *(*avcodec_find_encoder_by_name)(const char *name);
    void *(*avcodec_alloc_context3)(const void *codec);
    int (*avcodec_open2)(void *ctx, const void *codec, void *options);
    int (*avcodec_send_frame)(void *ctx, void *frame);
    int (*avcodec_receive_packet)(void *ctx, void *packet);
    void (*avcodec_free_context)(void **ctx);
    int (*avcodec_parameters_from_context)(void *par, const void *ctx);

    int (*avformat_alloc_output_context2)(void **ctx, void *oformat, const char *format_name, const char *filename);
    void *(*avformat_new_stream)(void *ctx, const void *c);
    int (*avformat_write_header)(void *ctx, void *options);
    int (*av_interleaved_write_frame)(void *ctx, void *packet);
    int (*av_write_trailer)(void *ctx);
    void (*avformat_free_context)(void *ctx);

    int (*avio_open2)(void **s, const char *url, int flags, const void *int_cb, void *options);
    int (*avio_closep)(void **s);

    void *(*av_frame_alloc)(void);
    void (*av_frame_free)(void **frame);
    int (*av_frame_get_buffer)(void *frame, int align);
    int (*av_frame_make_writable)(void *frame);
    void (*av_frame_unref)(void *frame);

    void *(*av_packet_alloc)(void);
    void (*av_packet_free)(void **packet);
    void (*av_packet_unref)(void *packet);
    void (*av_packet_rescale_ts)(void *packet, AVRational tb_src, AVRational tb_dst);

    int (*av_strerror)(int errnum, char *errbuf, size_t errbuf_size);

    char error[512];
} VSEncoder;

static void set_error(VSEncoder *e, const char *msg) {
    if (!e) return;
    snprintf(e->error, sizeof(e->error), "%s", msg ? msg : "unknown native error");
}

static void set_errorf(VSEncoder *e, const char *prefix, int code) {
    char detail[256];
    detail[0] = '\0';
    if (e && e->av_strerror) {
        e->av_strerror(code, detail, sizeof(detail));
    }
    if (!detail[0]) {
        snprintf(detail, sizeof(detail), "error %d", code);
    }
    if (e) {
        snprintf(e->error, sizeof(e->error), "%s: %s", prefix, detail);
    }
}

static void *load_from_path(const char *path) {
    return dlopen(path, RTLD_NOW | RTLD_LOCAL);
}

static void *try_directory(const char *dir, const char *soname) {
    char candidate[1024];
    DIR *dp;
    struct dirent *ent;
    const char *base;

    snprintf(candidate, sizeof(candidate), "%s/%s", dir, soname);
    {
        void *h = load_from_path(candidate);
        if (h) return h;
    }

    dp = opendir(dir);
    if (!dp) return NULL;
    base = soname;
    while ((ent = readdir(dp)) != NULL) {
        if (strncmp(ent->d_name, base, strlen(base)) != 0) continue;
        if (ent->d_name[strlen(base)] != '.') continue;
        snprintf(candidate, sizeof(candidate), "%s/%s", dir, ent->d_name);
        {
            void *h = load_from_path(candidate);
            if (h) {
                closedir(dp);
                return h;
            }
        }
    }
    closedir(dp);
    return NULL;
}

static void *load_any(const char *soname) {
    const char *override = getenv("VSCRIPT_LIB_PATH");
    const char *prefix = getenv("PREFIX");
    const char *termux_prefix = "/data/data/com.termux/files/usr";
    const char *fixed_dirs[] = {
        "/lib",
        "/usr/lib",
        "/lib64",
        "/usr/lib64",
        "/usr/local/lib",
        "/lib/x86_64-linux-gnu",
        "/lib/aarch64-linux-gnu",
        "/lib/arm-linux-gnueabihf",
        "/usr/lib/x86_64-linux-gnu",
        "/usr/lib/aarch64-linux-gnu",
        "/usr/lib/arm-linux-gnueabihf",
        NULL
    };
    char candidate[1024];

    if (override && override[0]) {
        const char *p = override;
        while (*p) {
            const char *end = strchr(p, ':');
            size_t len = end ? (size_t)(end - p) : strlen(p);
            if (len > 0 && len < sizeof(candidate) - 256) {
                snprintf(candidate, sizeof(candidate), "%.*s", (int)len, p);
                {
                    void *h = try_directory(candidate, soname);
                    if (h) return h;
                }
            }
            if (!end) break;
            p = end + 1;
        }
    }

    if (prefix && prefix[0]) {
        snprintf(candidate, sizeof(candidate), "%s/lib", prefix);
        {
            void *h = try_directory(candidate, soname);
            if (h) return h;
        }
    }

    snprintf(candidate, sizeof(candidate), "%s/lib", termux_prefix);
    {
        void *h = try_directory(candidate, soname);
        if (h) return h;
    }

    {
        size_t i;
        for (i = 0; fixed_dirs[i]; i++) {
            void *h = try_directory(fixed_dirs[i], soname);
            if (h) return h;
        }
    }

    return dlopen(soname, RTLD_NOW | RTLD_LOCAL);
}

static void close_lib(void **h) {
    if (h && *h) {
        dlclose(*h);
        *h = NULL;
    }
}

static int resolve_symbol(void *lib, void **dst, const char *name, VSEncoder *e) {
    if (!lib) return -1;
    *dst = dlsym(lib, name);
    if (!*dst) {
        char buf[256];
        snprintf(buf, sizeof(buf), "simbol native tidak ditemukan: %s", name);
        set_error(e, buf);
        return -1;
    }
    return 0;
}

static int resolve_all(VSEncoder *e) {
#define RESOLVE(lib, field, name) do { if (resolve_symbol((lib), (void **)&(e->field), (name), e) != 0) return -1; } while (0)
    RESOLVE(e->lib_codec, avcodec_find_encoder_by_name, "avcodec_find_encoder_by_name");
    RESOLVE(e->lib_codec, avcodec_alloc_context3, "avcodec_alloc_context3");
    RESOLVE(e->lib_codec, avcodec_open2, "avcodec_open2");
    RESOLVE(e->lib_codec, avcodec_send_frame, "avcodec_send_frame");
    RESOLVE(e->lib_codec, avcodec_receive_packet, "avcodec_receive_packet");
    RESOLVE(e->lib_codec, avcodec_free_context, "avcodec_free_context");
    RESOLVE(e->lib_codec, avcodec_parameters_from_context, "avcodec_parameters_from_context");

    RESOLVE(e->lib_format, avformat_alloc_output_context2, "avformat_alloc_output_context2");
    RESOLVE(e->lib_format, avformat_new_stream, "avformat_new_stream");
    RESOLVE(e->lib_format, avformat_write_header, "avformat_write_header");
    RESOLVE(e->lib_format, av_interleaved_write_frame, "av_interleaved_write_frame");
    RESOLVE(e->lib_format, av_write_trailer, "av_write_trailer");
    RESOLVE(e->lib_format, avformat_free_context, "avformat_free_context");
    RESOLVE(e->lib_format, avio_open2, "avio_open2");
    RESOLVE(e->lib_format, avio_closep, "avio_closep");

    RESOLVE(e->lib_util, av_frame_alloc, "av_frame_alloc");
    RESOLVE(e->lib_util, av_frame_free, "av_frame_free");
    RESOLVE(e->lib_util, av_frame_get_buffer, "av_frame_get_buffer");
    RESOLVE(e->lib_util, av_frame_make_writable, "av_frame_make_writable");
    RESOLVE(e->lib_util, av_frame_unref, "av_frame_unref");
    RESOLVE(e->lib_codec, av_packet_alloc, "av_packet_alloc");
    RESOLVE(e->lib_codec, av_packet_free, "av_packet_free");
    RESOLVE(e->lib_codec, av_packet_unref, "av_packet_unref");
    RESOLVE(e->lib_codec, av_packet_rescale_ts, "av_packet_rescale_ts");
    RESOLVE(e->lib_util, av_strerror, "av_strerror");
    RESOLVE(e->lib_util, av_opt_set_int, "av_opt_set_int");
    RESOLVE(e->lib_util, av_opt_set_q, "av_opt_set_q");
    RESOLVE(e->lib_util, av_opt_set, "av_opt_set");
    RESOLVE(e->lib_util, av_opt_set_image_size, "av_opt_set_image_size");
    RESOLVE(e->lib_util, av_opt_set_pixel_fmt, "av_opt_set_pixel_fmt");
#undef RESOLVE
    return 0;
}

static int configure_codec(VSEncoder *e, const char *codec_name, const char *format_hint, const char *output) {
    const void *codec = NULL;
    void *fmt_ctx = NULL;
    void *codec_ctx = NULL;
    void *stream = NULL;
    AVOutputFormatPrefix *ofmt;
    AVStreamPrefix *sp;
    AVRational fps_q;
    AVRational tb_q;
    int ret;

    codec = e->avcodec_find_encoder_by_name(codec_name);
    if (!codec && strcmp(codec_name, "h264") != 0) {
        codec = e->avcodec_find_encoder_by_name("h264");
    }
    if (!codec) {
        codec = e->avcodec_find_encoder_by_name("mpeg4");
    }
    if (!codec) {
        set_error(e, "encoder video tidak tersedia di libavcodec");
        return -1;
    }

    ret = e->avformat_alloc_output_context2(&fmt_ctx, NULL, format_hint && format_hint[0] ? format_hint : NULL, output);
    if (ret < 0 || !fmt_ctx) {
        set_errorf(e, "gagal membuat output format", ret);
        return -1;
    }

    codec_ctx = e->avcodec_alloc_context3(codec);
    if (!codec_ctx) {
        e->avformat_free_context(fmt_ctx);
        set_error(e, "gagal membuat codec context");
        return -1;
    }

    fps_q.num = e->fps;
    fps_q.den = 1;
    tb_q.num = 1;
    tb_q.den = e->fps;

    ret = e->av_opt_set_image_size(codec_ctx, "video_size", e->width, e->height, 0);
    if (ret < 0) { set_errorf(e, "gagal mengatur ukuran video", ret); e->avcodec_free_context(&codec_ctx); e->avformat_free_context(fmt_ctx); return -1; }
    ret = e->av_opt_set_pixel_fmt(codec_ctx, "pixel_format", AV_PIX_FMT_YUV420P, 0);
    if (ret < 0) { set_errorf(e, "gagal mengatur pixel format", ret); e->avcodec_free_context(&codec_ctx); e->avformat_free_context(fmt_ctx); return -1; }
    ret = e->av_opt_set_q(codec_ctx, "time_base", tb_q, 0);
    if (ret < 0) { set_errorf(e, "gagal mengatur time base", ret); e->avcodec_free_context(&codec_ctx); e->avformat_free_context(fmt_ctx); return -1; }
    ret = e->av_opt_set_int(codec_ctx, "b", e->bitrate, 0);
    if (ret < 0) { set_errorf(e, "gagal mengatur bitrate", ret); e->avcodec_free_context(&codec_ctx); e->avformat_free_context(fmt_ctx); return -1; }
    e->av_opt_set_int(codec_ctx, "gop_size", e->fps * 2, 0);
    e->av_opt_set_int(codec_ctx, "max_b_frames", 2, 0);
    e->av_opt_set(codec_ctx, "preset", "medium", 0);

    ofmt = (AVOutputFormatPrefix *)((AVFormatContextPrefix *)fmt_ctx)->oformat;
    if (ofmt && (ofmt->flags & AVFMT_NOFILE) == 0) {
        ret = e->avio_open2(&((AVFormatContextPrefix *)fmt_ctx)->pb, output, 2, NULL, NULL);
        if (ret < 0) {
            set_errorf(e, "gagal membuka output", ret);
            e->avcodec_free_context(&codec_ctx);
            e->avformat_free_context(fmt_ctx);
            return -1;
        }
    }

    stream = e->avformat_new_stream(fmt_ctx, NULL);
    if (!stream) {
        set_error(e, "gagal membuat stream video");
        if (((AVFormatContextPrefix *)fmt_ctx)->pb) {
            e->avio_closep(&((AVFormatContextPrefix *)fmt_ctx)->pb);
        }
        e->avcodec_free_context(&codec_ctx);
        e->avformat_free_context(fmt_ctx);
        return -1;
    }

    sp = (AVStreamPrefix *)stream;
    sp->time_base = tb_q;

    ret = e->avcodec_open2(codec_ctx, codec, NULL);
    if (ret < 0) {
        set_errorf(e, "gagal membuka encoder", ret);
        if (((AVFormatContextPrefix *)fmt_ctx)->pb) {
            e->avio_closep(&((AVFormatContextPrefix *)fmt_ctx)->pb);
        }
        e->avcodec_free_context(&codec_ctx);
        e->avformat_free_context(fmt_ctx);
        return -1;
    }

    ret = e->avcodec_parameters_from_context(sp->codecpar, codec_ctx);
    if (ret < 0) {
        set_errorf(e, "gagal mengisi codec parameters", ret);
        if (((AVFormatContextPrefix *)fmt_ctx)->pb) {
            e->avio_closep(&((AVFormatContextPrefix *)fmt_ctx)->pb);
        }
        e->avcodec_free_context(&codec_ctx);
        e->avformat_free_context(fmt_ctx);
        return -1;
    }

    ((AVFormatContextPrefix *)fmt_ctx)->oformat = ofmt;

    ret = e->avformat_write_header(fmt_ctx, NULL);
    if (ret < 0) {
        set_errorf(e, "gagal menulis header output", ret);
        if (((AVFormatContextPrefix *)fmt_ctx)->pb) {
            e->avio_closep(&((AVFormatContextPrefix *)fmt_ctx)->pb);
        }
        e->avcodec_free_context(&codec_ctx);
        e->avformat_free_context(fmt_ctx);
        return -1;
    }

    e->codec = codec;
    e->codec_ctx = codec_ctx;
    e->fmt_ctx = fmt_ctx;
    e->stream = stream;
    e->stream_time_base = tb_q;
    return 0;
}

VSEncoder *vscript_encoder_create(void) {
    VSEncoder *e = (VSEncoder *)calloc(1, sizeof(VSEncoder));
    if (!e) return NULL;
    snprintf(e->error, sizeof(e->error), "ok");

    e->lib_codec = load_any("libavcodec.so");
    e->lib_format = load_any("libavformat.so");
    e->lib_util = load_any("libavutil.so");

    if (!e->lib_codec || !e->lib_format || !e->lib_util) {
        set_error(e, "library libavcodec/libavformat/libavutil tidak ditemukan; cek instalasi ffmpeg atau VSCRIPT_LIB_PATH");
        close_lib(&e->lib_codec);
        close_lib(&e->lib_format);
        close_lib(&e->lib_util);
        return e;
    }

    if (resolve_all(e) != 0) {
        close_lib(&e->lib_codec);
        close_lib(&e->lib_format);
        close_lib(&e->lib_util);
        return e;
    }

    e->ready = 1;
    return e;
}

int vscript_encoder_open(VSEncoder *e, const char *output, int width, int height, int fps, const char *codec, const char *format, int64_t bitrate) {
    int ret;
    if (!e) return -1;
    if (e->codec_ctx || e->fmt_ctx) {
        set_error(e, "encoder sudah dibuka");
        return -1;
    }
    if (!output || !output[0] || width <= 0 || height <= 0 || fps <= 0) {
        set_error(e, "parameter encoder tidak valid");
        return -1;
    }

    if (!e->ready) {
        if (!e->error[0]) set_error(e, "native multimedia backend belum siap");
        return -1;
    }

    e->width = width;
    e->height = height;
    e->fps = fps;
    e->bitrate = bitrate > 0 ? bitrate : 4000000;
    e->next_pts = 0;

    ret = configure_codec(e, codec && codec[0] ? codec : "h264", format, output);
    if (ret != 0) return ret;

    e->frame = e->av_frame_alloc();
    e->packet = e->av_packet_alloc();
    if (!e->frame || !e->packet) {
        set_error(e, "gagal alokasi frame atau packet");
        return -1;
    }

    {
        AVFramePrefix *frame = (AVFramePrefix *)e->frame;
        frame->format = AV_PIX_FMT_YUV420P;
        frame->width = width;
        frame->height = height;
    }

    ret = e->av_frame_get_buffer(e->frame, 32);
    if (ret < 0) {
        set_errorf(e, "gagal mengalokasikan buffer frame", ret);
        return -1;
    }

    return 0;
}

static inline uint8_t clamp_u8(int v) {
    if (v < 0) return 0;
    if (v > 255) return 255;
    return (uint8_t)v;
}

static void rgba_to_yuv420(const uint8_t *rgba, int rgba_stride, AVFramePrefix *frame, int width, int height) {
    int y, x;
    for (y = 0; y < height; y++) {
        const uint8_t *src = rgba + (size_t)y * (size_t)rgba_stride;
        uint8_t *dst_y = frame->data[0] + (size_t)y * (size_t)frame->linesize[0];
        for (x = 0; x < width; x++) {
            int r = src[x * 4 + 0];
            int g = src[x * 4 + 1];
            int b = src[x * 4 + 2];
            int yy = (77 * r + 150 * g + 29 * b + 128) >> 8;
            dst_y[x] = clamp_u8(yy);
        }
    }

    for (y = 0; y < height; y += 2) {
        const uint8_t *src0 = rgba + (size_t)y * (size_t)rgba_stride;
        const uint8_t *src1 = rgba + (size_t)(y + 1 < height ? y + 1 : y) * (size_t)rgba_stride;
        uint8_t *dst_u = frame->data[1] + (size_t)(y / 2) * (size_t)frame->linesize[1];
        uint8_t *dst_v = frame->data[2] + (size_t)(y / 2) * (size_t)frame->linesize[2];
        for (x = 0; x < width; x += 2) {
            int x1 = x + 1 < width ? x + 1 : x;
            int r = src0[x * 4 + 0] + src0[x1 * 4 + 0] + src1[x * 4 + 0] + src1[x1 * 4 + 0];
            int g = src0[x * 4 + 1] + src0[x1 * 4 + 1] + src1[x * 4 + 1] + src1[x1 * 4 + 1];
            int b = src0[x * 4 + 2] + src0[x1 * 4 + 2] + src1[x * 4 + 2] + src1[x1 * 4 + 2];
            r >>= 2;
            g >>= 2;
            b >>= 2;
            dst_u[x / 2] = clamp_u8(((-43 * r - 85 * g + 128 * b + 32768) >> 8));
            dst_v[x / 2] = clamp_u8(((128 * r - 107 * g - 21 * b + 32768) >> 8));
        }
    }
}

int vscript_encoder_write_frame(VSEncoder *e, const uint8_t *rgba, int stride, int64_t pts) {
    AVFramePrefix *frame;
    AVPacketPrefix *packet;
    int ret;

    if (!e || !e->codec_ctx || !e->frame || !e->packet || !rgba) return -1;

    ret = e->av_frame_make_writable(e->frame);
    if (ret < 0) {
        set_errorf(e, "frame tidak writable", ret);
        return -1;
    }

    frame = (AVFramePrefix *)e->frame;
    packet = (AVPacketPrefix *)e->packet;
    rgba_to_yuv420(rgba, stride, frame, e->width, e->height);
    frame->pts = pts;

    ret = e->avcodec_send_frame(e->codec_ctx, e->frame);
    if (ret < 0) {
        set_errorf(e, "gagal mengirim frame ke encoder", ret);
        return -1;
    }

    for (;;) {
        ret = e->avcodec_receive_packet(e->codec_ctx, e->packet);
        if (ret == AVERROR_EAGAIN || ret == AVERROR_EOF) break;
        if (ret < 0) {
            set_errorf(e, "gagal menerima packet encoder", ret);
            return -1;
        }

        packet->stream_index = ((AVStreamPrefix *)e->stream)->index;
        e->av_packet_rescale_ts(e->packet, e->stream_time_base, ((AVStreamPrefix *)e->stream)->time_base);

        ret = e->av_interleaved_write_frame(e->fmt_ctx, e->packet);
        e->av_packet_unref(e->packet);
        if (ret < 0) {
            set_errorf(e, "gagal menulis packet output", ret);
            return -1;
        }
    }

    return 0;
}

int vscript_encoder_close(VSEncoder *e) {
    int ret;
    int status = 0;

    if (!e) return -1;

    if (e->codec_ctx) {
        ret = e->avcodec_send_frame(e->codec_ctx, NULL);
        if (ret < 0 && ret != AVERROR_EOF) {
            set_errorf(e, "gagal flush encoder", ret);
            status = -1;
        }

        while (ret >= 0) {
            ret = e->avcodec_receive_packet(e->codec_ctx, e->packet);
            if (ret == AVERROR_EAGAIN || ret == AVERROR_EOF) break;
            if (ret < 0) {
                set_errorf(e, "gagal flush packet encoder", ret);
                status = -1;
                break;
            }
            ((AVPacketPrefix *)e->packet)->stream_index = ((AVStreamPrefix *)e->stream)->index;
            e->av_packet_rescale_ts(e->packet, e->stream_time_base, ((AVStreamPrefix *)e->stream)->time_base);
            ret = e->av_interleaved_write_frame(e->fmt_ctx, e->packet);
            e->av_packet_unref(e->packet);
            if (ret < 0) {
                set_errorf(e, "gagal menulis packet flush", ret);
                status = -1;
                break;
            }
        }
    }

    if (e->fmt_ctx) {
        ret = e->av_write_trailer(e->fmt_ctx);
        if (ret < 0 && status == 0) {
            set_errorf(e, "gagal menulis trailer output", ret);
            status = -1;
        }

        if (((AVFormatContextPrefix *)e->fmt_ctx)->pb) {
            e->avio_closep(&((AVFormatContextPrefix *)e->fmt_ctx)->pb);
        }
        e->avformat_free_context(e->fmt_ctx);
        e->fmt_ctx = NULL;
    }

    if (e->packet) {
        e->av_packet_free(&e->packet);
    }
    if (e->frame) {
        e->av_frame_free(&e->frame);
    }
    if (e->codec_ctx) {
        e->avcodec_free_context(&e->codec_ctx);
    }

    return status;
}

const char *vscript_encoder_error(VSEncoder *e) {
    if (!e) return "encoder handle invalid";
    return e->error;
}

const char *vscript_ffmpeg_path_report(void) {
    static char report[2048];
    const char *prefix = getenv("PREFIX");
    const char *override = getenv("VSCRIPT_LIB_PATH");
    if (override && override[0]) {
        snprintf(report, sizeof(report), "override=%s; prefix=%s; termux=/data/data/com.termux/files/usr/lib; linux=/lib:/usr/lib:/usr/lib/x86_64-linux-gnu:/usr/lib/aarch64-linux-gnu", override, prefix ? prefix : "(unset)");
    } else {
        snprintf(report, sizeof(report), "prefix=%s; termux=/data/data/com.termux/files/usr/lib; linux=/lib:/usr/lib:/usr/lib/x86_64-linux-gnu:/usr/lib/aarch64-linux-gnu", prefix ? prefix : "(unset)");
    }
    return report;
}

void vscript_encoder_destroy(VSEncoder *e) {
    if (!e) return;
    vscript_encoder_close(e);
    close_lib(&e->lib_codec);
    close_lib(&e->lib_format);
    close_lib(&e->lib_util);
    free(e);
}

/* -------------------------------------------------------------------------
 * Patch 2: FreeType font rasterizer
 * ------------------------------------------------------------------------- */

typedef struct VSFT_Generic {
    void *data;
    void (*finalizer)(void *object);
} VSFT_Generic;

typedef struct VSFT_BBox {
    long xMin, yMin, xMax, yMax;
} VSFT_BBox;

typedef struct VSFT_Bitmap {
    unsigned int rows;
    unsigned int width;
    int pitch;
    unsigned char *buffer;
    unsigned short num_grays;
    unsigned char pixel_mode;
    unsigned char palette_mode;
    void *palette;
} VSFT_Bitmap;

typedef struct VSFT_GlyphMetrics {
    long width, height;
    long horiBearingX, horiBearingY, horiAdvance;
    long vertBearingX, vertBearingY, vertAdvance;
} VSFT_GlyphMetrics;

typedef struct VSFT_Vector {
    long x, y;
} VSFT_Vector;

typedef struct VSFT_GlyphSlotRecPrefix {
    void *library;
    void *face;
    void *next;
    unsigned int glyph_index;
    VSFT_Generic generic;
    VSFT_GlyphMetrics metrics;
    long linearHoriAdvance;
    long linearVertAdvance;
    VSFT_Vector advance;
    int format;
    int _format_padding;
    VSFT_Bitmap bitmap;
    int bitmap_left;
    int bitmap_top;
} VSFT_GlyphSlotRecPrefix;

typedef struct VSFT_FaceRecPrefix {
    long num_faces;
    long face_index;
    long face_flags;
    long style_flags;
    long num_glyphs;
    char *family_name;
    char *style_name;
    int num_fixed_sizes;
    void *available_sizes;
    int num_charmaps;
    void *charmaps;
    VSFT_Generic generic;
    VSFT_BBox bbox;
    unsigned short units_per_EM;
    short ascender;
    short descender;
    short height;
    short max_advance_width;
    short max_advance_height;
    short underline_position;
    short underline_thickness;
    VSFT_GlyphSlotRecPrefix *glyph;
} VSFT_FaceRecPrefix;

typedef struct VSFont {
    void *lib;
    void *dl_handle;
    void *face;
    int size;
    int ready;
    char error[256];
    int (*FT_Done_Face)(void *face);
    int (*FT_Done_FreeType)(void *library);
} VSFont;

static int utf8_next(const unsigned char **pp, unsigned long *out) {
    const unsigned char *p = *pp;
    unsigned long cp;
    if (!p || !*p) return 0;
    if (p[0] < 0x80) {
        cp = p[0];
        *pp = p + 1;
    } else if ((p[0] & 0xE0) == 0xC0 && p[1]) {
        cp = ((unsigned long)(p[0] & 0x1F) << 6) | (unsigned long)(p[1] & 0x3F);
        *pp = p + 2;
    } else if ((p[0] & 0xF0) == 0xE0 && p[1] && p[2]) {
        cp = ((unsigned long)(p[0] & 0x0F) << 12) | ((unsigned long)(p[1] & 0x3F) << 6) | (unsigned long)(p[2] & 0x3F);
        *pp = p + 3;
    } else if ((p[0] & 0xF8) == 0xF0 && p[1] && p[2] && p[3]) {
        cp = ((unsigned long)(p[0] & 0x07) << 18) | ((unsigned long)(p[1] & 0x3F) << 12) | ((unsigned long)(p[2] & 0x3F) << 6) | (unsigned long)(p[3] & 0x3F);
        *pp = p + 4;
    } else {
        cp = 0xFFFD;
        *pp = p + 1;
    }
    *out = cp;
    return 1;
}

static void font_set_error(VSFont *f, const char *msg) {
    if (!f) return;
    snprintf(f->error, sizeof(f->error), "%s", msg ? msg : "unknown font error");
}

VSFont *vscript_font_create(const char *path, int size) {
    typedef int (*FT_Init_FreeType_Fn)(void **);
    typedef int (*FT_New_Face_Fn)(void *, const char *, long, void **);
    typedef int (*FT_Set_Pixel_Sizes_Fn)(void *, unsigned int, unsigned int);
    typedef int (*FT_Load_Char_Fn)(void *, unsigned long, int);

    VSFont *f = (VSFont *)calloc(1, sizeof(VSFont));
    void *lib = NULL;
    void *face = NULL;
    FT_Init_FreeType_Fn FT_Init_FreeType;
    FT_New_Face_Fn FT_New_Face;
    FT_Set_Pixel_Sizes_Fn FT_Set_Pixel_Sizes;
    if (!f) return NULL;
    snprintf(f->error, sizeof(f->error), "ok");
    if (!path || !path[0]) {
        font_set_error(f, "path font kosong");
        return f;
    }
    lib = load_any("libfreetype.so");
    f->dl_handle = lib;
    if (!lib) {
        font_set_error(f, "libfreetype tidak ditemukan; instal FreeType atau set VSCRIPT_LIB_PATH");
        return f;
    }
    FT_Init_FreeType = (FT_Init_FreeType_Fn)dlsym(lib, "FT_Init_FreeType");
    FT_New_Face = (FT_New_Face_Fn)dlsym(lib, "FT_New_Face");
    FT_Set_Pixel_Sizes = (FT_Set_Pixel_Sizes_Fn)dlsym(lib, "FT_Set_Pixel_Sizes");
    f->FT_Done_Face = (int (*)(void *))dlsym(lib, "FT_Done_Face");
    f->FT_Done_FreeType = (int (*)(void *))dlsym(lib, "FT_Done_FreeType");
    if (!FT_Init_FreeType || !FT_New_Face || !FT_Set_Pixel_Sizes || !f->FT_Done_Face || !f->FT_Done_FreeType) {
        font_set_error(f, "simbol FreeType tidak lengkap");
        dlclose(lib);
        f->dl_handle = NULL;
        return f;
    }
    if (FT_Init_FreeType(&f->lib) != 0 || !f->lib) {
        font_set_error(f, "FT_Init_FreeType gagal");
        dlclose(lib);
        f->dl_handle = NULL;
        return f;
    }
    face = NULL;
    if (FT_New_Face(f->lib, path, 0, &face) != 0 || !face) {
        font_set_error(f, "font tidak dapat dibuka; pastikan TTF/OTF valid");
        f->FT_Done_FreeType(f->lib);
        f->lib = NULL;
        dlclose(lib);
        f->dl_handle = NULL;
        return f;
    }
    if (FT_Set_Pixel_Sizes(face, 0, (unsigned int)(size > 0 ? size : 48)) != 0) {
        font_set_error(f, "ukuran font tidak dapat diatur");
        f->FT_Done_Face(face);
        f->FT_Done_FreeType(f->lib);
        f->lib = NULL;
        dlclose(lib);
        f->dl_handle = NULL;
        return f;
    }
    f->face = face;
    f->size = size > 0 ? size : 48;
    f->ready = 1;
    /* keep FreeType library handle open; face owns the library state */
    return f;
}

static int font_load_char(VSFont *f, unsigned long cp) {
    typedef int (*FT_Load_Char_Fn)(void *, unsigned long, int);
    static const int FT_LOAD_RENDER_FLAG = (1 << 2);
    FT_Load_Char_Fn fn;
    if (!f || !f->lib) return -1;
    fn = (FT_Load_Char_Fn)dlsym(f->dl_handle, "FT_Load_Char");
    if (!fn) {
        /* FT_Load_Char is exported from the same shared object as FT_Init_FreeType. */
        return -1;
    }
    return fn(f->face, cp, FT_LOAD_RENDER_FLAG);
}

int vscript_font_draw_rgba(VSFont *f, uint8_t *rgba, int width, int height, int stride, int cx, int cy, uint8_t r, uint8_t g, uint8_t b, uint8_t a, const char *text) {
    const unsigned char *p;
    long pen_x = 0;
    int max_top = -100000;
    int min_bottom = 100000;
    int count = 0;
    VSFT_FaceRecPrefix *face;

    if (!f || !f->ready || !f->face) return -1;
    if (!rgba || width <= 0 || height <= 0 || stride < width * 4 || !text) {
        font_set_error(f, "buffer text tidak valid");
        return -1;
    }
    face = (VSFT_FaceRecPrefix *)f->face;

    p = (const unsigned char *)text;
    while (*p) {
        unsigned long cp;
        if (!utf8_next(&p, &cp)) break;
        if (font_load_char(f, cp) != 0) continue;
        {
            VSFT_GlyphSlotRecPrefix *slot = face->glyph;
            int top = slot->bitmap_top;
            int bottom = slot->bitmap_top - (int)slot->bitmap.rows;
            if (top > max_top) max_top = top;
            if (bottom < min_bottom) min_bottom = bottom;
            pen_x += slot->advance.x;
            count++;
        }
    }
    if (count == 0) return 0;

    {
        long pen = -(pen_x / 2);
        int baseline = cy + (max_top + min_bottom) / 2;
        p = (const unsigned char *)text;
        while (*p) {
            unsigned long cp;
            if (!utf8_next(&p, &cp)) break;
            if (font_load_char(f, cp) != 0) continue;
            {
                VSFT_GlyphSlotRecPrefix *slot = face->glyph;
                VSFT_Bitmap *bm = &slot->bitmap;
                int gx0 = cx + (int)(pen >> 6) + slot->bitmap_left;
                int gy0 = baseline - slot->bitmap_top;
                unsigned int yy;
                for (yy = 0; yy < bm->rows; yy++) {
                    int row_index = bm->pitch >= 0 ? (int)yy : (int)(bm->rows - 1 - yy);
                    const unsigned char *src = bm->buffer + (size_t)row_index * (size_t)(bm->pitch >= 0 ? bm->pitch : -bm->pitch);
                    unsigned int xx;
                    for (xx = 0; xx < bm->width; xx++) {
                        int px = gx0 + (int)xx;
                        int py = gy0 + (int)yy;
                        unsigned char coverage = src[xx];
                        if (px >= 0 && py >= 0 && px < width && py < height && coverage) {
                            unsigned char *dst = rgba + (size_t)py * (size_t)stride + (size_t)px * 4u;
                            unsigned int alpha = (unsigned int)coverage * (unsigned int)a / 255u;
                            unsigned int inv = 255u - alpha;
                            dst[0] = (uint8_t)(((unsigned int)r * alpha + (unsigned int)dst[0] * inv) / 255u);
                            dst[1] = (uint8_t)(((unsigned int)g * alpha + (unsigned int)dst[1] * inv) / 255u);
                            dst[2] = (uint8_t)(((unsigned int)b * alpha + (unsigned int)dst[2] * inv) / 255u);
                            dst[3] = (uint8_t)(alpha + ((unsigned int)dst[3] * inv) / 255u);
                        }
                    }
                }
                pen += slot->advance.x;
            }
        }
    }
    return 0;
}

const char *vscript_font_error(VSFont *f) {
    if (!f) return "font handle invalid";
    return f->error;
}

void vscript_font_destroy(VSFont *f) {
    if (!f) return;
    if (f->face && f->FT_Done_Face) f->FT_Done_Face(f->face);
    if (f->lib && f->FT_Done_FreeType) f->FT_Done_FreeType(f->lib);
    if (f->dl_handle) { dlclose(f->dl_handle); f->dl_handle = NULL; }
    free(f);
}

/* -------------------------------------------------------------------------
 * Patch 2: native audio remuxing
 * ------------------------------------------------------------------------- */

typedef struct AVFormatContextMuxPrefix {
    const void *av_class;
    void *iformat;
    void *oformat;
    void *priv_data;
    void *pb;
    int ctx_flags;
    unsigned int nb_streams;
    void **streams;
} AVFormatContextMuxPrefix;

static char g_mux_error[512] = "ok";

typedef struct VSMuxer {
    void *lib_codec;
    void *lib_format;
    void *lib_util;
    void *video_ctx;
    void *audio_ctx;
    void *out_ctx;
    void *video_stream;
    void *audio_stream;
    void *packet_video;
    void *packet_audio;
    char error[512];
    int out_video_index;
    int out_audio_index;
    AVRational video_tb;
    AVRational audio_tb;
    AVRational out_video_tb;
    AVRational out_audio_tb;
    int (*avformat_open_input)(void **, const char *, void *, void **);
    int (*avformat_find_stream_info)(void *, void *);
    void (*avformat_close_input)(void **);
    int (*av_read_frame)(void *, void *);
    int (*avformat_alloc_output_context2)(void **, void *, const char *, const char *);
    void *(*avformat_new_stream)(void *, const void *);
    int (*avformat_write_header)(void *, void *);
    int (*av_interleaved_write_frame)(void *, void *);
    int (*av_write_trailer)(void *);
    void (*avformat_free_context)(void *);
    int (*avio_open2)(void **, const char *, int, const void *, void *);
    int (*avio_closep)(void **);
    int (*avcodec_parameters_copy)(void *, const void *);
    void *(*av_packet_alloc)(void);
    void (*av_packet_free)(void **);
    void (*av_packet_unref)(void *);
    void (*av_packet_rescale_ts)(void *, AVRational, AVRational);
} VSMuxer;

static void mux_set_error(VSMuxer *m, const char *msg) {
    if (!m) return;
    snprintf(m->error, sizeof(m->error), "%s", msg ? msg : "unknown mux error");
    snprintf(g_mux_error, sizeof(g_mux_error), "%s", msg ? msg : "unknown mux error");
}

static int mux_resolve(VSMuxer *m) {
#define MRES(lib, field) do { m->field = dlsym((lib), #field); if (!m->field) return -1; } while (0)
    MRES(m->lib_format, avformat_open_input);
    MRES(m->lib_format, avformat_find_stream_info);
    MRES(m->lib_format, avformat_close_input);
    MRES(m->lib_format, av_read_frame);
    MRES(m->lib_format, avformat_alloc_output_context2);
    MRES(m->lib_format, avformat_new_stream);
    MRES(m->lib_format, avformat_write_header);
    MRES(m->lib_format, av_interleaved_write_frame);
    MRES(m->lib_format, av_write_trailer);
    MRES(m->lib_format, avformat_free_context);
    MRES(m->lib_format, avio_open2);
    MRES(m->lib_format, avio_closep);
    MRES(m->lib_codec, avcodec_parameters_copy);
    MRES(m->lib_codec, av_packet_alloc);
    MRES(m->lib_codec, av_packet_free);
    MRES(m->lib_codec, av_packet_unref);
    MRES(m->lib_codec, av_packet_rescale_ts);
#undef MRES
    return 0;
}

static void mux_close_inputs(VSMuxer *m) {
    if (!m) return;
    if (m->video_ctx && m->avformat_close_input) m->avformat_close_input(&m->video_ctx);
    if (m->audio_ctx && m->avformat_close_input) m->avformat_close_input(&m->audio_ctx);
    m->video_ctx = NULL;
    m->audio_ctx = NULL;
}

static int mux_find_stream(VSMuxer *m, void *ctx, int codec_type) {
    AVFormatContextMuxPrefix *fc = (AVFormatContextMuxPrefix *)ctx;
    unsigned int i;
    if (!fc || !fc->streams) return -1;
    for (i = 0; i < fc->nb_streams; i++) {
        AVStreamPrefix *st = (AVStreamPrefix *)fc->streams[i];
        AVCodecParametersPrefix *par;
        if (!st || !st->codecpar) continue;
        par = (AVCodecParametersPrefix *)st->codecpar;
        if (par->codec_type == codec_type) return (int)i;
    }
    return -1;
}

static int mux_read_selected(VSMuxer *m, void *ctx, void *packet, int wanted) {
    for (;;) {
        int ret = m->av_read_frame(ctx, packet);
        if (ret < 0) return 0;
        if (((AVPacketPrefix *)packet)->stream_index == wanted) return 1;
        m->av_packet_unref(packet);
    }
}

static long double mux_packet_time(void *packet, AVRational tb) {
    AVPacketPrefix *p = (AVPacketPrefix *)packet;
    int64_t ts = p->dts;
    if (ts == INT64_MIN) ts = p->pts;
    if (ts == INT64_MIN || tb.den == 0) return 1e100L;
    return ((long double)ts * (long double)tb.num) / (long double)tb.den;
}

const char *vscript_mux_audio_error(void) { return g_mux_error; }

int vscript_mux_audio(const char *video_path, const char *audio_path, const char *output_path) {
    VSMuxer m;
    AVFormatContextMuxPrefix *ofc;
    AVStreamPrefix *vout;
    AVStreamPrefix *aout;
    AVFormatContextMuxPrefix *vfc;
    AVFormatContextMuxPrefix *afc;
    int vindex, aindex;
    int have_v = 0, have_a = 0;
    int ret;
    if (!video_path || !audio_path || !output_path || !video_path[0] || !audio_path[0] || !output_path[0]) return -1;
    memset(&m, 0, sizeof(m));
    snprintf(m.error, sizeof(m.error), "ok");
    snprintf(g_mux_error, sizeof(g_mux_error), "ok");

    m.lib_codec = load_any("libavcodec.so");
    m.lib_format = load_any("libavformat.so");
    m.lib_util = load_any("libavutil.so");
    if (!m.lib_codec || !m.lib_format || !m.lib_util || mux_resolve(&m) != 0) {
        mux_set_error(&m, "library FFmpeg atau simbol muxing tidak lengkap");
        return -1;
    }

    ret = m.avformat_open_input(&m.video_ctx, video_path, NULL, NULL);
    if (ret < 0) { mux_set_error(&m, "gagal membuka video sementara"); goto fail; }
    ret = m.avformat_find_stream_info(m.video_ctx, NULL);
    if (ret < 0) { mux_set_error(&m, "gagal membaca stream video"); goto fail; }
    ret = m.avformat_open_input(&m.audio_ctx, audio_path, NULL, NULL);
    if (ret < 0) { mux_set_error(&m, "gagal membuka file audio"); goto fail; }
    ret = m.avformat_find_stream_info(m.audio_ctx, NULL);
    if (ret < 0) { mux_set_error(&m, "gagal membaca stream audio"); goto fail; }

    vindex = mux_find_stream(&m, m.video_ctx, AVMEDIA_TYPE_VIDEO);
    aindex = mux_find_stream(&m, m.audio_ctx, 1 /* AVMEDIA_TYPE_AUDIO */);
    if (vindex < 0 || aindex < 0) { mux_set_error(&m, "stream video atau audio tidak ditemukan"); goto fail; }
    vfc = (AVFormatContextMuxPrefix *)m.video_ctx;
    afc = (AVFormatContextMuxPrefix *)m.audio_ctx;
    m.video_stream = vfc->streams[vindex];
    m.audio_stream = afc->streams[aindex];
    m.video_tb = ((AVStreamPrefix *)m.video_stream)->time_base;
    m.audio_tb = ((AVStreamPrefix *)m.audio_stream)->time_base;

    ret = m.avformat_alloc_output_context2(&m.out_ctx, NULL, NULL, output_path);
    if (ret < 0 || !m.out_ctx) { mux_set_error(&m, "gagal membuat container output"); goto fail; }
    vout = (AVStreamPrefix *)m.avformat_new_stream(m.out_ctx, NULL);
    aout = (AVStreamPrefix *)m.avformat_new_stream(m.out_ctx, NULL);
    if (!vout || !aout) { mux_set_error(&m, "gagal membuat stream output"); goto fail; }
    if (m.avcodec_parameters_copy(vout->codecpar, ((AVStreamPrefix *)m.video_stream)->codecpar) < 0 ||
        m.avcodec_parameters_copy(aout->codecpar, ((AVStreamPrefix *)m.audio_stream)->codecpar) < 0) {
        mux_set_error(&m, "gagal menyalin parameter codec"); goto fail;
    }
    vout->time_base = m.video_tb;
    aout->time_base = m.audio_tb;
    m.out_video_index = vout->index;
    m.out_audio_index = aout->index;
    m.out_video_tb = vout->time_base;
    m.out_audio_tb = aout->time_base;
    ofc = (AVFormatContextMuxPrefix *)m.out_ctx;
    if (ofc->oformat && (((AVOutputFormatPrefix *)ofc->oformat)->flags & AVFMT_NOFILE) == 0) {
        ret = m.avio_open2(&ofc->pb, output_path, 2, NULL, NULL);
        if (ret < 0) { mux_set_error(&m, "gagal membuka container output"); goto fail; }
    }
    ret = m.avformat_write_header(m.out_ctx, NULL);
    if (ret < 0) { mux_set_error(&m, "gagal menulis header container"); goto fail; }
    /* FFmpeg may choose a container-specific stream time base during header write. */
    m.out_video_tb = vout->time_base;
    m.out_audio_tb = aout->time_base;

    m.packet_video = m.av_packet_alloc();
    m.packet_audio = m.av_packet_alloc();
    if (!m.packet_video || !m.packet_audio) { mux_set_error(&m, "gagal alokasi packet muxer"); goto fail; }
    have_v = mux_read_selected(&m, m.video_ctx, m.packet_video, vindex);
    have_a = mux_read_selected(&m, m.audio_ctx, m.packet_audio, aindex);

    while (have_v || have_a) {
        int choose_video;
        if (!have_a) choose_video = 1;
        else if (!have_v) choose_video = 0;
        else choose_video = mux_packet_time(m.packet_video, m.video_tb) <= mux_packet_time(m.packet_audio, m.audio_tb);
        if (choose_video) {
            AVPacketPrefix *p = (AVPacketPrefix *)m.packet_video;
            p->stream_index = m.out_video_index;
            m.av_packet_rescale_ts(m.packet_video, m.video_tb, m.out_video_tb);
            ret = m.av_interleaved_write_frame(m.out_ctx, m.packet_video);
            m.av_packet_unref(m.packet_video);
            have_v = mux_read_selected(&m, m.video_ctx, m.packet_video, vindex);
        } else {
            AVPacketPrefix *p = (AVPacketPrefix *)m.packet_audio;
            p->stream_index = m.out_audio_index;
            m.av_packet_rescale_ts(m.packet_audio, m.audio_tb, m.out_audio_tb);
            ret = m.av_interleaved_write_frame(m.out_ctx, m.packet_audio);
            m.av_packet_unref(m.packet_audio);
            have_a = mux_read_selected(&m, m.audio_ctx, m.packet_audio, aindex);
        }
        if (ret < 0) { mux_set_error(&m, "gagal menulis packet audio/video"); goto fail; }
    }

    ret = m.av_write_trailer(m.out_ctx);
    if (ret < 0) { mux_set_error(&m, "gagal menulis trailer container"); goto fail; }
    if (ofc->pb) m.avio_closep(&ofc->pb);
    m.avformat_free_context(m.out_ctx);
    m.out_ctx = NULL;
    m.av_packet_free(&m.packet_video);
    m.av_packet_free(&m.packet_audio);
    mux_close_inputs(&m);
    return 0;

fail:
    if (m.out_ctx) {
        ofc = (AVFormatContextMuxPrefix *)m.out_ctx;
        if (ofc->pb) m.avio_closep(&ofc->pb);
        m.avformat_free_context(m.out_ctx);
    }
    if (m.packet_video) m.av_packet_free(&m.packet_video);
    if (m.packet_audio) m.av_packet_free(&m.packet_audio);
    mux_close_inputs(&m);
    return -1;
}
