# changelog

## 0.2.3

- memperbaiki bug built-in `text()` yang sebelumnya memakai nilai `size` sebagai skala mentah setiap cell 5×7 sehingga text berukuran besar berubah menjadi blok dan keluar dari layout
- menjadikan `size` pada `text()` sebagai tinggi glyph kira-kira untuk built-in renderer
- menambahkan glyph punctuation ASCII umum pada built-in renderer
- menyamakan ukuran `text(..., size, ..., font)` dengan ukuran yang diminta pada pemanggilan `text()` dan mengosongkan cache glyph ketika size berubah
- mempertahankan perbaikan `FT_Bitmap.pitch` negatif dari 0.2.1
- memperbaiki perhitungan corner rounded rectangle
- membuat duplicate keyframe pada timestamp yang sama menggantikan frame sebelumnya
- mencegah interpolasi numeric memaksa tipe yang tidak kompatibel menjadi angka nol
- memperbaiki error handling `break`, `continue`, dan `return` di konteks yang tidak valid
- meneruskan error dari proses close/trailer encoder ke caller
- membuat codec encoder strict: codec yang diminta tidak lagi diam-diam diganti ke encoder lain
- memperketat validasi stride frame RGBA
- memperbaiki cleanup dynamic library pada native audio muxer
- membuat muxer menolak output yang sama dengan input
- membuat error pembacaan packet audio/video tidak dianggap sebagai EOF
- membuat `getMedia()` mengambil extension dari URL path dan bukan query string
- memperbaiki klasifikasi media berdasarkan extension agar tidak menggunakan substring matching yang ambigu
- membuat parsing warna hex strict dengan dukungan `#RGB`, `#RGBA`, `#RRGGBB`, dan `#RRGGBBAA`
- menambahkan validasi design resolution dan multiple video block
- memperbaiki equality agar string tidak dibandingkan setelah dipaksa ke representasi text yang sama
- memperbaiki modulo dengan nol agar menghasilkan runtime error
- menambahkan regression test untuk text, font size, geometry, timeline, runtime control flow, equality, dan color parsing
- memperbarui build script dan CLI ke versi 0.2.3
- menambahkan LICENSE MIT dan `.gitignore`

## 0.2.1

- memperbaiki pembacaan scanline FreeType ketika `FT_Bitmap.pitch` bernilai negatif
- memperbaiki alignment prefix `FT_GlyphSlotRec` pada ABI 32-bit agar offset `bitmap` tidak bergeser
- menormalisasi bitmap glyph ke buffer grayscale internal sehingga renderer tidak lagi membaca stride/pixel mode mentah pada setiap pixel output
- menambahkan validasi stride dan dukungan bitmap MONO, GRAY2, GRAY4, GRAY, serta BGRA alpha coverage
- menyimpan pointer `FT_Load_Char` sekali per font, bukan melakukan `dlsym()` pada setiap glyph
- menambahkan cache glyph per font agar glyph yang sama tidak dirasterisasi ulang pada setiap frame
- menggunakan framebuffer RGBA yang sama sepanjang export untuk mengurangi alokasi dan tekanan garbage collector
- menambahkan regresi test untuk renderer font dan stabilitas frame berulang

## 0.2.0 — patch 2

- menambahkan coordinate space `design W H` untuk output multi-resolusi
- memperbaiki `else if`
- menambahkan timeline, keyframe, interpolasi object/array/numeric
- menambahkan pemanggilan method object
- menambahkan dukungan font TTF/OTF melalui FreeType native dynamic loading
- menambahkan audio remux native dan keyword `audio` pada export
- menambahkan `cropAspect`, easing, `lerp`, `clamp`, dan `env`
- menambahkan test parser dan timeline
- menambahkan build script Termux
