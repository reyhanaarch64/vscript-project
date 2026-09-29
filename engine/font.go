package engine

/*
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include "ffmpeg_bridge.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type Font struct {
	ptr  *C.VSFont
	Path string
	Size int
}

func NewFont(path string, size int) (*Font, error) {
	if path == "" {
		return nil, fmt.Errorf("path font kosong")
	}
	if size <= 0 {
		size = 48
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	ptr := C.vscript_font_create(cpath, C.int(size))
	if ptr == nil {
		return nil, fmt.Errorf("gagal membuat font")
	}
	msg := C.GoString(C.vscript_font_error(ptr))
	if msg != "ok" && msg != "" {
		C.vscript_font_destroy(ptr)
		return nil, fmt.Errorf("font: %s", msg)
	}
	return &Font{ptr: ptr, Path: path, Size: size}, nil
}

func (f *Font) DrawRGBA(img []byte, width, height, stride, cx, cy, size int, rgba [4]uint8, text string) error {
	if f == nil || f.ptr == nil {
		return fmt.Errorf("font belum dibuka")
	}
	if len(img) < stride*height || width <= 0 || height <= 0 || stride < width*4 {
		return fmt.Errorf("buffer gambar tidak valid")
	}
	if size <= 0 {
		size = f.Size
	}
	if size != f.Size {
		if C.vscript_font_set_size(f.ptr, C.int(size)) < 0 {
			return fmt.Errorf("font: %s", C.GoString(C.vscript_font_error(f.ptr)))
		}
		f.Size = size
	}
	ctext := C.CString(text)
	defer C.free(unsafe.Pointer(ctext))
	ret := C.vscript_font_draw_rgba(
		f.ptr,
		(*C.uchar)(unsafe.Pointer(&img[0])),
		C.int(width), C.int(height), C.int(stride),
		C.int(cx), C.int(cy),
		C.uchar(rgba[0]), C.uchar(rgba[1]), C.uchar(rgba[2]), C.uchar(rgba[3]),
		ctext,
	)
	if ret < 0 {
		return fmt.Errorf("font: %s", C.GoString(C.vscript_font_error(f.ptr)))
	}
	return nil
}

func (f *Font) Close() {
	if f == nil || f.ptr == nil {
		return
	}
	C.vscript_font_destroy(f.ptr)
	f.ptr = nil
}
