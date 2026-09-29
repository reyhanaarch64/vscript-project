#ifndef VSCRIPT_FFMPEG_BRIDGE_H
#define VSCRIPT_FFMPEG_BRIDGE_H

#include <stdint.h>

typedef struct VSEncoder VSEncoder;

VSEncoder *vscript_encoder_create(void);
int vscript_encoder_open(VSEncoder *enc, const char *output, int width, int height, int fps, const char *codec_name, const char *format, int64_t bitrate);
int vscript_encoder_write_frame(VSEncoder *enc, const uint8_t *rgba, int stride, int64_t pts);
int vscript_encoder_close(VSEncoder *enc);
const char *vscript_encoder_error(VSEncoder *enc);
void vscript_encoder_destroy(VSEncoder *enc);
const char *vscript_ffmpeg_path_report(void);

typedef struct VSFont VSFont;
VSFont *vscript_font_create(const char *path, int size);
int vscript_font_set_size(VSFont *font, int size);
int vscript_font_draw_rgba(VSFont *font, uint8_t *rgba, int width, int height, int stride, int cx, int cy, uint8_t r, uint8_t g, uint8_t b, uint8_t a, const char *text);
const char *vscript_font_error(VSFont *font);
void vscript_font_destroy(VSFont *font);

int vscript_mux_audio(const char *video_path, const char *audio_path, const char *output_path);
const char *vscript_mux_audio_error(void);

#endif
