package engine

/*
#cgo CFLAGS: -I.
#cgo LDFLAGS: -ldl -lm
#include <stdlib.h>
#include "ffmpeg_bridge.h"
*/
import "C"

import (
	"fmt"
	"strings"
	"unsafe"
)

type FFmpegEncoder struct {
	ptr        *C.VSEncoder
	stride     int
	frameBytes int
	height     int
}

func NewFFmpegEncoder(path string, width, height, fps int, codec string, bitrate int64) (*FFmpegEncoder, error) {
	enc := C.vscript_encoder_create()
	if enc == nil {
		return nil, fmt.Errorf("native multimedia backend gagal dialokasikan")
	}
	if msg := C.GoString(C.vscript_encoder_error(enc)); msg != "ok" && msg != "" {
		C.vscript_encoder_destroy(enc)
		return nil, fmt.Errorf("ffmpeg: %s", msg)
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	ccodec := C.CString(codec)
	defer C.free(unsafe.Pointer(ccodec))
	var cformat *C.char
	format := inferContainer(path)
	if format != "" {
		cformat = C.CString(format)
		defer C.free(unsafe.Pointer(cformat))
	}
	if C.vscript_encoder_open(enc, cpath, C.int(width), C.int(height), C.int(fps), ccodec, cformat, C.int64_t(bitrate)) < 0 {
		msg := C.GoString(C.vscript_encoder_error(enc))
		C.vscript_encoder_destroy(enc)
		return nil, fmt.Errorf("ffmpeg: %s", msg)
	}
	return &FFmpegEncoder{ptr: enc, stride: width * 4, frameBytes: width * height * 4, height: height}, nil
}

func (e *FFmpegEncoder) WriteRGBA(rgba []byte, pts int64) error {
	if e == nil || e.ptr == nil {
		return fmt.Errorf("encoder belum dibuka")
	}
	if len(rgba) == 0 {
		return fmt.Errorf("frame kosong")
	}
	if e.stride <= 0 || e.frameBytes <= 0 || len(rgba) < e.frameBytes {
		return fmt.Errorf("ukuran frame rgba tidak valid: butuh minimal %d byte", e.frameBytes)
	}
	if C.vscript_encoder_write_frame(e.ptr, (*C.uchar)(unsafe.Pointer(&rgba[0])), C.int(e.stride), C.int64_t(pts)) < 0 {
		return fmt.Errorf("ffmpeg: %s", C.GoString(C.vscript_encoder_error(e.ptr)))
	}
	return nil
}

func (e *FFmpegEncoder) Close() error {
	if e == nil || e.ptr == nil {
		return nil
	}
	code := C.vscript_encoder_close(e.ptr)
	if code < 0 {
		err := fmt.Errorf("ffmpeg: %s", C.GoString(C.vscript_encoder_error(e.ptr)))
		C.vscript_encoder_destroy(e.ptr)
		e.ptr = nil
		return err
	}
	C.vscript_encoder_destroy(e.ptr)
	e.ptr = nil
	return nil
}

func FFmpegSearchPaths() []string {
	return []string{C.GoString(C.vscript_ffmpeg_path_report())}
}

func inferContainer(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return strings.ToLower(path[i+1:])
		}
		if path[i] == '/' || path[i] == '\\' {
			break
		}
	}
	return ""
}

func muxAudio(videoPath, audioPath, outputPath string) error {
	cvideo := C.CString(videoPath)
	defer C.free(unsafe.Pointer(cvideo))
	caudio := C.CString(audioPath)
	defer C.free(unsafe.Pointer(caudio))
	cout := C.CString(outputPath)
	defer C.free(unsafe.Pointer(cout))
	if C.vscript_mux_audio(cvideo, caudio, cout) < 0 {
		return fmt.Errorf("ffmpeg mux audio: %s", C.GoString(C.vscript_mux_audio_error()))
	}
	return nil
}
