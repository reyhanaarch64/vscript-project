# VScript 0.2.3

VScript adalah bahasa pemrograman umum dengan fokus kuat pada **procedural dan generative video**.

Dengan VScript, video tidak dibuat dengan menyusun ratusan file gambar secara manual. Scene, bentuk, teks, media, animasi, filter, audio, dan output video ditulis sebagai kode lalu dirender oleh runtime VScript.

VScript dibangun dengan **Go** dan menggunakan library native multimedia secara langsung melalui bridge native, terutama **FFmpeg libraries** dan **FreeType**. VScript tidak perlu menjalankan executable `ffmpeg` untuk proses export video.

> Status: **open source / stable development release**
>
> Versi awal tetap dapat mengalami perubahan API besar pada rilis berikutnya. Perubahan yang masuk 0.2.3 ditujukan untuk memperbaiki correctness, error reporting, renderer text, timeline, dan native resource cleanup.

## Mengapa VScript?

VScript ditujukan untuk orang yang ingin menggabungkan programming dengan pembuatan video.

```plaintext
kode VScript
    ↓
parser + runtime
    ↓
scene / timeline / animation
    ↓
renderer
    ↓
raw video frame
    ↓
FFmpeg native encoder + muxer
    ↓
MP4 / video output
```

Pendekatan ini membuat video dapat diperlakukan sebagai program: posisi objek dapat dihitung dengan matematika, animation dapat dibuat melalui function dan keyframe, dan seluruh proses render dapat direproduksi dari source code.

## Perubahan penting pada 0.2.3

- Memperbaiki bug serius pada built-in `text()` yang sebelumnya memperlakukan `size` sebagai skala setiap sel glyph 5×7. Pada 0.2.3, `size` adalah tinggi glyph kira-kira dalam pixel/design units.
- Memperbaiki dukungan karakter ASCII umum pada built-in text renderer.
- `text(..., size, color, font)` sekarang memakai `size` dari pemanggilan `text()` dan melakukan update ukuran FreeType dengan cache glyph yang di-reset saat ukuran berubah.
- Mempertahankan perbaikan `FT_Bitmap.pitch` negatif dan normalisasi bitmap FreeType dari 0.2.1.
- Memperbaiki perhitungan rounded rectangle.
- Memperbaiki duplicate keyframe pada timestamp yang sama agar keyframe baru menggantikan nilai lama.
- Interpolasi timeline tidak lagi memaksa tipe yang berbeda menjadi angka nol.
- `break`, `continue`, dan `return` yang berada di konteks tidak valid sekarang menghasilkan error runtime, bukan panic yang tidak terkontrol.
- Kesalahan penutupan encoder dan trailer output diteruskan ke caller.
- Encoder tidak lagi diam-diam mengganti codec yang diminta ke codec lain.
- Native audio muxer menutup dynamic library handle pada semua jalur selesai/error.
- `getMedia()` mengambil extension dari URL path, bukan query string.
- Parsing warna hex menjadi strict dan mendukung `#RGB`, `#RGBA`, `#RRGGBB`, dan `#RRGGBBAA`.
- Menambah regression test untuk text renderer, font size override, timeline, equality, color parsing, control flow, dan multiple video block.

## Fitur utama

- General-purpose language basics: variable, string, number, boolean, array, object, function, return, conditional, loop, dan runtime error reporting.
- Video rendering dengan resolusi output yang dapat ditentukan.
- Logical/design resolution menggunakan `design W H` agar layout dapat ditulis pada coordinate space logis.
- Primitive graphics: rectangle, rounded rectangle, circle, line, polygon, text, dan image.
- Font TTF/OTF melalui FreeType.
- Timeline dan keyframe dengan interpolasi numeric, array, dan object.
- Filter bawaan dan custom filter berbasis function.
- Crop, crop berdasarkan aspect ratio, pixelate, serta save frame.
- Audio track dan native remux ke output video.
- Filesystem API.
- HTTP request dan media download API.
- Native multimedia pipeline melalui FFmpeg libraries.
- Dynamic library loading yang kompatibel dengan environment seperti Termux maupun Linux.

## Quick start

### Instalasi otomatis

Target installer publik VScript menggunakan pola berikut:

```plaintext
wget https://vscript-docts.vercel.app/install.sh && bash install.sh
```

Installer dirancang untuk:

1. mendeteksi platform dan arsitektur CPU;
2. memilih binary yang sesuai;
3. mengunduh binary dari direktori `assets/`;
4. memasangnya ke prefix/bin yang sesuai;
5. membuat perintah `vscript` tersedia langsung di shell.

> Catatan: URL installer tersebut baru dapat digunakan setelah dokumentasi dan asset binary target benar-benar dideploy.

### Build dari source

VScript menggunakan Go dengan CGO untuk bridge native.

Pada Termux, pastikan Go, Clang, FFmpeg development/runtime libraries, dan dependency native yang diperlukan sudah terpasang.

Contoh build:

```plaintext
chmod +x build.sh
./build.sh
```

Binary hasil build berada di:

```plaintext
dist/vscript
```

Untuk menjalankan binary lokal:

```plaintext
./dist/vscript examples/hello.vs
```

## Program VScript pertama

Contoh minimal:

```plaintext
let name = "Rehan";

if name == "Rehan" {
    print("Halo, " + name);
} else {
    print("Halo");
}
```

VScript juga dapat digunakan untuk merender video.

```plaintext
let W = 1280;
let H = 720;
let FPS = 30;
let DURATION = 10;

video W H FPS DURATION {
    background("#07111f");

    circle(640, 360, 120, "#2088d8");

    text(
        "Hello from VScript",
        640,
        360,
        48,
        "#ffffff"
    );
}

export video "output/hello.mp4"
    codec "libx264";
```

Perintah:

```plaintext
vscript hello.vs
```

## Tutorial: membuat video procedural

### 1. Tentukan resolution dan frame rate

Gunakan output resolution untuk hasil akhir dan `design` untuk logical coordinate system.

```plaintext
let W = 1920;
let H = 1080;
let FPS = 30;
let DURATION = 60;

video W H FPS DURATION design 640 360 {
    background("#060b14");
}
```

`design 640 360` berarti scene menggunakan coordinate space 640×360, sedangkan framebuffer akhir adalah 1920×1080.

Pemetaan dilakukan secara terpisah untuk sumbu X dan Y. Karena itu, untuk mempertahankan bentuk tanpa distorsi, aspect ratio `design` sebaiknya sama dengan aspect ratio output.

Contoh pasangan yang konsisten:

```plaintext
640x360  → 1280x720
640x360  → 1920x1080
480x270  → 1280x720
```

### 2. Pahami sistem koordinat

API primitive 0.2.3 menggunakan anchor berikut:

```plaintext
rect(x, y, width, height, ...)   → x/y adalah sudut kiri-atas
circle(x, y, radius, ...)        → x/y adalah pusat
line(x1, y1, x2, y2, ...)         → x/y adalah endpoint
text(text, x, y, ...)             → x/y adalah pusat teks
polygon(points, ...)              → titik memakai koordinat absolut
image(x, y, width, height, ...)   → x/y adalah sudut kiri-atas
```

`design` hanya mengubah pemetaan coordinate space ke output. Ia tidak mengubah anchor primitive.

Satu script 0.2.3 hanya boleh memiliki satu blok `video` top-level. Beberapa `export video` dapat memakai blok yang sama untuk menghasilkan beberapa file output.

### 3. Buat animasi berdasarkan waktu

Di dalam video block, runtime menyediakan `t`, `frame`, `fps`, `width`, dan `height`.

```plaintext
let x = 320 + sin(t * 2) * 80;
let y = 180;

circle(x, y, 32, "#66ccff");
```

Karena `t` berubah pada setiap frame, object bergerak secara otomatis.

### 4. Gunakan conditional

Conditional dapat digunakan untuk mengganti isi scene berdasarkan waktu.

```plaintext
if t < 10 {
    text("INTRO", 320, 80, 32, "#ffffff");
} else if t < 30 {
    text("ABOUT ME", 320, 80, 32, "#66ccff");
} else {
    text("VScript", 320, 80, 32, "#ffffff");
}
```

`else if` diparse sebagai conditional bertingkat di AST/runtime.

### 5. Gunakan function

Pisahkan bagian scene menjadi function agar script tetap mudah dirawat.

```plaintext
function drawTitle(title) {
    text(title, 320, 100, 42, "#ffffff");
}

video 1280 720 30 10s design 640 360 {
    background("#08101c");
    drawTitle("VScript");
}
```

### 6. Gunakan timeline dan keyframe

Timeline menyimpan nilai animasi sepanjang waktu.

```plaintext
let tl = timeline(60);

tl.keyframe("x", 0, -100);
tl.keyframe("x", 5, 100);
tl.keyframe("x", 20, 100);
tl.keyframe("x", 25, 700);

video 1280 720 30 60s design 640 360 {
    let x = tl.value("x", t);
    circle(x, 180, 32, "#66ccff");
}
```

Jika dua keyframe pada track yang sama memiliki timestamp identik, keyframe terakhir menggantikan nilai sebelumnya.

Interpolasi numerik bersifat linear. Array dengan panjang sama dan object dengan property yang cocok dapat diinterpolasikan secara rekursif. Tipe yang tidak kompatibel dipilih secara discrete berdasarkan posisi interpolasi, bukan dipaksa menjadi angka.

### 7. Membaca font dari file

Font eksternal dapat dimuat sebagai resource.

```plaintext
let titleFont = font("assets/Roboto-Bold.ttf", 64);

video 1280 720 30 10s design 640 360 {
    background("#07111f");
    text("VScript", 320, 180, 64, "#ffffff", titleFont);
}
```

`font(path, size)` membuat object font dengan ukuran awal. Ketika object font dipakai sebagai argumen keenam pada `text()`, parameter `size` pada `text()` menjadi ukuran render yang digunakan pada pemanggilan tersebut.

Path dapat berupa file font lokal yang dapat diakses oleh runtime, termasuk path seperti:

```plaintext
/system/fonts/Roboto-Regular.ttf
```

pada environment Android/Termux yang menyediakan file tersebut, atau path font lokal lain yang valid.

FreeType dimuat melalui dynamic loading dan bitmap glyph dinormalisasi ke buffer internal sebelum compositing ke framebuffer.

### 8. Menambahkan audio

Audio dapat dimuat sebagai media resource dan dimasukkan ke output saat export.

```plaintext
let music = audio("assets/music.mp3");

video 1280 720 30 60s design 640 360 {
    background("#060b14");
}

export video "output/video.mp4"
    codec "libx264"
    bitrate 8000000
    audio music;
```

Jalur export audio menggunakan native remux; audio tidak otomatis ditranscode. Karena itu, codec audio harus kompatibel dengan container output.

## Syntax dasar

### Variable

```plaintext
let username = "Rehan";
let count = 10;
let active = true;
```

### Constant-like configuration

VScript 0.2.3 belum memiliki keyword `const`. Gunakan `let` untuk nilai konfigurasi yang tidak dimaksudkan untuk diubah.

### String

```plaintext
let title = "VScript";
let full = "Hello, " + title;
```

### Number

```plaintext
let x = 120;
let speed = 2.5;
```

### Boolean

```plaintext
let ready = true;
let hidden = false;
```

### Array

```plaintext
let points = [10, 20, 30, 40];
```

### Object

```plaintext
let user = {
    name: "Rehan",
    language: "VScript"
};
```

### Conditional

```plaintext
if score >= 90 {
    print("excellent");
} else if score >= 75 {
    print("good");
} else {
    print("keep learning");
}
```

### For

Range `..` pada VScript bersifat inklusif dan dapat berjalan naik atau turun.

```plaintext
for i in 0..4 {
    print(i);
}
```

### While

```plaintext
let i = 0;

while i < 10 {
    print(i);
    i = i + 1;
}
```

### Break / continue

```plaintext
while true {
    if done {
        break;
    }

    continue;
}
```

`break` dan `continue` hanya valid di dalam loop. Keduanya tidak boleh digunakan untuk keluar dari function.

### Function

```plaintext
function add(a, b) {
    return a + b;
}

let result = add(10, 20);
```

## Video API

### `video`

Membuat rendering context video.

```plaintext
video WIDTH HEIGHT FPS DURATION {
    // scene
}
```

Dengan logical design size:

```plaintext
video 1920 1080 30 60s design 640 360 {
    // scene menggunakan coordinate 640x360
}
```

Output width dan height harus positif serta genap agar dapat diproses oleh pipeline YUV420P yang digunakan encoder saat ini.

### `background(color)`

Mengisi background frame.

```plaintext
background("#08101c");
```

### `rect(x, y, width, height, color, radius)`

Rectangle atau rounded rectangle. `x/y` adalah sudut kiri-atas.

```plaintext
rect(40, 40, 240, 120, "#11243a", 20);
```

`radius` otomatis dibatasi agar tidak melebihi setengah dimensi terkecil.

### `circle(x, y, radius, color)`

`x/y` adalah pusat circle.

```plaintext
circle(320, 180, 60, "#66ccff");
```

### `line(x1, y1, x2, y2, color, width)`

```plaintext
line(20, 20, 620, 20, "#18314a", 2);
```

### `polygon(points, color)`

Membuat polygon dari kumpulan titik.

```plaintext
polygon(
    [320, 100, 370, 40, 430, 100, 370, 150],
    "#2878b8"
);
```

### `text(text, x, y, size, color, font?)`

Tanpa argumen font, VScript menggunakan built-in 5×7 text renderer.

```plaintext
text("Hello", 320, 180, 32, "#ffffff");
```

Pada 0.2.3 `size` berarti tinggi glyph kira-kira, bukan skala sel bitmap mentah.

Dengan FreeType:

```plaintext
let f = font("assets/Roboto-Regular.ttf", 32);
text("Hello", 320, 180, 32, "#ffffff", f);
```

### Warna

Format yang diterima:

```plaintext
#RGB
#RGBA
#RRGGBB
#RRGGBBAA
```

Contoh:

```plaintext
background("#08101a");
rect(20, 20, 200, 100, "#0088ffaa", 12);
```

### `image(x, y, width, height, media)`

```plaintext
let logo = loadMedia("assets/logo.png");

video 640 360 30 3s {
    image(40, 40, 180, 100, logo);
}
```

### `crop(x, y, width, height)`

Crop source framebuffer ke region tertentu lalu memetakan hasil crop kembali ke ukuran framebuffer.

```plaintext
crop(100, 50, 440, 260);
```

### `cropAspect(ratio)`

Crop berdasarkan aspect ratio.

```plaintext
cropAspect("16:9");
```

### `pixelate(size)`

```plaintext
pixelate(8);
```

### `saveFrame(path)`

Menyimpan framebuffer saat frame tersebut dieksekusi.

```plaintext
saveFrame("output/frame.png");
```

Ekstensi `.png` menghasilkan PNG dan `.jpg`/`.jpeg` menghasilkan JPEG.

## Timeline API

### `timeline(duration)`

Membuat timeline.

```plaintext
let tl = timeline(60);
```

### `keyframe(name, time, value)`

Bisa digunakan sebagai function global:

```plaintext
keyframe(tl, "opacity", 0, 0);
```

atau sebagai method:

```plaintext
tl.keyframe("opacity", 0, 0);
```

### `value(name, time)`

```plaintext
let opacity = tl.value("opacity", t);
```

Alias method `at` juga tersedia.

### `has(name)`

```plaintext
if tl.has("x") {
    print("track tersedia");
}
```

## Utility matematika

```plaintext
let a = lerp(0, 100, 0.5);
let b = clamp(speed, 0, 1);
let c = easeIn(0.5);
let d = easeOut(0.5);
let e = easeInOut(0.5);
```

Helper matematika tambahan yang tersedia antara lain `sin`, `cos`, `tan`, `abs`, `floor`, `ceil`, `sqrt`, `min`, dan `max`.

## Filter

Filter bawaan yang tersedia pada 0.2.3:

```plaintext
gray:      filter("grayscale");
invert:    filter("invert");
brightness: filter("brightness", 8);
contrast:  filter("contrast", 1.08);
vignette:  filter("vignette", 0.2);
blur:      filter("blur", 2);
sepia:     filter("sepia");
scanlines: filter("scanlines", 0.15);
```

Contoh:

```plaintext
if t > 50 {
    filter("contrast", 1.12);
    filter("brightness", 6);
} else {
    filter("contrast", 1.04);
}
```

### Custom filter

Custom filter dapat menggunakan function yang menerima koordinat dan channel pixel:

```plaintext
function warm(x, y, r, g, b, a) {
    return [
        clamp(r + 10, 0, 255),
        clamp(g + 3, 0, 255),
        b,
        a
    ];
}

filter("warm");
```

Nama filter yang dievaluasi runtime akan dicari di environment terlebih dahulu. Jika nilainya merupakan function VScript, function tersebut dipakai sebagai custom per-pixel filter.

> Catatan: custom filter adalah operasi per-pixel dan dapat sangat mahal pada resolusi tinggi.

## Media

Resource media dapat dimuat dari local file atau source internet yang didukung.

```plaintext
let local = loadMedia("assets/image.png");
let remote = getMedia("https://example.com/image.png");
let music = audio("assets/music.mp3");
```

Object media mempunyai property:

```plaintext
media.path
media.type
media.size
```

`getMedia()` menyimpan hasil download ke file temporer dan mengembalikan object `Media`.

## Filesystem API

```plaintext
let content = readFile("data.txt");
writeFile("output.txt", content);
```

Utility yang tersedia:

```plaintext
exists(path)
isFile(path)
isDir(path)
fileSize(path)
mkdir(path)
remove(path)
copyFile(source, destination)
moveFile(source, destination)
listDir(path)
pathJoin(...parts)
```

## HTTP API

Bentuk dasar request:

```plaintext
let response = HttpReq("GET", "https://example.com/api/data");
print(response.status);
print(response.body);
```

Request body opsional. Object/array akan diencode sebagai JSON.

Header tambahan dapat diberikan sebagai object:

```plaintext
let response = HttpReq(
    "POST",
    "https://example.com/api/data",
    {hello: "world"},
    {"Accept": "application/json"}
);
```

Response memiliki `status`, `body`, `ok`, dan `headers`.

## CLI

Pola penggunaan dasar:

```plaintext
vscript file.vs
vscript run file.vs
vscript paths
vscript --version
```

## Struktur project

```plaintext
vscript-project/
├── cmd/
│   └── vscript/
├── engine/
│   ├── lexer.go
│   ├── parser.go
│   ├── ast.go
│   ├── runtime.go
│   ├── video.go
│   ├── native.go
│   ├── ffmpeg_bridge.c
│   └── ffmpeg_bridge.h
├── examples/
├── assets/
├── dist/
├── output/
├── build.sh
├── go.mod
├── LICENSE
└── README.md
```

Nama file internal dapat berubah ketika engine berkembang. Pemisahan utamanya tetap:

- `cmd/` — entry point CLI.
- `engine/` — lexer, parser, runtime, renderer, dan native bridge.
- `examples/` — contoh script.
- `assets/` — resource demo/release.
- `dist/` — binary hasil build.
- `output/` — hasil render lokal.

## Arsitektur native

VScript tidak memanggil command-line FFmpeg sebagai subprocess.

Renderer menghasilkan frame di memory lalu native bridge menyerahkan data tersebut ke stack multimedia.

Secara konseptual:

```plaintext
VScript source
    ↓
lexer
    ↓
parser / AST
    ↓
runtime
    ↓
scene + timeline
    ↓
renderer
    ↓
framebuffer RGBA
    ↓
native FFmpeg encoder
    ↓
container muxer
    ↓
output video
```

Audio mengikuti native remux pipeline ketika digunakan pada `export ... audio`.

Library native utama yang digunakan:

```plaintext
libavcodec
libavformat
libavutil
libfreetype
```

Resolver native mencoba `VSCRIPT_LIB_PATH`, `$PREFIX/lib` pada Termux, path Termux standar, beberapa path Linux umum, lalu dynamic loader sistem. Soname versioned juga dicari.

## Mengapa tidak cukup menjalankan FFmpeg CLI?

VScript sengaja tidak menjadikan executable `ffmpeg` sebagai dependency utama runtime.

Dengan native library integration, renderer dapat:

- mengirim frame langsung dari memory;
- menghindari temporary image sequence untuk setiap frame;
- mengontrol lifecycle encoder dan muxer dari runtime;
- menggabungkan rendering procedural dengan encoding secara langsung.

Pendekatan ini membuat VScript lebih dekat ke language/runtime multimedia daripada wrapper command-line sederhana.

## Platform dan binary

Pola asset release yang direncanakan:

```plaintext
Linux:
vscript-linux-amd64
vscript-linux-arm64
vscript-linux-armv7
vscript-linux-386

Android / Termux:
vscript-android-arm64
vscript-android-armv7
vscript-android-amd64
vscript-android-386
```

Nama di atas merupakan pola packaging. Target yang belum dibangun tidak boleh dianggap tersedia.

## Build untuk Termux

Contoh:

```plaintext
pkg install golang clang ffmpeg

chmod +x build.sh
./build.sh
```

Binary kemudian berada di:

```plaintext
dist/vscript
```

Untuk instalasi manual:

```plaintext
cp dist/vscript "$PREFIX/bin/vscript"
chmod +x "$PREFIX/bin/vscript"
```

Setelah itu:

```plaintext
vscript examples/hello.vs
```

## Security dan sandboxing

VScript menyediakan capability filesystem dan network pada runtime. Jangan menjalankan script VScript yang tidak dipercaya pada environment yang memberi akses penuh terhadap file system atau jaringan.

Untuk automation atau deployment, gunakan environment terisolasi dengan permission minimum yang diperlukan.

## Troubleshooting

### `libavcodec` atau library FFmpeg tidak ditemukan

Pastikan library native tersedia.

Pada Termux, periksa:

```plaintext
ls "$PREFIX/lib" | grep 'libavcodec'
ls "$PREFIX/lib" | grep 'libavformat'
ls "$PREFIX/lib" | grep 'libavutil'
```

Library dapat memiliki nama versioned seperti:

```plaintext
libavcodec.so.61
```

### Font gagal dibuka

Pastikan file TTF/OTF benar-benar ada dan dapat dibaca oleh proses VScript.

Pada Android/Termux, contoh path sistem adalah:

```plaintext
/system/fonts/Roboto-Regular.ttf
```

tetapi ketersediaannya dapat berbeda antar perangkat/ROM.

### `text()` tanpa font terlihat terlalu besar pada versi lama

Itu merupakan bug renderer built-in yang diperbaiki pada 0.2.3. `size` sekarang diperlakukan sebagai tinggi glyph kira-kira, bukan ukuran cell 5×7 mentah.

### `text()` dengan font berbeda ukuran

Ukuran pada pemanggilan `text()` sekarang disinkronkan ke FreeType. Cache glyph akan dikosongkan saat ukuran font berubah.

### Render terlalu lambat

Rendering procedural adalah pekerjaan CPU-intensive, terutama pada resolusi tinggi dan custom filter per-pixel.

Untuk pengujian awal gunakan:

```plaintext
640x360
```

atau

```plaintext
1280x720
```

dengan FPS lebih rendah sebelum berpindah ke 1080p/30 FPS atau lebih tinggi.

### Output terasa terdistorsi ketika memakai `design`

Pastikan aspect ratio design canvas sama dengan output.

```plaintext
16:9 → 16:9
4:3  → 4:3
1:1  → 1:1
```

`design` tidak melakukan letterboxing otomatis; coordinate X dan Y dipetakan ke output masing-masing.

### File MP4 kecil

Ukuran file tidak selalu menunjukkan bahwa render gagal. Content yang statis atau sederhana dapat dikompresi sangat efisien.

Periksa stream dan metadata menggunakan tool pengembangan seperti `ffprobe` pada mesin developer:

```plaintext
ffprobe output/video.mp4
```

## Regression testing

Jalankan seluruh test Go:

```plaintext
go test ./...
```

Untuk pemeriksaan tambahan:

```plaintext
go vet ./...
```

Regression test utama mencakup parser `else if`, `design`, timeline, font renderer, built-in text sizing, rounded rectangle, equality, color parsing, control-flow error handling, dan multiple video block validation.

Contoh text renderer:

```plaintext
vscript examples/text.vs
```

## Contoh project

Direktori `examples/` berisi contoh language dan multimedia:

```plaintext
examples/
├── hello.vs
├── filters.vs
├── font_timeline.vs
├── text.vs
└── patch2_demo.vs
```

Contoh `text.vs` secara khusus digunakan sebagai regression example untuk membandingkan built-in 5×7 renderer dengan FreeType font renderer.

## Dokumentasi web

Target dokumentasi web VScript:

```plaintext
https://vscript-docts.vercel.app/
```

Contoh video dan referensi API dapat ditempatkan di situs dokumentasi ketika deployment aktif.

## Repository

Source code utama:

```plaintext
https://github.com/reyhanaarch64/vscript-project
```

Repository ini adalah project **open source** untuk pengembangan runtime, bahasa, renderer, native bridge, contoh, dan dokumentasi.

## Kontribusi

Issue, pull request, contoh script, regression test, dokumentasi, dan improvement renderer dapat dikembangkan secara terbuka.

Sebelum membuat perubahan syntax/API besar, jelaskan tujuan perubahan agar behavior runtime dan dokumentasi dapat diperbarui secara konsisten.

Untuk kontribusi code:

```plaintext
git clone https://github.com/reyhanaarch64/vscript-project
cd vscript-project

go test ./...
go vet ./...
./build.sh
```

Kemudian buat branch:

```plaintext
git checkout -b feature/nama-fitur
```

Gunakan commit message yang jelas dan sertakan regression test ketika memperkenalkan perubahan parser, runtime, timeline, renderer, atau native bridge.

## Filosofi project

VScript dibuat dengan ide sederhana:

> **video adalah program.**

Scene dapat dihitung, dianimasikan, diubah oleh kondisi, dibangun dari matematika, mengambil data eksternal, dan dirender ulang dari source code yang sama.

Tujuan jangka panjangnya adalah membuat VScript tetap nyaman untuk programming biasa, tetapi memiliki multimedia/video sebagai kemampuan native kelas satu.

## Roadmap awal

Roadmap bersifat terbuka dan dapat berubah.

- lebih banyak native codec/container;
- audio timeline dan mixing yang lebih lengkap;
- transform/keyframe yang lebih kaya;
- bezier/path animation;
- shader/procedural graphics yang lebih dalam;
- compositing dan layer graph yang lebih lengkap;
- sistem asset/resource yang lebih portable;
- cross-platform release binary yang lebih luas;
- Android-native runtime dan tooling;
- package/module ecosystem;
- dokumentasi API yang dihasilkan dari source specification.

## License

VScript dirilis dengan lisensi [MIT](LICENSE).

VScript tidak membundel FFmpeg maupun FreeType. Keduanya dimuat dari environment host saat runtime dan mengikuti lisensi masing-masing. Build FFmpeg pada mesin target dapat memiliki ketentuan lisensi berbeda tergantung opsi dan komponen yang digunakan.

---

Dibuat untuk eksperimen bahasa pemrograman, procedural graphics, dan video generation.
