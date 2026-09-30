# rencana pengembangan vscript

## tujuan

vscript saat ini adalah bahasa pemrograman scripting dengan fokus utama pada procedural dan generative video. rencana ini bertujuan memperluas vscript supaya bisa dipakai juga untuk:

- scripting umum;
- automation;
- cli tool;
- data processing;
- api client;
- backend ringan;
- multimedia dan rendering video.

prioritas utama adalah memperkuat fondasi bahasa terlebih dahulu sebelum menambah fitur besar seperti class, package manager, atau concurrency.

## status fitur saat ini

fitur yang sudah tersedia:

- variable dengan `let`;
- string, number, boolean, array, dan object;
- function dan return;
- conditional `if`, `else if`, dan `else`;
- loop `for` range dan `while`;
- `break` dan `continue`;
- operator aritmatika, perbandingan, equality, dan boolean;
- builtin `print()`;
- json encode melalui `json(value)`;
- filesystem api;
- http client melalui `HttpReq()`;
- download media melalui `getMedia()`;
- video block dan export video;
- primitive rendering seperti rectangle, circle, line, polygon, text, dan image;
- font freetype;
- timeline dan keyframe;
- filter bawaan dan custom filter;
- audio track dan native remux;
- native ffmpeg bridge;
- error handling runtime dasar;
- unit test engine menggunakan go.

fitur yang belum tersedia atau masih perlu diperkuat:

- json decode;
- http server;
- database;
- module dan import;
- package manager;
- try/catch/throw;
- null value resmi;
- string methods;
- array methods;
- object utilities;
- operator assignment tambahan;
- ternary dan null fallback;
- loop langsung untuk array dan object;
- aturan scope yang lebih jelas;
- default parameter dan variadic function;
- anonymous function yang lebih nyaman;
- map, filter, dan reduce;
- class dan oop;
- enum dan struct;
- type system atau type checking;
- debugger;
- repl;
- standard library modular;
- test runner dari sisi bahasa;
- formatter dan linter;
- konfigurasi project;
- async dan concurrency;
- sandbox atau permission system.

## fase 1: json dan data interchange

### target

menjadikan vscript mampu memakai response api secara praktis.

### pekerjaan

- tambahkan `jsonDecode(string)` untuk mengubah json string menjadi object atau array;
- tambahkan error yang jelas untuk json tidak valid;
- pastikan hasil decode mendukung nested object dan array;
- dukung akses property dan index pada hasil decode;
- tambahkan validasi tipe dasar jika diperlukan;
- tambahkan test untuk object, array, string, number, boolean, dan null;
- dokumentasikan perbedaan `json(value)` sebagai encode dan `jsonDecode(value)` sebagai decode.

### api yang diharapkan

```plaintext
let response = HttpReq("GET", "https://example.com/api/data");
let data = jsonDecode(response.body);

if data.ok {
    print(data.name);
}
```

## fase 2: error handling bahasa

### target

membuat error dari http, filesystem, json, media, dan database bisa ditangani oleh script.

### pekerjaan

- tambahkan `try`;
- tambahkan `catch`;
- tambahkan `throw`;
- buat object error dengan property seperti `message`, `type`, dan `stack` jika memungkinkan;
- bedakan error syntax, runtime, network, filesystem, dan native;
- pastikan error memiliki nama file, nomor baris, dan kolom;
- pastikan error di dalam function tetap membawa konteks pemanggilan;
- tambahkan regression test untuk nested try/catch dan error yang tidak ditangani.

### api yang diharapkan

```plaintext
try {
    let response = HttpReq("GET", url);
    let data = jsonDecode(response.body);
} catch err {
    print("request gagal: " + err.message);
}
```

## fase 3: module dan import

### target

memecah script besar menjadi beberapa file yang bisa dipakai ulang.

### pekerjaan

- tambahkan `import` untuk file lokal;
- tentukan format export function, value, dan object;
- dukung path relatif dari file yang sedang dijalankan;
- cegah module yang sama dimuat berulang kali tanpa perlu;
- deteksi circular dependency;
- tentukan scope module supaya variable internal tidak bocor;
- tambahkan module standard library;
- tambahkan test untuk import satu file, nested import, dan circular import.

### api yang diharapkan

```plaintext
import { add, formatUser } from "./utils.vs";

let result = add(10, 20);
print(formatUser(result));
```

## fase 4: string api

### target

mempermudah pemrosesan teks, response api, path, dan input user.

### pekerjaan

tambahkan method atau builtin untuk:

- `toUpper`;
- `toLower`;
- `trim`;
- `contains`;
- `startsWith`;
- `endsWith`;
- `split`;
- `replace`;
- `substring` atau `slice`;
- `charAt`;
- `indexOf`;
- `join` untuk array;
- operasi yang aman untuk unicode.

### contoh

```plaintext
let raw = "  halo reyhan  ";
let text = raw.trim().toUpper();
print(text);
```

## fase 5: array dan object api

### target

membuat pengolahan data lebih ringkas tanpa selalu memakai loop manual.

### pekerjaan array

- `push`;
- `pop`;
- `shift`;
- `unshift`;
- `removeAt`;
- `contains`;
- `indexOf`;
- `slice`;
- `join`;
- `sort`;
- `reverse`;
- `map`;
- `filter`;
- `reduce`.

### pekerjaan object

- `keys`;
- `values`;
- `entries`;
- `hasKey`;
- `deleteKey`;
- object merge;
- object clone;
- konversi object ke array entry.

### contoh

```plaintext
let items = [1, 2, 3, 4];
let doubled = items.map(function(value) {
    return value * 2;
});

let user = {name: "reyhan", hobby: "ngoding"};
print(keys(user));
```

## fase 6: operator dan control flow tambahan

### target

membuat syntax sehari-hari lebih nyaman.

### pekerjaan

- `+=`;
- `-=`;
- `*=`;
- `/=`;
- `%=`;
- `++`;
- `--`;
- ternary operator;
- null fallback operator;
- loop langsung untuk array;
- loop key-value untuk object;
- validasi range kosong dan range turun;
- dokumentasikan perilaku assignment pada scope parent.

### contoh

```plaintext
count += 1;
let label = active ? "online" : "offline";
let name = user.name ?? "guest";

for item in items {
    print(item);
}
```

## fase 7: scope dan function yang lebih lengkap

### target

membuat perilaku function dan variable konsisten saat project membesar.

### pekerjaan

- tetapkan aturan scope untuk block, loop, dan function;
- pastikan closure bekerja konsisten;
- dokumentasikan apakah assignment dapat mengubah variable parent;
- tambahkan default parameter;
- tambahkan variadic parameter;
- dukung anonymous function dengan syntax resmi;
- dukung callback untuk map, filter, reduce, dan server route;
- tambahkan validasi jumlah argument jika diperlukan.

### contoh

```plaintext
function greet(name = "guest") {
    return "halo " + name;
}

let values = function(a, b) {
    return a + b;
};
```

## fase 8: http server

### target

memungkinkan vscript menjadi backend ringan tanpa runtime tambahan.

### pekerjaan

- buat builtin `httpServer(port)`;
- dukung route `get`, `post`, `put`, `patch`, dan `delete`;
- buat object request dengan method, path, query, headers, body, dan params;
- buat object response dengan status, headers, dan body;
- dukung response object atau string;
- tambahkan middleware sederhana;
- dukung json response;
- tambahkan graceful shutdown;
- tambahkan batas ukuran body;
- tambahkan timeout request;
- tambahkan logging request;
- tentukan model concurrency yang aman;
- tambahkan contoh api sederhana.

### api yang diharapkan

```plaintext
let server = httpServer(8080);

server.get("/api/hello", function(req) {
    return {
        status: 200,
        body: json({message: "halo dari vscript"}),
        headers: {"content-type": "application/json"}
    };
});

server.listen();
```

## fase 9: database

### target

menyediakan penyimpanan data untuk backend kecil dan automation.

### urutan implementasi

1. sqlite;
2. mysql atau mariadb;
3. postgresql;
4. redis atau key-value storage jika memang dibutuhkan.

### pekerjaan sqlite

- builtin `sqlite(path)`;
- `exec(sql, params)`;
- `query(sql, params)`;
- `one(sql, params)`;
- transaction;
- close connection;
- parameterized query untuk mencegah sql injection;
- konversi row ke object;
- error database yang informatif;
- test database temporary.

### api yang diharapkan

```plaintext
let db = sqlite("data.db");
db.exec("create table users (id integer, name text)");
db.exec("insert into users values (?, ?)", [1, "reyhan"]);

let users = db.query("select * from users");
print(users);
db.close();
```

## fase 10: repl dan cli

### target

mempermudah belajar, eksperimen, dan debugging.

### pekerjaan

- tambahkan command `vscript repl`;
- dukung multiline block;
- tampilkan hasil expression;
- tampilkan error tanpa mematikan repl;
- tambahkan history jika environment mendukung;
- tambahkan command `help`, `clear`, dan `exit`;
- rapikan command cli seperti `run`, `paths`, dan `--version`;
- tambahkan opsi working directory;
- tambahkan opsi environment variable.

### contoh

```plaintext
vscript repl

> let x = 10;
> print(x * 2);
20
```

## fase 11: standard library modular

### target

mengurangi terlalu banyak builtin global dan membuat api lebih teratur.

### module yang direncanakan

- `std.math`;
- `std.fs`;
- `std.http`;
- `std.json`;
- `std.time`;
- `std.path`;
- `std.media`;
- `std.video`;
- `std.crypto` jika nanti dibutuhkan.

### catatan

builtin lama sebaiknya tetap dipertahankan sementara untuk compatibility. module baru dapat menjadi api resmi jangka panjang.

## fase 12: testing dari sisi vscript

### target

pengguna bisa menguji script tanpa harus menulis test dalam go.

### pekerjaan

- tambahkan `assert`;
- tambahkan block `test`;
- tambahkan test runner `vscript test`;
- tampilkan jumlah test passed dan failed;
- dukung test file dengan pola tertentu;
- dukung fixture sederhana;
- tambahkan mode verbose.

### contoh

```plaintext
test "addition" {
    assert(add(2, 3) == 5);
}
```

## fase 13: source location, debugger, dan diagnostics

### target

membuat debugging lebih mudah daripada hanya menerima pesan runtime error.

### pekerjaan

- simpan line dan column pada token;
- teruskan source location ke ast;
- tampilkan file, baris, dan kolom saat error;
- tampilkan potongan source yang bermasalah;
- tampilkan stack trace function;
- tambahkan mode debug;
- tambahkan inspect variable;
- rencanakan breakpoint jika interpreter sudah mendukungnya.

### contoh error yang diharapkan

```plaintext
main.vs:14:9: variabel "username" belum didefinisikan
```

## fase 14: formatter dan linter

### target

menjaga script tetap rapi dan menemukan masalah sebelum runtime.

### pekerjaan formatter

- tambahkan `vscript fmt file.vs`;
- dukung format in-place dan stdout;
- tentukan indentasi resmi;
- pertahankan string dan komentar;
- tambahkan format untuk object dan multiline call.

### pekerjaan linter

- variable tidak digunakan;
- import tidak digunakan;
- shadowing variable;
- unreachable code;
- jumlah argument salah jika bisa dideteksi;
- penggunaan api deprecated;
- path media yang berpotensi tidak ditemukan.

## fase 15: konfigurasi project dan package manager

### target

mengelola project vscript yang terdiri dari banyak file dan dependency.

### pekerjaan

- buat manifest `vscript.toml`;
- simpan nama, versi, entry point, dan target project;
- definisikan dependency;
- buat command `vscript init`;
- buat command `vscript install`;
- buat command `vscript run`;
- buat command `vscript build`;
- buat lock file untuk versi dependency;
- tentukan registry atau sumber dependency;
- validasi keamanan package sebelum eksekusi.

### contoh manifest

```plaintext
name = "my-project"
version = "1.0.0"
entry = "main.vs"
```

## fase 16: type system opsional

### target

memberi pilihan type checking tanpa menghilangkan fleksibilitas dynamic typing.

### pekerjaan

- type annotation untuk variable dan parameter;
- type annotation untuk return value;
- type checking mode opsional;
- type dasar: string, number, boolean, array, object, function, media;
- union type;
- validasi runtime untuk api publik;
- pesan error tipe yang jelas.

### contoh

```plaintext
function add(a: number, b: number): number {
    return a + b;
}
```

fitur ini sebaiknya tidak menjadi prioritas sebelum module, error handling, dan standard library stabil.

## fase 17: class, enum, dan struct

### target

menyediakan abstraksi tambahan untuk project besar.

### pekerjaan class

- `class`;
- constructor;
- method;
- property;
- `this`;
- inheritance jika benar-benar diperlukan;
- visibility atau convention private;
- error yang jelas untuk property yang tidak ada.

### pekerjaan enum dan struct

- enum untuk nilai konstan terbatas;
- struct untuk data terstruktur;
- validasi field;
- konversi struct ke object dan json.

fitur ini bukan prioritas awal karena object dan function sudah cukup untuk banyak script kecil.

## fase 18: async dan concurrency

### target

mendukung server dan pekerjaan network-heavy tanpa memblokir seluruh runtime.

### pekerjaan

- tentukan model concurrency;
- async request;
- task atau future;
- `await`;
- channel atau queue jika memang dibutuhkan;
- pembatasan jumlah worker;
- cancellation;
- timeout;
- error propagation;
- pastikan renderer dan environment tidak mengalami race condition.

fitur ini sebaiknya dibuat setelah http server dan model scope sudah stabil.

## fase 19: security dan sandbox

### target

menjalankan script dengan permission minimum, terutama untuk script dari sumber tidak dipercaya.

### pekerjaan

- permission filesystem read;
- permission filesystem write;
- permission network;
- permission subprocess jika nanti ditambahkan;
- batas ukuran download;
- batas waktu request;
- batas memori dan durasi render jika memungkinkan;
- path allowlist;
- mode sandbox;
- dokumentasi risiko menjalankan script pihak lain.

### contoh cli

```plaintext
vscript run app.vs \
    --allow-net \
    --allow-read=assets \
    --allow-write=output
```

## fase 20: compatibility dan release

### target

menjaga perubahan api tidak merusak script lama.

### pekerjaan

- gunakan semantic versioning;
- catat perubahan di `changelog.md`;
- tandai api experimental;
- buat compatibility test untuk setiap release;
- dokumentasikan deprecation;
- sediakan migration guide;
- uji di linux dan termux;
- uji arsitektur target yang didukung;
- uji native library version yang berbeda;
- siapkan binary release setelah pipeline build stabil.

## prioritas implementasi ringkas

urutan paling masuk akal:

1. json decode;
2. source location dan stack trace dasar;
3. try/catch/throw;
4. string methods;
5. array methods dan object utilities;
6. module/import;
7. repl;
8. http server;
9. sqlite;
10. standard library modular;
11. test runner vscript;
12. formatter dan linter;
13. project manifest;
14. permission/sandbox;
15. async/concurrency;
16. package manager;
17. optional type checking;
18. class, enum, dan struct.

## prinsip pengembangan

- jangan merusak syntax vscript yang sudah dipakai tanpa migration path;
- prioritaskan error message yang jelas daripada fitur syntax yang terlalu banyak;
- semua builtin baru harus punya test;
- api multimedia tetap menjadi kekuatan utama vscript;
- fitur backend harus tetap ringan dan tidak menghilangkan kemampuan rendering;
- hindari menambah dependency native jika fitur bisa dibuat dengan standard library go;
- gunakan parameterized query untuk database;
- batasi akses filesystem dan network pada mode sandbox;
- dokumentasikan semua syntax baru di `readme.md`;
- setiap fitur besar perlu contoh `.vs` di directory `examples/`;
- setiap perubahan release perlu dicatat di `changelog.md`.

## kriteria vscript siap menjadi bahasa general-purpose ringan

vscript bisa dianggap siap untuk scripting umum dan backend ringan setelah memiliki minimal:

- json encode dan decode;
- module/import;
- error handling try/catch/throw;
- string, array, dan object api;
- http client dan http server;
- sqlite;
- repl;
- source location dan stack trace;
- test runner;
- formatter;
- permission dasar;
- dokumentasi standard library;
- compatibility test untuk release.
