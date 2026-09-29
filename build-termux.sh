#!/data/data/com.termux/files/usr/bin/bash
set -e

VERSION="0.2.0"
OUT="dist/vscript"

if ! command -v go >/dev/null 2>&1; then
    echo "error: golang belum terpasang"
    exit 1
fi
if ! command -v clang >/dev/null 2>&1; then
    echo "error: clang belum terpasang"
    exit 1
fi

if [ -z "${PREFIX:-}" ]; then
    echo "warning: PREFIX tidak terdeteksi; build tetap dicoba"
fi

if [ -n "${PREFIX:-}" ]; then
    for lib in libavcodec libavformat libavutil; do
        if ! ls "$PREFIX/lib/${lib}.so"* >/dev/null 2>&1; then
            echo "error: $PREFIX/lib/${lib}.so* tidak ditemukan"
            echo "pastikan paket ffmpeg tersedia di Termux"
            exit 1
        fi
    done
    for lib in libfreetype; do
        if ! ls "$PREFIX/lib/${lib}.so"* >/dev/null 2>&1; then
            echo "warning: FreeType tidak ditemukan; fitur font file tidak akan tersedia saat runtime"
        fi
    done
fi

mkdir -p dist
echo "membangun vscript ${VERSION}"
echo "arsitektur : $(uname -m)"
echo "prefix     : ${PREFIX:-unset}"
echo

CGO_ENABLED=1 CC=clang go build \
    -trimpath \
    -ldflags="-s -w" \
    -o "$OUT" \
    ./cmd/vscript

chmod +x "$OUT"
"$OUT" --version

echo
echo "build berhasil: $(pwd)/$OUT"
