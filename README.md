# VScript

VScript adalah bahasa pemrograman umum dengan fokus kuat pada **procedural dan generative video**.

Dengan VScript, video tidak dibuat dengan menyusun ratusan file gambar secara manual. Scene, bentuk, teks, media, animasi, filter, audio, dan output video ditulis sebagai kode lalu dirender oleh runtime VScript.

VScript dibangun dengan **Go** dan menggunakan library native multimedia secara langsung melalui bridge native, terutama **FFmpeg libraries** dan **FreeType**. VScript tidak perlu menjalankan executable `ffmpeg` untuk proses export video.

> Status: **open source / early development**
>
> API masih dapat berubah pada versi awal, termasuk syntax multimedia dan detail renderer.

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

## Fitur utama

- General-purpose language basics: variable, constant, string, number, boolean, array, object, function, return, conditional, loop, dan error handling.
- Video rendering dengan resolusi output yang dapat ditentukan.
- Logical/design resolution menggunakan `design W H` agar layout dapat dibuat sekali lalu diskalakan ke berbagai output.
- Primitive graphics seperti rectangle, circle, line, polygon, path, dan text.
- Image dan media loading.
- Font TTF/OTF melalui FreeType.
- Timeline dan keyframe dengan interpolasi nilai.
- Transform, crop, resize, rotation, dan filter.
- Custom/procedural filtering melalui function.
- Audio track dan mux ke video.
- Filesystem API.
- HTTP/media download API.
- Native multimedia pipeline melalui FFmpeg libraries.
- Dynamic library loading sehingga runtime dapat mencari library pada environment seperti Termux maupun Linux.

## Quick start

### Instalasi otomatis

Target installer publik VScript:

```plaintext
wget https://vscript-docts.vercel.app/install.sh && bash install.sh
```

Installer dirancang untuk:

1. mendeteksi platform dan arsitektur CPU;
2. memilih binary yang sesuai;
3. mengunduh binary dari direktori `assets/`;
4. memasangnya ke `PREFIX/bin` pada Termux atau prefix/bin pada environment yang sesuai;
5. membuat perintah `vscript` tersedia langsung di shell.

Setelah terpasang, penggunaan menjadi:

```plaintext
vscript video.vs
```

> Catatan: URL installer di atas baru dapat digunakan setelah situs dokumentasi dideploy pada domain tersebut dan asset binary yang sesuai tersedia.

### Build dari source

VScript menggunakan Go dengan CGO untuk bridge native.

Pada Termux, pastikan Go, Clang, FFmpeg development libraries, dan dependency native yang diperlukan sudah terpasang.

Contoh build:

```plaintext
chmod +x build.sh
./build.sh
```

Binary hasil build biasanya berada di:

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

## Tutorial: membuat video 1 menit

Bagian ini menunjukkan pola umum untuk membuat video procedural sekitar satu menit.

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

`design 640 360` berarti scene menggunakan coordinate space 640×360, sedangkan framebuffer akhir adalah 1920×1080. Renderer melakukan scaling ke output resolution.

Ini berguna ketika source yang sama perlu dirender ke beberapa ukuran:

```plaintext
1280x720
1920x1080
1080x1920
640x360
```

### 2. Buat animasi berdasarkan waktu

Di dalam video block, runtime menyediakan nilai waktu/frame yang dapat digunakan untuk menghitung posisi.

```plaintext
let x = 320 + sin(t * 2) * 80;
let y = 180;

circle(x, y, 32, "#66ccff");
```

Karena `t` berubah pada setiap frame, object bergerak secara otomatis.

### 3. Gunakan conditional

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

`else if` dan `else` adalah bagian dari grammar conditional, bukan workaround runtime.

### 4. Gunakan function

Pisahkan bagian scene menjadi function agar script tetap mudah dirawat.

```plaintext
function drawTitle(title) {
    text(title, 320, 100, 42, "#ffffff");
}

video 1280 720 30 60 design 640 360 {
    background("#08101c");
    drawTitle("VScript");
}
```

### 5. Gunakan timeline dan keyframe

Timeline menyimpan nilai animasi sepanjang waktu.

```plaintext
let tl = timeline(60);

tl.keyframe("x", 0, -100);
tl.keyframe("x", 5, 100);
tl.keyframe("x", 20, 100);
tl.keyframe("x", 25, 700);

video 1280 720 30 60 design 640 360 {
    let x = tl.value("x", t);
    circle(x, 180, 32, "#66ccff");
}
```

Konsep ini berguna untuk animation yang memiliki titik-titik waktu tertentu tanpa harus menulis rumus untuk setiap perubahan.

### 6. Membaca font dari file

Font eksternal dapat dimuat sebagai resource.

```plaintext
let titleFont = font("assets/Roboto-Bold.ttf", 64);

video 1280 720 30 10 design 640 360 {
    background("#07111f");
    text("VScript", 320, 180, 42, "#ffffff", titleFont);
}
```

Format font yang ditujukan untuk API ini adalah TTF/OTF yang dapat dibaca FreeType.

Path dapat berupa path file biasa, termasuk path yang dapat diakses dari Termux.

Untuk environment Android/Termux, penggunaan file font langsung biasanya lebih portable daripada mengandalkan nama font sistem. Dukungan `systemFont()` dapat ditambahkan sebagai API convenience pada versi mendatang.

### 7. Menambahkan audio

Audio dapat dimuat sebagai media resource dan dimasukkan ke output saat export.

```plaintext
let music = audio("assets/music.mp3");

video 1280 720 30 60 design 640 360 {
    background("#060b14");
}

export video "output/video.mp4"
    codec "libx264"
    bitrate 8000000
    audio music;
```

Runtime menggunakan native multimedia pipeline. VScript tidak perlu menjalankan command `ffmpeg` sebagai child process.

Format audio yang didukung untuk workflow tertentu mengikuti encoder/container yang tersedia di build FFmpeg pada mesin target.

## Syntax dasar

### Variable

```plaintext
let username = "Rehan";
let count = 10;
let active = true;
```

### Constant

```plaintext
const FPS = 30;
```

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
video 1920 1080 30 60 design 640 360 {
    // scene menggunakan coordinate 640x360
}
```

### `background(color)`

Mengisi background frame.

```plaintext
background("#08101c");
```

### `rect(x, y, width, height, color, radius)`

Rectangle atau rounded rectangle.

```plaintext
rect(40, 40, 240, 120, "#11243a", 20);
```

### `circle(x, y, radius, color)`

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

Polygon cocok untuk bentuk procedural yang tidak terbatas pada primitive standar.

### `text(text, x, y, size, color, font)`

```plaintext
text("Hello", 320, 180, 32, "#ffffff", titleFont);
```

### `font(path, size)`

Memuat font file menggunakan native font rasterization.

```plaintext
let font64 = font("assets/Roboto-Bold.ttf", 64);
```

### `cropAspect`

Crop berdasarkan aspect ratio.

```plaintext
cropAspect("16:9");
```

Aspect ratio dapat digunakan sebagai bagian dari preprocessing/render workflow sesuai resource yang sedang diproses.

## Timeline API

### `timeline(duration)`

Membuat timeline.

```plaintext
let tl = timeline(60);
```

### `keyframe(name, time, value)`

Menambahkan keyframe.

```plaintext
tl.keyframe("opacity", 0, 0);
tl.keyframe("opacity", 1, 1);
tl.keyframe("opacity", 3, 1);
tl.keyframe("opacity", 4, 0);
```

### `value(name, time)`

Mengambil nilai yang telah diinterpolasi.

```plaintext
let opacity = tl.value("opacity", t);
```

Nilai yang dapat dianimasikan tidak terbatas pada satu angka sederhana selama tipe datanya didukung oleh interpolator runtime.

## Utility matematika

Beberapa helper yang berguna untuk animasi:

```plaintext
let a = lerp(0, 100, 0.5);
let b = clamp(speed, 0, 1);
let c = easeIn(0.5);
let d = easeOut(0.5);
let e = easeInOut(0.5);
```

Animasi procedural juga dapat dibuat langsung dengan fungsi matematika seperti `sin()` dan `cos()`.

## Filter

VScript mendukung filter bawaan dan custom filter.

Contoh filter bawaan:

```plaintext
filter("contrast", 1.08);
filter("brightness", 4);
filter("vignette", 0.15);
filter("pixelate", 4);
```

Filter dapat dipakai secara conditional:

```plaintext
if t > 50 {
    filter("contrast", 1.12);
    filter("brightness", 5);
} else {
    filter("contrast", 1.04);
}
```

### Custom filter

Konsep custom filter memungkinkan efek dibentuk dari kode, bukan hanya memilih preset.

```plaintext
function warm(x, y, r, g, b, a) {
    return [
        clamp(r + 10, 0, 255),
        clamp(g + 3, 0, 255),
        b,
        a
    ];
}

customFilter("warm", warm);
```

Signature function harus mengikuti bentuk yang dipahami runtime saat ini: `x, y, r, g, b, a`.

## Media dan image

Resource media dapat dimuat dari local file atau source yang didukung runtime.

```plaintext
let logo = loadMedia("assets/logo.png");
```

Remote media dapat diambil dengan media downloader:

```plaintext
let clip = getMedia("https://example.com/video.mp4");
```

Untuk media internet, format akhir tetap bergantung pada parser/decoder yang tersedia di native stack target.

## Filesystem API

API filesystem digunakan untuk script yang perlu membaca atau menulis file.

```plaintext
let content = readFile("data.txt");
writeFile("output.txt", content);
```

Utility yang tersedia pada runtime mencakup pola berikut:

```plaintext
exists(path)
mkdir(path)
deleteFile(path)
copyFile(source, destination)
moveFile(source, destination)
```

## HTTP API

HTTP request dapat digunakan untuk mengambil data dari endpoint web.

Contoh konsep request:

```plaintext
let response = HttpReq("https://example.com/api/data", {
    method: "GET",
    headers: {
        "Accept": "application/json"
    }
});

print(response.status);
print(response.body);
```

VScript juga menyediakan pendekatan media-oriented melalui `getMedia()` untuk source yang memang ditujukan menjadi resource multimedia.

## CLI

Pola penggunaan dasar:

```plaintext
vscript file.vs
```

Melihat versi:

```plaintext
vscript --version
```

Build source:

```plaintext
./build.sh
```

> Detail flag CLI dapat berkembang seiring runtime bertambah. Referensi CLI yang mengikat adalah implementasi pada `cmd/vscript` pada source tree.

## Struktur project

Struktur project runtime kira-kira:

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
└── README.md
```

Nama file dapat berubah ketika engine berkembang, tetapi pemisahan utamanya tetap:

- `cmd/` — entry point CLI.
- `engine/` — language engine, runtime, renderer, dan native bridge.
- `examples/` — contoh script.
- `assets/` — resource runtime/release.
- `dist/` — binary hasil build.
- `output/` — hasil render.

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
scene graph + timeline
    ↓
renderer
    ↓
framebuffer
    ↓
libswscale / native pixel processing
    ↓
libavcodec
    ↓
libavformat
    ↓
output container
```

Audio mengikuti pipeline native yang sama melalui codec/resampler/muxer yang tersedia pada build.

Library yang digunakan atau dituju oleh native multimedia layer mencakup:

```plaintext
libavcodec
libavformat
libavutil
libswscale
libswresample
```

Untuk font:

```plaintext
libfreetype
```

Runtime dirancang untuk mencari library dari environment host. Pada Termux, library umumnya dicari dari:

```plaintext
$PREFIX/lib
```

Pada Linux, resolver dapat mencari lokasi library standar host termasuk path multiarch yang tersedia.

## Mengapa tidak cukup menjalankan FFmpeg CLI?

VScript sengaja tidak menjadikan executable `ffmpeg` sebagai dependency utama runtime.

Dengan native library integration, renderer dapat:

- mengirim frame langsung dari memory;
- menghindari temporary image sequence untuk setiap frame;
- mengontrol encoder dan muxer dari runtime;
- menggabungkan rendering procedural dengan encoding secara langsung.

Pendekatan ini juga membuat VScript lebih cocok sebagai language/runtime mandiri dibanding wrapper command-line sederhana.

## Platform dan binary

Release binary ditempatkan di `assets/` dan dipilih oleh `install.sh` berdasarkan platform/arsitektur.

Pola nama yang digunakan:

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

Binary harus benar-benar dibangun untuk target OS tersebut. Binary Linux tidak boleh dianggap sebagai pengganti binary Android hanya karena arsitektur CPU-nya sama.

Pada release tertentu mungkin hanya sebagian target yang tersedia.

## Build untuk Termux

Contoh target Termux ARM64:

```plaintext
pkg install golang clang ffmpeg

chmod +x build.sh
./build.sh
```

Runtime kemudian dapat dijalankan:

```plaintext
./dist/vscript examples/hello.vs
```

Untuk instalasi manual ke prefix:

```plaintext
cp dist/vscript "$PREFIX/bin/vscript"
chmod +x "$PREFIX/bin/vscript"
```

Setelah itu:

```plaintext
vscript examples/hello.vs
```

## Security dan sandboxing

VScript adalah bahasa general-purpose, sehingga script yang dieksekusi dapat menggunakan capability runtime yang memang tersedia, misalnya filesystem atau network API.

Jangan menjalankan script VScript yang tidak dipercaya pada environment yang memberikan akses filesystem/network penuh.

Untuk deployment atau automation, pertimbangkan menjalankan runtime pada environment terisolasi dengan permission minimum yang diperlukan.

## Troubleshooting

### `libavcodec` atau library FFmpeg tidak ditemukan

Pastikan paket/library native tersedia dan runtime mengetahui lokasi library tersebut.

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

bukan hanya `libavcodec.so`.

### `vscript` bisa dijalankan tetapi output MP4 belum terlihat

Encoding video tidak selalu selesai seketika. Periksa proses:

```plaintext
ps -A | grep vscript
```

Dan periksa direktori output:

```plaintext
ls -lh output/
```

### Render terlalu lambat

Rendering procedural adalah pekerjaan CPU-intensive, terutama pada resolusi tinggi.

Untuk pengujian awal gunakan:

```plaintext
640x360
```

atau

```plaintext
1280x720
```

dengan FPS lebih rendah sebelum berpindah ke 1080p/30 FPS atau lebih tinggi.

### File MP4 kecil

Ukuran file tidak selalu menunjukkan bahwa render gagal. Content yang statis atau sederhana dapat dikompresi sangat efisien.

Periksa stream dan metadata menggunakan tool seperti `ffprobe` pada mesin pengembangan:

```plaintext
ffprobe output/video.mp4
```

## Contoh project

Direktori `examples/` digunakan untuk contoh language dan multimedia.

Contoh yang ideal untuk dipelajari bertahap:

```plaintext
examples/
├── hello.vs
├── filters.vs
├── timeline.vs
├── font.vs
└── rehan_intro_1min.vs
```

Script contoh satu menit dapat menggabungkan:

- logical design resolution;
- primitive graphics;
- font;
- timeline;
- keyframe;
- conditional;
- procedural movement;
- filter;
- audio;
- H.264 export.

## Dokumentasi web

Dokumentasi web VScript menyediakan tutorial dan API reference yang lebih interaktif dibanding README.

Target URL dokumentasi:

```plaintext
https://vscript-docts.vercel.app/
```

Contoh video juga disediakan di halaman dokumentasi.

## Repository

Source code utama:

```plaintext
https://github.com/reyhanaarch64/vscript-project
```

Repository ini adalah project **open source** dan ditujukan agar runtime, bahasa, contoh, dokumentasi, serta tooling dapat dikembangkan secara terbuka.

## Kontribusi

Pull request, issue, contoh script, perbaikan dokumentasi, dan improvement pada runtime sangat diterima.

Sebelum membuat perubahan besar, sebaiknya jelaskan tujuan perubahan pada issue agar desain syntax/API tidak saling bertabrakan.

Untuk kontribusi code:

```plaintext
git clone https://github.com/reyhanaarch64/vscript-project
cd vscript-project

go test ./...

./build.sh
```

Kemudian buat branch:

```plaintext
git checkout -b feature/nama-fitur
```

Gunakan commit message yang jelas dan sertakan contoh/regression test ketika memperkenalkan perubahan pada parser, runtime, atau renderer.

## Filosofi project

VScript dibuat dengan ide sederhana:

> **video adalah program.**

Scene dapat dihitung, dianimasikan, diubah oleh kondisi, dibangun dari matematika, mengambil data eksternal, dan dirender ulang kapan pun dari source code yang sama.

Tujuan jangka panjangnya adalah membuat VScript menjadi bahasa yang tetap nyaman untuk programming biasa, tetapi memiliki multimedia/video sebagai kemampuan native kelas satu.

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
- dokumentasi API yang sepenuhnya dihasilkan dari source specification;
- test suite renderer dan golden-frame regression testing.

## License

VScript dirilis dengan lisensi [MIT](LICENSE).

VScript tidak membundel FFmpeg maupun FreeType. Keduanya dimuat dari environment host saat runtime, jadi tetap mengikuti lisensi masing-masing (FFmpeg bisa LGPL atau GPL tergantung opsi build yang dipakai di mesin kamu, misalnya jika memakai libx264).

---

Dibuat untuk eksperimen bahasa pemrograman, procedural graphics, dan video generation.
