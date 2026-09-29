// audio harus berupa stream yang kompatibel dengan container output.
// MP3 dapat langsung diremux ke MP4 tanpa executable ffmpeg.

let music = audio("assets/demo-tone.mp3");
video 640 360 24fps 6s {
    background("#0b111a");
    circle(320, 180, 90 + sin(t * 8) * 12, "#245b90");
    text("native audio", 320, 175, 18, "#ffffff");
}
export video "output/audio-export.mp4" codec "libx264" bitrate 3000000 audio music;
